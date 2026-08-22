// Package projectstore persists Projects and durable work in PostgreSQL.
package projectstore

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/leohteixeira/opensource-project-intelligence/internal/access"
	"github.com/leohteixeira/opensource-project-intelligence/internal/job"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/database"
	"github.com/leohteixeira/opensource-project-intelligence/internal/project"
)

const (
	defaultPageLimit   = 50
	maxRepositories    = 20
	maxSources         = 50
	registrationRoute  = "POST /api/v1/projects"
	defaultInitialDays = project.DefaultHistoryDays
)

type IDSource interface {
	Next(context.Context) (int64, error)
}

type Store struct {
	pool *pgxpool.Pool
	ids  IDSource
	now  func() time.Time
}

type Registration struct {
	Project project.Project `json:"project"`
	Job     job.Job         `json:"job"`
	Replay  bool            `json:"replay"`
}

type JobEvent struct {
	ID  int64   `json:"id,string"`
	Job job.Job `json:"job"`
}

type Filter struct {
	State  string
	Query  string
	Kind   string
	Limit  int
	Offset int
}

type Portfolio struct {
	Projects       []project.Project `json:"projects"`
	AttentionCount int               `json:"attention_count"`
	ActiveJobs     []job.Job         `json:"active_jobs"`
	GeneratedAt    time.Time         `json:"generated_at"`
}

func New(pool *database.Pool, ids IDSource) *Store {
	return &Store{pool: pool.Unwrap(), ids: ids, now: time.Now}
}

