//go:build integration

package database_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pgvector/pgvector-go"

	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/database"
	dbgen "github.com/leohteixeira/opensource-project-intelligence/internal/platform/database/sqlc"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/id"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/recovery"
)

func TestPlatformDatabaseIntegrationContract(t *testing.T) {
	ctx := context.Background()
	baseURL := requireIntegrationDatabaseURL(t)
	root := repositoryRoot(t)
	sourceDatabase := createDatabase(t, ctx, baseURL, "foundation")

	t.Run("IT-097 migration round trip", func(t *testing.T) {
		runMigration(t, root, sourceDatabase, "up")

		connection, err := pgx.Connect(ctx, sourceDatabase)
		if err != nil {
			t.Fatalf("connect to migrated database: %v", err)
		}
		defer connection.Close(ctx)

		var vectorVersion string
		if err := connection.QueryRow(ctx,
			"SELECT extversion FROM pg_extension WHERE extname = 'vector'",
		).Scan(&vectorVersion); err != nil {
			t.Fatalf("vector extension was not installed: %v", err)
		}
		if vectorVersion == "" {
			t.Fatal("vector extension has no version")
		}
		if _, err := connection.Exec(ctx,
			"INSERT INTO jobs (id, kind, state) VALUES (0, 'invalid', 'queued')",
		); err == nil {
			t.Fatal("jobs positive-identifier constraint accepted zero")
		}
		var migrationCount int
		if err := connection.QueryRow(ctx, "SELECT count(*) FROM schema_migrations").Scan(&migrationCount); err != nil {
			t.Fatalf("count applied migrations: %v", err)
		}
		if migrationCount == 0 {
			t.Fatal("migration runner applied no migrations")
		}
		connection.Close(ctx)

		for range migrationCount {
			runMigration(t, root, sourceDatabase, "down")
		}
		connection, err = pgx.Connect(ctx, sourceDatabase)
		if err != nil {
			t.Fatalf("connect after rollback: %v", err)
		}
		var jobsTable pgtype.Text
		if err := connection.QueryRow(ctx, "SELECT to_regclass('public.jobs')::text").Scan(&jobsTable); err != nil {
			t.Fatalf("check rolled-back schema: %v", err)
		}
		connection.Close(ctx)
		if jobsTable.Valid {
			t.Fatalf("jobs table remained after rollback: %q", jobsTable.String)
		}

		runMigration(t, root, sourceDatabase, "up")
		connection, err = pgx.Connect(ctx, sourceDatabase)
		if err != nil {
			t.Fatalf("connect after reapply: %v", err)
		}
		defer connection.Close(ctx)
		var applied, reapplied int
		if err := connection.QueryRow(ctx,
			"SELECT count(*) FROM schema_migrations WHERE version = '0001_platform'",
		).Scan(&applied); err != nil {
			t.Fatalf("read migration history: %v", err)
		}
		if applied != 1 {
			t.Fatalf("migration history rows = %d, want 1", applied)
		}
		if err := connection.QueryRow(ctx, "SELECT count(*) FROM schema_migrations").Scan(&reapplied); err != nil {
			t.Fatalf("count reapplied migrations: %v", err)
		}
		if reapplied != migrationCount {
			t.Fatalf("reapplied migrations = %d, want %d", reapplied, migrationCount)
		}
	})

	t.Run("IT-098 generated SQL access", func(t *testing.T) {
		pool, err := database.Open(ctx, sourceDatabase)
		if err != nil {
			t.Fatalf("open generated-query pool: %v", err)
		}
		defer pool.Close()

		queries := dbgen.New(pool.Unwrap())
		created, err := queries.CreateEvidenceVector(ctx, dbgen.CreateEvidenceVectorParams{
			ID:                734_003_200_000_000_001,
			ObjectReferenceID: pgtype.Int8{Valid: false},
			Embedding:         pgvector.NewVector([]float32{0.25, -0.5, 1}),
			Model:             "integration-3d",
		})
		if err != nil {
			t.Fatalf("CreateEvidenceVector() error = %v", err)
		}
		read, err := queries.GetEvidenceVector(ctx, created.ID)
		if err != nil {
			t.Fatalf("GetEvidenceVector() error = %v", err)
		}
		if read.ID != created.ID || read.ObjectReferenceID.Valid {
			t.Fatalf("generated bigint/NULL scan = %#v", read)
		}
		if got := read.Embedding.Slice(); len(got) != 3 || got[0] != 0.25 || got[1] != -0.5 || got[2] != 1 {
			t.Fatalf("generated vector scan = %v", got)
		}
	})

	t.Run("IT-099 Snowflake lease failover", func(t *testing.T) {
		pool, err := database.Open(ctx, sourceDatabase)
		if err != nil {
			t.Fatalf("open lease pool: %v", err)
		}
		defer pool.Close()
		leaser := database.NewSnowflakeLeaser(pool)
		now := time.Date(2026, 8, 21, 20, 0, 0, 0, time.UTC)
		const ttl = 50 * time.Millisecond

		first, err := id.New(ctx, leaser, id.Config{
			Holder: "api-before-restart", LeaseTTL: ttl, Now: func() time.Time { return now },
		})
		if err != nil {
			t.Fatalf("create first generator: %v", err)
		}
		firstID, err := first.Next(ctx)
		if err != nil {
			t.Fatalf("first Next() error = %v", err)
		}

		now = now.Add(ttl + time.Millisecond)
		if _, err := first.Next(ctx); !errors.Is(err, id.ErrLeaseLost) {
			t.Fatalf("expired holder Next() error = %v, want ErrLeaseLost", err)
		}
		second, err := id.New(ctx, leaser, id.Config{
			Holder: "api-after-restart", LeaseTTL: ttl, Now: func() time.Time { return now },
		})
		if err != nil {
			t.Fatalf("create failover generator: %v", err)
		}
		secondID, err := second.Next(ctx)
		if err != nil {
			t.Fatalf("failover Next() error = %v", err)
		}
		const sequenceBits = 12
		if firstID>>sequenceBits&1023 != secondID>>sequenceBits&1023 {
			t.Fatalf("node IDs differ before/after failover: %d, %d", firstID, secondID)
		}
		if secondID <= firstID {
			t.Fatalf("failover ID %d is not after %d", secondID, firstID)
		}
	})

	t.Run("IT-138 backup restore", func(t *testing.T) {
		connection, err := pgx.Connect(ctx, sourceDatabase)
		if err != nil {
			t.Fatalf("connect to backup source: %v", err)
		}
		evidence := []byte("reviewed source evidence")
		digest := sha256.Sum256(evidence)
		const objectKey = "projects/734003200000000001/evidence/source.json"
		statements := []struct {
			sql  string
			args []any
		}{
			{"INSERT INTO object_references (id, object_key, sha256, size_bytes, media_type) VALUES ($1, $2, $3, $4, $5)", []any{int64(734_003_200_000_000_002), objectKey, digest[:], len(evidence), "application/json"}},
			{"INSERT INTO jobs (id, kind, state, checkpoint) VALUES ($1, $2, $3, $4)", []any{int64(734_003_200_000_000_003), "sync", "succeeded", []byte(`{"page":3}`)}},
			{"INSERT INTO audit_events (id, action, resource_type, resource_id, details) VALUES ($1, $2, $3, $4, $5)", []any{int64(734_003_200_000_000_004), "sync.completed", "project", int64(734_003_200_000_000_001), []byte(`{"coverage":"complete"}`)}},
		}
		for _, statement := range statements {
			if _, err := connection.Exec(ctx, statement.sql, statement.args...); err != nil {
				connection.Close(ctx)
				t.Fatalf("seed backup source: %v", err)
			}
		}
		connection.Close(ctx)

		container := postgresContainer(t, root)
		backupDirectory := t.TempDir()
		runRecoveryScript(t, root, "backup.sh", sourceDatabase, backupDirectory, container)

		restoreDatabase := createDatabase(t, ctx, baseURL, "restore")
		runRecoveryScript(t, root, "restore.sh", restoreDatabase, backupDirectory, container)
		restored, err := pgx.Connect(ctx, restoreDatabase)
		if err != nil {
			t.Fatalf("connect to restored database: %v", err)
		}
		defer restored.Close(ctx)

		var restoredDigest []byte
		var jobs, auditEvents int
		if err := restored.QueryRow(ctx,
			"SELECT sha256 FROM object_references WHERE object_key = $1", objectKey,
		).Scan(&restoredDigest); err != nil {
			t.Fatalf("read restored object reference: %v", err)
		}
		if err := restored.QueryRow(ctx, "SELECT count(*) FROM jobs WHERE state = 'succeeded'").Scan(&jobs); err != nil {
			t.Fatalf("read restored jobs: %v", err)
		}
		if err := restored.QueryRow(ctx, "SELECT count(*) FROM audit_events").Scan(&auditEvents); err != nil {
			t.Fatalf("read restored audit history: %v", err)
		}
		if jobs != 1 || auditEvents != 1 || !bytes.Equal(restoredDigest, digest[:]) {
			t.Fatalf("restored state: jobs=%d audit=%d digest=%x", jobs, auditEvents, restoredDigest)
		}

		reconciliation := recovery.Reconcile(
			[]recovery.Object{{Key: objectKey, SHA256: hex.EncodeToString(restoredDigest)}},
			map[string]string{objectKey: hex.EncodeToString(digest[:])},
		)
		if !reconciliation.Clean() {
			t.Fatalf("restored evidence did not reconcile: %#v", reconciliation)
		}
	})
}

