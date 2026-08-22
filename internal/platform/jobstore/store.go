// Package jobstore owns PostgreSQL job leases, attempts, checkpoints, and terminal events.
package jobstore

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/leohteixeira/opensource-project-intelligence/internal/job"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/database"
	"github.com/leohteixeira/opensource-project-intelligence/internal/project"
)

type IDSource interface {
	Next(context.Context) (int64, error)
}

type Store struct {
	pool  *pgxpool.Pool
	ids   IDSource
	now   func() time.Time
	blobs BlobDeleter
}

type BlobDeleter interface {
	Delete(context.Context, string) error
}

type Lease struct {
	Job       job.Job
	AttemptID int64
	Attempt   int
	Holder    string
	ExpiresAt time.Time
}

type GitHubSource struct {
	ProjectID    int64
	RepositoryID int64
	SourceID     int64
	Owner        string
	Repository   string
	Pages        map[string]int
	CoverageFrom time.Time
}

func New(pool *database.Pool, ids IDSource) *Store {
	return &Store{pool: pool.Unwrap(), ids: ids, now: time.Now}
}

func (s *Store) UseBlobStore(blobs BlobDeleter) error {
	if blobs == nil {
		return errors.New("blob store is required")
	}
	s.blobs = blobs
	return nil
}

func (s *Store) GitHubSource(ctx context.Context, projectID int64) (GitHubSource, error) {
	var target GitHubSource
	var canonicalURL string
	var issueCursor, pullRequestCursor, releaseCursor, commitCursor string
	err := s.pool.QueryRow(ctx, `SELECT s.project_id,s.repository_id,s.id,s.canonical_url,
		COALESCE((SELECT cursor FROM sync_checkpoints WHERE source_id=s.id AND scope='github_issues'),''),
		COALESCE((SELECT cursor FROM sync_checkpoints WHERE source_id=s.id AND scope='github_pull_requests'),''),
		COALESCE((SELECT cursor FROM sync_checkpoints WHERE source_id=s.id AND scope='github_releases'),''),
		COALESCE((SELECT cursor FROM sync_checkpoints WHERE source_id=s.id AND scope='github_commits'),''),
		COALESCE(s.coverage_from,now()-interval '180 days')
		FROM sources s
		WHERE s.project_id=$1 AND s.kind='github' AND s.state='available'
		ORDER BY s.id LIMIT 1`, projectID).Scan(&target.ProjectID, &target.RepositoryID,
		&target.SourceID, &canonicalURL, &issueCursor, &pullRequestCursor, &releaseCursor,
		&commitCursor, &target.CoverageFrom)
	if errors.Is(err, pgx.ErrNoRows) {
		return GitHubSource{}, project.ErrNotFound
	}
	if err != nil {
		return GitHubSource{}, fmt.Errorf("read GitHub collection target: %w", err)
	}
	parts := strings.Split(strings.Trim(strings.TrimPrefix(canonicalURL, "https://github.com/"), "/"), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return GitHubSource{}, fmt.Errorf("invalid canonical GitHub source URL")
	}
	target.Owner, target.Repository = parts[0], strings.TrimSuffix(parts[1], ".git")
	target.Pages = make(map[string]int, 4)
	for scope, cursor := range map[string]string{
		"github_issues": issueCursor, "github_pull_requests": pullRequestCursor,
		"github_releases": releaseCursor, "github_commits": commitCursor,
	} {
		page, parseErr := checkpointPage(cursor)
		if parseErr != nil {
			return GitHubSource{}, fmt.Errorf("invalid %s checkpoint cursor: %w", scope, parseErr)
		}
		target.Pages[scope] = page
	}
	return target, nil
}

func checkpointPage(cursor string) (int, error) {
	if cursor == "complete" {
		// A completed checkpoint starts the next incremental synchronization
		// from the provider's newest page. Canonical upserts make the overlap
		// idempotent while allowing updated records to be observed.
		return 1, nil
	}
	if cursor == "" {
		return 1, nil
	}
	page, err := strconv.Atoi(cursor)
	if err != nil || page <= 0 {
		return 0, errors.New("cursor must be a positive page or complete")
	}
	return page, nil
}

