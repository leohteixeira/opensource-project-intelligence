// Package agent owns the bounded assistant proposal and confirmation protocol.
//
// Model adapters may suggest a typed action, but authorization, validation,
// expiry, optimistic concurrency, quota, and execution remain deterministic.
package agent

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/leohteixeira/opensource-project-intelligence/internal/access"
)

const ProposalLifetime = 10 * time.Minute

type Action string

const ActionRepositoryAdd Action = "repository.add"

type Status string

const (
	AwaitingConfirmation Status = "awaiting_confirmation"
	Executing            Status = "executing"
	Executed             Status = "executed"
	Failed               Status = "failed"
	Expired              Status = "expired"
)

var (
	ErrInvalid          = errors.New("invalid assistant proposal")
	ErrActionNotAllowed = errors.New("assistant action not allowed")
	ErrExpired          = errors.New("assistant proposal expired")
	ErrAlreadyUsed      = errors.New("assistant proposal already used")
	ErrStateChanged     = errors.New("assistant proposal state changed")
	ErrQuotaExceeded    = errors.New("assistant action quota exceeded")
)

// RepositoryAdd is the complete, typed input accepted by the sole executable
// assistant tool. No map or provider-owned value reaches the executor.
type RepositoryAdd struct {
	ProjectID      int64  `json:"project_id,string"`
	ProjectVersion int64  `json:"project_version"`
	URL            string `json:"url"`
	Role           string `json:"role"`
}

type Draft struct {
	Action       Action        `json:"action"`
	Repository   RepositoryAdd `json:"repository"`
	Effect       string        `json:"effect"`
	QuotaName    string        `json:"quota_name"`
	QuotaCost    int           `json:"quota_cost"`
	QuotaLimit   int           `json:"quota_limit"`
	QuotaUsed    int           `json:"quota_used"`
	ActionCount  int           `json:"action_count"`
	Unrecognized []string      `json:"unrecognized,omitempty"`
}

func (draft Draft) Validate() error {
	if draft.Action != ActionRepositoryAdd || draft.ActionCount != 1 || len(draft.Unrecognized) != 0 {
		return ErrActionNotAllowed
	}
	input := draft.Repository
	if input.ProjectID <= 0 || input.ProjectVersion <= 0 ||
		!strings.HasPrefix(input.URL, "https://") || strings.TrimSpace(input.Role) == "" ||
		strings.TrimSpace(draft.Effect) == "" || draft.QuotaName != "project_repositories" ||
		draft.QuotaCost != 1 || draft.QuotaLimit <= 0 || draft.QuotaUsed < 0 {
		return ErrInvalid
	}
	if draft.QuotaUsed+draft.QuotaCost > draft.QuotaLimit {
		return ErrQuotaExceeded
	}
	return nil
}

type Proposal struct {
	ID                int64         `json:"id,string"`
	WorkspaceID       int64         `json:"-"`
	ActorID           int64         `json:"-"`
	ActorVersion      int64         `json:"-"`
	Status            Status        `json:"status"`
	Action            Action        `json:"action"`
	Inputs            RepositoryAdd `json:"inputs"`
	Resources         []string      `json:"resources"`
	Effect            string        `json:"effect"`
	Quota             Quota         `json:"quota"`
	ExpiresAt         time.Time     `json:"expires_at"`
	ConfirmationToken string        `json:"confirmation_token,omitempty"`
	TokenDigest       [32]byte      `json:"-"`
	CreatedAt         time.Time     `json:"created_at"`
	ConsumedAt        *time.Time    `json:"consumed_at,omitempty"`
	Result            Result        `json:"result,omitempty"`
}

type Quota struct {
	Name      string `json:"name"`
	Cost      int    `json:"cost"`
	Remaining int    `json:"remaining"`
}

type Result struct {
	RepositoryID int64 `json:"repository_id,string,omitempty"`
	AuditEventID int64 `json:"audit_event_id,string,omitempty"`
}

type Snapshot struct {
	ProjectVersion int64
	Lifecycle      string
	QuotaUsed      int
	QuotaLimit     int
}

type Planner interface {
	Plan(context.Context, access.Principal, string) (Draft, error)
}

type StateReader interface {
	Snapshot(context.Context, access.Principal, RepositoryAdd) (Snapshot, error)
}

type Executor interface {
	AddRepository(context.Context, access.Principal, RepositoryAdd, string) (Result, error)
}

type Store interface {
	Create(context.Context, Proposal, string, [32]byte) (Proposal, bool, error)
	Begin(context.Context, int64, access.Principal, [32]byte, string, time.Time) (Proposal, bool, error)
	Finish(context.Context, int64, Status, Result, time.Time) (Proposal, error)
}

type IDSource interface {
	Next(context.Context) (int64, error)
}

type TokenSource func() (string, error)

type Service struct {
	planner  Planner
	state    StateReader
	executor Executor
	store    Store
	ids      IDSource
	now      func() time.Time
	token    TokenSource
}