func (s *Store) Register(
	ctx context.Context,
	principal access.Principal,
	repositoryURL string,
	historyDays int,
	idempotencyKey, requestID string,
) (Registration, error) {
	if err := access.Authorize(principal, access.ActionProjectWrite); err != nil {
		return Registration{}, err
	}
	if strings.TrimSpace(idempotencyKey) == "" {
		return Registration{}, fmt.Errorf("%w: Idempotency-Key is required", project.ErrInvalid)
	}
	provider, _, repositoryName, canonicalURL, err := project.CanonicalRepositoryURL(repositoryURL)
	if err != nil {
		return Registration{}, err
	}
	if historyDays <= 0 {
		historyDays = defaultInitialDays
	}
	if historyDays > 3650 {
		return Registration{}, fmt.Errorf("%w: history exceeds the operational limit", project.ErrInvalid)
	}
	slug, err := project.Slug(repositoryName)
	if err != nil {
		return Registration{}, err
	}
	digest := sha256.Sum256([]byte(canonicalURL + "\x00" + fmt.Sprint(historyDays)))

	// The idempotency key and canonical URL are independent convergence keys:
	// one key cannot be reused for another request, while different requests for
	// one public repository must all resolve the already-created Project. Take
	// transaction-scoped locks before reading either key so concurrent callers
	// observe the winner instead of surfacing a uniqueness race.
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Registration{}, fmt.Errorf("begin project registration: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, $2))`,
		"registration-idempotency:"+fmt.Sprint(principal.Workspace)+":"+
			fmt.Sprint(principal.ActorID)+":"+idempotencyKey, int64(0)); err != nil {
		return Registration{}, fmt.Errorf("lock registration idempotency: %w", err)
	}
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, $2))`,
		"registration-repository:"+canonicalURL, int64(0)); err != nil {
		return Registration{}, fmt.Errorf("lock canonical repository: %w", err)
	}

	var replayProjectID, replayJobID int64
	var replayDigest []byte
	err = tx.QueryRow(ctx, `
		SELECT request_digest, COALESCE(resource_id, 0), COALESCE(job_id, 0)
		FROM idempotency_records
		WHERE workspace_id = $1 AND actor_id = $2 AND route = $3 AND idempotency_key = $4
		FOR UPDATE`, principal.Workspace, principal.ActorID, registrationRoute, idempotencyKey).
		Scan(&replayDigest, &replayProjectID, &replayJobID)
	if err == nil {
		if !equalDigest(replayDigest, digest[:]) {
			return Registration{}, fmt.Errorf("%w: idempotency key was used with another request", project.ErrConflict)
		}
		if replayProjectID == 0 || replayJobID == 0 {
			return Registration{}, fmt.Errorf("%w: matching registration is still in progress", project.ErrConflict)
		}
		registered, loadErr := s.registrationTx(ctx, tx, replayProjectID, replayJobID)
		if loadErr != nil {
			return Registration{}, loadErr
		}
		registered.Replay = true
		return registered, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Registration{}, fmt.Errorf("read registration idempotency: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO idempotency_records
			(workspace_id, actor_id, route, idempotency_key, request_digest)
		VALUES ($1, $2, $3, $4, $5)`,
		principal.Workspace, principal.ActorID, registrationRoute, idempotencyKey, digest[:])
	if err != nil {
		return Registration{}, mapConflict("reserve registration idempotency", err)
	}

	var existingProjectID, existingJobID int64
	err = tx.QueryRow(ctx, `
		SELECT r.project_id, j.id
		FROM repositories r
		JOIN projects p ON p.id = r.project_id
		JOIN LATERAL (
			SELECT id FROM jobs WHERE project_id = p.id ORDER BY created_at, id LIMIT 1
		) j ON true
		WHERE r.canonical_url = $1 AND p.workspace_id = $2`, canonicalURL, principal.Workspace).
		Scan(&existingProjectID, &existingJobID)
	if err == nil {
		existing, loadErr := s.registrationTx(ctx, tx, existingProjectID, existingJobID)
		if loadErr != nil {
			return Registration{}, loadErr
		}
		existing.Replay = true
		responseBody, _ := json.Marshal(map[string]any{
			"project_id": existingProjectID,
			"job_id":     existingJobID,
			"existing":   true,
		})
		if _, updateErr := tx.Exec(ctx, `
			UPDATE idempotency_records SET response_status = 200, response_body = $1,
				resource_id = $2, job_id = $3, completed_at = now()
			WHERE workspace_id = $4 AND actor_id = $5 AND route = $6 AND idempotency_key = $7`,
			responseBody, existingProjectID, existingJobID, principal.Workspace, principal.ActorID,
			registrationRoute, idempotencyKey); updateErr != nil {
			return Registration{}, fmt.Errorf("complete existing registration idempotency: %w", updateErr)
		}
		if commitErr := tx.Commit(ctx); commitErr != nil {
			return Registration{}, fmt.Errorf("commit existing registration resolution: %w", commitErr)
		}
		return existing, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Registration{}, fmt.Errorf("check canonical repository: %w", err)
	}
	var projectCount int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM projects WHERE workspace_id = $1 AND state <> 'deleted'`,
		principal.Workspace).Scan(&projectCount); err != nil {
		return Registration{}, fmt.Errorf("count projects: %w", err)
	}
	if projectCount >= 1000 {
		return Registration{}, fmt.Errorf("%w: project quota reached", project.ErrConflict)
	}

	projectID, repositoryID, sourceID, jobID, err := s.nextFour(ctx)
	if err != nil {
		return Registration{}, err
	}
	now := s.now().UTC()
	primary := project.Repository{
		ID: repositoryID, ProjectID: projectID, Provider: provider,
		CanonicalURL: canonicalURL, Role: project.RolePrimary, Version: 1,
	}
	created, err := project.New(projectID, principal.Workspace, repositoryName, slug, primary)
	if err != nil {
		return Registration{}, err
	}
	initialRange := project.InitialHistoryRange(now, historyDays)
	queued, err := job.New(jobID, projectID, "project_initial_sync", "sources", nil, true, now)
	if err != nil {
		return Registration{}, err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO projects (id, workspace_id, name, slug, state) VALUES ($1, $2, $3, $4, 'active')`,
		projectID, principal.Workspace, created.Name, created.Slug); err != nil {
		return Registration{}, mapConflict("create project", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO repositories (id, project_id, provider, canonical_url, role)
		VALUES ($1, $2, $3, $4, 'primary')`, repositoryID, projectID, provider, canonicalURL); err != nil {
		return Registration{}, mapConflict("create primary repository", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO sources
			(id, project_id, repository_id, kind, canonical_url, coverage_from, coverage_to)
		VALUES ($1, $2, $3, $4, $5, $6, NULL)`,
		sourceID, projectID, repositoryID, provider, canonicalURL, initialRange.From); err != nil {
		return Registration{}, fmt.Errorf("create repository source: %w", err)
	}
	if err := insertJob(ctx, tx, queued, principal, requestID, "initial:"+fmt.Sprint(projectID)); err != nil {
		return Registration{}, err
	}
	if err := s.recordJobEvent(ctx, tx, queued); err != nil {
		return Registration{}, err
	}
	if err := s.writeOutbox(ctx, tx, projectID, jobID, "project.registered", principal, requestID,
		map[string]any{"project_id": projectID, "job_id": jobID}); err != nil {
		return Registration{}, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO public_catalog_projects (id, name, slug, source_links, state)
		VALUES ($1, $2, $3, jsonb_build_array($4::text), 'active')`,
		projectID, created.Name, created.Slug, canonicalURL); err != nil {
		return Registration{}, fmt.Errorf("publish project catalog row: %w", err)
	}
	if err := s.audit(ctx, tx, principal, "project.register", "project", projectID, requestID,
		map[string]any{"repository_url": canonicalURL, "history_days": historyDays}); err != nil {
		return Registration{}, err
	}
	responseBody, _ := json.Marshal(map[string]any{"project_id": projectID, "job_id": jobID})
	if _, err := tx.Exec(ctx, `
		UPDATE idempotency_records SET response_status = 202, response_body = $1,
			resource_id = $2, job_id = $3, completed_at = now()
		WHERE workspace_id = $4 AND actor_id = $5 AND route = $6 AND idempotency_key = $7`,
		responseBody, projectID, jobID, principal.Workspace, principal.ActorID,
		registrationRoute, idempotencyKey); err != nil {
		return Registration{}, fmt.Errorf("complete registration idempotency: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Registration{}, mapConflict("commit project registration", err)
	}
	created.Sources = []project.Source{{
		ID: sourceID, ProjectID: projectID, Kind: project.SourceKind(provider),
		CanonicalURL: canonicalURL, State: project.SourceAvailable, Public: true,
		CoverageFrom: &initialRange.From, Version: 1,
	}}
	return Registration{Project: created, Job: queued}, nil
}

func (s *Store) Portfolio(ctx context.Context, principal access.Principal) (Portfolio, error) {
	if err := access.Authorize(principal, access.ActionIntelligenceRead); err != nil {
		return Portfolio{}, err
	}
	projects, err := s.ListProjects(ctx, principal, Filter{State: "active", Limit: 12})
	if err != nil {
		return Portfolio{}, err
	}
	if len(projects) > 12 {
		projects = projects[:12]
	}
	jobs, err := s.ListJobs(ctx, principal, 0, Filter{State: "active", Limit: 12})
	if err != nil {
		return Portfolio{}, err
	}
	if len(jobs) > 12 {
		jobs = jobs[:12]
	}
	var attentionCount int
	err = s.pool.QueryRow(ctx, `
		SELECT count(*)
		FROM projects p
		WHERE p.workspace_id=$1 AND p.state='active' AND (
			EXISTS (SELECT 1 FROM sources s WHERE s.project_id=p.id AND s.state='unavailable') OR
			EXISTS (SELECT 1 FROM source_associations a WHERE a.project_id=p.id AND a.status='unresolved') OR
			EXISTS (SELECT 1 FROM jobs j WHERE j.project_id=p.id AND j.state IN ('failed','partial'))
		)`, principal.Workspace).Scan(&attentionCount)
	if err != nil {
		return Portfolio{}, fmt.Errorf("count projects requiring attention: %w", err)
	}
	return Portfolio{
		Projects: projects, AttentionCount: attentionCount,
		ActiveJobs: jobs, GeneratedAt: s.now().UTC(),
	}, nil
}

func (s *Store) ListProjects(ctx context.Context, principal access.Principal, filter Filter) ([]project.Project, error) {
	if err := access.Authorize(principal, access.ActionIntelligenceRead); err != nil {
		return nil, err
	}
	limit, offset := page(filter.Limit, filter.Offset)
	rows, err := s.pool.Query(ctx, `
		SELECT id, workspace_id, name, slug, description, state, version, deleted_at
		FROM projects
		WHERE workspace_id = $1
		  AND ($2 = '' OR state = $2)
		  AND ($3 = '' OR name ILIKE '%' || $3 || '%' OR slug ILIKE '%' || $3 || '%')
		ORDER BY updated_at DESC, id DESC LIMIT $4 OFFSET $5`,
		principal.Workspace, filter.State, strings.TrimSpace(filter.Query), limit+1, offset)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()
	projects := make([]project.Project, 0, limit)
	for rows.Next() {
		value, scanErr := scanProject(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		projects = append(projects, value)
	}
	return projects, rows.Err()
}

func (s *Store) GetProject(ctx context.Context, principal access.Principal, id int64) (project.Project, error) {
	if err := access.Authorize(principal, access.ActionIntelligenceRead); err != nil {
		return project.Project{}, err
	}
	value, err := scanProject(s.pool.QueryRow(ctx, `
		SELECT id, workspace_id, name, slug, description, state, version, deleted_at
		FROM projects WHERE id = $1 AND workspace_id = $2
			AND state NOT IN ('deleting','deleted')`, id, principal.Workspace))
	if err != nil {
		return project.Project{}, mapNotFound("get project", err)
	}
	value.Repositories, err = s.ListRepositories(ctx, principal, id, Filter{})
	if err != nil {
		return project.Project{}, err
	}
	value.Sources, err = s.ListSources(ctx, principal, id, Filter{})
	return value, err
}

func (s *Store) UpdateProject(
	ctx context.Context,
	principal access.Principal,
	id, expectedVersion int64,
	name, description, requestID string,
) (project.Project, error) {
	if err := access.Authorize(principal, access.ActionProjectWrite); err != nil {
		return project.Project{}, err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return project.Project{}, fmt.Errorf("%w: project name is required", project.ErrInvalid)
	}
	var value project.Project
	err := s.withTx(ctx, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, `
			UPDATE projects SET name = $1, description = $2, version = version + 1, updated_at = now()
			WHERE id = $3 AND workspace_id = $4 AND version = $5 AND state IN ('active', 'paused')
			RETURNING id, workspace_id, name, slug, description, state, version, deleted_at`,
			name, description, id, principal.Workspace, expectedVersion)
		var err error
		value, err = scanProject(row)
		if errors.Is(err, pgx.ErrNoRows) {
			return project.ErrVersionConflict
		}
		if err != nil {
			return fmt.Errorf("update project: %w", err)
		}
		_, err = tx.Exec(ctx, `UPDATE public_catalog_projects SET name=$1, description=$2,
			updated_at=now() WHERE id=$3`, name, description, id)
		if err != nil {
			return fmt.Errorf("update catalog project: %w", err)
		}
		if err := s.audit(ctx, tx, principal, "project.update", "project", id, requestID,
			map[string]any{"name": name}); err != nil {
			return err
		}
		return s.writeOutbox(ctx, tx, id, 0, "project.updated", principal, requestID,
			map[string]any{"project_id": id, "version": value.Version})
	})
	return value, err
}

func (s *Store) Transition(
	ctx context.Context,
	principal access.Principal,
	id, expectedVersion int64,
	to project.State,
	reason string,
	requestID string,
) (Registration, error) {
	if err := access.Authorize(principal, access.ActionProjectLifecycle); err != nil {
		return Registration{}, err
	}
	current, err := s.GetProject(ctx, principal, id)
	if err != nil {
		return Registration{}, err
	}
	next, err := current.Transition(to, expectedVersion, true)
	if err != nil {
		return Registration{}, err
	}
	if next.Version == current.Version {
		return Registration{Project: current}, nil
	}
	jobID, err := s.ids.Next(ctx)
	if err != nil {
		return Registration{}, fmt.Errorf("issue lifecycle job ID: %w", err)
	}
	queued, err := job.New(jobID, id, "project_transition", "effects", nil, false, s.now())
	if err != nil {
		return Registration{}, err
	}
	err = s.withTx(ctx, func(tx pgx.Tx) error {
		command, commandErr := tx.Exec(ctx, `
			UPDATE projects SET state=$1, version=$2, updated_at=now(),
				unavailable_at=CASE WHEN $1 IN ('deleting','deleted') THEN COALESCE(unavailable_at, now()) ELSE NULL END,
				deleted_at=CASE WHEN $1='deleted' THEN now() ELSE NULL END
			WHERE id=$3 AND workspace_id=$4 AND version=$5`,
			to, next.Version, id, principal.Workspace, expectedVersion)
		if commandErr != nil {
			return fmt.Errorf("transition project: %w", commandErr)
		}
		if command.RowsAffected() != 1 {
			return project.ErrVersionConflict
		}
		catalogState := string(to)
		if to == project.StateDeleting {
			catalogState = "deleted"
		}
		if _, commandErr = tx.Exec(ctx, `UPDATE public_catalog_projects SET state=$1, updated_at=now() WHERE id=$2`,
			catalogState, id); commandErr != nil {
			return fmt.Errorf("transition catalog project: %w", commandErr)
		}
		if commandErr = insertJob(ctx, tx, queued, principal, requestID,
			fmt.Sprintf("transition:%d:%s:%d", id, to, next.Version)); commandErr != nil {
			return commandErr
		}
		if commandErr = s.recordJobEvent(ctx, tx, queued); commandErr != nil {
			return commandErr
		}
		if commandErr = s.audit(ctx, tx, principal, "project.transition", "project", id, requestID,
			map[string]any{"from": current.State, "to": to, "reason": strings.TrimSpace(reason)}); commandErr != nil {
			return commandErr
		}
		return s.writeOutbox(ctx, tx, id, jobID, "project.transitioned", principal, requestID,
			map[string]any{"project_id": id, "from": current.State, "to": to, "job_id": jobID})
	})
	return Registration{Project: next, Job: queued}, err
}

func (s *Store) RequestDeletion(
	ctx context.Context,
	principal access.Principal,
	id, expectedVersion int64,
	confirmation, reason, requestID string,
) (Registration, error) {
	current, err := s.GetProject(ctx, principal, id)
	if err != nil {
		return Registration{}, err
	}
	if confirmation != "DELETE "+current.Slug {
		return Registration{}, fmt.Errorf("%w: exact project identity confirmation is required", project.ErrInvalid)
	}
	next, err := current.Transition(project.StateDeleting, expectedVersion, principal.Role == access.RoleAdmin)
	if err != nil {
		return Registration{}, err
	}
	jobID, err := s.ids.Next(ctx)
	if err != nil {
		return Registration{}, fmt.Errorf("issue deletion job ID: %w", err)
	}
	queued, err := job.New(jobID, id, "project_purge", "objects", nil, false, s.now())
	if err != nil {
		return Registration{}, err
	}
	err = s.withTx(ctx, func(tx pgx.Tx) error {
		command, commandErr := tx.Exec(ctx, `
			UPDATE projects SET state='deleting', version=$1, unavailable_at=COALESCE(unavailable_at, now()),
				deletion_actor_id=$2, deletion_reason=$3, updated_at=now()
			WHERE id=$4 AND workspace_id=$5 AND version=$6`,
			next.Version, principal.ActorID, strings.TrimSpace(reason), id, principal.Workspace, expectedVersion)
		if commandErr != nil {
			return fmt.Errorf("mark project deleting: %w", commandErr)
		}
		if command.RowsAffected() != 1 {
			return project.ErrVersionConflict
		}
		if _, commandErr = tx.Exec(ctx, `UPDATE public_catalog_projects SET state='deleted', updated_at=now() WHERE id=$1`, id); commandErr != nil {
			return fmt.Errorf("remove project from catalog: %w", commandErr)
		}
		if commandErr = insertJob(ctx, tx, queued, principal, requestID, "purge:"+fmt.Sprint(id)); commandErr != nil {
			return commandErr
		}
		if commandErr = s.recordJobEvent(ctx, tx, queued); commandErr != nil {
			return commandErr
		}
		return s.writeOutbox(ctx, tx, id, jobID, "project.deletion_requested", principal, requestID,
			map[string]any{"project_id": id, "job_id": jobID})
	})
	return Registration{Project: next, Job: queued}, err
}

func (s *Store) ListRepositories(
	ctx context.Context,
	principal access.Principal,
	projectID int64,
	filter Filter,
) ([]project.Repository, error) {
	if err := access.Authorize(principal, access.ActionIntelligenceRead); err != nil {
		return nil, err
	}
	limit, offset := page(filter.Limit, filter.Offset)
	rows, err := s.pool.Query(ctx, `
		SELECT r.id, r.project_id, r.provider, r.canonical_url, r.role, r.version
		FROM repositories r JOIN projects p ON p.id=r.project_id
		WHERE r.project_id=$1 AND p.workspace_id=$2 ORDER BY r.role='primary' DESC, r.id LIMIT $3 OFFSET $4`,
		projectID, principal.Workspace, limit+1, offset)
	if err != nil {
		return nil, fmt.Errorf("list repositories: %w", err)
	}
	defer rows.Close()
	values := make([]project.Repository, 0, limit)
	for rows.Next() {
		var value project.Repository
		if err := rows.Scan(&value.ID, &value.ProjectID, &value.Provider, &value.CanonicalURL,
			&value.Role, &value.Version); err != nil {
			return nil, fmt.Errorf("scan repository: %w", err)
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

func (s *Store) AddRepository(
	ctx context.Context,
	principal access.Principal,
	projectID int64,
	rawURL string,
	role project.RepositoryRole,
	requestID string,
) (project.Repository, error) {
	if err := access.Authorize(principal, access.ActionProjectWrite); err != nil {
		return project.Repository{}, err
	}
	provider, _, _, canonical, err := project.CanonicalRepositoryURL(rawURL)
	if err != nil {
		return project.Repository{}, err
	}
	repositoryID, err := s.ids.Next(ctx)
	if err != nil {
		return project.Repository{}, fmt.Errorf("issue repository ID: %w", err)
	}
	value := project.Repository{ID: repositoryID, ProjectID: projectID, Provider: provider,
		CanonicalURL: canonical, Role: role, Version: 1}
	err = s.withTx(ctx, func(tx pgx.Tx) error {
		current, loadErr := loadProjectForUpdate(ctx, tx, principal.Workspace, projectID)
		if loadErr != nil {
			return loadErr
		}
		if len(current.Repositories) >= maxRepositories {
			return fmt.Errorf("%w: repository limit reached", project.ErrConflict)
		}
		updated, ruleErr := current.AddRepository(value, maxRepositories)
		if ruleErr != nil {
			return ruleErr
		}
		for _, existing := range current.Repositories {
			if strings.EqualFold(existing.CanonicalURL, canonical) {
				value = existing
				return nil
			}
		}
		if role == project.RolePrimary {
			if _, loadErr = tx.Exec(ctx, `UPDATE repositories SET role='core', version=version+1,
				updated_at=now() WHERE project_id=$1 AND role='primary'`, projectID); loadErr != nil {
				return fmt.Errorf("replace primary repository: %w", loadErr)
			}
		}
		if _, loadErr = tx.Exec(ctx, `INSERT INTO repositories
			(id, project_id, provider, canonical_url, role) VALUES ($1,$2,$3,$4,$5)`,
			value.ID, projectID, provider, canonical, role); loadErr != nil {
			return mapConflict("attach repository", loadErr)
		}
		if _, loadErr = tx.Exec(ctx, `UPDATE projects SET version=$1, updated_at=now() WHERE id=$2`,
			updated.Version, projectID); loadErr != nil {
			return fmt.Errorf("version project repositories: %w", loadErr)
		}
		return s.writeOutbox(ctx, tx, projectID, 0, "repository.attached", principal, requestID, value)
	})
	return value, err
}

func (s *Store) ChangeRepositoryRole(
	ctx context.Context,
	principal access.Principal,
	projectID, repositoryID, expectedVersion int64,
	role project.RepositoryRole,
	requestID string,
) (project.Repository, error) {
	if err := access.Authorize(principal, access.ActionProjectWrite); err != nil {
		return project.Repository{}, err
	}
	var value project.Repository
	err := s.withTx(ctx, func(tx pgx.Tx) error {
		current, err := loadProjectForUpdate(ctx, tx, principal.Workspace, projectID)
		if err != nil {
			return err
		}
		updated, err := current.ChangeRepositoryRole(repositoryID, role)
		if err != nil {
			return err
		}
		if role == project.RolePrimary {
			if _, err = tx.Exec(ctx, `UPDATE repositories SET role='core', version=version+1,
				updated_at=now() WHERE project_id=$1 AND role='primary' AND id<>$2`, projectID, repositoryID); err != nil {
				return fmt.Errorf("replace primary repository: %w", err)
			}
		}
		err = tx.QueryRow(ctx, `UPDATE repositories SET role=$1, version=version+1, updated_at=now()
			WHERE id=$2 AND project_id=$3 AND version=$4 RETURNING id,project_id,provider,canonical_url,role,version`,
			role, repositoryID, projectID, expectedVersion).Scan(&value.ID, &value.ProjectID, &value.Provider,
			&value.CanonicalURL, &value.Role, &value.Version)
		if errors.Is(err, pgx.ErrNoRows) {
			return project.ErrVersionConflict
		}
		if err != nil {
			return fmt.Errorf("change repository role: %w", err)
		}
		_, err = tx.Exec(ctx, `UPDATE projects SET version=$1,updated_at=now() WHERE id=$2`, updated.Version, projectID)
		if err != nil {
			return fmt.Errorf("version project repositories: %w", err)
		}
		return s.writeOutbox(ctx, tx, projectID, 0, "repository.role_changed", principal, requestID, value)
	})
	return value, err
}

func (s *Store) RemoveRepository(
	ctx context.Context,
	principal access.Principal,
	projectID, repositoryID, expectedVersion int64,
	requestID string,
) error {
	if err := access.Authorize(principal, access.ActionProjectWrite); err != nil {
		return err
	}
	return s.withTx(ctx, func(tx pgx.Tx) error {
		current, err := loadProjectForUpdate(ctx, tx, principal.Workspace, projectID)
		if err != nil {
			return err
		}
		updated, err := current.RemoveRepository(repositoryID)
		if err != nil {
			return err
		}
		command, err := tx.Exec(ctx, `DELETE FROM repositories WHERE id=$1 AND project_id=$2 AND version=$3`,
			repositoryID, projectID, expectedVersion)
		if err != nil {
			return fmt.Errorf("remove repository: %w", err)
		}
		if command.RowsAffected() == 0 {
			return project.ErrVersionConflict
		}
		_, err = tx.Exec(ctx, `UPDATE projects SET version=$1,updated_at=now() WHERE id=$2`, updated.Version, projectID)
		if err != nil {
			return fmt.Errorf("version project repositories: %w", err)
		}
		return s.writeOutbox(ctx, tx, projectID, 0, "repository.removed", principal, requestID,
			map[string]any{"repository_id": repositoryID})
	})
}

func (s *Store) ListSources(
	ctx context.Context,
	principal access.Principal,
	projectID int64,
	filter Filter,
) ([]project.Source, error) {
	if err := access.Authorize(principal, access.ActionIntelligenceRead); err != nil {
		return nil, err
	}
	limit, offset := page(filter.Limit, filter.Offset)
	rows, err := s.pool.Query(ctx, `
		SELECT s.id,s.project_id,s.kind,s.canonical_url,s.state,s.public,s.coverage_from,s.coverage_to,
			s.last_attempt_at,s.last_success_at,s.next_run_at,s.failure_code,s.version
		FROM sources s JOIN projects p ON p.id=s.project_id
		WHERE s.project_id=$1 AND p.workspace_id=$2 AND ($3='' OR s.kind=$3) AND ($4='' OR s.state=$4)
		ORDER BY s.kind,s.id LIMIT $5 OFFSET $6`,
		projectID, principal.Workspace, filter.Kind, filter.State, limit+1, offset)
	if err != nil {
		return nil, fmt.Errorf("list sources: %w", err)
	}
	defer rows.Close()
	values := make([]project.Source, 0, limit)
	for rows.Next() {
		value, scanErr := scanSource(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

func (s *Store) AddSource(
	ctx context.Context,
	principal access.Principal,
	projectID int64,
	kind project.SourceKind,
	canonicalURL, requestID string,
) (project.Source, error) {
	if err := access.Authorize(principal, access.ActionProjectWrite); err != nil {
		return project.Source{}, err
	}
	if !project.ValidSourceKind(kind) || strings.TrimSpace(canonicalURL) == "" {
		return project.Source{}, fmt.Errorf("%w: supported source kind and URL are required", project.ErrInvalid)
	}
	sourceID, err := s.ids.Next(ctx)
	if err != nil {
		return project.Source{}, fmt.Errorf("issue source ID: %w", err)
	}
	value := project.Source{ID: sourceID, ProjectID: projectID, Kind: kind, CanonicalURL: canonicalURL,
		State: project.SourceAvailable, Public: true, Version: 1}
	err = s.withTx(ctx, func(tx pgx.Tx) error {
		current, err := loadProjectForUpdate(ctx, tx, principal.Workspace, projectID)
		if err != nil {
			return err
		}
		if current.State != project.StateActive && current.State != project.StatePaused {
			return fmt.Errorf("%w: project is read-only", project.ErrConflict)
		}
		var count int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM sources WHERE project_id=$1 AND state<>'removed'`, projectID).Scan(&count); err != nil {
			return fmt.Errorf("count project sources: %w", err)
		}
		if count >= maxSources {
			return fmt.Errorf("%w: source limit reached", project.ErrConflict)
		}
		err = tx.QueryRow(ctx, `INSERT INTO sources
			(id,project_id,kind,canonical_url,state,public) VALUES ($1,$2,$3,$4,'available',true)
			ON CONFLICT (project_id,kind,canonical_url) DO UPDATE SET state='available',public=true,
				failure_code='',version=sources.version+1,updated_at=now()
			RETURNING id,project_id,kind,canonical_url,state,public,coverage_from,coverage_to,
				last_attempt_at,last_success_at,next_run_at,failure_code,version`,
			sourceID, projectID, kind, canonicalURL).Scan(&value.ID, &value.ProjectID, &value.Kind,
			&value.CanonicalURL, &value.State, &value.Public, &value.CoverageFrom, &value.CoverageTo,
			&value.LastAttemptAt, &value.LastSuccessAt, &value.NextRunAt, &value.Failure, &value.Version)
		if err != nil {
			return mapConflict("attach source", err)
		}
		return s.writeOutbox(ctx, tx, projectID, 0, "source.attached", principal, requestID, value)
	})
	return value, err
}

