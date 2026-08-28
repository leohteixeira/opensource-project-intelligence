// Package assistantstore persists bounded assistant proposals and executes the
// single allowlisted mutation behind a PostgreSQL transaction.
package assistantstore

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/leohteixeira/opensource-project-intelligence/internal/access"
	"github.com/leohteixeira/opensource-project-intelligence/internal/analysis/agent"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/database"
	"github.com/leohteixeira/opensource-project-intelligence/internal/project"
)

const (
	proposalRoute   = "POST /api/v1/assistant/proposals"
	maxRepositories = 20
)

type IDSource interface {
	Next(context.Context) (int64, error)
}

type Store struct {
	pool *pgxpool.Pool
	ids  IDSource
}

func New(pool *database.Pool, ids IDSource) *Store {
	return &Store{pool: pool.Unwrap(), ids: ids}
}

func (s *Store) Snapshot(
	ctx context.Context,
	principal access.Principal,
	input agent.RepositoryAdd,
) (agent.Snapshot, error) {
	if err := access.Authorize(principal, access.ActionProjectWrite); err != nil {
		return agent.Snapshot{}, err
	}
	var snapshot agent.Snapshot
	err := s.pool.QueryRow(ctx, `SELECT p.version,p.state,count(r.id),$3
		FROM projects p LEFT JOIN repositories r ON r.project_id=p.id
		WHERE p.id=$1 AND p.workspace_id=$2
		GROUP BY p.id`, input.ProjectID, principal.Workspace, maxRepositories).Scan(
		&snapshot.ProjectVersion, &snapshot.Lifecycle, &snapshot.QuotaUsed, &snapshot.QuotaLimit)
	if errors.Is(err, pgx.ErrNoRows) {
		return agent.Snapshot{}, project.ErrNotFound
	}
	if err != nil {
		return agent.Snapshot{}, fmt.Errorf("read assistant proposal state: %w", err)
	}
	return snapshot, nil
}

