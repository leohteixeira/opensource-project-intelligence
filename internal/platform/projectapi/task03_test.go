package projectapi

import (
	"testing"

	"github.com/leohteixeira/opensource-project-intelligence/internal/project"
)

// UT-253: every mutating project request requires a valid current version.
func TestUT253IfMatchRejectsMissingMalformedAndStaleVersions(t *testing.T) {
	t.Parallel()
	for _, value := range []string{"", `""`, `"v0"`, `"v-1"`, `"vwat"`} {
		if _, err := parseETag(value); err == nil {
			t.Fatalf("parseETag(%q) succeeded", value)
		}
	}
	version, err := parseETag(`"v7"`)
	if err != nil || version != 7 {
		t.Fatalf("parseETag(valid) = %d, %v", version, err)
	}
	value, err := project.New(1, 1, "Versioned", "versioned", project.Repository{
		ID: 2, ProjectID: 1, CanonicalURL: "https://github.com/acme/versioned", Role: project.RolePrimary,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := value.Transition(project.StatePaused, version, true); err != project.ErrVersionConflict {
		t.Fatalf("stale expected version returned %v", err)
	}
}