func (s *Store) MarkSourceUnavailable(ctx context.Context, sourceID int64, code string) error {
	if sourceID <= 0 || strings.TrimSpace(code) == "" {
		return job.ErrInvalid
	}
	_, err := s.pool.Exec(ctx, `UPDATE sources SET state='unavailable',public=false,
		failure_code=$1,last_attempt_at=now(),next_run_at=NULL,version=version+1,updated_at=now()
		WHERE id=$2`, code, sourceID)
	if err != nil {
		return fmt.Errorf("mark source unavailable: %w", err)
	}
	return nil
}

func (s *Store) RecoverExpired(ctx context.Context) (int64, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin expired lease recovery: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	now := s.now().UTC()
	rows, err := tx.Query(ctx, `SELECT id,COALESCE(project_id,0),kind,version,max_attempts,correlation_id,
		(SELECT count(*) FROM job_attempts WHERE job_id=jobs.id)
		FROM jobs WHERE state='running' AND lease_expires_at <= $1
		ORDER BY id FOR UPDATE`, now)
	if err != nil {
		return 0, fmt.Errorf("lock expired job leases: %w", err)
	}
	type expiredJob struct {
		value         job.Job
		maxAttempts   int
		attemptCount  int
		correlationID string
	}
	expired := make([]expiredJob, 0)
	for rows.Next() {
		var target expiredJob
		if err := rows.Scan(&target.value.ID, &target.value.ProjectID, &target.value.Kind,
			&target.value.Version, &target.maxAttempts, &target.correlationID,
			&target.attemptCount); err != nil {
			rows.Close()
			return 0, fmt.Errorf("scan expired job lease: %w", err)
		}
		expired = append(expired, target)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, fmt.Errorf("read expired job leases: %w", err)
	}
	rows.Close()
	for _, target := range expired {
		if _, err := tx.Exec(ctx, `UPDATE job_attempts SET state='failed',finished_at=$1,
			failure_code='lease_expired' WHERE job_id=$2 AND state='running'`, now,
			target.value.ID); err != nil {
			return 0, fmt.Errorf("fail expired Job attempt: %w", err)
		}
		eventType := "job.queued"
		target.value.State = job.Queued
		target.value.Failure = ""
		if target.attemptCount >= target.maxAttempts {
			eventType = "job.dead_lettered"
			target.value.State = job.Failed
			target.value.Failure = "attempts_exhausted"
			target.value.FinishedAt = &now
		}
		target.value.Version++
		target.value.UpdatedAt = now
		if _, err := tx.Exec(ctx, `UPDATE jobs SET state=$1, failure_code=$2,
			finished_at=$3,lease_holder=NULL,lease_expires_at=NULL,heartbeat_at=NULL,
			available_at=$4,version=$5,updated_at=$4 WHERE id=$6`, target.value.State,
			target.value.Failure, target.value.FinishedAt, now, target.value.Version,
			target.value.ID); err != nil {
			return 0, fmt.Errorf("recover expired Job lease: %w", err)
		}
		if err := s.recordEvent(ctx, tx, target.value); err != nil {
			return 0, err
		}
		if err := s.recordDeliveryEvent(ctx, tx, target.value, eventType,
			target.correlationID, now); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit expired lease recovery: %w", err)
	}
	return int64(len(expired)), nil
}

func (s *Store) Claim(ctx context.Context, holder string, ttl time.Duration) (*Lease, error) {
	return s.claim(ctx, 0, holder, ttl)
}

// ClaimJob leases one broker-selected Job. PostgreSQL still decides whether
// the notification is current, already handled, delayed, or terminal.
func (s *Store) ClaimJob(ctx context.Context, jobID int64, holder string, ttl time.Duration) (*Lease, error) {
	if jobID <= 0 {
		return nil, job.ErrInvalid
	}
	return s.claim(ctx, jobID, holder, ttl)
}