func (s *Store) UpdateSource(
	ctx context.Context,
	principal access.Principal,
	projectID, sourceID, expectedVersion int64,
	state project.SourceState,
	requestID string,
) (project.Source, error) {
	if err := access.Authorize(principal, access.ActionProjectWrite); err != nil {
		return project.Source{}, err
	}
	if state != project.SourceAvailable && state != project.SourcePaused && state != project.SourceRemoved {
		return project.Source{}, fmt.Errorf("%w: unsupported source state", project.ErrInvalid)
	}
	var value project.Source
	err := s.withTx(ctx, func(tx pgx.Tx) error {
		current, err := loadProjectForUpdate(ctx, tx, principal.Workspace, projectID)
		if err != nil {
			return err
		}
		if current.State == project.StateArchived || current.State == project.StateDeleting || current.State == project.StateDeleted {
			return fmt.Errorf("%w: project is read-only", project.ErrConflict)
		}
		value, err = scanSource(tx.QueryRow(ctx, `UPDATE sources SET state=$1,version=version+1,
			updated_at=now(),next_run_at=CASE WHEN $1='paused' THEN NULL ELSE next_run_at END
			WHERE id=$2 AND project_id=$3 AND version=$4
			RETURNING id,project_id,kind,canonical_url,state,public,coverage_from,coverage_to,
				last_attempt_at,last_success_at,next_run_at,failure_code,version`,
			state, sourceID, projectID, expectedVersion))
		if errors.Is(err, pgx.ErrNoRows) {
			return project.ErrVersionConflict
		}
		if err != nil {
			return err
		}
		return s.writeOutbox(ctx, tx, projectID, 0, "source.updated", principal, requestID, value)
	})
	return value, err
}

