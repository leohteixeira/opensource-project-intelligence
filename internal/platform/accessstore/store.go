// Package accessstore persists local access authority in PostgreSQL.
package accessstore

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/leohteixeira/opensource-project-intelligence/internal/access"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/database"
)

const WorkspaceID int64 = 1

type IDSource interface {
	Next(context.Context) (int64, error)
}

type Store struct {
	pool *pgxpool.Pool
	ids  IDSource
	now  func() time.Time
}

type Session struct {
	ID        int64
	Member    access.Member
	Principal access.Principal
	CSRFHash  [32]byte
	ExpiresAt time.Time
}

type LoginFlow struct {
	Nonce     string
	Verifier  string
	ReturnTo  string
	ExpiresAt time.Time
}

type CatalogProject struct {
	ID          int64    `json:"id,string"`
	Name        string   `json:"name"`
	Slug        string   `json:"slug"`
	Description string   `json:"description"`
	SourceLinks []string `json:"source_links"`
}

type AuditEvent struct {
	ID           int64          `json:"id,string"`
	OccurredAt   time.Time      `json:"occurred_at"`
	ActorID      *int64         `json:"actor_id,omitempty"`
	ActorKind    string         `json:"actor_kind"`
	Action       string         `json:"action"`
	ResourceType string         `json:"resource_type"`
	ResourceID   *int64         `json:"resource_id,omitempty"`
	Outcome      string         `json:"outcome"`
	RequestID    string         `json:"request_id"`
	Changes      map[string]any `json:"changes"`
}

type AuditFilter struct {
	ActorID      *int64
	Action       string
	Resource     string
	Outcome      string
	OccurredFrom *time.Time
	OccurredTo   *time.Time
	Limit        int
	Offset       int
}

type MemberFilter struct {
	State  access.Status
	Role   access.Role
	Query  string
	Limit  int
	Offset int
}

var AllowedServiceScopes = map[string]struct{}{
	"projects:read": {}, "projects:write": {}, "exports:write": {},
}

func New(pool *database.Pool, ids IDSource) *Store {
	return &Store{pool: pool.Unwrap(), ids: ids, now: time.Now}
}

func (s *Store) EnsureWorkspace(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO workspaces (id, name) VALUES ($1, 'Open Source Project Intelligence')
		ON CONFLICT (id) DO NOTHING`, WorkspaceID)
	if err != nil {
		return fmt.Errorf("ensure workspace: %w", err)
	}
	return nil
}

func (s *Store) UpsertApplicant(
	ctx context.Context,
	key access.IdentityKey,
	displayName, email string,
) (access.Member, error) {
	if err := key.Validate(); err != nil {
		return access.Member{}, err
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return access.Member{}, fmt.Errorf("begin applicant transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	identityID, err := s.ids.Next(ctx)
	if err != nil {
		return access.Member{}, fmt.Errorf("issue identity ID: %w", err)
	}
	var storedIdentityID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO external_identities (id, issuer, subject, display_name, email)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (issuer, subject) DO UPDATE SET
			display_name = EXCLUDED.display_name,
			email = EXCLUDED.email,
			updated_at = now()
		RETURNING id`, identityID, key.Issuer, key.Subject, displayName, email).Scan(&storedIdentityID)
	if err != nil {
		return access.Member{}, fmt.Errorf("upsert external identity: %w", err)
	}

	memberID, err := s.ids.Next(ctx)
	if err != nil {
		return access.Member{}, fmt.Errorf("issue membership ID: %w", err)
	}
	member, err := scanMember(tx.QueryRow(ctx, `
		INSERT INTO memberships (id, workspace_id, identity_id, status)
		VALUES ($1, $2, $3, 'pending')
		ON CONFLICT (workspace_id, identity_id) DO UPDATE SET
			status = CASE
				WHEN memberships.status IN ('rejected', 'suspended', 'deleted') THEN memberships.status
				ELSE memberships.status
			END
		RETURNING id, identity_id, $4, $5, COALESCE(role, ''), status, locale, timezone,
			version, requested_at`, memberID, WorkspaceID, storedIdentityID, displayName, email))
	if err != nil {
		return access.Member{}, fmt.Errorf("upsert applicant: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return access.Member{}, fmt.Errorf("commit applicant transaction: %w", err)
	}
	return member, nil
}

