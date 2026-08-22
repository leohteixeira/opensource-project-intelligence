package export_test

import (
	"errors"
	"testing"
	"time"

	exportartifact "github.com/leohteixeira/opensource-project-intelligence/internal/export"
)

// UT-260: export authorization expires at the exact frozen boundary.
func TestUT260ExportExpiresAtTheFrozen24HourBoundary(t *testing.T) {
	t.Parallel()
	completed := time.Date(2026, 8, 22, 7, 0, 0, 0, time.UTC)
	artifact, err := exportartifact.NewArtifact(42, "projects/42/exports/report.csv", "text/csv",
		[]byte("project,value\n42,1\n"), completed)
	if err != nil {
		t.Fatal(err)
	}
	if err := artifact.Authorize(42, completed.Add(exportartifact.Lifetime-time.Nanosecond)); err != nil {
		t.Fatalf("download immediately before expiry failed: %v", err)
	}
	for _, at := range []time.Time{completed.Add(exportartifact.Lifetime), completed.Add(25 * time.Hour)} {
		if !errors.Is(artifact.Authorize(42, at), exportartifact.ErrGone) {
			t.Fatalf("download at %s did not return gone", at)
		}
	}
}