func New(planner Planner, state StateReader, executor Executor, store Store, ids IDSource) (*Service, error) {
	if planner == nil || state == nil || executor == nil || store == nil || ids == nil {
		return nil, errors.New("assistant planner, state reader, executor, store, and ID source are required")
	}
	return &Service{planner: planner, state: state, executor: executor, store: store, ids: ids,
		now: time.Now, token: randomToken}, nil
}

func (service *Service) Propose(
	ctx context.Context,
	principal access.Principal,
	message, idempotencyKey string,
) (Proposal, error) {
	if err := access.Authorize(principal, access.ActionProjectWrite); err != nil {
		return Proposal{}, err
	}
	message = strings.TrimSpace(message)
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if message == "" || len(message) > 4000 || idempotencyKey == "" {
		return Proposal{}, ErrInvalid
	}
	draft, err := service.planner.Plan(ctx, principal, message)
	if err != nil {
		return Proposal{}, err
	}
	if err := draft.Validate(); err != nil {
		return Proposal{}, err
	}
	snapshot, err := service.state.Snapshot(ctx, principal, draft.Repository)
	if err != nil {
		return Proposal{}, err
	}
	if snapshot.Lifecycle != "active" || snapshot.ProjectVersion != draft.Repository.ProjectVersion {
		return Proposal{}, ErrStateChanged
	}
	if snapshot.QuotaLimit <= 0 || snapshot.QuotaUsed+draft.QuotaCost > snapshot.QuotaLimit {
		return Proposal{}, ErrQuotaExceeded
	}
	id, err := service.ids.Next(ctx)
	if err != nil {
		return Proposal{}, fmt.Errorf("issue assistant proposal ID: %w", err)
	}
	token, err := service.token()
	if err != nil {
		return Proposal{}, fmt.Errorf("issue assistant confirmation token: %w", err)
	}
	now := service.now().UTC()
	proposal := Proposal{
		ID: id, WorkspaceID: principal.Workspace, ActorID: principal.ActorID,
		ActorVersion: principal.Version, Status: AwaitingConfirmation,
		Action: draft.Action, Inputs: draft.Repository,
		Resources: []string{fmt.Sprintf("project:%d", draft.Repository.ProjectID), draft.Repository.URL},
		Effect:    draft.Effect, Quota: Quota{Name: draft.QuotaName, Cost: draft.QuotaCost,
			Remaining: snapshot.QuotaLimit - snapshot.QuotaUsed - draft.QuotaCost},
		ExpiresAt: now.Add(ProposalLifetime), CreatedAt: now, TokenDigest: sha256.Sum256([]byte(token)),
	}
	digest := requestDigest(message)
	created, replay, err := service.store.Create(ctx, proposal, idempotencyKey, digest)
	if err != nil {
		return Proposal{}, err
	}
	if !replay {
		created.ConfirmationToken = token
	}
	return created, nil
}

func (service *Service) Confirm(
	ctx context.Context,
	principal access.Principal,
	proposalID int64,
	token, idempotencyKey string,
) (Proposal, error) {
	if err := access.Authorize(principal, access.ActionProjectWrite); err != nil {
		return Proposal{}, err
	}
	if proposalID <= 0 || strings.TrimSpace(token) == "" || strings.TrimSpace(idempotencyKey) == "" {
		return Proposal{}, ErrInvalid
	}
	now := service.now().UTC()
	proposal, replay, err := service.store.Begin(ctx, proposalID, principal,
		sha256.Sum256([]byte(token)), idempotencyKey, now)
	if err != nil || replay {
		return proposal, err
	}
	finish := func(status Status, result Result) (Proposal, error) {
		return service.store.Finish(ctx, proposal.ID, status, result, service.now().UTC())
	}
	if principal.Version != proposal.ActorVersion {
		_, _ = finish(Failed, Result{})
		return Proposal{}, ErrStateChanged
	}
	snapshot, err := service.state.Snapshot(ctx, principal, proposal.Inputs)
	if err != nil {
		_, _ = finish(Failed, Result{})
		return Proposal{}, err
	}
	if snapshot.Lifecycle != "active" || snapshot.ProjectVersion != proposal.Inputs.ProjectVersion {
		_, _ = finish(Failed, Result{})
		return Proposal{}, ErrStateChanged
	}
	if snapshot.QuotaUsed+proposal.Quota.Cost > snapshot.QuotaLimit {
		_, _ = finish(Failed, Result{})
		return Proposal{}, ErrQuotaExceeded
	}
	result, err := service.executor.AddRepository(ctx, principal, proposal.Inputs, idempotencyKey)
	if err != nil {
		_, finishErr := finish(Failed, Result{})
		return Proposal{}, errors.Join(err, finishErr)
	}
	return finish(Executed, result)
}

func randomToken() (string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func requestDigest(value string) [32]byte { return sha256.Sum256([]byte(strings.TrimSpace(value))) }