func (s *Store) ResolveIdentity(ctx context.Context, key access.IdentityKey) (access.Principal, error) {
	if err := key.Validate(); err != nil {
		return access.Principal{}, err
	}
	var principal access.Principal
	var kind string
	err := s.pool.QueryRow(ctx, `
		SELECT actor_id, actor_kind, role, status, version FROM (
			SELECT m.id AS actor_id, 'member'::text AS actor_kind, COALESCE(m.role, '') AS role,
				m.status, m.version
			FROM memberships m
			JOIN external_identities i ON i.id = m.identity_id
			WHERE m.workspace_id = $1 AND i.issuer = $2 AND i.subject = $3
			UNION ALL
			SELECT sa.id, 'service_account', sa.role, sa.status, sa.version
			FROM service_accounts sa
			WHERE sa.workspace_id = $1 AND sa.issuer = $2 AND sa.external_subject = $3
		) resolved
		LIMIT 1`, WorkspaceID, key.Issuer, key.Subject).Scan(
		&principal.ActorID, &kind, &principal.Role, &principal.Status, &principal.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		return access.Principal{}, access.ErrNotFound
	}
	if err != nil {
		return access.Principal{}, fmt.Errorf("resolve local identity: %w", err)
	}
	principal.Kind = access.ActorKind(kind)
	principal.Workspace = WorkspaceID
	if principal.Kind == access.ActorServiceAccount {
		rows, queryErr := s.pool.Query(ctx, `
			SELECT scope FROM service_account_scopes WHERE service_account_id = $1 ORDER BY scope`,
			principal.ActorID)
		if queryErr != nil {
			return access.Principal{}, fmt.Errorf("load service account scopes: %w", queryErr)
		}
		principal.Scopes, err = pgx.CollectRows(rows, pgx.RowTo[string])
		if err != nil {
			return access.Principal{}, fmt.Errorf("scan service account scopes: %w", err)
		}
	}
	return principal, nil
}

func (s *Store) CreateSession(
	ctx context.Context,
	memberID int64,
	verifierHash, csrfHash [32]byte,
	expiresAt time.Time,
) (int64, error) {
	id, err := s.ids.Next(ctx)
	if err != nil {
		return 0, fmt.Errorf("issue session ID: %w", err)
	}
	command, err := s.pool.Exec(ctx, `
		INSERT INTO browser_sessions (id, membership_id, verifier_hash, csrf_hash, expires_at)
		SELECT $1, id, $3, $4, $5 FROM memberships
		WHERE id = $2 AND workspace_id = $6 AND status <> 'deleted'`,
		id, memberID, verifierHash[:], csrfHash[:], expiresAt.UTC(), WorkspaceID)
	if err != nil {
		return 0, fmt.Errorf("create browser session: %w", err)
	}
	if command.RowsAffected() != 1 {
		return 0, access.ErrPermissionDenied
	}
	return id, nil
}

