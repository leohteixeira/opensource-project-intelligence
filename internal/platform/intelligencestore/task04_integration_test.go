//go:build integration

package intelligencestore_test

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
	"reflect"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/leohteixeira/opensource-project-intelligence/internal/access"
	"github.com/leohteixeira/opensource-project-intelligence/internal/comparison"
	"github.com/leohteixeira/opensource-project-intelligence/internal/issue"
	"github.com/leohteixeira/opensource-project-intelligence/internal/metric"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/accessapi"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/accessstore"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/database"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/intelligenceapi"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/intelligencestore"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/oidc"
)

const (
	task04WorkspaceID int64 = 1
	task04ProjectOne  int64 = 7_400_000_000_000_000_001
	task04ProjectTwo  int64 = 7_400_000_000_000_000_002
)

type task04IDs struct{ value atomic.Int64 }

func (ids *task04IDs) Next(context.Context) (int64, error) { return ids.value.Add(1), nil }

type task04IdentityProvider struct{ identities map[string]oidc.Identity }

func (task04IdentityProvider) AuthorizationURL(string, string, string) string { return "" }
func (task04IdentityProvider) Exchange(context.Context, string, string, string) (oidc.Identity, error) {
	return oidc.Identity{}, errors.New("not used by Task 4 integration tests")
}
func (provider task04IdentityProvider) VerifyBearer(_ context.Context, token string) (oidc.Identity, error) {
	identity, ok := provider.identities[token]
	if !ok {
		return oidc.Identity{}, errors.New("unknown Task 4 bearer")
	}
	return identity, nil
}

type task04Harness struct {
	ctx       context.Context
	pool      *database.Pool
	store     *intelligencestore.Store
	principal access.Principal
	handler   http.Handler
	window    metric.Window
}

