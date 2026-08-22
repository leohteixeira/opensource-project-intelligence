package evidence_test

import (
	"errors"
	"testing"

	"github.com/leohteixeira/opensource-project-intelligence/internal/evidence"
)

func TestEvidenceChecksumAndPurgeResume(t *testing.T) {
	t.Parallel()
	object, err := evidence.NewObject(1, 2, "application/json", []byte(`{"public":true}`))
	if err != nil {
		t.Fatal(err)
	}
	if err := object.Verify([]byte("corrupt")); !errors.Is(err, evidence.ErrCorrupt) {
		t.Fatalf("UT-259 got %v", err)
	}
	manifest := evidence.PurgeManifest{ProjectID: 2, Keys: []string{"a", "b"}}
	manifest, err = manifest.MarkDeleted("a")
	if err != nil || len(manifest.Remaining()) != 1 || manifest.Remaining()[0] != "b" {
		t.Fatalf("UT-268 got %#v, %v", manifest, err)
	}
}