func (s *Store) RemoveSource(
	ctx context.Context,
	principal access.Principal,
	projectID, sourceID, expectedVersion int64,
	requestID string,
) (job.Job, error) {
	if err := access.Authorize(principal, access.ActionProjectWrite); err != nil {
		return job.Job{}, err
	}
	jobID, err := s.ids.Next(ctx)
	if err != nil {
		return job.Job{}, fmt.Errorf("issue source recalculation job ID: %w", err)
	}
	queued, err := job.New(jobID, projectID, "source_recalculation", "snapshots", nil, true, s.now())
	if err != nil {
		return job.Job{}, err
	}
	err = s.withTx(ctx, func(tx pgx.Tx) error {
		current, loadErr := loadProjectForUpdate(ctx, tx, principal.Workspace, projectID)
		if loadErr != nil {
			return loadErr
		}
		if current.State == project.StateArchived || current.State == project.StateDeleting || current.State == project.StateDeleted {
			return fmt.Errorf("%w: project is read-only", project.ErrConflict)
		}
		command, loadErr := tx.Exec(ctx, `UPDATE sources SET state='removed',version=version+1,
			next_run_at=NULL,updated_at=now() WHERE id=$1 AND project_id=$2 AND version=$3`,
			sourceID, projectID, expectedVersion)
		if loadErr != nil {
			return fmt.Errorf("remove source: %w", loadErr)
		}
		if command.RowsAffected() != 1 {
			return project.ErrVersionConflict
		}
		if loadErr = insertJob(ctx, tx, queued, principal, requestID,
			fmt.Sprintf("source-removal:%d:%d", sourceID, expectedVersion)); loadErr != nil {
			return loadErr
		}
		if loadErr = s.recordJobEvent(ctx, tx, queued); loadErr != nil {
			return loadErr
		}
		return s.writeOutbox(ctx, tx, projectID, jobID, "source.removed", principal, requestID,
			map[string]any{"source_id": sourceID, "job_id": jobID})
	})
	return queued, err
}