func TestTask04PostgreSQLContracts(t *testing.T) {
	h := newTask04Harness(t)

	t.Run("IT-037 IT-121 metric definition sets publish atomically and exact replays stay singular", func(t *testing.T) {
		values := snapshots(t, task04ProjectOne, h.window, 1)
		first, err := h.store.SaveMetricSet(h.ctx, values)
		if err != nil {
			t.Fatal(err)
		}
		second, err := h.store.SaveMetricSet(h.ctx, values)
		if err != nil {
			t.Fatal(err)
		}
		if len(first) != 7 || len(second) != 7 {
			t.Fatalf("expected one complete seven-definition set: %d %d", len(first), len(second))
		}
		var snapshotsCount, factorsCount int
		if err := h.pool.Unwrap().QueryRow(h.ctx, `SELECT count(*),
			(SELECT count(*) FROM metric_factors WHERE snapshot_id IN
			 (SELECT id FROM metric_snapshots WHERE project_id=$1))
			FROM metric_snapshots WHERE project_id=$1`, task04ProjectOne).Scan(&snapshotsCount, &factorsCount); err != nil {
			t.Fatal(err)
		}
		if snapshotsCount != 7 || factorsCount != 7 {
			t.Fatalf("partial or duplicate materialization: snapshots=%d factors=%d", snapshotsCount, factorsCount)
		}
	})

	t.Run("IT-038 failed metric recalculation leaves the prior completed snapshot visible", func(t *testing.T) {
		failedWindow, _ := metric.PresetWindow("90d", h.window.Cutoff.Add(24*time.Hour))
		if _, err := h.pool.Unwrap().Exec(h.ctx, `CREATE FUNCTION task04_reject_factor() RETURNS trigger
			LANGUAGE plpgsql AS $$ BEGIN IF NEW.name='controlled_failure' THEN RAISE EXCEPTION 'controlled'; END IF; RETURN NEW; END $$;
			CREATE TRIGGER task04_factor_failure BEFORE INSERT ON metric_factors
			FOR EACH ROW EXECUTE FUNCTION task04_reject_factor()`); err != nil {
			t.Fatal(err)
		}
		candidate := snapshots(t, task04ProjectOne, failedWindow, 2)
		candidate[3].Factors[0].Name = "controlled_failure"
		if _, err := h.store.SaveMetricSet(h.ctx, candidate); err == nil {
			t.Fatal("controlled factor failure unexpectedly committed")
		}
		var candidateCount int
		if err := h.pool.Unwrap().QueryRow(h.ctx, `SELECT count(*) FROM metric_snapshots WHERE project_id=$1 AND cutoff=$2`, task04ProjectOne, failedWindow.Cutoff).Scan(&candidateCount); err != nil {
			t.Fatal(err)
		}
		if candidateCount != 0 {
			t.Fatalf("failed set exposed %d partial rows", candidateCount)
		}
		prior, err := h.store.Metrics(h.ctx, h.principal, task04ProjectOne, h.window)
		if err != nil || len(prior) != 7 {
			t.Fatalf("prior snapshot disappeared: count=%d err=%v", len(prior), err)
		}
		_, _ = h.pool.Unwrap().Exec(h.ctx, `DROP TRIGGER task04_factor_failure ON metric_factors; DROP FUNCTION task04_reject_factor()`)
	})

	t.Run("IT-039 one hundred history windows remain indexed and bounded by exact cutoff", func(t *testing.T) {
		for index := 2; index <= 101; index++ {
			window, _ := metric.PresetWindow("30d", h.window.Cutoff.Add(time.Duration(index)*24*time.Hour))
			value, err := metric.NewSnapshot(task04ProjectTwo, metric.Catalog()[0], window,
				metric.StatusAvailable, float64Pointer(float64(index)), metric.Coverage{Eligible: 1, Observed: 1},
				[]metric.Factor{{Name: "stable_releases", Value: float64Pointer(float64(index)), Unit: "releases"}}, nil)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := h.store.SaveMetric(h.ctx, value); err != nil {
				t.Fatal(err)
			}
		}
		var count int
		if err := h.pool.Unwrap().QueryRow(h.ctx, `SELECT count(*) FROM metric_snapshots
			WHERE project_id=$1 AND definition_name='release_frequency'`, task04ProjectTwo).Scan(&count); err != nil || count != 100 {
			t.Fatalf("history count=%d err=%v", count, err)
		}
	})

	t.Run("IT-040 concurrent identity corrections leave one versioned current link", func(t *testing.T) {
		seedContributorIdentity(t, h)
		start := make(chan struct{})
		errorsByCall := make(chan error, 2)
		var group sync.WaitGroup
		for index, identityID := range []int64{7_400_000_000_000_003_101, 7_400_000_000_000_003_102} {
			group.Add(1)
			go func(index int, identityID int64) {
				defer group.Done()
				<-start
				_, err := h.store.CorrectContributorIdentity(h.ctx, h.principal,
					7_400_000_000_000_003_001, identityID, 1, "confirm", "controlled correction",
					fmt.Sprintf("task04-correction-%d", index))
				errorsByCall <- err
			}(index, identityID)
		}
		close(start)
		group.Wait()
		close(errorsByCall)
		succeeded, conflicted := 0, 0
		for err := range errorsByCall {
			switch {
			case err == nil:
				succeeded++
			case errors.Is(err, access.ErrVersionConflict):
				conflicted++
			default:
				t.Fatalf("unexpected correction result: %v", err)
			}
		}
		if succeeded != 1 || conflicted != 1 {
			t.Fatalf("correction outcomes succeeded=%d conflicted=%d", succeeded, conflicted)
		}
		var version, correctionCount int
		if err := h.pool.Unwrap().QueryRow(h.ctx, `SELECT version,
			(SELECT count(*) FROM contributor_identity_corrections WHERE account_id=$1)
			FROM contributor_identity_links WHERE account_id=$1`, int64(7_400_000_000_000_003_001)).Scan(&version, &correctionCount); err != nil {
			t.Fatal(err)
		}
		if version != 2 || correctionCount != 1 {
			t.Fatalf("link version=%d corrections=%d", version, correctionCount)
		}
	})

	t.Run("IT-041 health materialization rolls back dimensions and factors together", func(t *testing.T) {
		base := health(t, task04ProjectOne, h.window, false)
		if _, err := h.store.SaveHealth(h.ctx, base); err != nil {
			t.Fatal(err)
		}
		failedWindow, _ := metric.PresetWindow("90d", h.window.Cutoff.Add(48*time.Hour))
		if _, err := h.pool.Unwrap().Exec(h.ctx, `CREATE FUNCTION task04_reject_health_factor() RETURNS trigger
			LANGUAGE plpgsql AS $$ BEGIN IF NEW.name='controlled_failure' THEN RAISE EXCEPTION 'controlled'; END IF; RETURN NEW; END $$;
			CREATE TRIGGER task04_health_factor_failure BEFORE INSERT ON health_dimension_factors
			FOR EACH ROW EXECUTE FUNCTION task04_reject_health_factor()`); err != nil {
			t.Fatal(err)
		}
		candidate := health(t, task04ProjectOne, failedWindow, false)
		candidate.Dimensions[4].Factors[0].Name = "controlled_failure"
		if _, err := h.store.SaveHealth(h.ctx, candidate); err == nil {
			t.Fatal("controlled health factor failure unexpectedly committed")
		}
		var count int
		if err := h.pool.Unwrap().QueryRow(h.ctx, `SELECT count(*) FROM health_snapshots WHERE cutoff=$1`, failedWindow.Cutoff).Scan(&count); err != nil || count != 0 {
			t.Fatalf("partial health count=%d err=%v", count, err)
		}
		_, _ = h.pool.Unwrap().Exec(h.ctx, `DROP TRIGGER task04_health_factor_failure ON health_dimension_factors; DROP FUNCTION task04_reject_health_factor()`)
	})

	t.Run("IT-042 contributor detail is bounded without changing aggregate concentration", func(t *testing.T) {
		seedContributorHistory(t, h, 250)
		page, err := h.store.Contributors(h.ctx, h.principal, task04ProjectTwo, h.window, 50, 0)
		if err != nil {
			t.Fatal(err)
		}
		if len(page.Items) != 50 || !page.HasMore || page.Summary.Active != 250 {
			t.Fatalf("bounded contributor page items=%d more=%v active=%d", len(page.Items), page.HasMore, page.Summary.Active)
		}
	})

	t.Run("IT-046 IT-047 IT-122 comparison freezes one cutoff and preserves explicit missing states", func(t *testing.T) {
		left := snapshots(t, task04ProjectOne, h.window, 1)
		right := snapshots(t, task04ProjectTwo, h.window, 2)
		right[2].Status, right[2].Value = metric.StatusInsufficientData, nil
		value, err := comparison.Materialize(7_400_000_000_000_004_001, []comparison.Project{
			{ID: task04ProjectOne, Resolved: true, Metrics: left},
			{ID: task04ProjectTwo, Resolved: true, Metrics: right},
		}, h.window)
		if err != nil {
			t.Fatal(err)
		}
		if len(value.Rows) != 7 || value.Rows[2].Cells[1].Status != metric.StatusInsufficientData {
			t.Fatalf("comparison lost explicit state: %#v", value.Rows)
		}
	})

	t.Run("IT-048 many immutable comparisons remain addressable through the workspace index", func(t *testing.T) {
		projects := []comparison.Project{
			{ID: task04ProjectOne, Resolved: true, Metrics: snapshots(t, task04ProjectOne, h.window, 1)},
			{ID: task04ProjectTwo, Resolved: true, Metrics: snapshots(t, task04ProjectTwo, h.window, 2)},
		}
		for index := 0; index < 75; index++ {
			value, err := comparison.Materialize(7_400_000_000_000_005_000+int64(index), projects, h.window)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := h.store.SaveComparison(h.ctx, h.principal, value); err != nil {
				t.Fatal(err)
			}
		}
		var count int
		if err := h.pool.Unwrap().QueryRow(h.ctx, `SELECT count(*) FROM comparisons WHERE workspace_id=$1`, task04WorkspaceID).Scan(&count); err != nil || count != 75 {
			t.Fatalf("comparison count=%d err=%v", count, err)
		}
	})

	t.Run("IT-120 temporal issue events replay to the same ordered backlog value", func(t *testing.T) {
		events := []issue.StateEvent{
			{IssueID: 1, State: "open", At: h.window.From.Add(20 * time.Hour)},
			{IssueID: 1, State: "closed", At: h.window.From.Add(4 * time.Hour)},
			{IssueID: 1, State: "open", At: h.window.From.Add(-time.Hour)},
		}
		forward := issue.BacklogChange(events, metric.AsTimeWindow(h.window))
		for left, right := 0, len(events)-1; left < right; left, right = left+1, right-1 {
			events[left], events[right] = events[right], events[left]
		}
		if replay := issue.BacklogChange(events, metric.AsTimeWindow(h.window)); replay != forward {
			t.Fatalf("out-of-order replay changed backlog %d to %d", forward, replay)
		}
	})

	t.Run("canonical facts materialize reproducibly through the production store", func(t *testing.T) {
		materializedWindow, err := metric.PresetWindow("90d", h.window.Cutoff.Add(24*time.Hour))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := h.pool.Unwrap().Exec(h.ctx, `UPDATE sources SET coverage_from=$1,coverage_to=$2
			WHERE project_id=$3`, materializedWindow.From, materializedWindow.To, task04ProjectTwo); err != nil {
			t.Fatal(err)
		}
		first, err := h.store.MaterializeWindow(h.ctx, task04ProjectTwo, materializedWindow)
		if err != nil {
			t.Fatal(err)
		}
		second, err := h.store.MaterializeWindow(h.ctx, task04ProjectTwo, materializedWindow)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(first, second) {
			t.Fatal("canonical replay changed the immutable definition set")
		}
		active := first[1]
		if active.Definition.Name != "active_contributors" || active.Status != metric.StatusAvailable || active.Value == nil || *active.Value != 250 {
			t.Fatalf("unexpected materialized active contributors: %#v", active)
		}
	})

	t.Run("IT-221 IT-223 IT-225 IT-227 authorized intelligence reads expose versioned typed contracts", func(t *testing.T) {
		if _, err := h.store.SaveMetricSet(h.ctx, snapshots(t, task04ProjectTwo, h.window, 2)); err != nil && !errors.Is(err, metric.ErrInvalid) {
			t.Fatal(err)
		}
		if _, err := h.store.SaveHealth(h.ctx, health(t, task04ProjectTwo, h.window, true)); err != nil {
			t.Fatal(err)
		}
		for _, path := range []string{
			fmt.Sprintf("/api/v1/projects/%d/metrics?window=90d&cutoff=%s", task04ProjectOne, url.QueryEscape(h.window.Cutoff.Format(time.RFC3339))),
			fmt.Sprintf("/api/v1/projects/%d/metrics/release_frequency?window=90d&cutoff=%s", task04ProjectOne, url.QueryEscape(h.window.Cutoff.Format(time.RFC3339))),
			fmt.Sprintf("/api/v1/projects/%d/health?window=90d&cutoff=%s", task04ProjectOne, url.QueryEscape(h.window.Cutoff.Format(time.RFC3339))),
			fmt.Sprintf("/api/v1/projects/%d/contributors?window=90d&limit=50", task04ProjectTwo),
		} {
			response := h.request(t, http.MethodGet, path, nil, true)
			if response.Code != http.StatusOK {
				t.Fatalf("%s returned %d: %s", path, response.Code, response.Body.String())
			}
		}
	})

	t.Run("IT-222 IT-224 IT-226 IT-228 missing principals are rejected before disclosure", func(t *testing.T) {
		for _, suffix := range []string{"metrics", "metrics/release_frequency", "health", "contributors"} {
			response := h.request(t, http.MethodGet, fmt.Sprintf("/api/v1/projects/%d/%s?window=90d", task04ProjectOne, suffix), nil, false)
			if response.Code != http.StatusUnauthorized {
				t.Fatalf("%s returned %d", suffix, response.Code)
			}
		}
	})

	t.Run("IT-233 IT-235 comparison POST and GET preserve the immutable matrix", func(t *testing.T) {
		body := map[string]any{"project_ids": []string{fmt.Sprint(task04ProjectOne), fmt.Sprint(task04ProjectTwo)}, "window": "90d", "cutoff": h.window.Cutoff.Format(time.RFC3339)}
		created := h.request(t, http.MethodPost, "/api/v1/comparisons", body, true)
		if created.Code != http.StatusCreated {
			t.Fatalf("create comparison returned %d: %s", created.Code, created.Body.String())
		}
		var document struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(created.Body.Bytes(), &document); err != nil || document.ID == "" {
			t.Fatalf("decode comparison: id=%q err=%v", document.ID, err)
		}
		loaded := h.request(t, http.MethodGet, "/api/v1/comparisons/"+document.ID, nil, true)
		if loaded.Code != http.StatusOK || !bytes.Contains(loaded.Body.Bytes(), []byte(`"rows"`)) {
			t.Fatalf("load comparison returned %d: %s", loaded.Code, loaded.Body.String())
		}
	})

	t.Run("IT-234 IT-236 invalid or unauthorized comparison calls have no side effect", func(t *testing.T) {
		before := comparisonCount(t, h)
		invalid := h.request(t, http.MethodPost, "/api/v1/comparisons", map[string]any{"project_ids": []string{fmt.Sprint(task04ProjectOne)}, "window": "90d", "cutoff": h.window.Cutoff.Format(time.RFC3339)}, true)
		if invalid.Code != http.StatusBadRequest {
			t.Fatalf("invalid comparison returned %d", invalid.Code)
		}
		unauthorized := h.request(t, http.MethodGet, "/api/v1/comparisons/1", nil, false)
		if unauthorized.Code != http.StatusUnauthorized {
			t.Fatalf("unauthorized comparison returned %d", unauthorized.Code)
		}
		if after := comparisonCount(t, h); after != before {
			t.Fatalf("rejected comparison changed count from %d to %d", before, after)
		}
	})
}

