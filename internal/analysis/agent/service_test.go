package agent

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/leohteixeira/opensource-project-intelligence/internal/access"
)

type fakeIDs struct{ next int64 }

func (ids *fakeIDs) Next(context.Context) (int64, error) { ids.next++; return ids.next, nil }

type fakePlanner struct {
	draft Draft
	err   error
}

func (planner fakePlanner) Plan(context.Context, access.Principal, string) (Draft, error) {
	return planner.draft, planner.err
}

type fakeState struct {
	snapshot Snapshot
	err      error
}

func (state *fakeState) Snapshot(context.Context, access.Principal, RepositoryAdd) (Snapshot, error) {
	return state.snapshot, state.err
}

type fakeExecutor struct {
	mu    sync.Mutex
	calls int
	err   error
}

func (executor *fakeExecutor) AddRepository(context.Context, access.Principal, RepositoryAdd, string) (Result, error) {
	executor.mu.Lock()
	defer executor.mu.Unlock()
	executor.calls++
	return Result{RepositoryID: 88, AuditEventID: 99}, executor.err
}

func validPrincipal() access.Principal {
	return access.Principal{ActorID: 7, Workspace: 3, Version: 4, Kind: access.ActorMember,
		Role: access.RoleAnalyst, Status: access.StatusActive}
}

func validDraft() Draft {
	return Draft{Action: ActionRepositoryAdd, ActionCount: 1,
		Repository: RepositoryAdd{ProjectID: 42, ProjectVersion: 5,
			URL: "https://github.com/temporalio/sdk-go", Role: "sdk"},
		Effect:    "Adds one public Repository. Synchronization is a separate action.",
		QuotaName: "project_repositories", QuotaCost: 1, QuotaLimit: 20, QuotaUsed: 1}
}

func serviceFixture(t *testing.T) (*Service, *fakeState, *fakeExecutor) {
	t.Helper()
	state := &fakeState{snapshot: Snapshot{ProjectVersion: 5, Lifecycle: "active", QuotaUsed: 1, QuotaLimit: 20}}
	executor := &fakeExecutor{}
	service, err := New(fakePlanner{draft: validDraft()}, state, executor, NewMemoryStore(), &fakeIDs{next: 100})
	if err != nil {
		t.Fatal(err)
	}
	service.now = func() time.Time { return time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC) }
	service.token = func() (string, error) { return "single-use-secret", nil }
	return service, state, executor
}

func TestTypedRepositoryProposalContainsExactPreview(t *testing.T) {
	service, _, _ := serviceFixture(t)
	proposal, err := service.Propose(context.Background(), validPrincipal(), "Add the SDK repository", "create-1")
	if err != nil {
		t.Fatal(err)
	}
	if proposal.Action != ActionRepositoryAdd || len(proposal.Resources) != 2 || proposal.Quota.Cost != 1 ||
		proposal.ConfirmationToken == "" || proposal.ExpiresAt.Sub(proposal.CreatedAt) != ProposalLifetime {
		t.Fatalf("proposal = %#v", proposal)
	}
}

func TestUT169RejectsUntypedUnsupportedProposalFields(t *testing.T) {
	for _, change := range []func(*Draft){
		func(d *Draft) { d.Action = "credentials.rotate" },
		func(d *Draft) { d.Unrecognized = []string{"raw_sql"} },
		func(d *Draft) { d.ActionCount = 2 },
	} {
		draft := validDraft()
		change(&draft)
		if err := draft.Validate(); !errors.Is(err, ErrActionNotAllowed) {
			t.Fatalf("Validate() = %v", err)
		}
	}
}

func TestUT170MissingRequiredValueRequiresClarification(t *testing.T) {
	draft := validDraft()
	draft.Repository.URL = ""
	if !errors.Is(draft.Validate(), ErrInvalid) {
		t.Fatal("missing URL was accepted")
	}
}

func TestUT171QuotaLimitedProposalCannotBeApproved(t *testing.T) {
	draft := validDraft()
	draft.QuotaUsed = draft.QuotaLimit
	if !errors.Is(draft.Validate(), ErrQuotaExceeded) {
		t.Fatal("exhausted quota was accepted")
	}
}

func TestConfirmationIsActorBoundAndTokenBound(t *testing.T) {
	service, _, executor := serviceFixture(t)
	proposal, _ := service.Propose(context.Background(), validPrincipal(), "Add", "create-1")
	other := validPrincipal()
	other.ActorID++
	if _, err := service.Confirm(context.Background(), other, proposal.ID, proposal.ConfirmationToken, "confirm-1"); !errors.Is(err, access.ErrPermissionDenied) {
		t.Fatalf("other actor error = %v", err)
	}
	if _, err := service.Confirm(context.Background(), validPrincipal(), proposal.ID, "wrong", "confirm-1"); !errors.Is(err, access.ErrPermissionDenied) {
		t.Fatalf("wrong token error = %v", err)
	}
	if executor.calls != 0 {
		t.Fatalf("executor calls = %d", executor.calls)
	}
}