func (s *Store) claim(ctx context.Context, jobID int64, holder string, ttl time.Duration) (*Lease, error) {
	if strings.TrimSpace(holder) == "" || ttl <= 0 {
		return nil, job.ErrInvalid
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin job claim: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	now := s.now().UTC()
	expires := now.Add(ttl)
	var value job.Job
	var progress, checkpoint []byte
	err = tx.QueryRow(ctx, `
		SELECT id,COALESCE(project_id,0),kind,state,progress,checkpoint,requested_from,requested_to,version,coalesced_requests,
			cancellable,created_at,started_at,updated_at,finished_at,failure_code
		FROM jobs
		WHERE state='queued' AND available_at<=now() AND ($1::bigint=0 OR id=$1)
		ORDER BY available_at,created_at,id
		FOR UPDATE SKIP LOCKED LIMIT 1`, jobID).Scan(&value.ID, &value.ProjectID, &value.Kind, &value.State,
		&progress, &checkpoint, &value.RequestedFrom, &value.RequestedTo, &value.Version,
		&value.CoalescedRequests, &value.Cancellable,
		&value.CreatedAt, &value.StartedAt, &value.UpdatedAt, &value.FinishedAt, &value.Failure)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("select claimable job: %w", err)
	}
	if err := decodeJobState(&value, progress, checkpoint); err != nil {
		return nil, err
	}
	var attempt int
	if err := tx.QueryRow(ctx, `SELECT count(*)+1 FROM job_attempts WHERE job_id=$1`, value.ID).Scan(&attempt); err != nil {
		return nil, fmt.Errorf("count job attempts: %w", err)
	}
	var maxAttempts int
	if err := tx.QueryRow(ctx, `SELECT max_attempts FROM jobs WHERE id=$1`, value.ID).Scan(&maxAttempts); err != nil {
		return nil, fmt.Errorf("read job attempt limit: %w", err)
	}
	if attempt > maxAttempts {
		if err := markExhausted(ctx, tx, value.ID, now); err != nil {
			return nil, err
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("commit exhausted job: %w", err)
		}
		return nil, nil
	}
	attemptID, err := s.ids.Next(ctx)
	if err != nil {
		return nil, fmt.Errorf("issue job attempt ID: %w", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO job_attempts
		(id,job_id,attempt,worker_id,state,started_at) VALUES ($1,$2,$3,$4,'running',$5)`,
		attemptID, value.ID, attempt, holder, now); err != nil {
		return nil, fmt.Errorf("start job attempt: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE jobs SET state='running',version=version+1,
		started_at=COALESCE(started_at,$1),updated_at=$1,lease_holder=$2,lease_expires_at=$3,
		heartbeat_at=$1 WHERE id=$4`, now, holder, expires, value.ID); err != nil {
		return nil, fmt.Errorf("claim job: %w", err)
	}
	value.State = job.Running
	value.Version++
	value.UpdatedAt = now
	if value.StartedAt == nil {
		value.StartedAt = &now
	}
	if err := s.recordEvent(ctx, tx, value); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit job claim: %w", err)
	}
	return &Lease{Job: value, AttemptID: attemptID, Attempt: attempt, Holder: holder, ExpiresAt: expires}, nil
}

func (s *Store) Heartbeat(ctx context.Context, lease Lease, ttl time.Duration) (Lease, error) {
	if ttl <= 0 {
		return Lease{}, job.ErrInvalid
	}
	now := s.now().UTC()
	expires := now.Add(ttl)
	command, err := s.pool.Exec(ctx, `UPDATE jobs SET heartbeat_at=$1,lease_expires_at=$2,updated_at=$1
		WHERE id=$3 AND state='running' AND lease_holder=$4 AND lease_expires_at>$1`,
		now, expires, lease.Job.ID, lease.Holder)
	if err != nil {
		return Lease{}, fmt.Errorf("heartbeat job lease: %w", err)
	}
	if command.RowsAffected() != 1 {
		return Lease{}, job.ErrLeaseUnavailable
	}
	lease.ExpiresAt = expires
	return lease, nil
}