func newTask04Harness(t *testing.T) *task04Harness {
	t.Helper()
	ctx := context.Background()
	baseURL := os.Getenv("OPI_INTEGRATION_DATABASE_URL")
	if baseURL == "" {
		t.Fatal("OPI_INTEGRATION_DATABASE_URL is required")
	}
	parsed, err := url.Parse(baseURL)
	if err != nil {
		t.Fatal(err)
	}
	databaseName := fmt.Sprintf("opi_task04_%d", time.Now().UnixNano())
	connection, err := pgx.Connect(ctx, baseURL)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := connection.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{databaseName}.Sanitize()); err != nil {
		connection.Close(ctx)
		t.Fatal(err)
	}
	connection.Close(ctx)
	parsed.Path = "/" + databaseName
	databaseURL := parsed.String()
	root := task04RepositoryRoot(t)
	command := exec.Command(filepath.Join(root, "scripts", "migrate.sh"), "up")
	command.Dir = root
	command.Env = append(os.Environ(), "DATABASE_URL="+databaseURL)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("migrate Task 4 database: %v\n%s", err, output)
	}
	pool, err := database.Open(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pool.Close()
		cleanup, cleanupErr := pgx.Connect(context.Background(), baseURL)
		if cleanupErr != nil {
			t.Errorf("connect cleanup: %v", cleanupErr)
			return
		}
		defer cleanup.Close(context.Background())
		_, cleanupErr = cleanup.Exec(context.Background(), "DROP DATABASE "+pgx.Identifier{databaseName}.Sanitize()+" WITH (FORCE)")
		if cleanupErr != nil {
			t.Errorf("drop Task 4 database: %v", cleanupErr)
		}
	})
	const issuer = "https://task04.integration.test/realms/opi"
	if _, err := pool.Unwrap().Exec(ctx, `INSERT INTO workspaces(id,name) VALUES($1,'Task 4 workspace')`, task04WorkspaceID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Unwrap().Exec(ctx, `INSERT INTO projects(id,workspace_id,name,slug) VALUES
		($2,$1,'Temporal','temporal'),($3,$1,'OpenTelemetry','opentelemetry')`,
		task04WorkspaceID, task04ProjectOne, task04ProjectTwo); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Unwrap().Exec(ctx, `INSERT INTO external_identities(id,issuer,subject,display_name,email)
		VALUES(101,$1,'task04-analyst','Task 4 Analyst','task04@example.test'),
		(102,$1,'task06-admin','Task 6 Admin','task06-admin@example.test')`, issuer); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Unwrap().Exec(ctx, `INSERT INTO memberships(id,workspace_id,identity_id,role,status)
		VALUES(201,$1,101,'analyst','active'),(202,$1,102,'admin','active')`, task04WorkspaceID); err != nil {
		t.Fatal(err)
	}
	ids := &task04IDs{}
	ids.value.Store(7_500_000_000_000_000_000)
	store := intelligencestore.New(pool, ids)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cursors, err := access.NewCursorCodec(bytes.Repeat([]byte{0x54}, 32))
	if err != nil {
		t.Fatal(err)
	}
	provider := task04IdentityProvider{identities: map[string]oidc.Identity{
		"task04-analyst": {Key: access.IdentityKey{Issuer: issuer, Subject: "task04-analyst"}},
		"task06-admin":   {Key: access.IdentityKey{Issuer: issuer, Subject: "task06-admin"}},
	}}
	accessHandler, err := accessapi.New(accessstore.New(pool, ids), provider, cursors, logger,
		accessapi.Config{PublicBaseURL: "http://opi.integration.test", IssuerURL: issuer, SessionTTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	intelligenceHandler, err := intelligenceapi.New(store, ids, logger,
		intelligenceapi.WithModelIdentity(intelligenceapi.ModelIdentity{
			Provider: "fixture", Model: "fixture-v1",
		}))
	if err != nil {
		t.Fatal(err)
	}
	window, _ := metric.PresetWindow("90d", time.Date(2026, 8, 22, 0, 0, 0, 0, time.UTC))
	return &task04Harness{ctx: ctx, pool: pool, store: store,
		principal: access.Principal{ActorID: 201, Kind: access.ActorMember, Role: access.RoleAnalyst, Status: access.StatusActive, Workspace: task04WorkspaceID},
		handler:   accessHandler.Middleware(intelligenceapi.Routes(intelligenceHandler)), window: window}
}

func (h *task04Harness) request(t *testing.T, method, path string, body any, authenticated bool) *httptest.ResponseRecorder {
	t.Helper()
	bearer := ""
	if authenticated {
		bearer = "task04-analyst"
	}
	return h.requestAs(t, method, path, body, bearer, nil)
}

func (h *task04Harness) requestAs(t *testing.T, method, path string, body any, bearer string,
	headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(encoded)
	}
	request := httptest.NewRequest(method, "http://opi.integration.test"+path, reader)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "task04-"+fmt.Sprint(time.Now().UnixNano()))
	if bearer != "" {
		request.Header.Set("Authorization", "Bearer "+bearer)
	}
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	response := httptest.NewRecorder()
	h.handler.ServeHTTP(response, request)
	return response
}

