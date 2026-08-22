package recovery_test

import (
	"slices"
	"testing"

	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/recovery"
)

func TestReconcileObjectManifest(t *testing.T) {
	t.Parallel()

	result := recovery.Reconcile(
		[]recovery.Object{
			{Key: "projects/1/sha256/aaa", SHA256: "aaa"},
			{Key: "projects/1/sha256/bbb", SHA256: "bbb"},
		},
		map[string]string{
			"projects/1/sha256/aaa": "changed",
			"projects/2/sha256/ccc": "ccc",
		},
	)

	if !slices.Equal(result.Missing, []string{"projects/1/sha256/bbb"}) {
		t.Errorf("Missing = %v", result.Missing)
	}
	if !slices.Equal(result.Mismatched, []string{"projects/1/sha256/aaa"}) {
		t.Errorf("Mismatched = %v", result.Mismatched)
	}
	if !slices.Equal(result.Orphaned, []string{"projects/2/sha256/ccc"}) {
		t.Errorf("Orphaned = %v", result.Orphaned)
	}
}