func (s *Store) CreateLoginFlow(
	ctx context.Context,
	stateHash, nonceHash, verifierHash [32]byte,
	returnTo string,
	expiresAt time.Time,
) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO oidc_login_flows
			(state_hash, nonce_hash, verifier_hash, return_to, expires_at)
		VALUES ($1, $2, $3, $4, $5)`, stateHash[:], nonceHash[:], verifierHash[:], returnTo, expiresAt.UTC())
	if err != nil {
		return fmt.Errorf("create OIDC login flow: %w", err)
	}
	return nil
}

func (s *Store) ConsumeLoginFlow(ctx context.Context, state, nonce, verifier string) (LoginFlow, error) {
	stateHash := sha256.Sum256([]byte(state))
	nonceHash := sha256.Sum256([]byte(nonce))
	verifierHash := sha256.Sum256([]byte(verifier))
	var flow LoginFlow
	err := s.pool.QueryRow(ctx, `
		UPDATE oidc_login_flows SET consumed_at = now()
		WHERE state_hash = $1 AND nonce_hash = $2 AND verifier_hash = $3
			AND consumed_at IS NULL AND expires_at > now()
		RETURNING return_to, expires_at`, stateHash[:], nonceHash[:], verifierHash[:]).Scan(
		&flow.ReturnTo, &flow.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return LoginFlow{}, access.ErrAuthenticationRequired
	}
	if err != nil {
		return LoginFlow{}, fmt.Errorf("consume OIDC login flow: %w", err)
	}
	flow.Nonce = nonce
	flow.Verifier = verifier
	return flow, nil
}

func (s *Store) ResolveSession(ctx context.Context, id int64, verifier string) (Session, error) {
	var result Session
	var hash []byte
	var csrf []byte
	var kind string
	err := s.pool.QueryRow(ctx, `
		SELECT s.id, s.verifier_hash, s.csrf_hash, s.expires_at,
			m.id, m.identity_id, i.display_name, i.email, COALESCE(m.role, ''), m.status,
			m.locale, m.timezone, m.version, m.requested_at
		FROM browser_sessions s
		JOIN memberships m ON m.id = s.membership_id
		JOIN external_identities i ON i.id = m.identity_id
		WHERE s.id = $1 AND s.revoked_at IS NULL AND s.expires_at > $2`, id, s.now().UTC()).Scan(
		&result.ID, &hash, &csrf, &result.ExpiresAt,
		&result.Member.ID, &result.Member.IdentityID, &result.Member.DisplayName, &result.Member.Email,
		&result.Member.Role, &result.Member.Status, &result.Member.Locale, &result.Member.Timezone,
		&result.Member.Version, &result.Member.RequestedAt)
	_ = kind
	if errors.Is(err, pgx.ErrNoRows) {
		return Session{}, access.ErrAuthenticationRequired
	}
	if err != nil {
		return Session{}, fmt.Errorf("resolve browser session: %w", err)
	}
	if len(hash) != sha256.Size || len(csrf) != sha256.Size {
		return Session{}, errors.New("stored session hash has an invalid length")
	}
	var expected [32]byte
	copy(expected[:], hash)
	if !access.VerifySecret(verifier, expected) {
		return Session{}, access.ErrAuthenticationRequired
	}
	copy(result.CSRFHash[:], csrf)
	result.Principal = access.Principal{
		ActorID: result.Member.ID, Kind: access.ActorMember, Role: result.Member.Role,
		Status: result.Member.Status, Version: result.Member.Version, Workspace: WorkspaceID,
	}
	return result, nil
}

func (s *Store) RevokeSession(ctx context.Context, id int64) error {
	_, err := s.pool.Exec(ctx, `UPDATE browser_sessions SET revoked_at = now() WHERE id = $1 AND revoked_at IS NULL`, id)
	if err != nil {
		return fmt.Errorf("revoke browser session: %w", err)
	}
	return nil
}

func (s *Store) RevokeMemberSessions(ctx context.Context, memberID int64, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		UPDATE browser_sessions SET revoked_at = now()
		WHERE membership_id = $1 AND revoked_at IS NULL`, memberID)
	if err != nil {
		return fmt.Errorf("revoke member sessions: %w", err)
	}
	return nil
}