func TestUT172ApprovalRechecksCurrentRoleScopeAndResource(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*access.Principal, *fakeState)
	}{
		{"identity version", func(p *access.Principal, _ *fakeState) { p.Version++ }},
		{"resource version", func(_ *access.Principal, s *fakeState) { s.snapshot.ProjectVersion++ }},
		{"lifecycle", func(_ *access.Principal, s *fakeState) { s.snapshot.Lifecycle = "archived" }},
		{"quota", func(_ *access.Principal, s *fakeState) { s.snapshot.QuotaUsed = s.snapshot.QuotaLimit }},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			service, state, executor := serviceFixture(t)
			proposal, _ := service.Propose(context.Background(), validPrincipal(), "Add", "create-1")
			principal := validPrincipal()
			test.mutate(&principal, state)
			if _, err := service.Confirm(context.Background(), principal, proposal.ID, proposal.ConfirmationToken, "confirm-1"); err == nil {
				t.Fatal("changed state was accepted")
			}
			if executor.calls != 0 {
				t.Fatalf("executor calls = %d", executor.calls)
			}
		})
	}
}

func TestUT174ApprovalBeforePreviewOrAfterExpiryIsRejected(t *testing.T) {
	service, _, executor := serviceFixture(t)
	proposal, _ := service.Propose(context.Background(), validPrincipal(), "Add", "create-1")
	service.now = func() time.Time { return proposal.ExpiresAt }
	if _, err := service.Confirm(context.Background(), validPrincipal(), proposal.ID, proposal.ConfirmationToken, "confirm-1"); !errors.Is(err, ErrExpired) {
		t.Fatalf("expiry error = %v", err)
	}
	if executor.calls != 0 {
		t.Fatalf("executor calls = %d", executor.calls)
	}
}

func TestUT173ReplayedConfirmationCannotRepeatMutation(t *testing.T) {
	service, _, executor := serviceFixture(t)
	proposal, _ := service.Propose(context.Background(), validPrincipal(), "Add", "create-1")
	first, err := service.Confirm(context.Background(), validPrincipal(), proposal.ID, proposal.ConfirmationToken, "confirm-1")
	if err != nil || first.Status != Executed {
		t.Fatalf("Confirm() = %#v, %v", first, err)
	}
	replay, err := service.Confirm(context.Background(), validPrincipal(), proposal.ID, proposal.ConfirmationToken, "confirm-1")
	if err != nil || replay.Result.RepositoryID != first.Result.RepositoryID {
		t.Fatalf("replay = %#v, %v", replay, err)
	}
	if _, err := service.Confirm(context.Background(), validPrincipal(), proposal.ID, proposal.ConfirmationToken, "confirm-2"); !errors.Is(err, ErrAlreadyUsed) {
		t.Fatalf("second key error = %v", err)
	}
	if executor.calls != 1 {
		t.Fatalf("executor calls = %d", executor.calls)
	}
}

func TestViewerAdminOnlyAndServiceActorsCannotUseAssistant(t *testing.T) {
	roles := []access.Principal{
		{ActorID: 1, Kind: access.ActorMember, Role: access.RoleViewer, Status: access.StatusActive, Workspace: 1},
		{ActorID: 1, Kind: access.ActorServiceAccount, Role: access.RoleAnalyst, Status: access.StatusActive, Workspace: 1, Scopes: []string{"projects:read"}},
	}
	for _, principal := range roles {
		service, _, _ := serviceFixture(t)
		if _, err := service.Propose(context.Background(), principal, "Add", "key"); err == nil {
			t.Fatal("actor was allowed")
		}
	}
}

func TestProhibitedCategoriesNeverReachExecutor(t *testing.T) {
	for _, action := range []Action{"member.update", "credential.rotate", "policy.create", "project.archive", "project.delete"} {
		draft := validDraft()
		draft.Action = action
		state := &fakeState{snapshot: Snapshot{ProjectVersion: 5, Lifecycle: "active", QuotaLimit: 20}}
		executor := &fakeExecutor{}
		service, _ := New(fakePlanner{draft: draft}, state, executor, NewMemoryStore(), &fakeIDs{})
		if _, err := service.Propose(context.Background(), validPrincipal(), "unsafe", "key"); !errors.Is(err, ErrActionNotAllowed) {
			t.Fatalf("action %s error = %v", action, err)
		}
		if executor.calls != 0 {
			t.Fatal("prohibited action reached executor")
		}
	}
}

func TestCreateIdempotencyCannotChangeRequest(t *testing.T) {
	service, _, _ := serviceFixture(t)
	if _, err := service.Propose(context.Background(), validPrincipal(), "Add A", "same"); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Propose(context.Background(), validPrincipal(), "Add B", "same"); !errors.Is(err, ErrStateChanged) {
		t.Fatalf("idempotency mismatch error = %v", err)
	}
}