func (s *Store) CheckCancelled(ctx context.Context, lease Lease) (bool, error) {
	var state job.State
	err := s.pool.QueryRow(ctx, `SELECT state FROM jobs WHERE id=$1`, lease.Job.ID).Scan(&state)
	if err != nil {
		return false, fmt.Errorf("read job cancellation: %w", err)
	}
	return state == job.Cancelled, nil
}

// MarkCollectionSuccess advances source freshness only after a collector page
// has been normalized and committed. PostgreSQL remains authoritative if an
// optional cache is stale or unavailable.
func (s *Store) MarkCollectionSuccess(ctx context.Context, lease Lease) error {
	now := s.now().UTC()
	command, err := s.pool.Exec(ctx, `UPDATE sources SET last_attempt_at=$1,last_success_at=$1,
		coverage_to=GREATEST(COALESCE(coverage_to,$1),$1),failure_code='',next_run_at=$1+interval '1 hour',
		version=version+1,updated_at=$1
		WHERE project_id=$2 AND state='available' AND EXISTS (
			SELECT 1 FROM jobs WHERE id=$3 AND state='running' AND lease_holder=$4 AND lease_expires_at>$1
		)`, now, lease.Job.ProjectID, lease.Job.ID, lease.Holder)
	if err != nil {
		return fmt.Errorf("commit source collection freshness: %w", err)
	}
	if command.RowsAffected() == 0 {
		var sourceCount int
		if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM sources WHERE project_id=$1`,
			lease.Job.ProjectID).Scan(&sourceCount); err != nil {
			return fmt.Errorf("check collection sources: %w", err)
		}
		if sourceCount > 0 {
			return job.ErrLeaseUnavailable
		}
	}
	return nil
}

func (s *Store) Checkpoint(
	ctx context.Context,
	lease Lease,
	completed int64,
	checkpoint job.Checkpoint,
) (Lease, error) {
	now := s.now().UTC()
	encoded, err := json.Marshal(checkpoint)
	if err != nil {
		return Lease{}, fmt.Errorf("encode job checkpoint: %w", err)
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Lease{}, fmt.Errorf("begin job checkpoint: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var progress []byte
	var version int64
	err = tx.QueryRow(ctx, `UPDATE jobs SET
		progress=jsonb_set(progress,'{completed}',to_jsonb($1::bigint),true),checkpoint=$2,
		version=version+1,updated_at=$3
		WHERE id=$4 AND state='running' AND lease_holder=$5 AND lease_expires_at>$3
		RETURNING progress,version`, completed, encoded, now, lease.Job.ID, lease.Holder).Scan(&progress, &version)
	if errors.Is(err, pgx.ErrNoRows) {
		return Lease{}, job.ErrLeaseUnavailable
	}
	if err != nil {
		return Lease{}, fmt.Errorf("commit job checkpoint: %w", err)
	}
	lease.Job.Progress.Completed = completed
	lease.Job.Checkpoint = &checkpoint
	lease.Job.Version = version
	lease.Job.UpdatedAt = now
	if err := json.Unmarshal(progress, &lease.Job.Progress); err != nil {
		return Lease{}, fmt.Errorf("decode committed progress: %w", err)
	}
	if err := s.recordEvent(ctx, tx, lease.Job); err != nil {
		return Lease{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Lease{}, fmt.Errorf("commit job checkpoint transaction: %w", err)
	}
	return lease, nil
}

func (s *Store) Complete(ctx context.Context, lease Lease) error {
	return s.finish(ctx, lease, job.Succeeded, "", 0)
}

func (s *Store) Fail(ctx context.Context, lease Lease, code string, retryAfter time.Duration) error {
	if strings.TrimSpace(code) == "" {
		code = "worker_error"
	}
	return s.finish(ctx, lease, job.Failed, code, retryAfter)
}

func (s *Store) finish(ctx context.Context, lease Lease, state job.State, code string, retryAfter time.Duration) error {
	now := s.now().UTC()
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin job completion: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var currentState job.State
	var maxAttempts int
	var correlationID string
	err = tx.QueryRow(ctx, `SELECT state,max_attempts,correlation_id FROM jobs WHERE id=$1 AND lease_holder=$2
		FOR UPDATE`, lease.Job.ID, lease.Holder).Scan(&currentState, &maxAttempts, &correlationID)
	if errors.Is(err, pgx.ErrNoRows) || currentState != job.Running {
		return job.ErrLeaseUnavailable
	}
	if err != nil {
		return fmt.Errorf("lock finishing job: %w", err)
	}
	finalState := state
	availableAt := now
	if state == job.Failed && lease.Attempt < maxAttempts && retryAfter >= 0 {
		finalState = job.Queued
		if retryAfter == 0 {
			retryAfter = time.Duration(1<<min(lease.Attempt, 8)) * time.Second
		}
		availableAt = now.Add(retryAfter)
	}
	command, err := tx.Exec(ctx, `UPDATE jobs SET state=$1::text,version=version+1,updated_at=$2::timestamptz,
		finished_at=CASE WHEN $1::text IN ('succeeded','failed','cancelled') THEN $2::timestamptz ELSE NULL END,
		failure_code=$3,available_at=$4,lease_holder=NULL,lease_expires_at=NULL,heartbeat_at=NULL
		WHERE id=$5 AND state='running' AND lease_holder=$6`,
		finalState, now, code, availableAt, lease.Job.ID, lease.Holder)
	if err != nil {
		return fmt.Errorf("finish job: %w", err)
	}
	if command.RowsAffected() != 1 {
		return job.ErrLeaseUnavailable
	}
	attemptState := string(state)
	if _, err := tx.Exec(ctx, `UPDATE job_attempts SET state=$1,finished_at=$2,failure_code=$3
		WHERE id=$4 AND state='running'`, attemptState, now, code, lease.AttemptID); err != nil {
		return fmt.Errorf("finish job attempt: %w", err)
	}
	lease.Job.State = finalState
	lease.Job.Version++
	lease.Job.UpdatedAt = now
	lease.Job.Failure = code
	if finalState == job.Succeeded || finalState == job.Failed || finalState == job.Cancelled {
		lease.Job.FinishedAt = &now
	}
	if err := s.recordEvent(ctx, tx, lease.Job); err != nil {
		return err
	}
	if finalState == job.Queued {
		if err := s.recordDeliveryEvent(ctx, tx, lease.Job, "job.queued", correlationID, availableAt); err != nil {
			return err
		}
	} else if finalState == job.Failed {
		if err := s.recordDeliveryEvent(ctx, tx, lease.Job, "job.dead_lettered", correlationID, now); err != nil {
			return err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit job completion: %w", err)
	}
	return nil
}

func (s *Store) recordDeliveryEvent(
	ctx context.Context,
	tx pgx.Tx,
	value job.Job,
	eventType string,
	correlationID string,
	availableAt time.Time,
) error {
	id, err := s.ids.Next(ctx)
	if err != nil {
		return fmt.Errorf("issue Job delivery event ID: %w", err)
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode Job delivery event: %w", err)
	}
	aggregateID := value.ProjectID
	if aggregateID == 0 {
		aggregateID = value.ID
	}
	_, err = tx.Exec(ctx, `INSERT INTO outbox_events
		(id,aggregate_type,aggregate_id,event_type,schema_version,payload,job_id,
		 correlation_id,causation_id,available_at)
		VALUES($1,'job',$2,$3,1,$4,$5,$6,$6,$7)`, id, aggregateID, eventType,
		payload, value.ID, correlationID, availableAt)
	if err != nil {
		return fmt.Errorf("record %s delivery event: %w", eventType, err)
	}
	return nil
}

func (s *Store) PurgeProject(ctx context.Context, lease Lease, batchSize int) (bool, Lease, error) {
	if batchSize <= 0 || batchSize > 1000 {
		batchSize = 100
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return false, Lease{}, fmt.Errorf("begin project purge: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var state string
	var slug, reason string
	var actorID *int64
	err = tx.QueryRow(ctx, `SELECT state,slug,deletion_reason,deletion_actor_id FROM projects
		WHERE id=$1 FOR UPDATE`, lease.Job.ProjectID).Scan(&state, &slug, &reason, &actorID)
	if errors.Is(err, pgx.ErrNoRows) {
		return true, lease, nil
	}
	if err != nil {
		return false, Lease{}, fmt.Errorf("lock deleting project: %w", err)
	}
	if state == "deleted" {
		return true, lease, nil
	}
	if state != "deleting" {
		return false, Lease{}, fmt.Errorf("project purge requires deleting state")
	}
	manifestID, err := s.ensureManifest(ctx, tx, lease.Job.ProjectID, lease.Job.ID)
	if err != nil {
		return false, Lease{}, err
	}
	rows, err := tx.Query(ctx, `SELECT objects.object_reference_id,owned.object_key
		FROM purge_manifest_objects objects JOIN object_references owned
		ON owned.id=objects.object_reference_id
		WHERE manifest_id=$1 AND deleted_at IS NULL ORDER BY object_reference_id LIMIT $2 FOR UPDATE SKIP LOCKED`,
		manifestID, batchSize)
	if err != nil {
		return false, Lease{}, fmt.Errorf("read purge batch: %w", err)
	}
	type purgeObject struct {
		id  int64
		key string
	}
	objects := make([]purgeObject, 0, batchSize)
	for rows.Next() {
		var object purgeObject
		if err := rows.Scan(&object.id, &object.key); err != nil {
			rows.Close()
			return false, Lease{}, fmt.Errorf("scan purge object: %w", err)
		}
		objects = append(objects, object)
	}
	rows.Close()
	if len(objects) > 0 && s.blobs == nil {
		return false, Lease{}, errors.New("object storage is required to purge owned evidence")
	}
	for _, object := range objects {
		if err := s.blobs.Delete(ctx, object.key); err != nil {
			_, _ = tx.Exec(ctx, `UPDATE purge_manifest_objects SET attempts=attempts+1,
				last_error='object deletion failed' WHERE manifest_id=$1 AND object_reference_id=$2`,
				manifestID, object.id)
			return false, Lease{}, fmt.Errorf("delete owned object %d: %w", object.id, err)
		}
		if _, err := tx.Exec(ctx, `UPDATE object_references SET retention_state='purged'
			WHERE id=$1 AND project_id=$2`, object.id, lease.Job.ProjectID); err != nil {
			return false, Lease{}, fmt.Errorf("purge object ownership: %w", err)
		}
		if _, err := tx.Exec(ctx, `UPDATE purge_manifest_objects SET deleted_at=now()
			WHERE manifest_id=$1 AND object_reference_id=$2`, manifestID, object.id); err != nil {
			return false, Lease{}, fmt.Errorf("checkpoint purged object: %w", err)
		}
	}
	var remaining int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM purge_manifest_objects
		WHERE manifest_id=$1 AND deleted_at IS NULL`, manifestID).Scan(&remaining); err != nil {
		return false, Lease{}, fmt.Errorf("count purge remainder: %w", err)
	}
	if remaining == 0 {
		fingerprint := sha256Text(fmt.Sprintf("%d:%s", lease.Job.ProjectID, slug))
		tombstoneID, err := s.ids.Next(ctx)
		if err != nil {
			return false, Lease{}, fmt.Errorf("issue tombstone ID: %w", err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO project_tombstones
			(id,project_id,workspace_id,slug_hash,deletion_actor_id,deletion_reason,deleted_at)
			SELECT $1,id,workspace_id,$2,$3,$4,now() FROM projects WHERE id=$5
			ON CONFLICT (project_id) DO NOTHING`, tombstoneID, fingerprint, actorID, reason,
			lease.Job.ProjectID); err != nil {
			return false, Lease{}, fmt.Errorf("write project tombstone: %w", err)
		}
		if _, err := tx.Exec(ctx, `DELETE FROM projects WHERE id=$1`, lease.Job.ProjectID); err != nil {
			return false, Lease{}, fmt.Errorf("purge project rows: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return false, Lease{}, fmt.Errorf("commit project purge batch: %w", err)
	}
	checkpoint := job.Checkpoint{Scope: "purge", Cursor: fmt.Sprint(len(objects)),
		CoverageTo: s.now().UTC(), Version: lease.Job.Version + 1}
	lease, err = s.Checkpoint(ctx, lease, lease.Job.Progress.Completed+int64(len(objects)), checkpoint)
	return remaining == 0, lease, err
}

func (s *Store) ensureManifest(ctx context.Context, tx pgx.Tx, projectID, jobID int64) (int64, error) {
	var manifestID int64
	err := tx.QueryRow(ctx, `SELECT id FROM purge_manifests WHERE project_id=$1 FOR UPDATE`, projectID).Scan(&manifestID)
	if err == nil {
		return manifestID, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, fmt.Errorf("read purge manifest: %w", err)
	}
	manifestID, err = s.ids.Next(ctx)
	if err != nil {
		return 0, fmt.Errorf("issue purge manifest ID: %w", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO purge_manifests (id,project_id,job_id,state)
		VALUES ($1,$2,$3,'purging')`, manifestID, projectID, jobID); err != nil {
		return 0, fmt.Errorf("create purge manifest: %w", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO purge_manifest_objects (manifest_id,object_reference_id)
		SELECT $1,id FROM object_references WHERE project_id=$2`, manifestID, projectID); err != nil {
		return 0, fmt.Errorf("snapshot project object ownership: %w", err)
	}
	return manifestID, nil
}

func (s *Store) recordEvent(ctx context.Context, tx pgx.Tx, value job.Job) error {
	id, err := s.ids.Next(ctx)
	if err != nil {
		return fmt.Errorf("issue job event ID: %w", err)
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode job event: %w", err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO job_events (id,job_id,job_version,representation,occurred_at)
		VALUES ($1,$2,$3,$4,$5)`, id, value.ID, value.Version, encoded, value.UpdatedAt)
	if err != nil {
		return fmt.Errorf("record job event: %w", err)
	}
	return nil
}

func decodeJobState(value *job.Job, progress, checkpoint []byte) error {
	if err := json.Unmarshal(progress, &value.Progress); err != nil {
		return fmt.Errorf("decode job progress: %w", err)
	}
	if len(checkpoint) > 0 && string(checkpoint) != "{}" && string(checkpoint) != "null" {
		var decoded job.Checkpoint
		if err := json.Unmarshal(checkpoint, &decoded); err != nil {
			return fmt.Errorf("decode job checkpoint: %w", err)
		}
		value.Checkpoint = &decoded
	}
	return nil
}

func markExhausted(ctx context.Context, tx pgx.Tx, id int64, now time.Time) error {
	_, err := tx.Exec(ctx, `UPDATE jobs SET state='failed',failure_code='attempts_exhausted',
		finished_at=$1,updated_at=$1,version=version+1 WHERE id=$2`, now, id)
	if err != nil {
		return fmt.Errorf("mark exhausted job: %w", err)
	}
	return nil
}

func sha256Text(value string) []byte {
	digest := sha256.Sum256([]byte(value))
	return digest[:]
}