func (s *Store) ListCatalog(ctx context.Context, query string, limit, offset int) ([]CatalogProject, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, slug, description, source_links
		FROM public_catalog_projects
		WHERE state IN ('active', 'paused')
		AND ($1 = '' OR name ILIKE '%' || $1 || '%' OR description ILIKE '%' || $1 || '%')
		ORDER BY lower(name), id
		LIMIT $2 OFFSET $3`, strings.TrimSpace(query), limit, max(offset, 0))
	if err != nil {
		return nil, fmt.Errorf("list public catalog: %w", err)
	}
	defer rows.Close()
	projects := make([]CatalogProject, 0)
	for rows.Next() {
		var project CatalogProject
		var links []byte
		if err := rows.Scan(&project.ID, &project.Name, &project.Slug, &project.Description, &links); err != nil {
			return nil, fmt.Errorf("scan public catalog project: %w", err)
		}
		if err := json.Unmarshal(links, &project.SourceLinks); err != nil {
			return nil, fmt.Errorf("decode public catalog links: %w", err)
		}
		projects = append(projects, project)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate public catalog: %w", err)
	}
	return projects, nil
}

func (s *Store) GetCatalogProject(ctx context.Context, id int64) (CatalogProject, error) {
	var project CatalogProject
	var links []byte
	err := s.pool.QueryRow(ctx, `
		SELECT id, name, slug, description, source_links FROM public_catalog_projects
		WHERE id = $1 AND state IN ('active', 'paused')`, id).Scan(
		&project.ID, &project.Name, &project.Slug, &project.Description, &links)
	if errors.Is(err, pgx.ErrNoRows) {
		return CatalogProject{}, access.ErrNotFound
	}
	if err != nil {
		return CatalogProject{}, fmt.Errorf("get public catalog project: %w", err)
	}
	if err := json.Unmarshal(links, &project.SourceLinks); err != nil {
		return CatalogProject{}, fmt.Errorf("decode public catalog links: %w", err)
	}
	return project, nil
}

func (s *Store) ListMembers(ctx context.Context, filter MemberFilter) ([]access.Member, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	rows, err := s.pool.Query(ctx, `
		SELECT m.id, m.identity_id, i.display_name, i.email, COALESCE(m.role, ''), m.status,
			m.locale, m.timezone, m.version, m.requested_at
		FROM memberships m JOIN external_identities i ON i.id = m.identity_id
		WHERE m.workspace_id = $1 AND m.status <> 'deleted'
			AND ($2 = '' OR m.status = $2)
			AND ($3 = '' OR m.role = $3)
			AND ($4 = '' OR i.display_name ILIKE '%' || $4 || '%' OR i.email ILIKE '%' || $4 || '%')
		ORDER BY m.requested_at, m.id LIMIT $5 OFFSET $6`, WorkspaceID, filter.State, filter.Role,
		strings.TrimSpace(filter.Query), limit, max(filter.Offset, 0))
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}
	defer rows.Close()
	members := make([]access.Member, 0)
	for rows.Next() {
		member, scanErr := scanMember(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan member: %w", scanErr)
		}
		members = append(members, member)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate members: %w", err)
	}
	return members, nil
}

func (s *Store) ApproveMember(
	ctx context.Context,
	actor access.Principal,
	memberID int64,
	decision string,
	role access.Role,
	requestID string,
) (access.Member, error) {
	if err := access.Authorize(actor, access.ActionMembershipGovern); err != nil {
		return access.Member{}, err
	}
	if decision != "approve" && decision != "reject" {
		return access.Member{}, fmt.Errorf("%w: decision must be approve or reject", access.ErrInvalidInput)
	}
	if decision == "approve" {
		if err := access.ValidateRole(role, false); err != nil {
			return access.Member{}, err
		}
	} else {
		role = ""
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return access.Member{}, fmt.Errorf("begin member approval: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	current, err := scanMember(tx.QueryRow(ctx, `
		SELECT m.id, m.identity_id, i.display_name, i.email, COALESCE(m.role, ''), m.status,
			m.locale, m.timezone, m.version, m.requested_at
		FROM memberships m JOIN external_identities i ON i.id = m.identity_id
		WHERE m.id = $1 AND m.workspace_id = $2 FOR UPDATE`, memberID, WorkspaceID))
	if errors.Is(err, pgx.ErrNoRows) {
		return access.Member{}, access.ErrNotFound
	}
	if err != nil {
		return access.Member{}, fmt.Errorf("lock member approval: %w", err)
	}
	status := access.StatusRejected
	if decision == "approve" {
		status = access.StatusActive
	}
	if current.Status == status && current.Role == role {
		if err := tx.Commit(ctx); err != nil {
			return access.Member{}, fmt.Errorf("commit repeated member approval: %w", err)
		}
		return current, nil
	}
	member, err := scanMember(tx.QueryRow(ctx, `
		UPDATE memberships m SET role = NULLIF($2, ''), status = $3, decided_at = now(),
			version = CASE WHEN role IS DISTINCT FROM NULLIF($2, '') OR status <> $3 THEN version + 1 ELSE version END
		FROM external_identities i
		WHERE m.id = $1 AND m.identity_id = i.id AND m.workspace_id = $4
			AND m.status IN ('pending', 'active', 'rejected', 'suspended')
		RETURNING m.id, m.identity_id, i.display_name, i.email, COALESCE(m.role, ''), m.status,
			m.locale, m.timezone, m.version, m.requested_at`, memberID, role, status, WorkspaceID))
	if errors.Is(err, pgx.ErrNoRows) {
		return access.Member{}, access.ErrNotFound
	}
	if err != nil {
		return access.Member{}, fmt.Errorf("approve member: %w", err)
	}
	if err := s.appendAudit(ctx, tx, actor, "membership."+decision, "membership", memberID,
		"succeeded", requestID, map[string]any{"role": role, "status": status}); err != nil {
		return access.Member{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return access.Member{}, fmt.Errorf("commit member approval: %w", err)
	}
	return member, nil
}

func (s *Store) UpdateMember(
	ctx context.Context,
	actor access.Principal,
	memberID, expectedVersion int64,
	role access.Role,
	status access.Status,
	requestID string,
) (access.Member, error) {
	if err := access.Authorize(actor, access.ActionMembershipGovern); err != nil {
		return access.Member{}, err
	}
	if role != "" {
		if err := access.ValidateRole(role, false); err != nil {
			return access.Member{}, err
		}
	}
	if status != "" && status != access.StatusActive && status != access.StatusSuspended {
		return access.Member{}, fmt.Errorf("%w: member state must be active or suspended", access.ErrInvalidInput)
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return access.Member{}, fmt.Errorf("begin member update: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	target, err := scanMember(tx.QueryRow(ctx, `
		SELECT m.id, m.identity_id, i.display_name, i.email, COALESCE(m.role, ''), m.status,
			m.locale, m.timezone, m.version, m.requested_at
		FROM memberships m JOIN external_identities i ON i.id = m.identity_id
		WHERE m.id = $1 AND m.workspace_id = $2 FOR UPDATE`, memberID, WorkspaceID))
	if errors.Is(err, pgx.ErrNoRows) {
		return access.Member{}, access.ErrNotFound
	}
	if err != nil {
		return access.Member{}, fmt.Errorf("lock member: %w", err)
	}
	if target.Version != expectedVersion {
		return access.Member{}, access.ErrVersionConflict
	}
	if role == "" {
		role = target.Role
	}
	if status == "" {
		status = target.Status
	}
	var activeAdmins int
	if err := tx.QueryRow(ctx, `
		SELECT count(*) FROM memberships WHERE workspace_id = $1 AND role = 'admin' AND status = 'active'`,
		WorkspaceID).Scan(&activeAdmins); err != nil {
		return access.Member{}, fmt.Errorf("count active admins: %w", err)
	}
	if err := access.ProtectLastAdmin(target, activeAdmins, role, status); err != nil {
		return access.Member{}, err
	}
	if target.Role != role || target.Status != status {
		if err := s.RevokeMemberSessions(ctx, memberID, tx); err != nil {
			return access.Member{}, err
		}
	}
	updated, err := scanMember(tx.QueryRow(ctx, `
		UPDATE memberships m SET role = $2, status = $3, version = version + 1
		FROM external_identities i WHERE m.id = $1 AND m.identity_id = i.id
		RETURNING m.id, m.identity_id, i.display_name, i.email, m.role, m.status,
			m.locale, m.timezone, m.version, m.requested_at`, memberID, role, status))
	if err != nil {
		return access.Member{}, fmt.Errorf("update member: %w", err)
	}
	if err := s.appendAudit(ctx, tx, actor, "membership.update", "membership", memberID,
		"succeeded", requestID, map[string]any{"role": role, "status": status}); err != nil {
		return access.Member{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return access.Member{}, fmt.Errorf("commit member update: %w", err)
	}
	return updated, nil
}

func (s *Store) UpdatePreferences(
	ctx context.Context,
	actor access.Principal,
	expectedVersion int64,
	locale, timezone, requestID string,
) (access.Member, error) {
	if !actor.IsApproved() || actor.Kind != access.ActorMember {
		return access.Member{}, access.ErrPermissionDenied
	}
	if err := access.ValidatePreferences(locale, timezone); err != nil {
		return access.Member{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return access.Member{}, fmt.Errorf("begin preference update: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	member, err := scanMember(tx.QueryRow(ctx, `
		UPDATE memberships m SET locale = $2, timezone = $3, version = version + 1
		FROM external_identities i
		WHERE m.id = $1 AND m.identity_id = i.id AND m.version = $4 AND m.status = 'active'
		RETURNING m.id, m.identity_id, i.display_name, i.email, m.role, m.status,
			m.locale, m.timezone, m.version, m.requested_at`, actor.ActorID, locale, timezone, expectedVersion))
	if errors.Is(err, pgx.ErrNoRows) {
		return access.Member{}, access.ErrVersionConflict
	}
	if err != nil {
		return access.Member{}, fmt.Errorf("update preferences: %w", err)
	}
	if err := s.appendAudit(ctx, tx, actor, "member.preferences.update", "membership", actor.ActorID,
		"succeeded", requestID, map[string]any{"locale": locale, "timezone": timezone}); err != nil {
		return access.Member{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return access.Member{}, fmt.Errorf("commit preference update: %w", err)
	}
	return member, nil
}

func (s *Store) DeleteAccount(
	ctx context.Context,
	actor access.Principal,
	confirmation, requestID string,
) (int64, error) {
	if actor.Kind != access.ActorMember || actor.ActorID == 0 {
		return 0, access.ErrPermissionDenied
	}
	if err := access.ValidateDeletionConfirmation(confirmation); err != nil {
		return 0, err
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return 0, fmt.Errorf("begin account deletion: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	target, err := scanMember(tx.QueryRow(ctx, `
		SELECT m.id, m.identity_id, i.display_name, i.email, COALESCE(m.role, ''), m.status,
			m.locale, m.timezone, m.version, m.requested_at
		FROM memberships m JOIN external_identities i ON i.id = m.identity_id
		WHERE m.id = $1 FOR UPDATE`, actor.ActorID))
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, access.ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("lock account for deletion: %w", err)
	}
	if target.Status == access.StatusDeleted {
		var existingJobID int64
		if err := tx.QueryRow(ctx, `SELECT deletion_job_id FROM memberships WHERE id = $1`, actor.ActorID).
			Scan(&existingJobID); err != nil {
			return 0, fmt.Errorf("read completed account deletion: %w", err)
		}
		return existingJobID, nil
	}
	var activeAdmins int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM memberships
		WHERE workspace_id = $1 AND role = 'admin' AND status = 'active'`, WorkspaceID).Scan(&activeAdmins); err != nil {
		return 0, fmt.Errorf("count active admins: %w", err)
	}
	if err := access.ProtectLastAdmin(target, activeAdmins, "", access.StatusDeleted); err != nil {
		return 0, err
	}
	if err := s.RevokeMemberSessions(ctx, actor.ActorID, tx); err != nil {
		return 0, err
	}
	_, err = tx.Exec(ctx, `UPDATE external_identities
		SET display_name = '', email = '', updated_at = now() WHERE id = $1`, target.IdentityID)
	if err != nil {
		return 0, fmt.Errorf("delete account profile: %w", err)
	}
	jobID, err := s.ids.Next(ctx)
	if err != nil {
		return 0, fmt.Errorf("issue account deletion job ID: %w", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO jobs (id, kind, state, checkpoint)
		VALUES ($1, 'account_deletion', 'succeeded', '{"personal_data":"purged"}')`, jobID); err != nil {
		return 0, fmt.Errorf("record account deletion job: %w", err)
	}
	_, err = tx.Exec(ctx, `UPDATE memberships
		SET role = NULL, status = 'deleted', locale = 'en', timezone = 'UTC',
			version = version + 1, deleted_at = now(), deletion_job_id = $2 WHERE id = $1`, actor.ActorID, jobID)
	if err != nil {
		return 0, fmt.Errorf("delete account membership: %w", err)
	}
	deletedActor := actor
	deletedActor.Kind = access.ActorDeleted
	if err := s.appendAudit(ctx, tx, deletedActor, "member.delete", "membership", actor.ActorID,
		"succeeded", requestID, map[string]any{"personal_data": "purged"}); err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit account deletion: %w", err)
	}
	return jobID, nil
}

func (s *Store) ListServiceAccounts(ctx context.Context, query string, limit, offset int) ([]access.ServiceAccount, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	rows, err := s.pool.Query(ctx, `
		SELECT sa.id, sa.issuer, sa.external_subject, sa.name, sa.role, sa.status, sa.version,
			COALESCE(array_agg(s.scope ORDER BY s.scope) FILTER (WHERE s.scope IS NOT NULL), '{}')
		FROM service_accounts sa LEFT JOIN service_account_scopes s ON s.service_account_id = sa.id
		WHERE sa.workspace_id = $1 AND sa.status <> 'deleted'
			AND ($2 = '' OR sa.name ILIKE '%' || $2 || '%' OR sa.external_subject ILIKE '%' || $2 || '%')
		GROUP BY sa.id ORDER BY lower(sa.name), sa.id LIMIT $3 OFFSET $4`, WorkspaceID,
		strings.TrimSpace(query), limit, max(offset, 0))
	if err != nil {
		return nil, fmt.Errorf("list service accounts: %w", err)
	}
	defer rows.Close()
	accounts := make([]access.ServiceAccount, 0)
	for rows.Next() {
		var account access.ServiceAccount
		if err := rows.Scan(&account.ID, &account.Issuer, &account.ExternalSubject, &account.Name,
			&account.Role, &account.Status, &account.Version, &account.Scopes); err != nil {
			return nil, fmt.Errorf("scan service account: %w", err)
		}
		accounts = append(accounts, account)
	}
	return accounts, rows.Err()
}

func (s *Store) CreateServiceAccount(
	ctx context.Context,
	actor access.Principal,
	account access.ServiceAccount,
	requestID string,
) (access.ServiceAccount, error) {
	if err := access.Authorize(actor, access.ActionServiceGovern); err != nil {
		return access.ServiceAccount{}, err
	}
	if err := access.ValidateServiceAccount(account, AllowedServiceScopes); err != nil {
		return access.ServiceAccount{}, err
	}
	id, err := s.ids.Next(ctx)
	if err != nil {
		return access.ServiceAccount{}, fmt.Errorf("issue service account ID: %w", err)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return access.ServiceAccount{}, fmt.Errorf("begin service account creation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	_, err = tx.Exec(ctx, `INSERT INTO service_accounts
		(id, workspace_id, issuer, external_subject, name, role, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`, id, WorkspaceID, account.Issuer,
		account.ExternalSubject, account.Name, account.Role, account.Status)
	if err != nil {
		return access.ServiceAccount{}, fmt.Errorf("create service account: %w", err)
	}
	for _, scope := range account.Scopes {
		if _, err := tx.Exec(ctx, `INSERT INTO service_account_scopes (service_account_id, scope)
			VALUES ($1, $2)`, id, scope); err != nil {
			return access.ServiceAccount{}, fmt.Errorf("create service account scope: %w", err)
		}
	}
	account.ID, account.Version = id, 1
	if err := s.appendAudit(ctx, tx, actor, "service_account.create", "service_account", id,
		"succeeded", requestID, map[string]any{"role": account.Role, "scopes": account.Scopes}); err != nil {
		return access.ServiceAccount{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return access.ServiceAccount{}, fmt.Errorf("commit service account creation: %w", err)
	}
	return account, nil
}

func (s *Store) UpdateServiceAccount(
	ctx context.Context,
	actor access.Principal,
	accountID, expectedVersion int64,
	role access.Role,
	status access.Status,
	scopes []string,
	requestID string,
) (access.ServiceAccount, error) {
	if err := access.Authorize(actor, access.ActionServiceGovern); err != nil {
		return access.ServiceAccount{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return access.ServiceAccount{}, fmt.Errorf("begin service account update: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var current access.ServiceAccount
	err = tx.QueryRow(ctx, `SELECT id, issuer, external_subject, name, role, status, version
		FROM service_accounts WHERE id = $1 AND workspace_id = $2 FOR UPDATE`, accountID, WorkspaceID).Scan(
		&current.ID, &current.Issuer, &current.ExternalSubject, &current.Name,
		&current.Role, &current.Status, &current.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		return access.ServiceAccount{}, access.ErrNotFound
	}
	if err != nil {
		return access.ServiceAccount{}, fmt.Errorf("lock service account: %w", err)
	}
	if current.Version != expectedVersion {
		return access.ServiceAccount{}, access.ErrVersionConflict
	}
	if role == "" {
		role = current.Role
	}
	if status == "" {
		status = current.Status
	}
	if scopes == nil {
		rows, queryErr := tx.Query(ctx, `SELECT scope FROM service_account_scopes
			WHERE service_account_id = $1 ORDER BY scope`, accountID)
		if queryErr != nil {
			return access.ServiceAccount{}, fmt.Errorf("load service account scopes: %w", queryErr)
		}
		scopes, err = pgx.CollectRows(rows, pgx.RowTo[string])
		if err != nil {
			return access.ServiceAccount{}, fmt.Errorf("scan service account scopes: %w", err)
		}
	}
	next := current
	next.Role, next.Status, next.Scopes = role, status, scopes
	if err := access.ValidateServiceAccount(next, AllowedServiceScopes); err != nil {
		return access.ServiceAccount{}, err
	}
	_, err = tx.Exec(ctx, `UPDATE service_accounts SET role = $2, status = $3,
		version = version + 1, updated_at = now() WHERE id = $1`, accountID, role, status)
	if err != nil {
		return access.ServiceAccount{}, fmt.Errorf("update service account: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM service_account_scopes WHERE service_account_id = $1`, accountID); err != nil {
		return access.ServiceAccount{}, fmt.Errorf("replace service account scopes: %w", err)
	}
	for _, scope := range scopes {
		if _, err := tx.Exec(ctx, `INSERT INTO service_account_scopes (service_account_id, scope)
			VALUES ($1, $2)`, accountID, scope); err != nil {
			return access.ServiceAccount{}, fmt.Errorf("replace service account scope: %w", err)
		}
	}
	next.Version++
	if err := s.appendAudit(ctx, tx, actor, "service_account.update", "service_account", accountID,
		"succeeded", requestID, map[string]any{"role": role, "status": status, "scopes": scopes}); err != nil {
		return access.ServiceAccount{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return access.ServiceAccount{}, fmt.Errorf("commit service account update: %w", err)
	}
	return next, nil
}

func (s *Store) ListAudit(ctx context.Context, actor access.Principal, filter AuditFilter) ([]AuditEvent, error) {
	if err := access.Authorize(actor, access.ActionAuditRead); err != nil {
		return nil, err
	}
	if filter.Limit <= 0 {
		filter.Limit = 50
	}
	if filter.Limit > 200 {
		filter.Limit = 200
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, occurred_at, actor_id, actor_kind, action, resource_type, resource_id,
			outcome, request_id, changes FROM audit_events
		WHERE ($1::bigint IS NULL OR actor_id=$1)
		  AND ($2='' OR action=$2)
		  AND ($3='' OR resource_type=$3)
		  AND ($4='' OR outcome=$4)
		  AND ($5::timestamptz IS NULL OR occurred_at >= $5)
		  AND ($6::timestamptz IS NULL OR occurred_at <= $6)
		ORDER BY occurred_at DESC, id DESC LIMIT $7 OFFSET $8`, filter.ActorID,
		filter.Action, filter.Resource, filter.Outcome, filter.OccurredFrom, filter.OccurredTo,
		filter.Limit, max(filter.Offset, 0))
	if err != nil {
		return nil, fmt.Errorf("list audit events: %w", err)
	}
	defer rows.Close()
	events := make([]AuditEvent, 0)
	for rows.Next() {
		var event AuditEvent
		var changes []byte
		if err := rows.Scan(&event.ID, &event.OccurredAt, &event.ActorID, &event.ActorKind,
			&event.Action, &event.ResourceType, &event.ResourceID, &event.Outcome,
			&event.RequestID, &changes); err != nil {
			return nil, fmt.Errorf("scan audit event: %w", err)
		}
		if err := json.Unmarshal(changes, &event.Changes); err != nil {
			return nil, fmt.Errorf("decode audit changes: %w", err)
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func (s *Store) appendAudit(
	ctx context.Context,
	tx pgx.Tx,
	actor access.Principal,
	action, resourceType string,
	resourceID int64,
	outcome, requestID string,
	changes map[string]any,
) error {
	id, err := s.ids.Next(ctx)
	if err != nil {
		return fmt.Errorf("issue audit event ID: %w", err)
	}
	encoded, err := json.Marshal(changes)
	if err != nil {
		return fmt.Errorf("encode audit changes: %w", err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO audit_events
		(id, actor_id, actor_kind, action, resource_type, resource_id, outcome, request_id, changes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`, id, actor.ActorID, actor.Kind,
		action, resourceType, resourceID, outcome, requestID, encoded)
	if err != nil {
		return fmt.Errorf("append audit event: %w", err)
	}
	return nil
}

// RecordHTTPMutation appends the transport outcome without retaining request
// bodies, credentials, query strings, or response payloads. Domain audit rows
// remain the detailed successful command record; this boundary row proves that
// denied, stale, and failed attempts are also attributable and immutable.
func (s *Store) RecordHTTPMutation(
	ctx context.Context,
	actor access.Principal,
	method, path string,
	status int,
	requestID string,
) error {
	if actor.Kind == "" {
		actor.Kind = access.ActorSystem
	}
	outcome := "succeeded"
	switch {
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		outcome = "denied"
	case status == http.StatusConflict || status == http.StatusPreconditionFailed ||
		status == http.StatusPreconditionRequired:
		outcome = "stale"
	case status >= http.StatusBadRequest:
		outcome = "failed"
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin HTTP mutation audit: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := s.appendAudit(ctx, tx, actor, strings.ToLower(method)+" "+path,
		"http_route", 0, outcome, requestID, map[string]any{"status": status}); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit HTTP mutation audit: %w", err)
	}
	return nil
}

func scanMember(row pgx.Row) (access.Member, error) {
	var member access.Member
	err := row.Scan(&member.ID, &member.IdentityID, &member.DisplayName, &member.Email, &member.Role,
		&member.Status, &member.Locale, &member.Timezone, &member.Version, &member.RequestedAt)
	return member, err
}