func TestIT136GeneratedContractDrift(t *testing.T) {
	root := repositoryRoot(t)
	temporaryRoot := t.TempDir()
	manifest, err := os.ReadFile(filepath.Join(root, "generated.sha256"))
	if err != nil {
		t.Fatalf("read generated manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(temporaryRoot, "generated.sha256"), manifest, 0o600); err != nil {
		t.Fatalf("write temporary generated manifest: %v", err)
	}
	for _, line := range strings.Split(strings.TrimSpace(string(manifest)), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			t.Fatalf("invalid generated manifest line %q", line)
		}
		source := filepath.Join(root, fields[1])
		destination := filepath.Join(temporaryRoot, fields[1])
		if err := os.MkdirAll(filepath.Dir(destination), 0o750); err != nil {
			t.Fatalf("create temporary generated directory: %v", err)
		}
		contents, err := os.ReadFile(source)
		if err != nil {
			t.Fatalf("read generated file %s: %v", fields[1], err)
		}
		if err := os.WriteFile(destination, contents, 0o600); err != nil {
			t.Fatalf("copy generated file %s: %v", fields[1], err)
		}
	}

	check := exec.Command(filepath.Join(root, "scripts/check-generated.sh"))
	check.Env = append(os.Environ(), "REPOSITORY_ROOT="+temporaryRoot)
	if output, err := check.CombinedOutput(); err != nil {
		t.Fatalf("clean generated check failed: %v\n%s", err, output)
	}
	contract := filepath.Join(temporaryRoot, "api/openapi.yaml")
	file, err := os.OpenFile(contract, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatalf("open temporary contract: %v", err)
	}
	if _, err := file.WriteString("\n# intentional integration-test drift\n"); err != nil {
		file.Close()
		t.Fatalf("modify temporary contract: %v", err)
	}
	file.Close()
	check = exec.Command(filepath.Join(root, "scripts/check-generated.sh"))
	check.Env = append(os.Environ(), "REPOSITORY_ROOT="+temporaryRoot)
	if output, err := check.CombinedOutput(); err == nil {
		t.Fatalf("drift check passed after source modification:\n%s", output)
	}
}