func (s *Store) QueueSync(
	ctx context.Context,
	principal access.Principal,
	projectID int64,
	scope, requestID string,
) (job.Job, error) {
	if err := access.Authorize(principal, access.ActionProjectWrite); err != nil {
		return job.Job{}, err
	}
	if scope == "" {
		scope = "all"
	}
	if scope != "all" && scope != "repository" && scope != "issues" && scope != "releases" {
		return job.Job{}, fmt.Errorf("%w: unsupported synchronization scope", project.ErrInvalid)
	}
	current, err := s.GetProject(ctx, principal, projectID)
	if err != nil {
		return job.Job{}, err
	}
	if err := current.CanSynchronize(); err != nil {
		return job.Job{}, err
	}
	return s.queueJob(ctx, principal, projectID, "project_sync", scope,
		"sync:"+fmt.Sprint(projectID)+":"+scope, requestID, nil, nil)
}

func (s *Store) QueueHistory(
	ctx context.Context,
	principal access.Principal,
	projectID int64,
	from, to time.Time,
	requestID string,
) (job.Job, error) {
	if err := access.Authorize(principal, access.ActionProjectWrite); err != nil {
		return job.Job{}, err
	}
	current, err := s.GetProject(ctx, principal, projectID)
	if err != nil {
		return job.Job{}, err
	}
	if err := current.CanSynchronize(); err != nil {
		return job.Job{}, err
	}
	rangeValue, err := project.ValidateHistoryRange(from, to, s.now(), 3650)
	if err != nil {
		return job.Job{}, err
	}
	var covered bool
	if err := s.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM sources
			WHERE project_id=$1 AND state='available'
				AND coverage_from IS NOT NULL AND coverage_from <= $2
				AND coverage_to IS NOT NULL AND coverage_to >= $3
		)`, projectID, rangeValue.From, rangeValue.To).Scan(&covered); err != nil {
		return job.Job{}, fmt.Errorf("check history coverage: %w", err)
	}
	if covered {
		return job.Job{}, fmt.Errorf("%w: requested history range is already covered", project.ErrConflict)
	}
	key := "history:" + fmt.Sprint(projectID)
	return s.queueJob(ctx, principal, projectID, "project_history", "history", key, requestID,
		&rangeValue.From, &rangeValue.To)
}

func (s *Store) ListJobs(ctx context.Context, principal access.Principal, projectID int64, filter Filter) ([]job.Job, error) {
	if err := access.Authorize(principal, access.ActionIntelligenceRead); err != nil {
		return nil, err
	}
	limit, offset := page(filter.Limit, filter.Offset)
	stateClause := filter.State
	rows, err := s.pool.Query(ctx, `
		SELECT id,COALESCE(project_id,0),kind,state,progress,checkpoint,requested_from,requested_to,version,coalesced_requests,
			cancellable,created_at,started_at,updated_at,finished_at,failure_code
		FROM jobs WHERE workspace_id=$1 AND ($2::bigint=0 OR project_id=$2::bigint)
			AND ($3='' OR ($3='active' AND state IN ('queued','running')) OR state=$3)
			AND ($4='' OR kind=$4)
		ORDER BY created_at DESC,id DESC LIMIT $5 OFFSET $6`,
		principal.Workspace, projectID, stateClause, filter.Kind, limit+1, offset)
	if err != nil {
		return nil, fmt.Errorf("list jobs: %w", err)
	}
	defer rows.Close()
	values := make([]job.Job, 0, limit)
	for rows.Next() {
		value, scanErr := scanJob(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

func (s *Store) GetJob(ctx context.Context, principal access.Principal, id int64) (job.Job, error) {
	if err := access.Authorize(principal, access.ActionIntelligenceRead); err != nil {
		return job.Job{}, err
	}
	value, err := scanJob(s.pool.QueryRow(ctx, `
		SELECT id,COALESCE(project_id,0),kind,state,progress,checkpoint,requested_from,requested_to,version,coalesced_requests,
			cancellable,created_at,started_at,updated_at,finished_at,failure_code
		FROM jobs WHERE id=$1 AND workspace_id=$2`, id, principal.Workspace))
	if err != nil {
		return job.Job{}, mapNotFound("get job", err)
	}
	return value, nil
}

func (s *Store) CancelJob(ctx context.Context, principal access.Principal, id int64, requestID string) (job.Job, error) {
	if err := access.Authorize(principal, access.ActionProjectWrite); err != nil {
		return job.Job{}, err
	}
	var value job.Job
	err := s.withTx(ctx, func(tx pgx.Tx) error {
		current, err := scanJob(tx.QueryRow(ctx, `
			SELECT id,COALESCE(project_id,0),kind,state,progress,checkpoint,requested_from,requested_to,version,coalesced_requests,
				cancellable,created_at,started_at,updated_at,finished_at,failure_code
			FROM jobs WHERE id=$1 AND workspace_id=$2 FOR UPDATE`, id, principal.Workspace))
		if err != nil {
			return mapNotFound("get job for cancellation", err)
		}
		if current.State == job.Cancelled {
			value = current
			return nil
		}
		value, err = current.Cancel(s.now())
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `UPDATE jobs SET state='cancelled',version=$1,updated_at=$2,
			finished_at=$2,lease_holder=NULL,lease_expires_at=NULL WHERE id=$3`,
			value.Version, value.UpdatedAt, id)
		if err != nil {
			return fmt.Errorf("cancel job: %w", err)
		}
		if err := s.recordJobEvent(ctx, tx, value); err != nil {
			return err
		}
		return s.writeOutbox(ctx, tx, value.ProjectID, value.ID, "job.cancelled", principal, requestID, value)
	})
	return value, err
}

func (s *Store) JobEvents(ctx context.Context, principal access.Principal, id, after int64, limit int) ([]JobEvent, error) {
	if err := access.Authorize(principal, access.ActionIntelligenceRead); err != nil {
		return nil, err
	}
	limit, _ = page(limit, 0)
	rows, err := s.pool.Query(ctx, `
		SELECT e.id,e.representation FROM job_events e JOIN jobs j ON j.id=e.job_id
		WHERE e.job_id=$1 AND j.workspace_id=$2 AND e.id>$3 ORDER BY e.id LIMIT $4`,
		id, principal.Workspace, after, limit)
	if err != nil {
		return nil, fmt.Errorf("list job events: %w", err)
	}
	defer rows.Close()
	values := make([]JobEvent, 0, limit)
	for rows.Next() {
		var encoded []byte
		var value JobEvent
		if err := rows.Scan(&value.ID, &encoded); err != nil {
			return nil, fmt.Errorf("scan job event: %w", err)
		}
		if err := json.Unmarshal(encoded, &value.Job); err != nil {
			return nil, fmt.Errorf("decode job event: %w", err)
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

func (s *Store) queueJob(
	ctx context.Context,
	principal access.Principal,
	projectID int64,
	kind, scope, coalescingKey, requestID string,
	requestedFrom, requestedTo *time.Time,
) (job.Job, error) {
	jobID, err := s.ids.Next(ctx)
	if err != nil {
		return job.Job{}, fmt.Errorf("issue job ID: %w", err)
	}
	queued, err := job.New(jobID, projectID, kind, scope, nil, true, s.now())
	if err != nil {
		return job.Job{}, err
	}
	queued.RequestedFrom = requestedFrom
	queued.RequestedTo = requestedTo
	err = s.withTx(ctx, func(tx pgx.Tx) error {
		if _, lockErr := tx.Exec(ctx,
			`SELECT pg_advisory_xact_lock(hashtextextended($1,$2))`,
			coalescingKey, principal.Workspace); lockErr != nil {
			return fmt.Errorf("lock job coalescing key: %w", lockErr)
		}
		var existingID int64
		err := tx.QueryRow(ctx, `SELECT id FROM jobs WHERE workspace_id=$1 AND coalescing_key=$2
			AND state IN ('queued','running') FOR UPDATE`, principal.Workspace, coalescingKey).Scan(&existingID)
		if err == nil {
			_, err = tx.Exec(ctx, `UPDATE jobs SET coalesced_requests=coalesced_requests+1,
				requested_from=CASE WHEN $2::timestamptz IS NULL THEN requested_from
					ELSE LEAST(COALESCE(requested_from,$2),$2) END,
				requested_to=CASE WHEN $3::timestamptz IS NULL THEN requested_to
					ELSE GREATEST(COALESCE(requested_to,$3),$3) END,
				version=version+1,updated_at=now() WHERE id=$1`, existingID, requestedFrom, requestedTo)
			if err != nil {
				return fmt.Errorf("coalesce job: %w", err)
			}
			queued, err = scanJob(tx.QueryRow(ctx, `
				SELECT id,COALESCE(project_id,0),kind,state,progress,checkpoint,requested_from,requested_to,version,coalesced_requests,
					cancellable,created_at,started_at,updated_at,finished_at,failure_code FROM jobs WHERE id=$1`, existingID))
			if err != nil {
				return err
			}
			return s.recordJobEvent(ctx, tx, queued)
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("find coalesced job: %w", err)
		}
		if err := insertJob(ctx, tx, queued, principal, requestID, coalescingKey); err != nil {
			return err
		}
		if err := s.recordJobEvent(ctx, tx, queued); err != nil {
			return err
		}
		return s.writeOutbox(ctx, tx, projectID, queued.ID, "job.queued", principal, requestID, queued)
	})
	return queued, err
}

func (s *Store) registrationTx(ctx context.Context, tx pgx.Tx, projectID, jobID int64) (Registration, error) {
	value, err := scanProject(tx.QueryRow(ctx, `SELECT id,workspace_id,name,slug,description,state,version,deleted_at
		FROM projects WHERE id=$1`, projectID))
	if err != nil {
		return Registration{}, mapNotFound("read replayed project", err)
	}
	repositories, err := tx.Query(ctx, `SELECT id,project_id,provider,canonical_url,role,version
		FROM repositories WHERE project_id=$1 ORDER BY role='primary' DESC,id`, projectID)
	if err != nil {
		return Registration{}, fmt.Errorf("read replayed repositories: %w", err)
	}
	for repositories.Next() {
		var repository project.Repository
		if err := repositories.Scan(&repository.ID, &repository.ProjectID, &repository.Provider,
			&repository.CanonicalURL, &repository.Role, &repository.Version); err != nil {
			repositories.Close()
			return Registration{}, fmt.Errorf("scan replayed repository: %w", err)
		}
		value.Repositories = append(value.Repositories, repository)
	}
	if err := repositories.Err(); err != nil {
		repositories.Close()
		return Registration{}, fmt.Errorf("read replayed repositories: %w", err)
	}
	repositories.Close()
	sources, err := tx.Query(ctx, `SELECT id,project_id,kind,canonical_url,state,public,coverage_from,
		coverage_to,last_attempt_at,last_success_at,next_run_at,failure_code,version
		FROM sources WHERE project_id=$1 ORDER BY kind,id`, projectID)
	if err != nil {
		return Registration{}, fmt.Errorf("read replayed sources: %w", err)
	}
	for sources.Next() {
		source, scanErr := scanSource(sources)
		if scanErr != nil {
			sources.Close()
			return Registration{}, scanErr
		}
		value.Sources = append(value.Sources, source)
	}
	if err := sources.Err(); err != nil {
		sources.Close()
		return Registration{}, fmt.Errorf("read replayed sources: %w", err)
	}
	sources.Close()
	queued, err := scanJob(tx.QueryRow(ctx, `SELECT id,COALESCE(project_id,0),kind,state,progress,checkpoint,
		requested_from,requested_to,version,coalesced_requests,cancellable,created_at,started_at,updated_at,finished_at,failure_code
		FROM jobs WHERE id=$1`, jobID))
	return Registration{Project: value, Job: queued}, err
}

func (s *Store) nextFour(ctx context.Context) (int64, int64, int64, int64, error) {
	values := [4]int64{}
	for index := range values {
		value, err := s.ids.Next(ctx)
		if err != nil {
			return 0, 0, 0, 0, fmt.Errorf("issue registration ID: %w", err)
		}
		values[index] = value
	}
	return values[0], values[1], values[2], values[3], nil
}

func insertJob(ctx context.Context, tx pgx.Tx, value job.Job, principal access.Principal, requestID, coalescingKey string) error {
	progress, _ := json.Marshal(value.Progress)
	_, err := tx.Exec(ctx, `INSERT INTO jobs
		(id,kind,state,workspace_id,project_id,version,progress,requested_from,requested_to,
		 coalescing_key,cancellable,requested_by,request_id,correlation_id,available_at,created_at,updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$13,$14,$14,$14)`,
		value.ID, value.Kind, value.State, principal.Workspace, nullableProject(value.ProjectID), value.Version,
		progress, value.RequestedFrom, value.RequestedTo, coalescingKey, value.Cancellable,
		principal.ActorID, requestID, value.CreatedAt)
	if err != nil {
		return mapConflict("create job", err)
	}
	return nil
}

func (s *Store) recordJobEvent(ctx context.Context, tx pgx.Tx, value job.Job) error {
	id, err := s.ids.Next(ctx)
	if err != nil {
		return fmt.Errorf("issue job event ID: %w", err)
	}
	representation, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode job event: %w", err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO job_events (id,job_id,job_version,representation,occurred_at)
		VALUES ($1,$2,$3,$4,$5)`, id, value.ID, value.Version, representation, value.UpdatedAt)
	if err != nil {
		return fmt.Errorf("record job event: %w", err)
	}
	return nil
}