func (s *Store) Create(
	ctx context.Context,
	proposal agent.Proposal,
	idempotencyKey string,
	digest [32]byte,
) (agent.Proposal, bool, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return agent.Proposal{}, false, fmt.Errorf("begin assistant proposal: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var storedDigest []byte
	var resourceID *int64
	err = tx.QueryRow(ctx, `SELECT request_digest,resource_id FROM idempotency_records
		WHERE workspace_id=$1 AND actor_id=$2 AND route=$3 AND idempotency_key=$4 FOR UPDATE`,
		proposal.WorkspaceID, proposal.ActorID, proposalRoute, idempotencyKey).Scan(&storedDigest, &resourceID)
	if err == nil {
		if len(storedDigest) != len(digest) || subtle.ConstantTimeCompare(storedDigest, digest[:]) != 1 || resourceID == nil {
			return agent.Proposal{}, false, agent.ErrIdempotencyKey
		}
		value, loadErr := scanProposal(tx.QueryRow(ctx, proposalSelect+` WHERE id=$1`, *resourceID))
		if loadErr != nil {
			return agent.Proposal{}, false, loadErr
		}
		if err := tx.Commit(ctx); err != nil {
			return agent.Proposal{}, false, fmt.Errorf("commit assistant replay: %w", err)
		}
		return value, true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return agent.Proposal{}, false, fmt.Errorf("read assistant idempotency: %w", err)
	}
	inputs, _ := json.Marshal(proposal.Inputs)
	resources, _ := json.Marshal(proposal.Resources)
	quota, _ := json.Marshal(proposal.Quota)
	result, _ := json.Marshal(proposal.Result)
	_, err = tx.Exec(ctx, `INSERT INTO assistant_proposals
		(id,workspace_id,actor_id,actor_version,status,action,inputs,resources,effect,quota,
		 confirmation_digest,result,expires_at,created_at,updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$14)`,
		proposal.ID, proposal.WorkspaceID, proposal.ActorID, proposal.ActorVersion, proposal.Status,
		proposal.Action, inputs, resources, proposal.Effect, quota, proposal.TokenDigest[:], result,
		proposal.ExpiresAt, proposal.CreatedAt)
	if err != nil {
		return agent.Proposal{}, false, fmt.Errorf("insert assistant proposal: %w", err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO idempotency_records
		(workspace_id,actor_id,route,idempotency_key,request_digest,response_status,resource_id,completed_at)
		VALUES ($1,$2,$3,$4,$5,201,$6,now())`, proposal.WorkspaceID, proposal.ActorID,
		proposalRoute, idempotencyKey, digest[:], proposal.ID)
	if err != nil {
		return agent.Proposal{}, false, fmt.Errorf("record assistant idempotency: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return agent.Proposal{}, false, fmt.Errorf("commit assistant proposal: %w", err)
	}
	return proposal, false, nil
}

func (s *Store) Begin(
	ctx context.Context,
	id int64,
	principal access.Principal,
	digest [32]byte,
	idempotencyKey string,
	now time.Time,
) (agent.Proposal, bool, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return agent.Proposal{}, false, fmt.Errorf("begin assistant confirmation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	value, storedDigest, confirmationKey, err := scanProposalForUpdate(ctx, tx, id)
	if errors.Is(err, pgx.ErrNoRows) || value.WorkspaceID != principal.Workspace || value.ActorID != principal.ActorID {
		return agent.Proposal{}, false, project.ErrNotFound
	}
	if err != nil {
		return agent.Proposal{}, false, err
	}
	if len(storedDigest) != len(digest) || subtle.ConstantTimeCompare(storedDigest, digest[:]) != 1 {
		return agent.Proposal{}, false, agent.ErrInvalid
	}
	if confirmationKey != nil && *confirmationKey == idempotencyKey &&
		(value.Status == agent.Executed || value.Status == agent.Failed) {
		if err := tx.Commit(ctx); err != nil {
			return agent.Proposal{}, false, fmt.Errorf("commit confirmation replay: %w", err)
		}
		return value, true, nil
	}
	if !now.Before(value.ExpiresAt) {
		_, _ = tx.Exec(ctx, `UPDATE assistant_proposals SET status='expired',updated_at=$2
			WHERE id=$1 AND status='awaiting_confirmation'`, id, now)
		if err := tx.Commit(ctx); err != nil {
			return agent.Proposal{}, false, fmt.Errorf("expire assistant proposal: %w", err)
		}
		return agent.Proposal{}, false, agent.ErrExpired
	}
	if value.Status != agent.AwaitingConfirmation || confirmationKey != nil {
		return agent.Proposal{}, false, agent.ErrAlreadyUsed
	}
	_, err = tx.Exec(ctx, `UPDATE assistant_proposals SET status='executing',confirmation_key=$2,
		consumed_at=$3,updated_at=$3 WHERE id=$1`, id, idempotencyKey, now)
	if err != nil {
		return agent.Proposal{}, false, fmt.Errorf("consume assistant proposal: %w", err)
	}
	value.Status, value.ConsumedAt = agent.Executing, &now
	if err := tx.Commit(ctx); err != nil {
		return agent.Proposal{}, false, fmt.Errorf("commit assistant confirmation: %w", err)
	}
	return value, false, nil
}

func (s *Store) Finish(
	ctx context.Context,
	id int64,
	status agent.Status,
	result agent.Result,
	now time.Time,
) (agent.Proposal, error) {
	if status != agent.Executed && status != agent.Failed {
		return agent.Proposal{}, agent.ErrInvalid
	}
	encoded, _ := json.Marshal(result)
	command, err := s.pool.Exec(ctx, `UPDATE assistant_proposals SET status=$2,result=$3,updated_at=$4
		WHERE id=$1 AND status='executing'`, id, status, encoded, now)
	if err != nil {
		return agent.Proposal{}, fmt.Errorf("finish assistant proposal: %w", err)
	}
	if command.RowsAffected() != 1 {
		return agent.Proposal{}, agent.ErrAlreadyUsed
	}
	return scanProposal(s.pool.QueryRow(ctx, proposalSelect+` WHERE id=$1`, id))
}

func (s *Store) AddRepository(
	ctx context.Context,
	principal access.Principal,
	input agent.RepositoryAdd,
	requestID string,
) (agent.Result, error) {
	if err := access.Authorize(principal, access.ActionProjectWrite); err != nil {
		return agent.Result{}, err
	}
	provider, _, _, canonical, err := project.CanonicalRepositoryURL(input.URL)
	if err != nil {
		return agent.Result{}, err
	}
	role := project.RepositoryRole(input.Role)
	if !validRole(role) {
		return agent.Result{}, agent.ErrInvalid
	}
	repositoryID, auditID, outboxID, err := s.nextThree(ctx)
	if err != nil {
		return agent.Result{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return agent.Result{}, fmt.Errorf("begin assistant action: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var state string
	var version int64
	var used int
	err = tx.QueryRow(ctx, `SELECT state,version FROM projects
		WHERE id=$1 AND workspace_id=$2 FOR UPDATE`, input.ProjectID, principal.Workspace).Scan(&state, &version)
	if errors.Is(err, pgx.ErrNoRows) {
		return agent.Result{}, project.ErrNotFound
	}
	if err != nil {
		return agent.Result{}, fmt.Errorf("lock assistant project: %w", err)
	}
	if state != "active" || version != input.ProjectVersion {
		return agent.Result{}, agent.ErrStateChanged
	}
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM repositories WHERE project_id=$1`,
		input.ProjectID).Scan(&used); err != nil {
		return agent.Result{}, fmt.Errorf("count assistant project repositories: %w", err)
	}
	if used >= maxRepositories {
		return agent.Result{}, agent.ErrQuotaExceeded
	}
	if role == project.RolePrimary {
		if _, err := tx.Exec(ctx, `UPDATE repositories SET role='core',version=version+1,
			updated_at=now() WHERE project_id=$1 AND role='primary'`, input.ProjectID); err != nil {
			return agent.Result{}, fmt.Errorf("replace primary repository: %w", err)
		}
	}
	_, err = tx.Exec(ctx, `INSERT INTO repositories
		(id,project_id,provider,canonical_url,role) VALUES ($1,$2,$3,$4,$5)`,
		repositoryID, input.ProjectID, provider, canonical, role)
	if err != nil {
		return agent.Result{}, fmt.Errorf("attach assistant repository: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE projects SET version=version+1,updated_at=now() WHERE id=$1`,
		input.ProjectID); err != nil {
		return agent.Result{}, fmt.Errorf("version assistant project: %w", err)
	}
	details, _ := json.Marshal(map[string]any{"role": role, "canonical_url": canonical})
	_, err = tx.Exec(ctx, `INSERT INTO audit_events
		(id,actor_id,actor_kind,action,resource_type,resource_id,outcome,request_id,details,changes)
		VALUES ($1,$2,$3,'assistant.repository.add','repository',$4,'succeeded',$5,$6,$6)`,
		auditID, principal.ActorID, principal.Kind, repositoryID, requestID, details)
	if err != nil {
		return agent.Result{}, fmt.Errorf("audit assistant action: %w", err)
	}
	payload, _ := json.Marshal(map[string]any{"project_id": input.ProjectID, "repository_id": repositoryID,
		"actor_id": principal.ActorID, "request_id": requestID})
	_, err = tx.Exec(ctx, `INSERT INTO outbox_events
		(id,aggregate_type,aggregate_id,event_type,schema_version,payload,correlation_id)
		VALUES ($1,'project',$2,'repository.attached',1,$3,$4)`, outboxID, input.ProjectID,
		payload, requestID)
	if err != nil {
		return agent.Result{}, fmt.Errorf("enqueue assistant action event: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return agent.Result{}, fmt.Errorf("commit assistant action: %w", err)
	}
	return agent.Result{RepositoryID: repositoryID, AuditEventID: auditID}, nil
}

const proposalFields = `id,workspace_id,actor_id,actor_version,status,action,inputs,
	resources,effect,quota,expires_at,created_at,consumed_at,result`

const proposalSelect = `SELECT ` + proposalFields + ` FROM assistant_proposals`

type rowScanner interface{ Scan(...any) error }

func scanProposal(row rowScanner) (agent.Proposal, error) {
	var value agent.Proposal
	var inputs, resources, quota, result []byte
	err := row.Scan(&value.ID, &value.WorkspaceID, &value.ActorID, &value.ActorVersion, &value.Status,
		&value.Action, &inputs, &resources, &value.Effect, &quota, &value.ExpiresAt,
		&value.CreatedAt, &value.ConsumedAt, &result)
	if err != nil {
		return agent.Proposal{}, err
	}
	if err := errors.Join(json.Unmarshal(inputs, &value.Inputs), json.Unmarshal(resources, &value.Resources),
		json.Unmarshal(quota, &value.Quota), json.Unmarshal(result, &value.Result)); err != nil {
		return agent.Proposal{}, fmt.Errorf("decode assistant proposal: %w", err)
	}
	return value, nil
}

func scanProposalForUpdate(
	ctx context.Context,
	tx pgx.Tx,
	id int64,
) (agent.Proposal, []byte, *string, error) {
	row := tx.QueryRow(ctx, `SELECT `+proposalFields+`,confirmation_digest,confirmation_key
		FROM assistant_proposals WHERE id=$1 FOR UPDATE`, id)
	var value agent.Proposal
	var inputs, resources, quota, result, digest []byte
	var confirmationKey *string
	err := row.Scan(&value.ID, &value.WorkspaceID, &value.ActorID, &value.ActorVersion, &value.Status,
		&value.Action, &inputs, &resources, &value.Effect, &quota, &value.ExpiresAt,
		&value.CreatedAt, &value.ConsumedAt, &result, &digest, &confirmationKey)
	if err != nil {
		return agent.Proposal{}, nil, nil, err
	}
	if err := errors.Join(json.Unmarshal(inputs, &value.Inputs), json.Unmarshal(resources, &value.Resources),
		json.Unmarshal(quota, &value.Quota), json.Unmarshal(result, &value.Result)); err != nil {
		return agent.Proposal{}, nil, nil, fmt.Errorf("decode locked assistant proposal: %w", err)
	}
	return value, digest, confirmationKey, nil
}

func validRole(role project.RepositoryRole) bool {
	switch role {
	case project.RolePrimary, project.RoleCore, project.RoleDocumentation, project.RoleExamples,
		project.RoleSDK, project.RoleOther:
		return true
	default:
		return false
	}
}

func (s *Store) nextThree(ctx context.Context) (int64, int64, int64, error) {
	values := [3]int64{}
	for index := range values {
		value, err := s.ids.Next(ctx)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("issue assistant action ID: %w", err)
		}
		values[index] = value
	}
	return values[0], values[1], values[2], nil
}