func requireIntegrationDatabaseURL(t *testing.T) string {
	t.Helper()
	value := os.Getenv("OPI_INTEGRATION_DATABASE_URL")
	if value == "" {
		t.Fatal("OPI_INTEGRATION_DATABASE_URL is required for integration tests")
	}
	return value
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve integration test path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", ".."))
}

func createDatabase(t *testing.T, ctx context.Context, baseURL, suffix string) string {
	t.Helper()
	name := fmt.Sprintf("opi_it_%s_%d", suffix, time.Now().UnixNano())
	connection, err := pgx.Connect(ctx, baseURL)
	if err != nil {
		t.Fatalf("connect to integration PostgreSQL: %v", err)
	}
	if _, err := connection.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{name}.Sanitize()); err != nil {
		connection.Close(ctx)
		t.Fatalf("create integration database: %v", err)
	}
	connection.Close(ctx)
	t.Cleanup(func() {
		cleanup, cleanupErr := pgx.Connect(context.Background(), baseURL)
		if cleanupErr != nil {
			t.Errorf("connect to drop integration database: %v", cleanupErr)
			return
		}
		defer cleanup.Close(context.Background())
		if _, cleanupErr = cleanup.Exec(context.Background(),
			"DROP DATABASE "+pgx.Identifier{name}.Sanitize()+" WITH (FORCE)",
		); cleanupErr != nil {
			t.Errorf("drop integration database: %v", cleanupErr)
		}
	})

	parsed, err := url.Parse(baseURL)
	if err != nil {
		t.Fatalf("parse integration database URL: %v", err)
	}
	parsed.Path = "/" + name
	return parsed.String()
}

func databaseName(t *testing.T, databaseURL string) string {
	t.Helper()
	parsed, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatalf("parse database URL: %v", err)
	}
	return strings.TrimPrefix(parsed.Path, "/")
}

func runMigration(t *testing.T, root, databaseURL, direction string) {
	t.Helper()
	command := exec.Command(filepath.Join(root, "scripts/migrate.sh"), direction)
	command.Dir = root
	command.Env = append(os.Environ(), "DATABASE_URL="+databaseURL)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("migrate %s: %v\n%s", direction, err, output)
	}
}

func postgresContainer(t *testing.T, root string) string {
	t.Helper()
	command := exec.Command("docker", "compose", "ps", "-q", "postgres")
	command.Dir = root
	output, err := command.Output()
	if err != nil {
		t.Fatalf("resolve Compose PostgreSQL container: %v", err)
	}
	container := strings.TrimSpace(string(output))
	if container == "" {
		t.Fatal("Compose PostgreSQL container is not running")
	}
	return container
}

func runRecoveryScript(t *testing.T, root, script, databaseURL, backupDirectory, container string) {
	t.Helper()
	command := exec.Command(filepath.Join(root, "scripts", script))
	command.Dir = root
	command.Env = append(os.Environ(),
		"DATABASE_URL="+databaseURL,
		"BACKUP_DIRECTORY="+backupDirectory,
		"POSTGRES_CONTAINER="+container,
		"POSTGRES_DATABASE="+databaseName(t, databaseURL),
		"POSTGRES_USER=opensource",
	)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("run %s: %v\n%s", script, err, output)
	}
}
