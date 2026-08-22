//go:build integration

package projectstore_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/leohteixeira/opensource-project-intelligence/internal/access"
	"github.com/leohteixeira/opensource-project-intelligence/internal/job"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/accessapi"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/accessstore"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/database"
	gh "github.com/leohteixeira/opensource-project-intelligence/internal/platform/github"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/ingestion"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/jobstore"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/oidc"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/projectapi"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/projectstore"
	"github.com/leohteixeira/opensource-project-intelligence/internal/project"
)

const task03WorkspaceID int64 = 1

type task03IDs struct{ value atomic.Int64 }

func (ids *task03IDs) Next(context.Context) (int64, error) { return ids.value.Add(1), nil }

type task03Harness struct {
	ctx       context.Context
	pool      *database.Pool
	store     *projectstore.Store
	jobs      *jobstore.Store
	ingestion *ingestion.Store
	admin     access.Principal
	analyst   access.Principal
	projects  *projectapi.Handler
	handler   http.Handler
	tokens    map[access.Role]string
}

type task03UnavailableWakeups struct{}

func (task03UnavailableWakeups) Subscribe(context.Context, string) (<-chan struct{}, error) {
	return nil, errors.New("controlled Valkey outage")
}

type task03IdentityProvider struct{ identities map[string]oidc.Identity }

func (provider *task03IdentityProvider) AuthorizationURL(string, string, string) string { return "" }
func (provider *task03IdentityProvider) Exchange(context.Context, string, string, string) (oidc.Identity, error) {
	return oidc.Identity{}, errors.New("not used by Task 3 integration tests")
}
func (provider *task03IdentityProvider) VerifyBearer(_ context.Context, token string) (oidc.Identity, error) {
	identity, ok := provider.identities[token]
	if !ok {
		return oidc.Identity{}, errors.New("unknown test bearer")
	}
	return identity, nil
}

type task03URLValidator struct{}

func (task03URLValidator) Validate(_ context.Context, raw string) (*url.URL, error) {
	parsed, err := url.ParseRequestURI(raw)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return nil, errors.New("unsafe test URL")
	}
	return parsed, nil
}