func (s *Store) writeOutbox(
	ctx context.Context,
	tx pgx.Tx,
	projectID, jobID int64,
	eventType string,
	principal access.Principal,
	requestID string,
	payload any,
) error {
	id, err := s.ids.Next(ctx)
	if err != nil {
		return fmt.Errorf("issue outbox ID: %w", err)
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode outbox event: %w", err)
	}
	aggregateID := projectID
	if aggregateID == 0 {
		aggregateID = jobID
	}
	_, err = tx.Exec(ctx, `INSERT INTO outbox_events
		(id,aggregate_type,aggregate_id,event_type,schema_version,payload,job_id,correlation_id,causation_id)
		VALUES ($1,'project',$2,$3,1,$4,$5,$6,$6)`,
		id, aggregateID, eventType, encoded, nullableProject(jobID), requestID)
	if err != nil {
		return fmt.Errorf("write outbox event: %w", err)
	}
	return nil
}

func (s *Store) audit(
	ctx context.Context,
	tx pgx.Tx,
	principal access.Principal,
	action, resourceType string,
	resourceID int64,
	requestID string,
	changes any,
) error {
	id, err := s.ids.Next(ctx)
	if err != nil {
		return fmt.Errorf("issue audit ID: %w", err)
	}
	encoded, err := json.Marshal(changes)
	if err != nil {
		return fmt.Errorf("encode audit changes: %w", err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO audit_events
		(id,actor_id,actor_kind,action,resource_type,resource_id,request_id,changes)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`, id, principal.ActorID, principal.Kind,
		action, resourceType, resourceID, requestID, encoded)
	if err != nil {
		return fmt.Errorf("write audit event: %w", err)
	}
	return nil
}

func (s *Store) withTx(ctx context.Context, run func(pgx.Tx) error) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin project transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := run(tx); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit project transaction: %w", err)
	}
	return nil
}

type rowScanner interface {
	Scan(...any) error
}

func scanProject(row rowScanner) (project.Project, error) {
	var value project.Project
	err := row.Scan(&value.ID, &value.WorkspaceID, &value.Name, &value.Slug, &value.Description,
		&value.State, &value.Version, &value.DeletedAt)
	if err != nil {
		return project.Project{}, err
	}
	return value, nil
}

func scanSource(row rowScanner) (project.Source, error) {
	var value project.Source
	err := row.Scan(&value.ID, &value.ProjectID, &value.Kind, &value.CanonicalURL, &value.State,
		&value.Public, &value.CoverageFrom, &value.CoverageTo, &value.LastAttemptAt,
		&value.LastSuccessAt, &value.NextRunAt, &value.Failure, &value.Version)
	if err != nil {
		return project.Source{}, fmt.Errorf("scan source: %w", err)
	}
	return value, nil
}

func scanJob(row rowScanner) (job.Job, error) {
	var value job.Job
	var progress, checkpoint []byte
	err := row.Scan(&value.ID, &value.ProjectID, &value.Kind, &value.State, &progress, &checkpoint,
		&value.RequestedFrom, &value.RequestedTo,
		&value.Version, &value.CoalescedRequests, &value.Cancellable, &value.CreatedAt,
		&value.StartedAt, &value.UpdatedAt, &value.FinishedAt, &value.Failure)
	if err != nil {
		return job.Job{}, err
	}
	if err := json.Unmarshal(progress, &value.Progress); err != nil {
		return job.Job{}, fmt.Errorf("decode job progress: %w", err)
	}
	if len(checkpoint) > 0 && string(checkpoint) != "{}" && string(checkpoint) != "null" {
		var decoded job.Checkpoint
		if err := json.Unmarshal(checkpoint, &decoded); err != nil {
			return job.Job{}, fmt.Errorf("decode job checkpoint: %w", err)
		}
		value.Checkpoint = &decoded
	}
	return value, nil
}

func loadProjectForUpdate(ctx context.Context, tx pgx.Tx, workspaceID, projectID int64) (project.Project, error) {
	value, err := scanProject(tx.QueryRow(ctx, `SELECT id,workspace_id,name,slug,description,state,version,deleted_at
		FROM projects WHERE id=$1 AND workspace_id=$2 FOR UPDATE`, projectID, workspaceID))
	if err != nil {
		return project.Project{}, mapNotFound("lock project", err)
	}
	rows, err := tx.Query(ctx, `SELECT id,project_id,provider,canonical_url,role,version
		FROM repositories WHERE project_id=$1 ORDER BY id FOR UPDATE`, projectID)
	if err != nil {
		return project.Project{}, fmt.Errorf("lock project repositories: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var repository project.Repository
		if err := rows.Scan(&repository.ID, &repository.ProjectID, &repository.Provider,
			&repository.CanonicalURL, &repository.Role, &repository.Version); err != nil {
			return project.Project{}, fmt.Errorf("scan locked repository: %w", err)
		}
		value.Repositories = append(value.Repositories, repository)
	}
	if err := rows.Err(); err != nil {
		return project.Project{}, fmt.Errorf("read locked repositories: %w", err)
	}
	return value, nil
}

func page(limit, offset int) (int, int) {
	if limit <= 0 || limit > 100 {
		limit = defaultPageLimit
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

func nullableProject(id int64) any {
	if id <= 0 {
		return nil
	}
	return id
}

func equalDigest(left, right []byte) bool {
	return len(left) == len(right) && string(left) == string(right)
}

func mapNotFound(operation string, err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return project.ErrNotFound
	}
	return fmt.Errorf("%s: %w", operation, err)
}

func mapConflict(operation string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && (pgErr.Code == "23505" || pgErr.Code == "40001") {
		return fmt.Errorf("%w: %s", project.ErrConflict, operation)
	}
	return fmt.Errorf("%s: %w", operation, err)
}