func TestUT175PausedArchivedAndDeletedProposalLifecycle(t *testing.T) {
	for _, lifecycle := range []string{"paused", "archived", "deleting", "deleted"} {
		service, state, _ := serviceFixture(t)
		state.snapshot.Lifecycle = lifecycle
		if _, err := service.Propose(context.Background(), validPrincipal(), "Add", lifecycle); !errors.Is(err, ErrStateChanged) {
			t.Fatalf("lifecycle %s error = %v", lifecycle, err)
		}
	}
}

func TestExecutionFailureIsTerminalAndAuditable(t *testing.T) {
	service, _, executor := serviceFixture(t)
	executor.err = errors.New("write failed")
	proposal, _ := service.Propose(context.Background(), validPrincipal(), "Add", "create")
	if _, err := service.Confirm(context.Background(), validPrincipal(), proposal.ID, proposal.ConfirmationToken, "confirm"); err == nil {
		t.Fatal("failure hidden")
	}
	if _, err := service.Confirm(context.Background(), validPrincipal(), proposal.ID, proposal.ConfirmationToken, "other"); !errors.Is(err, ErrAlreadyUsed) {
		t.Fatalf("retry error = %v", err)
	}
}

func TestConcurrentConfirmationsExecuteOnce(t *testing.T) {
	service, _, executor := serviceFixture(t)
	proposal, _ := service.Propose(context.Background(), validPrincipal(), "Add", "create")
	var wait sync.WaitGroup
	for index := 0; index < 16; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			_, _ = service.Confirm(context.Background(), validPrincipal(), proposal.ID, proposal.ConfirmationToken, "confirm")
		}()
	}
	wait.Wait()
	if executor.calls != 1 {
		t.Fatalf("executor calls = %d", executor.calls)
	}
}

func TestADKBoundaryHasOneNamedAllowlistedTool(t *testing.T) {
	if proposalToolName != "propose_repository_add" {
		t.Fatalf("tool name = %q", proposalToolName)
	}
}

func TestIT073ChangedResourceInvalidatesProposalPreview(t *testing.T) {
	service, state, executor := serviceFixture(t)
	proposal, _ := service.Propose(context.Background(), validPrincipal(), "Add", "create")
	state.snapshot.ProjectVersion++
	if _, err := service.Confirm(context.Background(), validPrincipal(), proposal.ID,
		proposal.ConfirmationToken, "confirm"); !errors.Is(err, ErrStateChanged) {
		t.Fatalf("changed resource error = %v", err)
	}
	if executor.calls != 0 {
		t.Fatal("changed preview executed")
	}
}

func TestIT074ExpiredConfirmationExecutesNothing(t *testing.T) {
	service, _, executor := serviceFixture(t)
	proposal, _ := service.Propose(context.Background(), validPrincipal(), "Add", "create")
	service.now = func() time.Time { return proposal.ExpiresAt }
	_, _ = service.Confirm(context.Background(), validPrincipal(), proposal.ID,
		proposal.ConfirmationToken, "confirm")
	if executor.calls != 0 {
		t.Fatal("expired proposal executed")
	}
}

func TestIT075ManyActionsAreRejectedAtomically(t *testing.T) {
	draft := validDraft()
	draft.ActionCount = 2
	state := &fakeState{snapshot: Snapshot{ProjectVersion: 5, Lifecycle: "active", QuotaLimit: 20}}
	executor := &fakeExecutor{}
	service, _ := New(fakePlanner{draft: draft}, state, executor, NewMemoryStore(), &fakeIDs{})
	if _, err := service.Propose(context.Background(), validPrincipal(), "two actions", "create"); !errors.Is(err, ErrActionNotAllowed) {
		t.Fatalf("multi-action error = %v", err)
	}
	if executor.calls != 0 {
		t.Fatal("part of a rejected multi-action proposal executed")
	}
}

func TestIT085SimultaneousConfirmationsHaveOneDeterministicReceipt(t *testing.T) {
	service, _, executor := serviceFixture(t)
	proposal, _ := service.Propose(context.Background(), validPrincipal(), "Add", "create")
	var wait sync.WaitGroup
	for range 8 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			_, _ = service.Confirm(context.Background(), validPrincipal(), proposal.ID,
				proposal.ConfirmationToken, "confirm")
		}()
	}
	wait.Wait()
	if executor.calls != 1 {
		t.Fatalf("mutation count = %d", executor.calls)
	}
}

func TestIT086FailedActionLeavesNoSuccessfulMutation(t *testing.T) {
	service, _, executor := serviceFixture(t)
	executor.err = errors.New("database unavailable")
	proposal, _ := service.Propose(context.Background(), validPrincipal(), "Add", "create")
	if _, err := service.Confirm(context.Background(), validPrincipal(), proposal.ID,
		proposal.ConfirmationToken, "confirm"); err == nil {
		t.Fatal("execution failure was hidden")
	}
	if executor.calls != 1 {
		t.Fatalf("executor calls = %d", executor.calls)
	}
}