func TestTask03PostgreSQLContracts(t *testing.T) {
	h := newTask03Harness(t)

	t.Run("UT-030 UT-031 UT-033 UT-035 IT-013 IT-015 portfolio is bounded read-only and attention aware", func(t *testing.T) {
		baseline, err := h.store.Portfolio(h.ctx, h.analyst)
		if err != nil {
			t.Fatal(err)
		}
		registered := h.register(t, "portfolio-attention", "portfolio-attention")
		if _, err := h.pool.Unwrap().Exec(h.ctx, `UPDATE sources SET state='unavailable',public=false,
			failure_code='provider_unavailable' WHERE id=$1`, registered.Project.Sources[0].ID); err != nil {
			t.Fatal(err)
		}
		attention, err := h.store.Portfolio(h.ctx, h.analyst)
		if err != nil {
			t.Fatal(err)
		}
		if attention.AttentionCount != baseline.AttentionCount+1 {
			t.Fatalf("attention count = %d, want %d", attention.AttentionCount, baseline.AttentionCount+1)
		}
		if len(attention.Projects) > 12 || len(attention.ActiveJobs) > 12 {
			t.Fatalf("portfolio summaries are unbounded: projects=%d jobs=%d",
				len(attention.Projects), len(attention.ActiveJobs))
		}
		if _, err := h.store.Transition(h.ctx, h.admin, registered.Project.ID,
			registered.Project.Version, project.StateArchived, "portfolio archive", "portfolio-archive"); err != nil {
			t.Fatal(err)
		}
		archived, err := h.store.Portfolio(h.ctx, h.analyst)
		if err != nil {
			t.Fatal(err)
		}
		for _, value := range archived.Projects {
			if value.ID == registered.Project.ID {
				t.Fatal("archived project remained in the active portfolio summary")
			}
		}
		archives, err := h.store.ListProjects(h.ctx, h.analyst,
			projectstore.Filter{State: "archived", Limit: 10})
		if err != nil {
			t.Fatal(err)
		}
		if len(archives) != 1 || archives[0].ID != registered.Project.ID {
			t.Fatalf("archive listing = %#v", archives)
		}
	})

	t.Run("IT-103 command transaction commits aggregate job and outbox together", func(t *testing.T) {
		baseline := map[string]int{}
		for _, table := range []string{"projects", "repositories", "sources", "jobs", "outbox_events", "idempotency_records"} {
			var count int
			if err := h.pool.Unwrap().QueryRow(h.ctx, "SELECT count(*) FROM "+table).Scan(&count); err != nil {
				t.Fatalf("count baseline %s: %v", table, err)
			}
			baseline[table] = count
		}
		_, err := h.pool.Unwrap().Exec(h.ctx, `
			CREATE FUNCTION task03_reject_outbox() RETURNS trigger LANGUAGE plpgsql AS $$
			BEGIN RAISE EXCEPTION 'controlled outbox failure'; END $$;
			CREATE TRIGGER task03_reject_outbox BEFORE INSERT ON outbox_events
			FOR EACH ROW EXECUTE FUNCTION task03_reject_outbox()`)
		if err != nil {
			t.Fatalf("install failure injection: %v", err)
		}
		_, registerErr := h.store.Register(h.ctx, h.analyst,
			"https://github.com/acme/atomic-failure", 180, "atomic-failure", "IT-103")
		if registerErr == nil {
			t.Fatal("registration unexpectedly survived the injected outbox failure")
		}
		if _, err := h.pool.Unwrap().Exec(h.ctx, `DROP TRIGGER task03_reject_outbox ON outbox_events;
			DROP FUNCTION task03_reject_outbox()`); err != nil {
			t.Fatalf("remove failure injection: %v", err)
		}
		for _, table := range []string{"projects", "repositories", "sources", "jobs", "outbox_events", "idempotency_records"} {
			var count int
			if err := h.pool.Unwrap().QueryRow(h.ctx, "SELECT count(*) FROM "+table).Scan(&count); err != nil {
				t.Fatalf("count %s: %v", table, err)
			}
			if count != baseline[table] {
				t.Fatalf("%s count = %d, want unchanged baseline %d", table, count, baseline[table])
			}
		}
	})

	t.Run("UT-040 UT-252 registration replay and mismatch are side-effect free", func(t *testing.T) {
		first := h.register(t, "replay", "registration-replay")
		replayed, err := h.store.Register(h.ctx, h.analyst,
			"https://github.com/acme/registration-replay.git", 180, "replay", "replay-2")
		if err != nil {
			t.Fatalf("replay registration: %v", err)
		}
		if !replayed.Replay || replayed.Project.ID != first.Project.ID || replayed.Job.ID != first.Job.ID ||
			len(replayed.Project.Repositories) != 1 || len(replayed.Project.Sources) != 1 {
			t.Fatalf("replay did not return the complete original aggregate: %#v", replayed)
		}
		_, err = h.store.Register(h.ctx, h.analyst,
			"https://github.com/acme/registration-replay", 365, "replay", "replay-3")
		if !errors.Is(err, project.ErrConflict) {
			t.Fatalf("mismatched idempotency error = %v", err)
		}
		var projects, jobs int
		_ = h.pool.Unwrap().QueryRow(h.ctx,
			`SELECT count(*) FROM projects WHERE slug='registration-replay'`).Scan(&projects)
		_ = h.pool.Unwrap().QueryRow(h.ctx,
			`SELECT count(*) FROM jobs WHERE project_id=$1`, first.Project.ID).Scan(&jobs)
		if projects != 1 || jobs != 1 {
			t.Fatalf("replay counts = projects %d jobs %d, want one each", projects, jobs)
		}
	})

	t.Run("UT-037 UT-039 UT-041 UT-042 UT-046 registration boundaries preserve the aggregate", func(t *testing.T) {
		viewer := access.Principal{ActorID: 9003, Kind: access.ActorMember, Role: access.RoleViewer,
			Status: access.StatusActive, Workspace: task03WorkspaceID}
		pending := access.Principal{ActorID: 9004, Kind: access.ActorMember, Role: access.RoleAnalyst,
			Status: access.StatusPending, Workspace: task03WorkspaceID}
		if _, err := h.store.Register(h.ctx, h.analyst, " ", 180, "blank-url", "UT-037"); !errors.Is(err, project.ErrInvalid) {
			t.Fatalf("blank registration URL error = %v", err)
		}
		for name, testCase := range map[string]struct {
			principal access.Principal
			want      error
		}{
			"viewer":  {principal: viewer, want: access.ErrPermissionDenied},
			"pending": {principal: pending, want: access.ErrAccessPending},
		} {
			if _, err := h.store.Register(h.ctx, testCase.principal, "https://github.com/acme/denied-"+name,
				180, "denied-"+name, "UT-039"); !errors.Is(err, testCase.want) {
				t.Fatalf("%s registration error = %v", name, err)
			}
		}
		if _, err := h.store.AddRepository(h.ctx, h.analyst, 8_888_888_888_888_888_888,
			"https://github.com/acme/orphan", project.RoleSDK, "UT-041"); !errors.Is(err, project.ErrNotFound) {
			t.Fatalf("orphan repository error = %v", err)
		}

		registered := h.register(t, "archive-register", "archive-register")
		archived, err := h.store.Transition(h.ctx, h.admin, registered.Project.ID,
			registered.Project.Version, project.StateArchived, "retention", "UT-042-archive")
		if err != nil {
			t.Fatal(err)
		}
		replayed, err := h.store.Register(h.ctx, h.analyst,
			"https://github.com/acme/archive-register.git", 180, "archive-register-retry", "UT-042")
		if err != nil || !replayed.Replay || replayed.Project.ID != registered.Project.ID ||
			replayed.Project.State != project.StateArchived {
			t.Fatalf("archived registration resolution = %#v, %v", replayed, err)
		}
		if archived.Project.State != replayed.Project.State {
			t.Fatalf("archived state changed during duplicate registration: %s -> %s",
				archived.Project.State, replayed.Project.State)
		}
		if _, err := h.store.AddRepository(h.ctx, viewer, registered.Project.ID,
			"https://github.com/acme/viewer-denied", project.RoleSDK, "UT-046"); !errors.Is(err, access.ErrPermissionDenied) {
			t.Fatalf("viewer repository mutation error = %v", err)
		}
	})

	t.Run("IT-017 failed initial collection remains one recoverable Project", func(t *testing.T) {
		registered := h.register(t, "recoverable", "recoverable")
		if _, err := h.pool.Unwrap().Exec(h.ctx, `UPDATE sources SET state='unavailable',
			failure_code='provider_unavailable' WHERE id=$1`, registered.Project.Sources[0].ID); err != nil {
			t.Fatal(err)
		}
		if _, err := h.pool.Unwrap().Exec(h.ctx, `UPDATE jobs SET state='failed',
			failure_code='provider_unavailable',finished_at=now() WHERE id=$1`, registered.Job.ID); err != nil {
			t.Fatal(err)
		}
		replayed, err := h.store.Register(h.ctx, h.analyst,
			"https://github.com/acme/recoverable.git", 180, "recoverable retry", "IT-017")
		if err != nil || !replayed.Replay || replayed.Project.ID != registered.Project.ID ||
			replayed.Project.Sources[0].State != project.SourceUnavailable {
			t.Fatalf("recoverable registration = %#v, %v", replayed, err)
		}
	})

	t.Run("IT-018 registration remains independent of another active backfill", func(t *testing.T) {
		backfill := h.register(t, "active-backfill", "active-backfill")
		lease, err := h.jobs.ClaimJob(h.ctx, backfill.Job.ID, "backfill-worker", time.Minute)
		if err != nil || lease == nil {
			t.Fatalf("claim active backfill = %#v, %v", lease, err)
		}
		ctx, cancel := context.WithTimeout(h.ctx, 2*time.Second)
		defer cancel()
		created, err := h.store.Register(ctx, h.analyst,
			"https://github.com/acme/responsive-registration", 180, "responsive", "IT-018")
		if err != nil || created.Project.ID == backfill.Project.ID {
			t.Fatalf("independent registration = %#v, %v", created, err)
		}
		var state string
		err = h.pool.Unwrap().QueryRow(h.ctx, `SELECT state FROM jobs WHERE id=$1`, backfill.Job.ID).Scan(&state)
		if err != nil || state != string(job.Running) {
			t.Fatalf("unrelated backfill state = %q, %v", state, err)
		}
	})

	t.Run("IT-020 IT-021 interrupted edits preserve a paged repository set", func(t *testing.T) {
		registered := h.register(t, "repository-pages", "repository-pages")
		for index := 1; index < 13; index++ {
			if _, err := h.store.AddRepository(h.ctx, h.analyst, registered.Project.ID,
				fmt.Sprintf("https://github.com/acme/repository-page-%02d", index), project.RoleSDK,
				fmt.Sprintf("IT-021-%d", index)); err != nil {
				t.Fatal(err)
			}
		}
		before, err := h.store.ListRepositories(h.ctx, h.analyst, registered.Project.ID,
			projectstore.Filter{Limit: 20})
		if err != nil || len(before) != 13 {
			t.Fatalf("repository summary = %d, %v", len(before), err)
		}
		if _, err := h.store.ChangeRepositoryRole(h.ctx, h.analyst, registered.Project.ID,
			before[1].ID, before[1].Version+10, project.RoleCore, "IT-020"); !errors.Is(err, project.ErrVersionConflict) {
			t.Fatalf("interrupted edit error = %v", err)
		}
		after, err := h.store.ListRepositories(h.ctx, h.analyst, registered.Project.ID,
			projectstore.Filter{Limit: 20})
		if err != nil || len(after) != len(before) {
			t.Fatalf("repository set after interruption = %d, %v", len(after), err)
		}
		for index := range before {
			if before[index] != after[index] {
				t.Fatalf("repository %d changed: %#v -> %#v", index, before[index], after[index])
			}
		}
		first, err := h.store.ListRepositories(h.ctx, h.analyst, registered.Project.ID,
			projectstore.Filter{Limit: 5})
		second, secondErr := h.store.ListRepositories(h.ctx, h.analyst, registered.Project.ID,
			projectstore.Filter{Limit: 5, Offset: 5})
		if err != nil || secondErr != nil || len(first) != 6 || len(second) != 6 || first[5].ID != second[0].ID {
			t.Fatalf("repository pages = %#v / %#v, errors %v / %v", first, second, err, secondErr)
		}
	})

	t.Run("IT-016 concurrent canonical registration creates exactly one project", func(t *testing.T) {
		const callers = 12
		var wait sync.WaitGroup
		var successes atomic.Int64
		projectIDs := make(chan int64, callers)
		for index := range callers {
			wait.Add(1)
			go func(index int) {
				defer wait.Done()
				registered, err := h.store.Register(h.ctx, h.analyst,
					"https://github.com/acme/concurrent-registration", 180,
					fmt.Sprintf("concurrent-%d", index), fmt.Sprintf("IT-016-%d", index))
				if err == nil {
					successes.Add(1)
					projectIDs <- registered.Project.ID
				}
			}(index)
		}
		wait.Wait()
		close(projectIDs)
		var count int
		if err := h.pool.Unwrap().QueryRow(h.ctx,
			`SELECT count(*) FROM projects WHERE slug='concurrent-registration'`).Scan(&count); err != nil {
			t.Fatal(err)
		}
		resolved := map[int64]struct{}{}
		for id := range projectIDs {
			resolved[id] = struct{}{}
		}
		if count != 1 || successes.Load() != callers || len(resolved) != 1 {
			t.Fatalf("projects = %d, successful commands = %d, resolved IDs = %v, want one Project resolved by all callers",
				count, successes.Load(), resolved)
		}
	})

	t.Run("IT-019 IT-142 concurrent primary changes retain one primary", func(t *testing.T) {
		registered := h.register(t, "primary-race", "primary-race")
		one, err := h.store.AddRepository(h.ctx, h.analyst, registered.Project.ID,
			"https://github.com/acme/primary-one", project.RoleCore, "primary-one")
		if err != nil {
			t.Fatal(err)
		}
		two, err := h.store.AddRepository(h.ctx, h.analyst, registered.Project.ID,
			"https://github.com/acme/primary-two", project.RoleSDK, "primary-two")
		if err != nil {
			t.Fatal(err)
		}
		var wait sync.WaitGroup
		for _, repository := range []project.Repository{one, two} {
			wait.Add(1)
			go func(repository project.Repository) {
				defer wait.Done()
				_, _ = h.store.ChangeRepositoryRole(h.ctx, h.analyst, registered.Project.ID,
					repository.ID, repository.Version, project.RolePrimary, "primary-race")
			}(repository)
		}
		wait.Wait()
		var primaries int
		if err := h.pool.Unwrap().QueryRow(h.ctx,
			`SELECT count(*) FROM repositories WHERE project_id=$1 AND role='primary'`,
			registered.Project.ID).Scan(&primaries); err != nil {
			t.Fatal(err)
		}
		if primaries != 1 {
			t.Fatalf("primary repositories = %d, want exactly one", primaries)
		}
	})

	t.Run("IT-028 simultaneous refreshes coalesce atomically", func(t *testing.T) {
		registered := h.register(t, "sync-coalescing", "sync-coalescing")
		const callers = 16
		ids := make(chan int64, callers)
		errs := make(chan error, callers)
		var wait sync.WaitGroup
		for index := range callers {
			wait.Add(1)
			go func(index int) {
				defer wait.Done()
				queued, err := h.store.QueueSync(h.ctx, h.analyst, registered.Project.ID,
					"issues", fmt.Sprintf("IT-028-%d", index))
				if err != nil {
					errs <- err
					return
				}
				ids <- queued.ID
			}(index)
		}
		wait.Wait()
		close(ids)
		close(errs)
		for err := range errs {
			t.Errorf("QueueSync() error = %v", err)
		}
		unique := map[int64]struct{}{}
		for id := range ids {
			unique[id] = struct{}{}
		}
		var count int
		var coalesced int64
		if err := h.pool.Unwrap().QueryRow(h.ctx, `SELECT count(*),COALESCE(max(coalesced_requests),0)
			FROM jobs WHERE project_id=$1 AND coalescing_key=$2`, registered.Project.ID,
			"sync:"+fmt.Sprint(registered.Project.ID)+":issues").Scan(&count, &coalesced); err != nil {
			t.Fatal(err)
		}
		if len(unique) != 1 || count != 1 || coalesced != callers-1 {
			t.Fatalf("unique IDs=%v count=%d coalesced=%d", unique, count, coalesced)
		}
	})

	t.Run("IT-031 overlapping history requests retain the broadest target", func(t *testing.T) {
		registered := h.register(t, "history-coalescing", "history-coalescing")
		to := time.Now().UTC().Truncate(24 * time.Hour)
		first, err := h.store.QueueHistory(h.ctx, h.analyst, registered.Project.ID,
			to.AddDate(0, -1, 0), to, "history-short")
		if err != nil {
			t.Fatal(err)
		}
		broad, err := h.store.QueueHistory(h.ctx, h.analyst, registered.Project.ID,
			to.AddDate(-1, 0, 0), to, "history-broad")
		if err != nil {
			t.Fatal(err)
		}
		if broad.ID != first.ID || broad.RequestedFrom == nil || broad.RequestedTo == nil ||
			!broad.RequestedFrom.Equal(to.AddDate(-1, 0, 0)) || !broad.RequestedTo.Equal(to) {
			t.Fatalf("coalesced history target = %#v", broad)
		}
	})

	t.Run("UT-064 UT-067 UT-074 UT-075 UT-076 UT-077 IT-033 coverage and scheduling remain authoritative", func(t *testing.T) {
		registered := h.register(t, "coverage", "coverage")
		viewer := access.Principal{ActorID: 9003, Kind: access.ActorMember, Role: access.RoleViewer,
			Status: access.StatusActive, Workspace: task03WorkspaceID}
		to := time.Now().UTC().Truncate(time.Second)
		from := to.AddDate(-1, 0, 0)
		if _, err := h.store.QueueSync(h.ctx, h.analyst, registered.Project.ID, "unsupported", "UT-064"); !errors.Is(err, project.ErrInvalid) {
			t.Fatalf("unsupported synchronization scope error = %v", err)
		}
		if _, err := h.store.QueueSync(h.ctx, viewer, registered.Project.ID, "all", "UT-067"); !errors.Is(err, access.ErrPermissionDenied) {
			t.Fatalf("viewer synchronization error = %v", err)
		}
		if _, err := h.store.QueueHistory(h.ctx, viewer, registered.Project.ID, from, to, "UT-074"); !errors.Is(err, access.ErrPermissionDenied) {
			t.Fatalf("viewer history error = %v", err)
		}
		if _, err := h.pool.Unwrap().Exec(h.ctx, `UPDATE sources SET coverage_from=$1,coverage_to=$2
			WHERE id=$3`, from.AddDate(-1, 0, 0), to, registered.Project.Sources[0].ID); err != nil {
			t.Fatal(err)
		}
		var jobsBefore int
		if err := h.pool.Unwrap().QueryRow(h.ctx, `SELECT count(*) FROM jobs WHERE project_id=$1`,
			registered.Project.ID).Scan(&jobsBefore); err != nil {
			t.Fatal(err)
		}
		if _, err := h.store.QueueHistory(h.ctx, h.analyst, registered.Project.ID, from, to, "UT-075"); !errors.Is(err, project.ErrConflict) {
			t.Fatalf("covered history request error = %v", err)
		}
		var jobsAfter int
		var coverageFrom, coverageTo time.Time
		if err := h.pool.Unwrap().QueryRow(h.ctx, `SELECT
			(SELECT count(*) FROM jobs WHERE project_id=$1),coverage_from,coverage_to
			FROM sources WHERE id=$2`, registered.Project.ID, registered.Project.Sources[0].ID).
			Scan(&jobsAfter, &coverageFrom, &coverageTo); err != nil {
			t.Fatal(err)
		}
		if jobsAfter != jobsBefore || !coverageFrom.Equal(from.AddDate(-1, 0, 0)) || !coverageTo.Equal(to) {
			t.Fatalf("covered request changed durable state: jobs %d->%d coverage %s..%s",
				jobsBefore, jobsAfter, coverageFrom, coverageTo)
		}
		archived, err := h.store.Transition(h.ctx, h.admin, registered.Project.ID,
			registered.Project.Version, project.StateArchived, "retention", "UT-077")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := h.store.QueueHistory(h.ctx, h.analyst, registered.Project.ID,
			from.AddDate(-1, 0, 0), to, "UT-077"); !errors.Is(err, project.ErrConflict) {
			t.Fatalf("archived history request error = %v", err)
		}
		if archived.Project.Sources[0].CoverageFrom == nil || archived.Project.Sources[0].CoverageTo == nil ||
			!archived.Project.Sources[0].CoverageFrom.Equal(coverageFrom) ||
			!archived.Project.Sources[0].CoverageTo.Equal(coverageTo) {
			t.Fatalf("archive changed coverage: %#v", archived.Project.Sources[0])
		}
		// The range request is a durable intent only; it cannot mutate snapshots before collectors
		// commit new facts and coverage in their own transaction (UT-076).
		var snapshots int
		if err := h.pool.Unwrap().QueryRow(h.ctx,
			`SELECT count(*) FROM raw_objects WHERE project_id=$1`, registered.Project.ID).Scan(&snapshots); err != nil {
			t.Fatal(err)
		}
		if snapshots != 0 {
			t.Fatalf("history scheduling rewrote %d evidence objects", snapshots)
		}
	})

	t.Run("UT-050 UT-051 UT-052 UT-053 UT-054 UT-056 IT-022 IT-023 IT-024 association correction is bounded", func(t *testing.T) {
		registered := h.register(t, "correction", "correction")
		associationID, _ := h.storeID()
		_, err := h.pool.Unwrap().Exec(h.ctx, `INSERT INTO source_associations
			(id,source_id,project_id,method,confidence,evidence_ids,decision_version,status)
			VALUES($1,$2,$3,'canonical_url',0.75,$4,'identity-v1','linked')`,
			associationID, registered.Project.Sources[0].ID, registered.Project.ID,
			[]int64{registered.Project.Sources[0].ID})
		if err != nil {
			t.Fatal(err)
		}
		viewer := access.Principal{ActorID: 9003, Kind: access.ActorMember, Role: access.RoleViewer,
			Status: access.StatusActive, Workspace: task03WorkspaceID}
		visible, err := h.store.ListAssociations(h.ctx, viewer, registered.Project.ID,
			projectstore.Filter{State: "linked", Limit: 1})
		if err != nil || len(visible) != 1 || len(visible[0].Evidence) == 0 || visible[0].DecisionVersion == "" {
			t.Fatalf("viewer provenance queue = %#v, %v", visible, err)
		}
		if _, _, err := h.store.CorrectAssociation(h.ctx, viewer, registered.Project.ID,
			associationID, 0, "split", "viewer denied", "UT-053"); !errors.Is(err, access.ErrPermissionDenied) {
			t.Fatalf("viewer correction error = %v", err)
		}
		if _, _, err := h.store.CorrectAssociation(h.ctx, h.analyst, registered.Project.ID,
			associationID, 8_888_888_888_888_888_888, "reassign", "invalid target", "UT-050"); !errors.Is(err, project.ErrNotFound) {
			t.Fatalf("invalid reassignment error = %v", err)
		}
		var unchangedProject int64
		if err := h.pool.Unwrap().QueryRow(h.ctx,
			`SELECT project_id FROM source_associations WHERE id=$1`, associationID).Scan(&unchangedProject); err != nil {
			t.Fatal(err)
		}
		if unchangedProject != registered.Project.ID {
			t.Fatalf("failed correction changed project to %d", unchangedProject)
		}
		unresolvedSource, err := h.store.AddSource(h.ctx, h.analyst, registered.Project.ID,
			project.SourceWebsite, "https://unresolved.example", "UT-052-unresolved-source")
		if err != nil {
			t.Fatal(err)
		}
		unresolvedID, _ := h.storeID()
		if _, err := h.pool.Unwrap().Exec(h.ctx, `INSERT INTO source_associations
			(id,source_id,project_id,method,confidence,evidence_ids,decision_version,status)
			VALUES($1,$2,$3,'insufficient_evidence',0.25,$4,'identity-v1','unresolved')`,
			unresolvedID, unresolvedSource.ID, registered.Project.ID,
			[]int64{unresolvedSource.ID}); err != nil {
			t.Fatal(err)
		}
		queue, err := h.store.ListAssociations(h.ctx, h.analyst, registered.Project.ID,
			projectstore.Filter{State: "unresolved", Limit: 1})
		if err != nil || len(queue) != 1 || queue[0].ID != unresolvedID || queue[0].Status != "unresolved" {
			t.Fatalf("bounded unresolved queue = %#v, %v", queue, err)
		}
		deleting := h.register(t, "correction-target", "correction-target")
		if _, err := h.store.RequestDeletion(h.ctx, h.admin, deleting.Project.ID,
			deleting.Project.Version, "DELETE "+deleting.Project.Slug, "target removed", "UT-056"); err != nil {
			t.Fatal(err)
		}
		if _, _, err := h.store.CorrectAssociation(h.ctx, h.analyst, registered.Project.ID,
			associationID, deleting.Project.ID, "reassign", "deleted target", "UT-056"); !errors.Is(err, project.ErrConflict) {
			t.Fatalf("deleting reassignment target error = %v", err)
		}
		jobID, changed, err := h.store.CorrectAssociation(h.ctx, h.analyst, registered.Project.ID,
			associationID, 0, "split", "different product", "correction-first")
		if err != nil || !changed || jobID == 0 {
			t.Fatalf("first correction = job %d changed %t error %v", jobID, changed, err)
		}
		replayedJob, changed, err := h.store.CorrectAssociation(h.ctx, h.analyst,
			registered.Project.ID, associationID, 0, "split", "different product", "correction-replay")
		if err != nil || changed || replayedJob != 0 {
			t.Fatalf("replayed correction = job %d changed %t error %v", replayedJob, changed, err)
		}
		var corrections, recalculations int
		_ = h.pool.Unwrap().QueryRow(h.ctx,
			`SELECT count(*) FROM identity_corrections WHERE association_id=$1`, associationID).Scan(&corrections)
		_ = h.pool.Unwrap().QueryRow(h.ctx,
			`SELECT count(*) FROM jobs WHERE project_id=$1 AND kind='association_recalculation'`,
			registered.Project.ID).Scan(&recalculations)
		if corrections != 1 || recalculations != 1 {
			t.Fatalf("corrections=%d recalculations=%d", corrections, recalculations)
		}
	})

	t.Run("IT-109 IT-110 IT-219 cancellation is durable repeatable and resumable", func(t *testing.T) {
		registered := h.register(t, "cancellation", "cancellation")
		queued, err := h.store.QueueSync(h.ctx, h.analyst, registered.Project.ID, "releases", "cancel")
		if err != nil {
			t.Fatal(err)
		}
		cancelled, err := h.store.CancelJob(h.ctx, h.analyst, queued.ID, "cancel-first")
		if err != nil || cancelled.State != job.Cancelled {
			t.Fatalf("cancel = %#v, %v", cancelled, err)
		}
		repeated, err := h.store.CancelJob(h.ctx, h.analyst, queued.ID, "cancel-repeat")
		if err != nil || repeated.Version != cancelled.Version {
			t.Fatalf("repeated cancel = %#v, %v", repeated, err)
		}
		events, err := h.store.JobEvents(h.ctx, h.analyst, queued.ID, 0, 10)
		if err != nil || len(events) != 2 || events[len(events)-1].Job.State != job.Cancelled {
			t.Fatalf("job events = %#v, %v", events, err)
		}
		resumed, err := h.store.JobEvents(h.ctx, h.analyst, queued.ID, events[0].ID, 10)
		if err != nil || len(resumed) != 1 || resumed[0].ID != events[1].ID {
			t.Fatalf("resumed events = %#v, %v", resumed, err)
		}
	})

	t.Run("UT-068 UT-256 IT-114 IT-119 core source pages and checkpoints commit atomically and replay safely", func(t *testing.T) {
		registered := h.register(t, "ingestion", "ingestion")
		now := time.Now().UTC().Truncate(time.Second)
		issue := gh.Issue{ExternalID: 42, Number: 7, Title: "Durable ingestion", State: "open",
			CreatedAt: now.Add(-time.Hour), UpdatedAt: now, Raw: []byte(`{"id":42,"number":7}`)}
		commit := func(cursor string, issues []gh.Issue) error {
			return h.ingestion.CommitGitHubIssues(h.ctx, registered.Project.ID,
				registered.Project.Repositories[0].ID, registered.Project.Sources[0].ID,
				issues, cursor, now.AddDate(0, -6, 0), now)
		}
		if err := commit("page-2", []gh.Issue{issue}); err != nil {
			t.Fatal(err)
		}
		if err := commit("page-2", []gh.Issue{issue}); err != nil {
			t.Fatalf("replay page: %v", err)
		}
		pullRequest := gh.PullRequest{ExternalID: 52, Number: 8, Title: "Durable pull request",
			State: "open", CreatedAt: now.Add(-2 * time.Hour), Raw: []byte(`{"id":52,"number":8}`)}
		if err := h.ingestion.CommitGitHubPullRequests(h.ctx, registered.Project.ID,
			registered.Project.Repositories[0].ID, registered.Project.Sources[0].ID,
			[]gh.PullRequest{pullRequest}, "complete", now.AddDate(0, -6, 0), now); err != nil {
			t.Fatalf("commit pull requests: %v", err)
		}
		publishedAt := now.Add(-30 * time.Minute)
		release := gh.Release{ExternalID: 62, Tag: "v1.0.0", PublishedAt: &publishedAt,
			Raw: []byte(`{"id":62,"tag_name":"v1.0.0"}`)}
		if err := h.ingestion.CommitGitHubReleases(h.ctx, registered.Project.ID,
			registered.Project.Repositories[0].ID, registered.Project.Sources[0].ID,
			[]gh.Release{release}, "complete", now.AddDate(0, -6, 0), now); err != nil {
			t.Fatalf("commit releases: %v", err)
		}
		commitValue := gh.Commit{ExternalID: "abc123", SHA: "abc123", AuthorExternalID: "73",
			CommittedAt: now.Add(-15 * time.Minute), DefaultBranch: true,
			Raw: []byte(`{"sha":"abc123"}`)}
		if err := h.ingestion.CommitGitHubCommits(h.ctx, registered.Project.ID,
			registered.Project.Repositories[0].ID, registered.Project.Sources[0].ID,
			[]gh.Commit{commitValue}, "complete", now.AddDate(0, -6, 0), now); err != nil {
			t.Fatalf("commit commits: %v", err)
		}
		bad := issue
		bad.ExternalID = 43
		bad.Raw = []byte(`{"unterminated"`)
		if err := commit("page-3", []gh.Issue{bad}); err == nil {
			t.Fatal("malformed page unexpectedly committed")
		}
		var rawObjects, canonicalIssues, canonicalPullRequests, canonicalReleases, canonicalCommits int
		var checkpointVersion, checkpointScopes int
		var cursor string
		err := h.pool.Unwrap().QueryRow(h.ctx, `SELECT
			(SELECT count(*) FROM raw_objects WHERE source_id=$1),
			(SELECT count(*) FROM canonical_issues WHERE source_id=$1),
			(SELECT count(*) FROM canonical_pull_requests WHERE source_id=$1),
			(SELECT count(*) FROM canonical_releases WHERE source_id=$1),
			(SELECT count(*) FROM canonical_commits WHERE source_id=$1),
			(SELECT count(*) FROM sync_checkpoints WHERE source_id=$1),cursor,version
			FROM sync_checkpoints WHERE source_id=$1 AND scope='github_issues'`,
			registered.Project.Sources[0].ID).Scan(&rawObjects, &canonicalIssues, &canonicalPullRequests,
			&canonicalReleases, &canonicalCommits, &checkpointScopes, &cursor, &checkpointVersion)
		if err != nil {
			t.Fatal(err)
		}
		if rawObjects != 4 || canonicalIssues != 1 || canonicalPullRequests != 1 ||
			canonicalReleases != 1 || canonicalCommits != 1 || checkpointScopes != 4 ||
			cursor != "page-2" || checkpointVersion != 2 {
			t.Fatalf("raw=%d issues=%d pull requests=%d releases=%d commits=%d scopes=%d cursor=%s version=%d",
				rawObjects, canonicalIssues, canonicalPullRequests, canonicalReleases,
				canonicalCommits, checkpointScopes, cursor, checkpointVersion)
		}
	})

	t.Run("IT-029 IT-108 expired lease resumes from the durable checkpoint", func(t *testing.T) {
		_, _ = h.pool.Unwrap().Exec(h.ctx, `UPDATE jobs SET state='succeeded',finished_at=now()
			WHERE state='queued'`)
		registered := h.register(t, "lease-recovery", "lease-recovery")
		lease, err := h.jobs.Claim(h.ctx, "worker-a", time.Minute)
		if err != nil || lease == nil || lease.Job.ID != registered.Job.ID {
			t.Fatalf("claim = %#v, %v", lease, err)
		}
		checkpoint := job.Checkpoint{Scope: "github_issues", Cursor: "page-4",
			CoverageTo: time.Now().UTC(), Version: 1}
		leaseValue, err := h.jobs.Checkpoint(h.ctx, *lease, 400, checkpoint)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = h.pool.Unwrap().Exec(h.ctx, `UPDATE jobs SET lease_expires_at=now()-interval '1 second'
			WHERE id=$1`, leaseValue.Job.ID)
		if recovered, err := h.jobs.RecoverExpired(h.ctx); err != nil || recovered != 1 {
			t.Fatalf("RecoverExpired() = %d, %v", recovered, err)
		}
		resumed, err := h.jobs.Claim(h.ctx, "worker-b", time.Minute)
		if err != nil || resumed == nil || resumed.Job.Checkpoint == nil ||
			resumed.Job.Checkpoint.Cursor != "page-4" || resumed.Job.Progress.Completed != 400 {
			t.Fatalf("resumed lease = %#v, %v", resumed, err)
		}
	})

	t.Run("IT-106 redelivery after committed completion is side-effect free", func(t *testing.T) {
		registered := h.register(t, "ack-loss", "ack-loss")
		lease, err := h.jobs.ClaimJob(h.ctx, registered.Job.ID, "worker-before-ack-loss", time.Minute)
		if err != nil || lease == nil {
			t.Fatalf("claim Job before simulated acknowledgement loss = %#v, %v", lease, err)
		}
		if err := h.jobs.Complete(h.ctx, *lease); err != nil {
			t.Fatalf("commit Job result before simulated acknowledgement loss: %v", err)
		}
		var attemptsBefore, eventsBefore, outboxBefore int
		if err := h.pool.Unwrap().QueryRow(h.ctx, `SELECT
			(SELECT count(*) FROM job_attempts WHERE job_id=$1),
			(SELECT count(*) FROM job_events WHERE job_id=$1),
			(SELECT count(*) FROM outbox_events WHERE job_id=$1)`, registered.Job.ID).
			Scan(&attemptsBefore, &eventsBefore, &outboxBefore); err != nil {
			t.Fatal(err)
		}
		redelivered, err := h.jobs.ClaimJob(h.ctx, registered.Job.ID, "worker-after-ack-loss", time.Minute)
		if err != nil || redelivered != nil {
			t.Fatalf("redelivered terminal Job = %#v, %v; want PostgreSQL no-op", redelivered, err)
		}
		var attemptsAfter, eventsAfter, outboxAfter int
		if err := h.pool.Unwrap().QueryRow(h.ctx, `SELECT
			(SELECT count(*) FROM job_attempts WHERE job_id=$1),
			(SELECT count(*) FROM job_events WHERE job_id=$1),
			(SELECT count(*) FROM outbox_events WHERE job_id=$1)`, registered.Job.ID).
			Scan(&attemptsAfter, &eventsAfter, &outboxAfter); err != nil {
			t.Fatal(err)
		}
		if attemptsAfter != attemptsBefore || eventsAfter != eventsBefore || outboxAfter != outboxBefore {
			t.Fatalf("redelivery changed durable state: attempts %d->%d events %d->%d outbox %d->%d",
				attemptsBefore, attemptsAfter, eventsBefore, eventsAfter, outboxBefore, outboxAfter)
		}
	})

	t.Run("IT-107 exhausted crashed Job records durable failure and advisory", func(t *testing.T) {
		registered := h.register(t, "poison", "poison")
		if _, err := h.pool.Unwrap().Exec(h.ctx, `UPDATE jobs SET max_attempts=1 WHERE id=$1`,
			registered.Job.ID); err != nil {
			t.Fatal(err)
		}
		lease, err := h.jobs.ClaimJob(h.ctx, registered.Job.ID, "poison-worker", time.Minute)
		if err != nil || lease == nil {
			t.Fatalf("claim poison Job = %#v, %v", lease, err)
		}
		if _, err := h.pool.Unwrap().Exec(h.ctx, `UPDATE jobs SET lease_expires_at=now()-interval '1 second'
			WHERE id=$1`, registered.Job.ID); err != nil {
			t.Fatal(err)
		}
		if recovered, err := h.jobs.RecoverExpired(h.ctx); err != nil || recovered != 1 {
			t.Fatalf("RecoverExpired() = %d, %v", recovered, err)
		}
		var state, failure, attemptState, attemptFailure string
		var deadLetters, terminalEvents int
		if err := h.pool.Unwrap().QueryRow(h.ctx, `SELECT j.state,j.failure_code,a.state,a.failure_code,
			(SELECT count(*) FROM outbox_events WHERE job_id=j.id AND event_type='job.dead_lettered'),
			(SELECT count(*) FROM job_events WHERE job_id=j.id AND representation->>'state'='failed')
			FROM jobs j JOIN job_attempts a ON a.job_id=j.id WHERE j.id=$1`, registered.Job.ID).
			Scan(&state, &failure, &attemptState, &attemptFailure, &deadLetters, &terminalEvents); err != nil {
			t.Fatal(err)
		}
		if state != "failed" || failure != "attempts_exhausted" || attemptState != "failed" ||
			attemptFailure != "lease_expired" || deadLetters != 1 || terminalEvents != 1 {
			t.Fatalf("poison result: Job=%s/%s attempt=%s/%s advisories=%d terminal events=%d",
				state, failure, attemptState, attemptFailure, deadLetters, terminalEvents)
		}
	})

	t.Run("UT-058 UT-059 IT-025 IT-026 IT-027 IT-133 IT-144 deletion is scoped unavailable first and leaves a tombstone", func(t *testing.T) {
		_, _ = h.pool.Unwrap().Exec(h.ctx, `UPDATE jobs SET state='succeeded',finished_at=now()
			WHERE state='queued'`)
		registered := h.register(t, "project-purge", "project-purge")
		unrelated := h.register(t, "unrelated-readable", "unrelated-readable")
		_, _ = h.pool.Unwrap().Exec(h.ctx, `UPDATE jobs SET state='succeeded',finished_at=now()
			WHERE id IN ($1,$2)`, registered.Job.ID, unrelated.Job.ID)
		if _, err := h.store.RequestDeletion(h.ctx, h.admin, registered.Project.ID,
			registered.Project.Version, "", "missing identity", "UT-058"); !errors.Is(err, project.ErrInvalid) {
			t.Fatalf("missing deletion identity error = %v", err)
		}
		if _, err := h.store.GetProject(h.ctx, h.analyst, registered.Project.ID); err != nil {
			t.Fatalf("invalid confirmation changed target project: %v", err)
		}
		deletion, err := h.store.RequestDeletion(h.ctx, h.admin, registered.Project.ID,
			registered.Project.Version, "DELETE "+registered.Project.Slug, "duplicate", "delete")
		if err != nil || deletion.Job.Cancellable {
			t.Fatalf("deletion = %#v, %v", deletion, err)
		}
		if _, err := h.store.GetProject(h.ctx, h.analyst, registered.Project.ID); !errors.Is(err, project.ErrNotFound) {
			t.Fatalf("deleting project remained available: %v", err)
		}
		if _, err := h.store.CancelJob(h.ctx, h.admin, deletion.Job.ID, "cancel-purge"); !errors.Is(err, job.ErrConflict) {
			t.Fatalf("purge cancellation error = %v", err)
		}
		lease, err := h.jobs.Claim(h.ctx, "purger", time.Minute)
		if err != nil || lease == nil || lease.Job.ID != deletion.Job.ID {
			t.Fatalf("purge claim = %#v, %v", lease, err)
		}
		done, leaseValue, err := h.jobs.PurgeProject(h.ctx, *lease, 10)
		if err != nil || !done {
			t.Fatalf("purge = done %t lease %#v error %v", done, leaseValue, err)
		}
		if err := h.jobs.Complete(h.ctx, leaseValue); err != nil {
			t.Fatalf("complete purge: %v", err)
		}
		var projects, tombstones int
		_ = h.pool.Unwrap().QueryRow(h.ctx, `SELECT count(*) FROM projects WHERE id=$1`,
			registered.Project.ID).Scan(&projects)
		_ = h.pool.Unwrap().QueryRow(h.ctx, `SELECT count(*) FROM project_tombstones WHERE project_id=$1`,
			registered.Project.ID).Scan(&tombstones)
		if projects != 0 || tombstones != 1 {
			t.Fatalf("projects=%d tombstones=%d", projects, tombstones)
		}
		if value, err := h.store.GetProject(h.ctx, h.analyst, unrelated.Project.ID); err != nil ||
			value.ID != unrelated.Project.ID {
			t.Fatalf("single-project purge affected unrelated project: %#v, %v", value, err)
		}
	})

	t.Run("UT-038 IT-036 IT-141 aggregate quota rejection has no partial state", func(t *testing.T) {
		var current int
		if err := h.pool.Unwrap().QueryRow(h.ctx,
			`SELECT count(*) FROM projects WHERE workspace_id=$1 AND state<>'deleted'`, task03WorkspaceID).
			Scan(&current); err != nil {
			t.Fatal(err)
		}
		missing := 1000 - current
		if missing > 0 {
			_, err := h.pool.Unwrap().Exec(h.ctx, `INSERT INTO projects(id,workspace_id,name,slug,state)
				SELECT 7000000000000000000+n,$1,'quota-'||n,'quota-'||n,'active'
				FROM generate_series(1,$2) AS n`, task03WorkspaceID, missing)
			if err != nil {
				t.Fatal(err)
			}
		}
		var repositoriesBefore, jobsBefore, outboxBefore int
		if err := h.pool.Unwrap().QueryRow(h.ctx, `SELECT
			(SELECT count(*) FROM repositories),
			(SELECT count(*) FROM jobs),
			(SELECT count(*) FROM outbox_events)`).
			Scan(&repositoriesBefore, &jobsBefore, &outboxBefore); err != nil {
			t.Fatal(err)
		}
		_, err := h.store.Register(h.ctx, h.analyst,
			"https://github.com/acme/quota-overflow", 180, "quota-overflow", "IT-141")
		if !errors.Is(err, project.ErrConflict) {
			t.Fatalf("quota registration error = %v", err)
		}
		var projectsAfter, repositoriesAfter, jobsAfter, outboxAfter, idempotencyAfter int
		if err := h.pool.Unwrap().QueryRow(h.ctx, `SELECT
			(SELECT count(*) FROM projects WHERE workspace_id=$1 AND state<>'deleted'),
			(SELECT count(*) FROM repositories),
			(SELECT count(*) FROM jobs),
			(SELECT count(*) FROM outbox_events),
			(SELECT count(*) FROM idempotency_records WHERE idempotency_key='quota-overflow')`,
			task03WorkspaceID).Scan(&projectsAfter, &repositoriesAfter, &jobsAfter, &outboxAfter,
			&idempotencyAfter); err != nil {
			t.Fatal(err)
		}
		if projectsAfter != 1000 || repositoriesAfter != repositoriesBefore || jobsAfter != jobsBefore ||
			outboxAfter != outboxBefore || idempotencyAfter != 0 {
			t.Fatalf("quota rejection leaked state: projects=%d repositories=%d->%d jobs=%d->%d outbox=%d->%d idempotency=%d",
				projectsAfter, repositoriesBefore, repositoriesAfter, jobsBefore, jobsAfter,
				outboxBefore, outboxAfter, idempotencyAfter)
		}
	})
}