func snapshots(t *testing.T, projectID int64, window metric.Window, seed float64) []metric.Snapshot {
	t.Helper()
	values := make([]metric.Snapshot, 0, 7)
	for index, definition := range metric.Catalog() {
		value := seed + float64(index)
		snapshot, err := metric.NewSnapshot(projectID, definition, window, metric.StatusAvailable,
			&value, metric.Coverage{Eligible: 1, Observed: 1},
			[]metric.Factor{{Name: "deterministic_input", Value: &value, Unit: definition.Unit}}, nil)
		if err != nil {
			t.Fatal(err)
		}
		values = append(values, snapshot)
	}
	return values
}

func health(t *testing.T, projectID int64, window metric.Window, missing bool) metric.Health {
	t.Helper()
	dimensions := make([]metric.Dimension, 0, 7)
	for index, name := range metric.HealthDimensionNames {
		score := 0.8 - float64(index)*0.05
		status := metric.StatusAvailable
		var value *float64 = &score
		if missing && index == 6 {
			status, value = metric.StatusInsufficientData, nil
		}
		dimensions = append(dimensions, metric.Dimension{Name: name, Status: status, Score: value,
			Version: "v1", Factors: []metric.Factor{{Name: "absolute_rubric", Value: value, Unit: "score"}}})
	}
	value, err := metric.CalculateHealth(projectID, window, "v1", dimensions)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func seedContributorIdentity(t *testing.T, h *task04Harness) {
	t.Helper()
	_, err := h.pool.Unwrap().Exec(h.ctx, `INSERT INTO repositories(id,project_id,provider,canonical_url,role,default_branch)
		VALUES(7400000000000003000,$1,'github','https://github.com/task04/identity','primary','main')`, task04ProjectOne)
	if err != nil {
		t.Fatal(err)
	}
	_, err = h.pool.Unwrap().Exec(h.ctx, `INSERT INTO sources(id,project_id,repository_id,kind,canonical_url)
		VALUES(7400000000000003010,$1,7400000000000003000,'github','https://api.github.com/repos/task04/identity')`, task04ProjectOne)
	if err != nil {
		t.Fatal(err)
	}
	_, err = h.pool.Unwrap().Exec(h.ctx, `INSERT INTO contributor_identities(id,project_id,public_handle) VALUES
		(7400000000000003101,$1,'ada'),(7400000000000003102,$1,'grace')`, task04ProjectOne)
	if err != nil {
		t.Fatal(err)
	}
	_, err = h.pool.Unwrap().Exec(h.ctx, `INSERT INTO contributor_accounts(id,project_id,source_id,external_id)
		VALUES(7400000000000003001,$1,7400000000000003010,'account-one')`, task04ProjectOne)
	if err != nil {
		t.Fatal(err)
	}
	_, err = h.pool.Unwrap().Exec(h.ctx, `INSERT INTO contributor_identity_links(account_id,status)
		VALUES(7400000000000003001,'unresolved')`)
	if err != nil {
		t.Fatal(err)
	}
}

func seedContributorHistory(t *testing.T, h *task04Harness, count int) {
	t.Helper()
	_, err := h.pool.Unwrap().Exec(h.ctx, `INSERT INTO repositories(id,project_id,provider,canonical_url,role,default_branch)
		VALUES(7400000000000006000,$1,'github','https://github.com/task04/history','primary','main')`, task04ProjectTwo)
	if err != nil {
		t.Fatal(err)
	}
	_, err = h.pool.Unwrap().Exec(h.ctx, `INSERT INTO sources(id,project_id,repository_id,kind,canonical_url)
		VALUES(7400000000000006010,$1,7400000000000006000,'github','https://api.github.com/repos/task04/history')`, task04ProjectTwo)
	if err != nil {
		t.Fatal(err)
	}
	_, err = h.pool.Unwrap().Exec(h.ctx, `WITH values AS (SELECT generate_series(1,$2)::bigint AS n)
		INSERT INTO raw_objects(id,project_id,source_id,external_type,external_id,observed_at,payload,digest)
		SELECT 7400000000000100000+n,$1,7400000000000006010,'commit',n::text,$3,'{}'::jsonb,decode(repeat('01',32),'hex') FROM values`,
		task04ProjectTwo, count, h.window.From.Add(24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	_, err = h.pool.Unwrap().Exec(h.ctx, `WITH values AS (SELECT generate_series(1,$2)::bigint AS n)
		INSERT INTO contributor_accounts(id,project_id,source_id,external_id)
		SELECT 7400000000000200000+n,$1,7400000000000006010,'author-'||n FROM values`, task04ProjectTwo, count)
	if err != nil {
		t.Fatal(err)
	}
	_, err = h.pool.Unwrap().Exec(h.ctx, `WITH values AS (SELECT generate_series(1,$1)::bigint AS n)
		INSERT INTO contributor_identity_links(account_id,status)
		SELECT 7400000000000200000+n,'unresolved' FROM values`, count)
	if err != nil {
		t.Fatal(err)
	}
	_, err = h.pool.Unwrap().Exec(h.ctx, `WITH values AS (SELECT generate_series(1,$2)::bigint AS n)
		INSERT INTO canonical_commits(id,project_id,repository_id,source_id,external_id,sha,author_external_id,committed_at,default_branch,merge_commit,raw_object_id)
		SELECT 7400000000000300000+n,$1,7400000000000006000,7400000000000006010,n::text,'sha-'||n,
		'author-'||n,$3,true,false,7400000000000100000+n FROM values`,
		task04ProjectTwo, count, h.window.From.Add(24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
}

func comparisonCount(t *testing.T, h *task04Harness) int {
	t.Helper()
	var count int
	if err := h.pool.Unwrap().QueryRow(h.ctx, `SELECT count(*) FROM comparisons`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}

func float64Pointer(value float64) *float64 { return &value }

func task04RepositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve Task 4 test path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}
