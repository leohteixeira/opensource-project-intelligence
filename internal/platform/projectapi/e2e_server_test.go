//go:build e2e

package projectapi_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/leohteixeira/opensource-project-intelligence/internal/access"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/accessapi"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/accessstore"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/database"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/oidc"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/projectapi"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/projectstore"
)

const (
	browserDatabase    = "opi_task03_browser"
	browserIssuer      = "https://task03.browser.test/realms/opi"
	browserToken       = "task03-browser-admin"
	browserWorkspaceID = int64(1)
)

type browserIDs struct{ value atomic.Int64 }

func (ids *browserIDs) Next(context.Context) (int64, error) { return ids.value.Add(1), nil }

type browserIdentityProvider struct{}

func (browserIdentityProvider) AuthorizationURL(string, string, string) string { return "" }
func (browserIdentityProvider) Exchange(context.Context, string, string, string) (oidc.Identity, error) {
	return oidc.Identity{}, errors.New("interactive login is disabled in the browser fixture")
}
func (browserIdentityProvider) VerifyBearer(_ context.Context, token string) (oidc.Identity, error) {
	if token != browserToken {
		return oidc.Identity{}, errors.New("unknown browser fixture bearer")
	}
	return oidc.Identity{Key: access.IdentityKey{Issuer: browserIssuer, Subject: "task03-browser-admin"}}, nil
}

type browserURLValidator struct{}

func (browserURLValidator) Validate(_ context.Context, raw string) (*url.URL, error) {
	parsed, err := url.ParseRequestURI(raw)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil {
		return nil, errors.New("browser fixture accepts only public-shaped HTTPS URLs")
	}
	return parsed, nil
}

// TestTask03E2EBackend is a deliberately long-lived test process used only by
// Playwright. It composes the production middleware, generated HTTP routes,
// migrations, and PostgreSQL stores; Playwright terminates the process after
// the browser suite settles.
func TestTask03E2EBackend(t *testing.T) {
	ctx := context.Background()
	baseURL := os.Getenv("OPI_INTEGRATION_DATABASE_URL")
	if baseURL == "" {
		t.Fatal("OPI_INTEGRATION_DATABASE_URL is required")
	}
	databaseURL := resetBrowserDatabase(t, ctx, baseURL)
	runBrowserMigrations(t, databaseURL)

	pool, err := database.Open(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open browser database: %v", err)
	}
	defer pool.Close()
	ids := &browserIDs{}
	ids.value.Store(8_200_000_000_000_000_000)
	seedBrowserAuthority(t, ctx, pool, ids)

	cursors, err := access.NewCursorCodec(bytes.Repeat([]byte{0x73}, 32))
	if err != nil {
		t.Fatal(err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	accessHandler, err := accessapi.New(
		accessstore.New(pool, ids), browserIdentityProvider{}, cursors, logger,
		accessapi.Config{PublicBaseURL: "http://127.0.0.1:8100", IssuerURL: browserIssuer, SessionTTL: time.Hour},
	)
	if err != nil {
		t.Fatalf("construct browser access handler: %v", err)
	}
	projects, err := projectapi.NewWithURLValidator(
		projectstore.New(pool, ids), cursors, logger, browserURLValidator{},
	)
	if err != nil {
		t.Fatalf("construct browser project handler: %v", err)
	}

	accessRoutes := accessapi.Routes(accessHandler)
	projectRoutes := accessHandler.Middleware(projectapi.Routes(projects))
	mux := http.NewServeMux()
	for _, prefix := range []string{"/api/v1/session", "/api/v1/session/"} {
		mux.Handle(prefix, accessRoutes)
	}
	for _, prefix := range []string{"/api/v1/portfolio", "/api/v1/projects", "/api/v1/projects/", "/api/v1/jobs/"} {
		mux.Handle(prefix, projectRoutes)
	}
	server := &http.Server{Addr: "127.0.0.1:8100", Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		t.Fatalf("serve browser API: %v", err)
	}
}

func resetBrowserDatabase(t *testing.T, ctx context.Context, baseURL string) string {
	t.Helper()
	parsed, err := url.Parse(baseURL)
	if err != nil {
		t.Fatalf("parse integration database URL: %v", err)
	}
	connection, err := pgx.Connect(ctx, baseURL)
	if err != nil {
		t.Fatalf("connect integration server: %v", err)
	}
	defer connection.Close(ctx)
	identifier := pgx.Identifier{browserDatabase}.Sanitize()
	if _, err := connection.Exec(ctx, "DROP DATABASE IF EXISTS "+identifier+" WITH (FORCE)"); err != nil {
		t.Fatalf("drop prior browser database: %v", err)
	}
	if _, err := connection.Exec(ctx, "CREATE DATABASE "+identifier); err != nil {
		t.Fatalf("create browser database: %v", err)
	}
	parsed.Path = "/" + browserDatabase
	return parsed.String()
}

func runBrowserMigrations(t *testing.T, databaseURL string) {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve browser fixture repository root")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
	command := exec.Command(filepath.Join(root, "scripts", "migrate.sh"), "up")
	command.Dir = root
	command.Env = append(os.Environ(), "DATABASE_URL="+databaseURL)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("migrate browser database: %v\n%s", err, output)
	}
}

func seedBrowserAuthority(t *testing.T, ctx context.Context, pool *database.Pool, ids *browserIDs) {
	t.Helper()
	if _, err := pool.Unwrap().Exec(ctx,
		`INSERT INTO workspaces(id,name) VALUES($1,'Task 3 browser workspace')`, browserWorkspaceID); err != nil {
		t.Fatalf("seed browser workspace: %v", err)
	}
	identityID, err := ids.Next(ctx)
	if err != nil {
		t.Fatal(err)
	}
	membershipID, err := ids.Next(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Unwrap().Exec(ctx, `INSERT INTO external_identities
		(id,issuer,subject,display_name,email) VALUES($1,$2,$3,$4,$5)`, identityID,
		browserIssuer, "task03-browser-admin", "Browser Admin", "browser-admin@example.test"); err != nil {
		t.Fatalf("seed browser identity: %v", err)
	}
	if _, err := pool.Unwrap().Exec(ctx, `INSERT INTO memberships
		(id,workspace_id,identity_id,role,status) VALUES($1,$2,$3,'admin','active')`,
		membershipID, browserWorkspaceID, identityID); err != nil {
		t.Fatalf("seed browser membership: %v", err)
	}
}