func TestTask03HTTPContracts(t *testing.T) {
	h := newTask03Harness(t)
	const (
		analyst = access.RoleAnalyst
		viewer  = access.RoleViewer
		admin   = access.RoleAdmin
	)

	requireHTTPStatus(t, h.request(http.MethodGet, "/api/v1/portfolio", "", "", "", ""), http.StatusUnauthorized, "IT-176")
	h.register(t, "http-seed", "http-seed")
	requireHTTPStatus(t, h.request(http.MethodGet, "/api/v1/portfolio", "", h.tokens[viewer], "", ""), http.StatusOK, "IT-175")
	requireHTTPStatus(t, h.request(http.MethodGet, "/api/v1/projects?state=unknown", "", h.tokens[viewer], "", ""),
		http.StatusBadRequest, "UT-029")
	requireHTTPStatus(t, h.request(http.MethodGet, "/api/v1/projects?state=active&limit=2", "", h.tokens[viewer], "", ""), http.StatusOK, "IT-177")
	requireHTTPStatus(t, h.request(http.MethodGet, "/api/v1/projects", "", "", "", ""), http.StatusUnauthorized, "IT-178")

	createdResponse := h.request(http.MethodPost, "/api/v1/projects",
		`{"repository_url":"https://github.com/temporalio/temporal","history_days":180}`,
		h.tokens[analyst], "http-register", "")
	requireHTTPStatus(t, createdResponse, http.StatusAccepted, "IT-179")
	var created projectstore.Registration
	if err := json.Unmarshal(createdResponse.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode IT-179 response: %v", err)
	}
	requireHTTPStatus(t, h.request(http.MethodPost, "/api/v1/projects",
		`{"repository_url":"https://github.com/temporalio/denied","history_days":180}`,
		h.tokens[viewer], "http-register-denied", ""), http.StatusForbidden, "IT-180")

	projectPath := fmt.Sprintf("/api/v1/projects/%d", created.Project.ID)
	getProject := h.request(http.MethodGet, projectPath, "", h.tokens[viewer], "", "")
	requireHTTPStatus(t, getProject, http.StatusOK, "IT-181")
	if getProject.Header().Get("ETag") != `"v1"` {
		t.Fatalf("IT-181 ETag = %q", getProject.Header().Get("ETag"))
	}
	requireHTTPStatus(t, h.request(http.MethodGet, projectPath, "", "", "", ""), http.StatusUnauthorized, "IT-182")
	updated := h.request(http.MethodPatch, projectPath,
		`{"name":"Temporal","description":"Durable execution"}`, h.tokens[analyst], "", `"v1"`)
	requireHTTPStatus(t, updated, http.StatusOK, "IT-183")
	requireHTTPStatus(t, h.request(http.MethodPatch, projectPath,
		`{"name":"Denied","description":""}`, h.tokens[viewer], "", `"v2"`), http.StatusForbidden, "IT-184")

	lifecycle := h.register(t, "http-lifecycle", "http-lifecycle")
	lifecyclePath := fmt.Sprintf("/api/v1/projects/%d/transition", lifecycle.Project.ID)
	requireHTTPStatus(t, h.request(http.MethodPost, lifecyclePath,
		`{"to":"paused","reason":"Maintenance review"}`, h.tokens[admin], "transition", `"v1"`), http.StatusAccepted, "IT-185")
	requireHTTPStatus(t, h.request(http.MethodPost, lifecyclePath,
		`{"to":"active","reason":"Denied"}`, h.tokens[analyst], "transition-denied", `"v2"`), http.StatusForbidden, "IT-186")
	deleting := h.register(t, "http-delete", "http-delete")
	deletionPath := fmt.Sprintf("/api/v1/projects/%d/deletion", deleting.Project.ID)
	requireHTTPStatus(t, h.request(http.MethodPost, deletionPath,
		`{"confirmation":"DELETE http-delete","reason":"Duplicate project"}`, h.tokens[admin], "deletion", `"v1"`), http.StatusAccepted, "IT-187")
	requireHTTPStatus(t, h.request(http.MethodPost, deletionPath,
		`{"confirmation":"DELETE wrong","reason":"Denied"}`, h.tokens[analyst], "deletion-denied", `"v2"`), http.StatusNotFound, "IT-188")

	repositoriesPath := projectPath + "/repositories"
	requireHTTPStatus(t, h.request(http.MethodGet, repositoriesPath+"?limit=10", "", h.tokens[viewer], "", ""), http.StatusOK, "IT-189")
	requireHTTPStatus(t, h.request(http.MethodGet, repositoriesPath, "", "", "", ""), http.StatusUnauthorized, "IT-190")
	addedRepository := h.request(http.MethodPost, repositoriesPath,
		`{"url":"https://github.com/temporalio/sdk-go","role":"sdk"}`, h.tokens[analyst], "repository-add", "")
	requireHTTPStatus(t, addedRepository, http.StatusCreated, "IT-191")
	var repository project.Repository
	if err := json.Unmarshal(addedRepository.Body.Bytes(), &repository); err != nil {
		t.Fatal(err)
	}
	requireHTTPStatus(t, h.request(http.MethodPost, repositoriesPath,
		`{"url":"https://github.com/temporalio/denied","role":"sdk"}`, h.tokens[viewer], "repository-denied", ""), http.StatusForbidden, "IT-192")
	repositoryPath := fmt.Sprintf("%s/%d", repositoriesPath, repository.ID)
	requireHTTPStatus(t, h.request(http.MethodPatch, repositoryPath,
		`{"role":"primary"}`, h.tokens[analyst], "", `"v1"`), http.StatusOK, "IT-193")
	requireHTTPStatus(t, h.request(http.MethodPatch, repositoryPath,
		`{"role":"invalid"}`, h.tokens[analyst], "", `"v2"`), http.StatusBadRequest, "IT-194")
	primaryID := created.Project.Repositories[0].ID
	requireHTTPStatus(t, h.request(http.MethodDelete, fmt.Sprintf("%s/%d", repositoriesPath, primaryID),
		"", h.tokens[analyst], "", `"v2"`), http.StatusNoContent, "IT-195")
	requireHTTPStatus(t, h.request(http.MethodDelete, repositoryPath, "", h.tokens[viewer], "", `"v2"`), http.StatusForbidden, "IT-196")

	sourcesPath := projectPath + "/sources"
	requireHTTPStatus(t, h.request(http.MethodGet, sourcesPath+"?limit=10", "", h.tokens[viewer], "", ""), http.StatusOK, "IT-197")
	requireHTTPStatus(t, h.request(http.MethodGet, sourcesPath, "", "", "", ""), http.StatusUnauthorized, "IT-198")
	addedSource := h.request(http.MethodPost, sourcesPath,
		`{"kind":"docs","url":"https://docs.temporal.io"}`, h.tokens[analyst], "source-add", "")
	requireHTTPStatus(t, addedSource, http.StatusCreated, "IT-199")
	var source project.Source
	if err := json.Unmarshal(addedSource.Body.Bytes(), &source); err != nil {
		t.Fatal(err)
	}
	requireHTTPStatus(t, h.request(http.MethodPost, sourcesPath,
		`{"kind":"docs","url":"https://denied.example"}`, h.tokens[viewer], "source-denied", ""), http.StatusForbidden, "IT-200")
	sourcePath := fmt.Sprintf("%s/%d", sourcesPath, source.ID)
	requireHTTPStatus(t, h.request(http.MethodPatch, sourcePath,
		`{"state":"paused"}`, h.tokens[analyst], "", `"v1"`), http.StatusOK, "IT-201")
	requireHTTPStatus(t, h.request(http.MethodPatch, sourcePath,
		`{"state":"available"}`, h.tokens[viewer], "", `"v2"`), http.StatusForbidden, "IT-202")
	removedSource := h.request(http.MethodDelete, sourcePath, "", h.tokens[analyst], "", `"v2"`)
	requireHTTPStatus(t, removedSource, http.StatusAccepted, "IT-203")
	requireHTTPStatus(t, h.request(http.MethodDelete, sourcePath, "", h.tokens[viewer], "", `"v3"`), http.StatusForbidden, "IT-204")

	associationSource, err := h.store.AddSource(h.ctx, h.analyst, created.Project.ID,
		project.SourceWebsite, "https://temporal.io", "association-source")
	if err != nil {
		t.Fatal(err)
	}
	associationID, _ := h.storeID()
	_, err = h.pool.Unwrap().Exec(h.ctx, `INSERT INTO source_associations
		(id,source_id,project_id,method,confidence,evidence_ids,decision_version,status)
		VALUES($1,$2,$3,'canonical_url',0.9,$4,'identity-v1','linked')`, associationID,
		associationSource.ID, created.Project.ID, []int64{associationSource.ID})
	if err != nil {
		t.Fatal(err)
	}
	associationsPath := projectPath + "/associations"
	requireHTTPStatus(t, h.request(http.MethodGet, associationsPath, "", h.tokens[viewer], "", ""), http.StatusOK, "IT-205")
	requireHTTPStatus(t, h.request(http.MethodGet, associationsPath, "", "", "", ""), http.StatusUnauthorized, "IT-206")
	correctionPath := fmt.Sprintf("%s/%d/correction", associationsPath, associationID)
	requireHTTPStatus(t, h.request(http.MethodPost, correctionPath,
		`{"action":"split","reason":"Different product"}`, h.tokens[analyst], "correction", ""), http.StatusAccepted, "IT-207")
	requireHTTPStatus(t, h.request(http.MethodPost, correctionPath,
		`{"action":"confirm","reason":"Denied"}`, h.tokens[viewer], "correction-denied", ""), http.StatusForbidden, "IT-208")

	syncResponse := h.request(http.MethodPost, projectPath+"/syncs", `{"scope":"all"}`,
		h.tokens[analyst], "sync", "")
	requireHTTPStatus(t, syncResponse, http.StatusAccepted, "IT-209")
	var syncJob job.Job
	if err := json.Unmarshal(syncResponse.Body.Bytes(), &syncJob); err != nil {
		t.Fatal(err)
	}
	requireHTTPStatus(t, h.request(http.MethodPost, projectPath+"/syncs", `{"scope":"all"}`,
		h.tokens[viewer], "sync-denied", ""), http.StatusForbidden, "IT-210")
	requireHTTPStatus(t, h.request(http.MethodPost, projectPath+"/history-requests",
		`{"from":"2025-08-20","reason":"Annual review"}`, h.tokens[analyst], "history", ""), http.StatusAccepted, "IT-211")
	requireHTTPStatus(t, h.request(http.MethodPost, projectPath+"/history-requests",
		`{"from":"not-a-date","reason":"Invalid"}`, h.tokens[analyst], "history-invalid", ""), http.StatusBadRequest, "IT-212")
	if _, err := h.store.ListJobs(h.ctx, h.analyst, created.Project.ID,
		projectstore.Filter{Limit: 20}); err != nil {
		t.Fatalf("list jobs before IT-213: %v", err)
	}
	requireHTTPStatus(t, h.request(http.MethodGet, projectPath+"/jobs?limit=20", "", h.tokens[viewer], "", ""), http.StatusOK, "IT-213")
	requireHTTPStatus(t, h.request(http.MethodGet, projectPath+"/jobs", "", "", "", ""), http.StatusUnauthorized, "IT-214")
	jobPath := fmt.Sprintf("/api/v1/jobs/%d", syncJob.ID)
	getJob := h.request(http.MethodGet, jobPath, "", h.tokens[viewer], "", "")
	requireHTTPStatus(t, getJob, http.StatusOK, "IT-215")
	if getJob.Header().Get("Retry-After") == "" {
		t.Fatal("IT-215 active Job omitted Retry-After")
	}
	requireHTTPStatus(t, h.request(http.MethodGet, jobPath, "", "", "", ""), http.StatusUnauthorized, "IT-216")
	eventsRequest := httptest.NewRequest(http.MethodGet, jobPath+"/events", nil)
	eventsRequest.Header.Set("Authorization", "Bearer "+h.tokens[viewer])
	eventsRequest.Header.Set("Accept", "text/event-stream")
	streamContext, stopStream := context.WithTimeout(eventsRequest.Context(), 100*time.Millisecond)
	defer stopStream()
	eventsRequest = eventsRequest.WithContext(streamContext)
	eventsResponse := httptest.NewRecorder()
	if err := h.projects.UseWakeups(task03UnavailableWakeups{}); err != nil {
		t.Fatalf("install IT-111 unavailable acceleration: %v", err)
	}
	h.handler.ServeHTTP(eventsResponse, eventsRequest)
	requireHTTPStatus(t, eventsResponse, http.StatusOK, "IT-111 IT-217")
	if eventsResponse.Header().Get("Content-Type") != "text/event-stream" ||
		!bytes.Contains(eventsResponse.Body.Bytes(), []byte("event: job.updated")) {
		t.Fatalf("IT-217 malformed SSE response: headers=%v body=%s", eventsResponse.Header(), eventsResponse.Body.String())
	}
	requireHTTPStatus(t, h.request(http.MethodGet, jobPath+"/events", "", "", "", ""), http.StatusUnauthorized, "IT-218")
	requireHTTPStatus(t, h.request(http.MethodPost, jobPath+"/cancellation", `{"reason":"No longer needed"}`,
		h.tokens[analyst], "cancel", ""), http.StatusAccepted, "IT-219")
	terminalRequest := httptest.NewRequest(http.MethodGet, jobPath+"/events", nil)
	terminalRequest.Header.Set("Authorization", "Bearer "+h.tokens[viewer])
	terminalRequest.Header.Set("Accept", "text/event-stream")
	terminalRequest.Header.Set("Last-Event-ID", "0")
	terminalResponse := httptest.NewRecorder()
	h.handler.ServeHTTP(terminalResponse, terminalRequest)
	requireHTTPStatus(t, terminalResponse, http.StatusOK, "IT-110")
	if !bytes.Contains(terminalResponse.Body.Bytes(), []byte(`"state":"cancelled"`)) {
		t.Fatalf("IT-110 terminal SSE did not replay and close: %s", terminalResponse.Body.String())
	}
	requireHTTPStatus(t, h.request(http.MethodPost, jobPath+"/cancellation", `{"reason":"Denied"}`,
		h.tokens[viewer], "cancel-denied", ""), http.StatusForbidden, "IT-220")

	requireHTTPStatus(t, h.request(http.MethodPatch, projectPath,
		`{"name":"Stale secret-value","description":"must not be audited"}`,
		h.tokens[analyst], "", `"v1"`), http.StatusPreconditionFailed, "IT-129 stale")
	rows, err := h.pool.Unwrap().Query(h.ctx, `SELECT id,outcome,action,changes::text
		FROM audit_events WHERE resource_type='http_route' ORDER BY id`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	outcomes := map[string]int{}
	var previous int64
	for rows.Next() {
		var id int64
		var outcome, action, changes string
		if err := rows.Scan(&id, &outcome, &action, &changes); err != nil {
			t.Fatal(err)
		}
		if id <= previous {
			t.Fatalf("IT-129 audit IDs are not ordered: %d after %d", id, previous)
		}
		previous = id
		outcomes[outcome]++
		serialized := strings.ToLower(action + " " + changes)
		for _, forbidden := range []string{"secret-value", "authorization", "bearer ", "csrf", "?"} {
			if strings.Contains(serialized, forbidden) {
				t.Fatalf("IT-129 audit retained forbidden content %q", serialized)
			}
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	for _, outcome := range []string{"succeeded", "denied", "stale", "failed"} {
		if outcomes[outcome] == 0 {
			t.Fatalf("IT-129 missing %s transport audit outcome: %#v", outcome, outcomes)
		}
	}

}

func newTask03Harness(t *testing.T) *task03Harness {
	t.Helper()
	ctx := context.Background()
	baseURL := os.Getenv("OPI_INTEGRATION_DATABASE_URL")
	if baseURL == "" {
		t.Fatal("OPI_INTEGRATION_DATABASE_URL is required")
	}
	parsed, err := url.Parse(baseURL)
	if err != nil {
		t.Fatalf("parse integration database URL: %v", err)
	}
	databaseName := fmt.Sprintf("opi_task03_%d", time.Now().UnixNano())
	connection, err := pgx.Connect(ctx, baseURL)
	if err != nil {
		t.Fatalf("connect integration server: %v", err)
	}
	if _, err = connection.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{databaseName}.Sanitize()); err != nil {
		connection.Close(ctx)
		t.Fatalf("create integration database: %v", err)
	}
	connection.Close(ctx)
	parsed.Path = "/" + databaseName
	databaseURL := parsed.String()
	root := task03RepositoryRoot(t)
	command := exec.Command(filepath.Join(root, "scripts", "migrate.sh"), "up")
	command.Dir = root
	command.Env = append(os.Environ(), "DATABASE_URL="+databaseURL)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("migrate task database: %v\n%s", err, output)
	}
	pool, err := database.Open(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open task database: %v", err)
	}
	if _, err := pool.Unwrap().Exec(ctx,
		`INSERT INTO workspaces(id,name) VALUES($1,'Task 3 workspace')`, task03WorkspaceID); err != nil {
		pool.Close()
		t.Fatalf("seed workspace: %v", err)
	}
	ids := &task03IDs{}
	ids.value.Store(8_100_000_000_000_000_000)
	h := &task03Harness{
		ctx: ctx, pool: pool, store: projectstore.New(pool, ids), jobs: jobstore.New(pool, ids),
		ingestion: ingestion.New(pool, ids),
		admin: access.Principal{ActorID: 9001, Kind: access.ActorMember, Role: access.RoleAdmin,
			Status: access.StatusActive, Workspace: task03WorkspaceID},
		analyst: access.Principal{ActorID: 9002, Kind: access.ActorMember, Role: access.RoleAnalyst,
			Status: access.StatusActive, Workspace: task03WorkspaceID},
		tokens: map[access.Role]string{},
	}
	const issuer = "https://task03.integration.test/realms/opi"
	provider := &task03IdentityProvider{identities: map[string]oidc.Identity{}}
	for index, role := range []access.Role{access.RoleViewer, access.RoleAnalyst, access.RoleAdmin} {
		identityID := int64(100 + index)
		memberID := int64(200 + index)
		subject := "task03-" + string(role)
		token := "task03-token-" + string(role)
		_, err := pool.Unwrap().Exec(ctx, `INSERT INTO external_identities
			(id,issuer,subject,display_name,email) VALUES($1,$2,$3,$4,$5)`, identityID,
			issuer, subject, subject, subject+"@example.test")
		if err != nil {
			pool.Close()
			t.Fatalf("seed %s identity: %v", role, err)
		}
		_, err = pool.Unwrap().Exec(ctx, `INSERT INTO memberships
			(id,workspace_id,identity_id,role,status) VALUES($1,$2,$3,$4,'active')`,
			memberID, task03WorkspaceID, identityID, role)
		if err != nil {
			pool.Close()
			t.Fatalf("seed %s membership: %v", role, err)
		}
		provider.identities[token] = oidc.Identity{Key: access.IdentityKey{Issuer: issuer, Subject: subject}}
		h.tokens[role] = token
	}
	cursors, err := access.NewCursorCodec(bytes.Repeat([]byte{0x53}, 32))
	if err != nil {
		pool.Close()
		t.Fatalf("construct cursor codec: %v", err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	accessHandler, err := accessapi.New(accessstore.New(pool, ids), provider, cursors, logger,
		accessapi.Config{PublicBaseURL: "http://opi.integration.test", IssuerURL: issuer,
			SessionTTL: time.Hour})
	if err != nil {
		pool.Close()
		t.Fatalf("construct access middleware: %v", err)
	}
	projectHandler, err := projectapi.NewWithURLValidator(h.store, cursors, logger, task03URLValidator{})
	if err != nil {
		pool.Close()
		t.Fatalf("construct project handler: %v", err)
	}
	h.projects = projectHandler
	h.handler = accessHandler.Middleware(projectapi.Routes(projectHandler))
	t.Cleanup(func() {
		pool.Close()
		cleanup, cleanupErr := pgx.Connect(context.Background(), baseURL)
		if cleanupErr != nil {
			t.Errorf("connect database cleanup: %v", cleanupErr)
			return
		}
		defer cleanup.Close(context.Background())
		if _, cleanupErr = cleanup.Exec(context.Background(),
			"DROP DATABASE "+pgx.Identifier{databaseName}.Sanitize()+" WITH (FORCE)"); cleanupErr != nil {
			t.Errorf("drop task database: %v", cleanupErr)
		}
	})
	return h
}

func (h *task03Harness) register(t *testing.T, key, repository string) projectstore.Registration {
	t.Helper()
	value, err := h.store.Register(h.ctx, h.analyst,
		"https://github.com/acme/"+repository, 180, key, "register-"+key)
	if err != nil {
		t.Fatalf("register %s: %v", repository, err)
	}
	return value
}

func (h *task03Harness) storeID() (int64, error) {
	var id int64
	err := h.pool.Unwrap().QueryRow(h.ctx,
		`SELECT 8200000000000000000 + count(*) FROM source_associations`).Scan(&id)
	return id, err
}

func (h *task03Harness) request(method, path, body, token, idempotency, etag string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	if idempotency != "" {
		request.Header.Set("Idempotency-Key", idempotency)
	}
	if etag != "" {
		request.Header.Set("If-Match", etag)
	}
	response := httptest.NewRecorder()
	h.handler.ServeHTTP(response, request)
	return response
}

func requireHTTPStatus(t *testing.T, response *httptest.ResponseRecorder, want int, id string) {
	t.Helper()
	if response.Code != want {
		t.Fatalf("%s status = %d, want %d: %s", id, response.Code, want, response.Body.String())
	}
}

func task03RepositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve integration root")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}
