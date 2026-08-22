//go:build integration

package objectstore

import (
	"context"
	"errors"
	"os"
	"strconv"
	"testing"
	"time"

	exportartifact "github.com/leohteixeira/opensource-project-intelligence/internal/export"
)

func TestRealS3AtomicPromotionAndPurge(t *testing.T) {
	endpoint := os.Getenv("OPI_INTEGRATION_S3_ENDPOINT")
	if endpoint == "" {
		t.Skip("OPI_INTEGRATION_S3_ENDPOINT is required for the real S3 contract")
	}
	store, err := NewS3(Config{
		Endpoint:  endpoint,
		Bucket:    envOr("OPI_INTEGRATION_S3_BUCKET", "opensource-project-intelligence"),
		AccessKey: envOr("OPI_INTEGRATION_S3_ACCESS_KEY", "opensource"),
		SecretKey: envOr("OPI_INTEGRATION_S3_SECRET_KEY", "opensource-development-only"),
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	suffix := strconv.FormatInt(time.Now().UnixNano(), 10)
	staged, final := "integration/staged-"+suffix, "integration/projects/1/evidence-"+suffix
	t.Cleanup(func() {
		_ = store.Delete(context.Background(), staged)
		_ = store.Delete(context.Background(), final)
	})
	body := []byte(`{"source":"public","test":"IT-112"}`)
	if err := store.Stage(ctx, staged, body, "application/json"); err != nil {
		t.Fatalf("IT-112 stage immutable bytes: %v", err)
	}
	if err := store.Promote(ctx, staged, final); err != nil {
		t.Fatalf("IT-112 atomically promote bytes: %v", err)
	}
	read, err := store.Read(ctx, final)
	if err != nil || string(read) != string(body) {
		t.Fatalf("IT-112 promoted bytes = %q: %v", read, err)
	}
	if err := store.Delete(ctx, final); err != nil {
		t.Fatalf("IT-113 purge object: %v", err)
	}
	if _, err := store.Read(ctx, final); err == nil {
		t.Fatal("IT-113 purged object remained readable")
	}
}

// IT-131: real S3 export generation preserves checksum, authorization, and expiry.
func TestIT131RealS3ExportGenerationChecksumAndExpiryLifecycle(t *testing.T) {
	endpoint := os.Getenv("OPI_INTEGRATION_S3_ENDPOINT")
	if endpoint == "" {
		t.Skip("OPI_INTEGRATION_S3_ENDPOINT is required for the real S3 export contract")
	}
	store, err := NewS3(Config{
		Endpoint: endpoint, Bucket: envOr("OPI_INTEGRATION_S3_BUCKET", "opensource-project-intelligence"),
		AccessKey: envOr("OPI_INTEGRATION_S3_ACCESS_KEY", "opensource"),
		SecretKey: envOr("OPI_INTEGRATION_S3_SECRET_KEY", "opensource-development-only"),
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	completed := time.Now().UTC()
	key := "integration/projects/42/exports/" + strconv.FormatInt(completed.UnixNano(), 10) + ".csv"
	body := []byte("project_id,name\n42,Projeto público\n")
	artifact, err := exportartifact.NewArtifact(42, key, "text/csv; charset=utf-8", body, completed)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Delete(context.Background(), key) })
	staged := key + ".staged"
	if err := store.Stage(ctx, staged, body, artifact.MediaType); err != nil {
		t.Fatal(err)
	}
	if err := store.Promote(ctx, staged, key); err != nil {
		t.Fatal(err)
	}
	read, err := store.Read(ctx, key)
	if err != nil || !artifact.Verify(read) {
		t.Fatalf("generated export checksum failed: %v", err)
	}
	if err := artifact.Authorize(42, completed.Add(exportartifact.Lifetime)); !errors.Is(err, exportartifact.ErrGone) {
		t.Fatalf("expired export authorization = %v", err)
	}
	if err := store.Delete(ctx, key); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Read(ctx, key); err == nil {
		t.Fatal("expired export remained readable from object storage")
	}
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
