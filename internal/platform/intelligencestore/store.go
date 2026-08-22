// Package intelligencestore persists immutable metric, health, contributor, and comparison facts.
package intelligencestore

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/leohteixeira/opensource-project-intelligence/internal/access"
	"github.com/leohteixeira/opensource-project-intelligence/internal/comparison"
	"github.com/leohteixeira/opensource-project-intelligence/internal/contributor"
	"github.com/leohteixeira/opensource-project-intelligence/internal/metric"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/database"
)

type IDSource interface {
	Next(context.Context) (int64, error)
}

type Store struct {
	pool *pgxpool.Pool
	ids  IDSource
}

func New(pool *database.Pool, ids IDSource) *Store { return &Store{pool: pool.Unwrap(), ids: ids} }

func (s *Store) SaveMetric(ctx context.Context, snapshot metric.Snapshot) (metric.Snapshot, error) {
	values, err := s.SaveMetricSet(ctx, []metric.Snapshot{snapshot})
	if err != nil {
		return metric.Snapshot{}, err
	}
	return values[0], nil
}

// SaveMetricSet publishes one complete definition set atomically. An identical computation is an
// idempotent replay; the same materialization key with different immutable content is rejected.
func (s *Store) SaveMetricSet(ctx context.Context, snapshots []metric.Snapshot) ([]metric.Snapshot, error) {
	if len(snapshots) == 0 {
		return nil, metric.ErrInvalid
	}
	values := slices.Clone(snapshots)
	for index := range values {
		if values[index].ID != 0 {
			continue
		}
		id, err := s.ids.Next(ctx)
		if err != nil {
			return nil, fmt.Errorf("metric snapshot ID: %w", err)
		}
		values[index].ID = id
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin metric materialization: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	for index := range values {
		stored, insertErr := insertMetric(ctx, tx, values[index])
		if insertErr != nil {
			return nil, insertErr
		}
		values[index] = stored
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit metric materialization: %w", err)
	}
	return values, nil
}

func insertMetric(ctx context.Context, tx pgx.Tx, snapshot metric.Snapshot) (metric.Snapshot, error) {
	snapshot = normalizeSnapshot(snapshot)
	inserted := false
	err := tx.QueryRow(ctx, `INSERT INTO metric_snapshots
		(id,project_id,definition_name,definition_version,window_from,window_to,cutoff,status,
		 numeric_value,eligible_count,observed_count,coverage_note,repository_ids,stale_reason)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		ON CONFLICT(project_id,definition_name,definition_version,window_from,window_to,cutoff)
		DO NOTHING RETURNING true`, snapshot.ID, snapshot.ProjectID, snapshot.Definition.Name,
		snapshot.Definition.Version, snapshot.Window.From, snapshot.Window.To, snapshot.Window.Cutoff,
		snapshot.Status, snapshot.Value, snapshot.Coverage.Eligible, snapshot.Coverage.Observed,
		snapshot.Coverage.Note, snapshot.Repositories, snapshot.StaleReason).Scan(&inserted)
	if errors.Is(err, pgx.ErrNoRows) {
		existing, loadErr := loadMetric(ctx, tx, snapshot.ProjectID, snapshot.Definition.Name,
			snapshot.Definition.Version, snapshot.Window)
		if loadErr != nil {
			return metric.Snapshot{}, loadErr
		}
		expected := normalizeSnapshot(snapshot)
		expected.ID = existing.ID
		if !reflect.DeepEqual(expected, normalizeSnapshot(existing)) {
			return metric.Snapshot{}, fmt.Errorf("%w: materialization key collision for %s", metric.ErrInvalid, snapshot.Definition.Name)
		}
		return existing, nil
	}
	if err != nil {
		return metric.Snapshot{}, fmt.Errorf("insert metric snapshot: %w", err)
	}
	for ordinal, factor := range snapshot.Factors {
		var evidenceID *int64
		if factor.EvidenceID > 0 {
			evidenceID = &factor.EvidenceID
		}
		if _, err := tx.Exec(ctx, `INSERT INTO metric_factors
			(snapshot_id,ordinal,name,numeric_value,unit,evidence_id) VALUES($1,$2,$3,$4,$5,$6)`,
			snapshot.ID, ordinal, factor.Name, factor.Value, factor.Unit, evidenceID); err != nil {
			return metric.Snapshot{}, fmt.Errorf("insert metric factor: %w", err)
		}
	}
	return snapshot, nil
}

func normalizeSnapshot(value metric.Snapshot) metric.Snapshot {
	if len(value.Factors) == 0 {
		value.Factors = []metric.Factor{}
	}
	if len(value.Repositories) == 0 {
		value.Repositories = []int64{}
	}
	return value
}

func loadMetric(ctx context.Context, query interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, projectID int64, name, version string, window metric.Window) (metric.Snapshot, error) {
	definition, ok := metric.DefinitionByName(name)
	if !ok || definition.Version != version {
		return metric.Snapshot{}, metric.ErrInvalid
	}
	var snapshot metric.Snapshot
	var value *float64
	var status metric.Status
	err := query.QueryRow(ctx, `SELECT id,status,numeric_value,eligible_count,observed_count,
		coverage_note,repository_ids,stale_reason FROM metric_snapshots
		WHERE project_id=$1 AND definition_name=$2 AND definition_version=$3
		AND window_from=$4 AND window_to=$5 AND cutoff=$6`, projectID, name, version,
		window.From, window.To, window.Cutoff).Scan(&snapshot.ID, &status, &value,
		&snapshot.Coverage.Eligible, &snapshot.Coverage.Observed, &snapshot.Coverage.Note,
		&snapshot.Repositories, &snapshot.StaleReason)
	if err != nil {
		return metric.Snapshot{}, fmt.Errorf("load metric snapshot: %w", err)
	}
	snapshot.ProjectID, snapshot.Definition, snapshot.Window, snapshot.Status, snapshot.Value = projectID, definition, window, status, value
	if snapshot.Coverage.Eligible > 0 {
		snapshot.Coverage.Ratio = float64(snapshot.Coverage.Observed) / float64(snapshot.Coverage.Eligible)
	}
	rows, err := sQuery(ctx, query, `SELECT name,numeric_value,unit,COALESCE(evidence_id,0)
		FROM metric_factors WHERE snapshot_id=$1 ORDER BY ordinal`, snapshot.ID)
	if err != nil {
		return metric.Snapshot{}, fmt.Errorf("load metric factors: %w", err)
	}
	defer rows.Close()
	snapshot.Factors = make([]metric.Factor, 0)
	for rows.Next() {
		var factor metric.Factor
		if err := rows.Scan(&factor.Name, &factor.Value, &factor.Unit, &factor.EvidenceID); err != nil {
			return metric.Snapshot{}, fmt.Errorf("scan metric factor: %w", err)
		}
		snapshot.Factors = append(snapshot.Factors, factor)
	}
	if err := rows.Err(); err != nil {
		return metric.Snapshot{}, fmt.Errorf("iterate metric factors: %w", err)
	}
	return snapshot, nil
}

type rowQuery interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func sQuery(ctx context.Context, query any, sql string, args ...any) (pgx.Rows, error) {
	if value, ok := query.(rowQuery); ok {
		return value.Query(ctx, sql, args...)
	}
	return nil, errors.New("query adapter does not support rows")
}

func (s *Store) Metrics(ctx context.Context, principal access.Principal, projectID int64, window metric.Window) ([]metric.Snapshot, error) {
	if err := s.authorizeProject(ctx, principal, projectID); err != nil {
		return nil, err
	}
	values := make([]metric.Snapshot, 0, len(metric.Catalog()))
	for _, definition := range metric.Catalog() {
		value, err := loadMetric(ctx, s.pool, projectID, definition.Name, definition.Version, window)
		if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, metric.ErrInvalid) || hasNoRows(err) {
			missing, makeErr := metric.NewSnapshot(projectID, definition, window, metric.StatusInsufficientData,
				nil, metric.Coverage{Note: "no immutable materialization exists for this cutoff"}, nil, nil)
			if makeErr != nil {
				return nil, makeErr
			}
			values = append(values, missing)
			continue
		}
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

func (s *Store) Metric(ctx context.Context, principal access.Principal, projectID int64, name string, window metric.Window) (metric.Snapshot, error) {
	if err := s.authorizeProject(ctx, principal, projectID); err != nil {
		return metric.Snapshot{}, err
	}
	definition, ok := metric.DefinitionByName(name)
	if !ok {
		return metric.Snapshot{}, metric.ErrInvalid
	}
	return loadMetric(ctx, s.pool, projectID, name, definition.Version, window)
}

func (s *Store) SaveHealth(ctx context.Context, health metric.Health) (metric.Health, error) {
	id, err := s.ids.Next(ctx)
	if err != nil {
		return metric.Health{}, fmt.Errorf("health snapshot ID: %w", err)
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return metric.Health{}, fmt.Errorf("begin health materialization: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var storedID int64
	err = tx.QueryRow(ctx, `INSERT INTO health_snapshots
		(id,project_id,definition_version,window_from,window_to,cutoff,overall_status,overall_score)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT(project_id,definition_version,window_from,window_to,cutoff)
		DO NOTHING RETURNING id`, id, health.ProjectID, health.Version,
		health.Window.From, health.Window.To, health.Window.Cutoff, health.OverallStatus, health.Overall).Scan(&storedID)
	if errors.Is(err, pgx.ErrNoRows) {
		existing, loadErr := s.loadHealth(ctx, tx, health.ProjectID, health.Window)
		if loadErr != nil {
			return metric.Health{}, loadErr
		}
		if !reflect.DeepEqual(normalizeHealth(health), normalizeHealth(existing)) {
			return metric.Health{}, fmt.Errorf("%w: health materialization key collision", metric.ErrInvalid)
		}
		return existing, nil
	}
	if err != nil {
		return metric.Health{}, fmt.Errorf("insert health snapshot: %w", err)
	}
	for ordinal, dimension := range health.Dimensions {
		if _, err := tx.Exec(ctx, `INSERT INTO health_dimensions
			(health_snapshot_id,ordinal,name,status,score,weight) VALUES($1,$2,$3,$4,$5,$6)`,
			id, ordinal, dimension.Name, dimension.Status, dimension.Score, dimension.Weight); err != nil {
			return metric.Health{}, fmt.Errorf("insert health dimension: %w", err)
		}
		for factorOrdinal, factor := range dimension.Factors {
			var evidenceID *int64
			if factor.EvidenceID > 0 {
				evidenceID = &factor.EvidenceID
			}
			if _, err := tx.Exec(ctx, `INSERT INTO health_dimension_factors
				(health_snapshot_id,dimension_ordinal,ordinal,name,numeric_value,unit,evidence_id)
				VALUES($1,$2,$3,$4,$5,$6,$7)`, id, ordinal, factorOrdinal, factor.Name,
				factor.Value, factor.Unit, evidenceID); err != nil {
				return metric.Health{}, fmt.Errorf("insert health dimension factor: %w", err)
			}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return metric.Health{}, fmt.Errorf("commit health materialization: %w", err)
	}
	return health, nil
}

func (s *Store) Health(ctx context.Context, principal access.Principal, projectID int64, window metric.Window) (metric.Health, error) {
	if err := s.authorizeProject(ctx, principal, projectID); err != nil {
		return metric.Health{}, err
	}
	return s.loadHealth(ctx, s.pool, projectID, window)
}

func (s *Store) loadHealth(ctx context.Context, query interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Query(context.Context, string, ...any) (pgx.Rows, error)
}, projectID int64, window metric.Window) (metric.Health, error) {
	var health metric.Health
	var id int64
	err := query.QueryRow(ctx, `SELECT id,definition_version,overall_status,overall_score
		FROM health_snapshots WHERE project_id=$1 AND window_from=$2 AND window_to=$3 AND cutoff=$4
		ORDER BY created_at DESC,id DESC LIMIT 1`, projectID, window.From, window.To, window.Cutoff).
		Scan(&id, &health.Version, &health.OverallStatus, &health.Overall)
	if err != nil {
		return metric.Health{}, fmt.Errorf("load health snapshot: %w", err)
	}
	health.ProjectID, health.Window = projectID, window
	rows, err := query.Query(ctx, `SELECT name,status,score,weight FROM health_dimensions
		WHERE health_snapshot_id=$1 ORDER BY ordinal`, id)
	if err != nil {
		return metric.Health{}, fmt.Errorf("load health dimensions: %w", err)
	}
	health.Dimensions = make([]metric.Dimension, 0, 7)
	for rows.Next() {
		var dimension metric.Dimension
		if err := rows.Scan(&dimension.Name, &dimension.Status, &dimension.Score, &dimension.Weight); err != nil {
			return metric.Health{}, fmt.Errorf("scan health dimension: %w", err)
		}
		dimension.Version = health.Version
		health.Dimensions = append(health.Dimensions, dimension)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return metric.Health{}, fmt.Errorf("iterate health dimensions: %w", err)
	}
	rows.Close()
	for index := range health.Dimensions {
		factors, factorErr := loadHealthFactors(ctx, query, id, index)
		if factorErr != nil {
			return metric.Health{}, factorErr
		}
		health.Dimensions[index].Factors = factors
	}
	return health, nil
}

func loadHealthFactors(ctx context.Context, query interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}, healthID int64, dimensionOrdinal int) ([]metric.Factor, error) {
	rows, err := query.Query(ctx, `SELECT name,numeric_value,unit,COALESCE(evidence_id,0)
		FROM health_dimension_factors WHERE health_snapshot_id=$1 AND dimension_ordinal=$2
		ORDER BY ordinal`, healthID, dimensionOrdinal)
	if err != nil {
		return nil, fmt.Errorf("load health dimension factors: %w", err)
	}
	defer rows.Close()
	factors := make([]metric.Factor, 0)
	for rows.Next() {
		var factor metric.Factor
		if err := rows.Scan(&factor.Name, &factor.Value, &factor.Unit, &factor.EvidenceID); err != nil {
			return nil, fmt.Errorf("scan health dimension factor: %w", err)
		}
		factors = append(factors, factor)
	}
	return factors, rows.Err()
}

func normalizeHealth(value metric.Health) metric.Health {
	if len(value.Dimensions) == 0 {
		value.Dimensions = []metric.Dimension{}
	}
	for index := range value.Dimensions {
		if len(value.Dimensions[index].Factors) == 0 {
			value.Dimensions[index].Factors = []metric.Factor{}
		}
	}
	return value
}

type ContributorPage struct {
	ProjectID  int64                     `json:"project_id,string"`
	Window     metric.Window             `json:"window"`
	Summary    contributor.Summary       `json:"summary"`
	Items      []contributor.Contributor `json:"items"`
	HasMore    bool                      `json:"has_more"`
	NextCursor string                    `json:"next_cursor,omitempty"`
}

type IdentityCorrection struct {
	ID               int64  `json:"id,string"`
	AccountID        int64  `json:"account_id,string"`
	FromIdentityID   int64  `json:"from_identity_id,string,omitempty"`
	ToIdentityID     int64  `json:"to_identity_id,string,omitempty"`
	Action           string `json:"action"`
	Reason           string `json:"reason"`
	RequestID        string `json:"request_id"`
	ResultingVersion int64  `json:"resulting_version"`
}

// CorrectContributorIdentity serializes corrections per source account, preserves the immutable
// correction trail, and updates only the current link. The canonical source account is never
// rewritten.
func (s *Store) CorrectContributorIdentity(ctx context.Context, principal access.Principal, accountID,
	targetIdentityID, expectedVersion int64, action, reason, requestID string) (IdentityCorrection, error) {
	if err := access.Authorize(principal, access.ActionProjectWrite); err != nil {
		return IdentityCorrection{}, err
	}
	if accountID <= 0 || expectedVersion <= 0 || (action != "confirm" && action != "split") ||
		requestID == "" || reason == "" || (action == "confirm" && targetIdentityID <= 0) {
		return IdentityCorrection{}, access.ErrInvalidInput
	}
	id, err := s.ids.Next(ctx)
	if err != nil {
		return IdentityCorrection{}, fmt.Errorf("identity correction ID: %w", err)
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return IdentityCorrection{}, fmt.Errorf("begin identity correction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var projectID, currentVersion int64
	var currentIdentityID *int64
	err = tx.QueryRow(ctx, `SELECT a.project_id,l.identity_id,l.version
		FROM contributor_accounts a JOIN contributor_identity_links l ON l.account_id=a.id
		WHERE a.id=$1 FOR UPDATE OF l`, accountID).Scan(&projectID, &currentIdentityID, &currentVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		return IdentityCorrection{}, access.ErrNotFound
	}
	if err != nil {
		return IdentityCorrection{}, fmt.Errorf("lock contributor identity link: %w", err)
	}
	if projectID <= 0 || currentVersion != expectedVersion {
		return IdentityCorrection{}, access.ErrVersionConflict
	}
	if err := s.authorizeProject(ctx, principal, projectID); err != nil {
		return IdentityCorrection{}, err
	}
	if targetIdentityID > 0 {
		var targetProjectID int64
		if err := tx.QueryRow(ctx, `SELECT project_id FROM contributor_identities WHERE id=$1`,
			targetIdentityID).Scan(&targetProjectID); err != nil || targetProjectID != projectID {
			return IdentityCorrection{}, access.ErrInvalidInput
		}
	}
	var fromID int64
	if currentIdentityID != nil {
		fromID = *currentIdentityID
	}
	var target *int64
	status := "unresolved"
	if action == "confirm" {
		target = &targetIdentityID
		status = "analyst_confirmed"
	}
	_, err = tx.Exec(ctx, `INSERT INTO contributor_identity_corrections
		(id,account_id,from_identity_id,to_identity_id,action,reason,actor_id,request_id)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, id, accountID, nullableID(fromID), target, action,
		reason, principal.ActorID, requestID)
	if err != nil {
		return IdentityCorrection{}, fmt.Errorf("insert contributor identity correction: %w", err)
	}
	result, err := tx.Exec(ctx, `UPDATE contributor_identity_links
		SET identity_id=$1,status=$2,version=version+1,updated_at=now()
		WHERE account_id=$3 AND version=$4`, target, status, accountID, expectedVersion)
	if err != nil {
		return IdentityCorrection{}, fmt.Errorf("update contributor identity link: %w", err)
	}
	if result.RowsAffected() != 1 {
		return IdentityCorrection{}, access.ErrVersionConflict
	}
	if err := tx.Commit(ctx); err != nil {
		return IdentityCorrection{}, fmt.Errorf("commit contributor identity correction: %w", err)
	}
	return IdentityCorrection{ID: id, AccountID: accountID, FromIdentityID: fromID,
		ToIdentityID: targetIdentityID, Action: action, Reason: reason, RequestID: requestID,
		ResultingVersion: expectedVersion + 1}, nil
}

func nullableID(value int64) *int64 {
	if value <= 0 {
		return nil
	}
	return &value
}

func (s *Store) Contributors(ctx context.Context, principal access.Principal, projectID int64, window metric.Window, limit, offset int) (ContributorPage, error) {
	if err := s.authorizeProject(ctx, principal, projectID); err != nil {
		return ContributorPage{}, err
	}
	rows, err := s.pool.Query(ctx, `SELECT c.id,c.author_external_id,c.committed_at,c.default_branch,c.merge_commit,
		COALESCE(a.bot,false),COALESCE(l.status,'unresolved'),COALESCE(l.identity_id,0)
		FROM canonical_commits c
		LEFT JOIN contributor_accounts a ON a.source_id=c.source_id AND a.external_id=c.author_external_id
		LEFT JOIN contributor_identity_links l ON l.account_id=a.id
		WHERE c.project_id=$1 AND c.committed_at >= $2 AND c.committed_at < $3
		ORDER BY c.committed_at,c.id`, projectID, window.From, window.To)
	if err != nil {
		return ContributorPage{}, fmt.Errorf("load contributor evidence: %w", err)
	}
	defer rows.Close()
	commits := make([]contributor.Commit, 0)
	for rows.Next() {
		var value contributor.Commit
		var identityID int64
		if err := rows.Scan(&value.ID, &value.AccountID, &value.CommittedAt, &value.DefaultBranch,
			&value.MergeCommit, &value.Bot, &value.LinkStatus, &identityID); err != nil {
			return ContributorPage{}, fmt.Errorf("scan contributor evidence: %w", err)
		}
		if identityID > 0 {
			value.IdentityID = fmt.Sprint(identityID)
		}
		commits = append(commits, value)
	}
	if err := rows.Err(); err != nil {
		return ContributorPage{}, fmt.Errorf("iterate contributor evidence: %w", err)
	}
	summary := contributor.Aggregate(commits, metric.AsTimeWindow(window))
	items := summary.Contributors
	if offset > len(items) {
		offset = len(items)
	}
	end := min(len(items), offset+limit)
	page := slices.Clone(items[offset:end])
	summary.Contributors = nil
	nextCursor := ""
	if end < len(items) {
		nextCursor = encodeOffset(end)
	}
	return ContributorPage{ProjectID: projectID, Window: window, Summary: summary, Items: page,
		HasMore: end < len(items), NextCursor: nextCursor}, nil
}

func encodeOffset(offset int) string {
	return base64.RawURLEncoding.EncodeToString([]byte(strconv.Itoa(offset)))
}

func (s *Store) SaveComparison(ctx context.Context, principal access.Principal, value comparison.Comparison) (comparison.Comparison, error) {
	if err := access.Authorize(principal, access.ActionIntelligenceRead); err != nil {
		return comparison.Comparison{}, err
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
	if err != nil {
		return comparison.Comparison{}, fmt.Errorf("begin comparison materialization: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	_, err = tx.Exec(ctx, `INSERT INTO comparisons
		(id,workspace_id,window_from,window_to,cutoff,definition_set,project_boundary,created_by)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, value.ID, principal.Workspace, value.Window.From,
		value.Window.To, value.Window.Cutoff, "metric-catalog-v1", value.ProjectIDs, principal.ActorID)
	if err != nil {
		return comparison.Comparison{}, fmt.Errorf("insert comparison: %w", err)
	}
	for ordinal, row := range value.Rows {
		for _, cell := range row.Cells {
			if _, err := tx.Exec(ctx, `INSERT INTO comparison_items
				(comparison_id,metric_name,project_id,ordinal,status,numeric_value,unit,definition_version)
				VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, value.ID, row.Metric, cell.ProjectID, ordinal,
				cell.Status, cell.Value, row.Unit, cell.Version); err != nil {
				return comparison.Comparison{}, fmt.Errorf("insert comparison item: %w", err)
			}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return comparison.Comparison{}, fmt.Errorf("commit comparison materialization: %w", err)
	}
	return value, nil
}

func (s *Store) Comparison(ctx context.Context, principal access.Principal, comparisonID int64) (comparison.Comparison, error) {
	if err := access.Authorize(principal, access.ActionIntelligenceRead); err != nil {
		return comparison.Comparison{}, err
	}
	var value comparison.Comparison
	err := s.pool.QueryRow(ctx, `SELECT project_boundary,window_from,window_to,cutoff,created_at
		FROM comparisons WHERE id=$1 AND workspace_id=$2`, comparisonID, principal.Workspace).
		Scan(&value.ProjectIDs, &value.Window.From, &value.Window.To, &value.Window.Cutoff, &value.CreatedAt)
	if err != nil {
		return comparison.Comparison{}, fmt.Errorf("load comparison: %w", err)
	}
	value.ID = comparisonID
	rows, err := s.pool.Query(ctx, `SELECT metric_name,unit,project_id,status,numeric_value,definition_version
		FROM comparison_items WHERE comparison_id=$1
		ORDER BY ordinal,array_position($2::bigint[],project_id)`, comparisonID, value.ProjectIDs)
	if err != nil {
		return comparison.Comparison{}, fmt.Errorf("load comparison items: %w", err)
	}
	defer rows.Close()
	byMetric := make(map[string]int)
	for rows.Next() {
		var name, unit string
		var cell comparison.Cell
		if err := rows.Scan(&name, &unit, &cell.ProjectID, &cell.Status, &cell.Value, &cell.Version); err != nil {
			return comparison.Comparison{}, fmt.Errorf("scan comparison item: %w", err)
		}
		index, ok := byMetric[name]
		if !ok {
			index = len(value.Rows)
			byMetric[name] = index
			value.Rows = append(value.Rows, comparison.Row{Metric: name, Unit: unit})
		}
		value.Rows[index].Cells = append(value.Rows[index].Cells, cell)
	}
	return value, rows.Err()
}

func (s *Store) authorizeProject(ctx context.Context, principal access.Principal, projectID int64) error {
	if err := access.Authorize(principal, access.ActionIntelligenceRead); err != nil {
		return err
	}
	var state string
	err := s.pool.QueryRow(ctx, `SELECT state FROM projects WHERE id=$1 AND workspace_id=$2`, projectID, principal.Workspace).Scan(&state)
	if errors.Is(err, pgx.ErrNoRows) {
		return access.ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("authorize Project intelligence: %w", err)
	}
	if state == "deleted" || state == "deleting" {
		return access.ErrNotFound
	}
	return nil
}

func hasNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows) || (err != nil && errors.Is(errors.Unwrap(err), pgx.ErrNoRows))
}
