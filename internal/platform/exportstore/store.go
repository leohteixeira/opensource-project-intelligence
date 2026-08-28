// Package exportstore owns durable export Jobs and their checksummed object references.
package exportstore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/leohteixeira/opensource-project-intelligence/internal/access"
	exportartifact "github.com/leohteixeira/opensource-project-intelligence/internal/export"
	"github.com/leohteixeira/opensource-project-intelligence/internal/job"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/database"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/jobstore"
	"github.com/leohteixeira/opensource-project-intelligence/internal/project"
)

const artifactLifetime = 24 * time.Hour

type IDSource interface {
	Next(context.Context) (int64, error)
}

type BlobStore interface {
	Stage(context.Context, string, []byte, string) error
	Promote(context.Context, string, string) error
	Read(context.Context, string) ([]byte, error)
	Delete(context.Context, string) error
}

type Store struct {
	pool  *pgxpool.Pool
	ids   IDSource
	blobs BlobStore
	gen   *exportartifact.Coordinator
	now   func() time.Time
}

type Export struct {
	ID          int64                  `json:"id,string"`
	JobID       int64                  `json:"job_id,string"`
	State       string                 `json:"state"`
	Request     exportartifact.Request `json:"request"`
	Rows        int64                  `json:"row_count"`
	MediaType   string                 `json:"media_type,omitempty"`
	SHA256      string                 `json:"sha256,omitempty"`
	SizeBytes   int64                  `json:"size_bytes,omitempty"`
	DownloadURL string                 `json:"download_url,omitempty"`
	Failure     string                 `json:"failure,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	ExpiresAt   *time.Time             `json:"expires_at,omitempty"`
	objectKey   string
	workspaceID int64
	actorID     int64
}

func New(pool *database.Pool, ids IDSource, blobs BlobStore, concurrency int) (*Store, error) {
	if pool == nil || ids == nil || blobs == nil {
		return nil, errors.New("export database, ID source, and object store are required")
	}
	coordinator, err := exportartifact.NewCoordinator(concurrency, exportartifact.DefaultMaxBytes)
	if err != nil {
		return nil, err
	}
	return &Store{pool: pool.Unwrap(), ids: ids, blobs: blobs, gen: coordinator, now: time.Now}, nil
}

func (s *Store) Create(
	ctx context.Context,
	principal access.Principal,
	request exportartifact.Request,
	requestID string,
) (Export, error) {
	if err := access.Authorize(principal, access.ActionExportWrite); err != nil {
		return Export{}, err
	}
	request = request.Normalize()
	if err := request.Validate(); err != nil {
		return Export{}, err
	}
	requestID = strings.TrimSpace(requestID)
	if requestID == "" || len(requestID) > 255 {
		return Export{}, exportartifact.ErrInvalidRequest
	}
	if err := s.authorizeProjects(ctx, principal.Workspace, request.ProjectIDs); err != nil {
		return Export{}, err
	}
	encodedRequest, _ := json.Marshal(request)
	var existingID int64
	var existingRequest []byte
	err := s.pool.QueryRow(ctx, `SELECT id,request FROM export_requests
		WHERE workspace_id=$1 AND actor_id=$2 AND request_id=$3`, principal.Workspace,
		principal.ActorID, requestID).Scan(&existingID, &existingRequest)
	if err == nil {
		if string(existingRequest) != string(encodedRequest) {
			return Export{}, exportartifact.ErrIdempotencyKey
		}
		return s.load(ctx, existingID, principal.Workspace, principal.ActorID)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Export{}, fmt.Errorf("read export idempotency: %w", err)
	}
	exportID, err := s.ids.Next(ctx)
	if err != nil {
		return Export{}, fmt.Errorf("issue export ID: %w", err)
	}
	jobID, err := s.ids.Next(ctx)
	if err != nil {
		return Export{}, fmt.Errorf("issue export Job ID: %w", err)
	}
	now := s.now().UTC()
	value, err := job.New(jobID, 0, "export", "rows", nil, true, now)
	if err != nil {
		return Export{}, err
	}
	encodedJob, _ := json.Marshal(value)
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Export{}, fmt.Errorf("begin export Job: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	_, err = tx.Exec(ctx, `INSERT INTO jobs
		(id,kind,state,workspace_id,version,progress,checkpoint,cancellable,requested_by,request_id,
		 max_attempts,available_at,created_at,updated_at)
		VALUES ($1,'export','queued',$2,1,$3,'{}',true,$4,$5,3,$6,$6,$6)`,
		jobID, principal.Workspace, mustJSON(value.Progress), principal.ActorID, requestID, now)
	if err != nil {
		return Export{}, fmt.Errorf("insert export Job: %w", err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO export_requests
		(id,workspace_id,actor_id,job_id,request,request_id,created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`, exportID, principal.Workspace, principal.ActorID,
		jobID, encodedRequest, requestID, now)
	if err != nil {
		return Export{}, fmt.Errorf("insert export request: %w", err)
	}
	for _, projectID := range request.ProjectIDs {
		if _, err = tx.Exec(ctx, `INSERT INTO export_request_projects (export_id,project_id)
			VALUES ($1,$2)`, exportID, projectID); err != nil {
			return Export{}, fmt.Errorf("record export project ownership: %w", err)
		}
	}
	eventID, err := s.ids.Next(ctx)
	if err != nil {
		return Export{}, fmt.Errorf("issue export Job event ID: %w", err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO job_events (id,job_id,job_version,representation,occurred_at)
		VALUES ($1,$2,1,$3,$4)`, eventID, jobID, encodedJob, now)
	if err != nil {
		return Export{}, fmt.Errorf("record export Job event: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Export{}, fmt.Errorf("commit export Job: %w", err)
	}
	return Export{ID: exportID, JobID: jobID, State: "queued", Request: request, CreatedAt: now}, nil
}

func (s *Store) Get(ctx context.Context, principal access.Principal, id int64) (Export, error) {
	if err := access.Authorize(principal, access.ActionIntelligenceRead); err != nil {
		return Export{}, err
	}
	value, err := s.load(ctx, id, principal.Workspace, principal.ActorID)
	if err != nil {
		return Export{}, err
	}
	if value.ExpiresAt != nil && !s.now().UTC().Before(*value.ExpiresAt) {
		value.State, value.DownloadURL = "expired", ""
	}
	if value.State == "succeeded" {
		value.DownloadURL = "/api/v1/exports/" + strconv.FormatInt(value.ID, 10) + "/download"
	}
	return value, nil
}

func (s *Store) Download(
	ctx context.Context,
	principal access.Principal,
	id int64,
) (Export, []byte, error) {
	value, err := s.Get(ctx, principal, id)
	if err != nil {
		return Export{}, nil, err
	}
	if value.State == "expired" {
		return Export{}, nil, exportartifact.ErrExpired
	}
	if value.State != "succeeded" || value.objectKey == "" {
		return Export{}, nil, exportartifact.ErrNotReady
	}
	body, err := s.blobs.Read(ctx, value.objectKey)
	if err != nil {
		return Export{}, nil, err
	}
	digest := sha256.Sum256(body)
	if hex.EncodeToString(digest[:]) != value.SHA256 || int64(len(body)) != value.SizeBytes {
		return Export{}, nil, errors.New("export artifact checksum mismatch")
	}
	return value, body, nil
}

// Process generates one leased export. The Runner owns the Job terminal state;
// this method atomically publishes the artifact metadata only after S3 promotion.
func (s *Store) Process(ctx context.Context, lease jobstore.Lease) error {
	var exportID int64
	var encoded []byte
	err := s.pool.QueryRow(ctx, `UPDATE export_requests SET state='running'
		WHERE job_id=$1 AND state IN ('queued','running') RETURNING id,request`, lease.Job.ID).Scan(&exportID, &encoded)
	if errors.Is(err, pgx.ErrNoRows) {
		return exportartifact.ErrInvalidRequest
	}
	if err != nil {
		return fmt.Errorf("start export generation: %w", err)
	}
	var request exportartifact.Request
	if err := json.Unmarshal(encoded, &request); err != nil {
		return fmt.Errorf("decode export request: %w", err)
	}
	records, err := s.records(ctx, request)
	if err != nil {
		return err
	}
	generated, err := s.gen.Generate(ctx, request, records)
	if err != nil {
		return err
	}
	objectID, err := s.ids.Next(ctx)
	if err != nil {
		return fmt.Errorf("issue export object ID: %w", err)
	}
	base := "exports/" + strconv.FormatInt(exportID, 10)
	stagedKey, finalKey := base+"/staged", base+"/artifact"
	if err := s.blobs.Stage(ctx, stagedKey, generated.Body, generated.MediaType); err != nil {
		return err
	}
	if err := s.blobs.Promote(ctx, stagedKey, finalKey); err != nil {
		_ = s.blobs.Delete(ctx, stagedKey)
		return err
	}
	now := s.now().UTC()
	expires := now.Add(artifactLifetime)
	provenance, _ := json.Marshal(map[string]any{"export_id": strconv.FormatInt(exportID, 10),
		"cutoff": request.Cutoff.UTC(), "generator": "opi.evidence-export/v1"})
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		_ = s.blobs.Delete(ctx, finalKey)
		return fmt.Errorf("begin export publication: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	_, err = tx.Exec(ctx, `INSERT INTO object_references
		(id,object_key,sha256,size_bytes,media_type,provenance,verified_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`, objectID, finalKey, generated.Digest[:],
		len(generated.Body), generated.MediaType, provenance, now)
	if err == nil {
		_, err = tx.Exec(ctx, `UPDATE export_requests SET state='succeeded',object_reference_id=$2,
			row_count=$3,completed_at=$4,expires_at=$5 WHERE id=$1 AND state='running'`,
			exportID, objectID, generated.Rows, now, expires)
	}
	if err != nil {
		_ = s.blobs.Delete(ctx, finalKey)
		return fmt.Errorf("publish export metadata: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		_ = s.blobs.Delete(ctx, finalKey)
		return fmt.Errorf("commit export publication: %w", err)
	}
	return nil
}

func (s *Store) records(ctx context.Context, request exportartifact.Request) ([]exportartifact.Record, error) {
	rows, err := s.pool.Query(ctx, `SELECT DISTINCT ON (m.project_id,m.definition_name)
		m.project_id,COALESCE(m.repository_ids[1],0),m.definition_name,m.numeric_value,m.status,
		d.unit,d.version,d.formula,m.observed_count,m.eligible_count,m.id
		FROM metric_snapshots m JOIN metric_definitions d
		  ON d.name=m.definition_name AND d.version=m.definition_version
		WHERE m.project_id=ANY($1) AND m.window_from=$2 AND m.window_to=$3 AND m.cutoff<=$4
		ORDER BY m.project_id,m.definition_name,m.cutoff DESC,m.id DESC`, request.ProjectIDs,
		request.WindowFrom, request.WindowTo, request.Cutoff)
	if err != nil {
		return nil, fmt.Errorf("read export records: %w", err)
	}
	defer rows.Close()
	records := make([]exportartifact.Record, 0)
	for rows.Next() {
		var record exportartifact.Record
		var rawStatus string
		var observed, eligible, snapshotID int64
		if err := rows.Scan(&record.ProjectID, &record.RepositoryID, &record.Metric, &record.Value,
			&rawStatus, &record.Unit, &record.Definition, &record.Formula, &observed, &eligible,
			&snapshotID); err != nil {
			return nil, fmt.Errorf("scan export record: %w", err)
		}
		record.Status = missingState(rawStatus)
		record.Coverage = fmt.Sprintf("%d/%d", observed, eligible)
		record.Provenance = []string{"metric_snapshot:" + strconv.FormatInt(snapshotID, 10)}
		records = append(records, record)
	}
	return records, rows.Err()
}

func (s *Store) authorizeProjects(ctx context.Context, workspaceID int64, ids []int64) error {
	var count int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM projects
		WHERE workspace_id=$1 AND id=ANY($2) AND state NOT IN ('deleting','deleted')`,
		workspaceID, ids).Scan(&count); err != nil {
		return fmt.Errorf("authorize export scope: %w", err)
	}
	if count != len(ids) {
		return project.ErrNotFound
	}
	return nil
}

func (s *Store) load(ctx context.Context, id, workspaceID, actorID int64) (Export, error) {
	var value Export
	var encoded, digest []byte
	var jobState string
	err := s.pool.QueryRow(ctx, `SELECT e.id,e.job_id,e.state,j.state,e.request,e.row_count,
		COALESCE(o.media_type,''),COALESCE(o.sha256,''::bytea),COALESCE(o.size_bytes,0),
		COALESCE(o.object_key,''),e.failure_code,e.created_at,e.completed_at,e.expires_at,e.workspace_id,e.actor_id
		FROM export_requests e JOIN jobs j ON j.id=e.job_id
		LEFT JOIN object_references o ON o.id=e.object_reference_id
		WHERE e.id=$1 AND e.workspace_id=$2 AND e.actor_id=$3`, id, workspaceID, actorID).Scan(
		&value.ID, &value.JobID,
		&value.State, &jobState, &encoded, &value.Rows, &value.MediaType, &digest, &value.SizeBytes,
		&value.objectKey, &value.Failure, &value.CreatedAt, &value.CompletedAt, &value.ExpiresAt,
		&value.workspaceID, &value.actorID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Export{}, project.ErrNotFound
	}
	if err != nil {
		return Export{}, fmt.Errorf("read export: %w", err)
	}
	if err := json.Unmarshal(encoded, &value.Request); err != nil {
		return Export{}, fmt.Errorf("decode export metadata: %w", err)
	}
	if value.State != "succeeded" && (jobState == "failed" || jobState == "cancelled") {
		value.State = jobState
	}
	value.SHA256 = hex.EncodeToString(digest)
	return value, nil
}

func missingState(value string) exportartifact.MissingState {
	switch exportartifact.MissingState(value) {
	case exportartifact.Available, exportartifact.Unknown, exportartifact.NotApplicable,
		exportartifact.InsufficientData:
		return exportartifact.MissingState(value)
	default:
		return exportartifact.Unknown
	}
}

func mustJSON(value any) []byte {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return encoded
}

// FailureCode is deliberately stable and contains no provider or object-store details.
func FailureCode(err error) string {
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return "interrupted"
	case errors.Is(err, exportartifact.ErrTooLarge):
		return "export_too_large"
	case errors.Is(err, exportartifact.ErrInvalidRequest):
		return "invalid_export"
	default:
		return "export_failed"
	}
}

func Filename(value Export) string {
	extension := ".json"
	if value.Request.Format == exportartifact.CSV {
		extension = ".csv"
	}
	return "opi-" + strings.ReplaceAll(value.Request.Resource, "_", "-") + "-" +
		strconv.FormatInt(value.ID, 10) + extension
}
