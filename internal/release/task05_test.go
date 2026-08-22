package release

import (
	"errors"
	"testing"
	"time"

	"github.com/leohteixeira/opensource-project-intelligence/internal/analysis"
)

func releaseFixture(id int64) Intelligence {
	return Intelligence{ID: id, ProjectID: 2, RepositoryID: 3, SourceID: 4,
		ExternalID: "release-" + time.Unix(id, 0).UTC().Format(time.RFC3339Nano), Tag: "v1.2.3",
		Title: "Stable release", URL: "https://example.test/releases/v1.2.3",
		PublishedAt: time.Date(2026, 8, 22, 10, 0, 0, 0, time.UTC), EvidenceID: id + 10}
}

func TestUT149ReleaseWithoutChangelogShowsLimitedEvidence(t *testing.T) {
	value, err := NewIntelligence(releaseFixture(1))
	if err != nil {
		t.Fatal(err)
	}
	view := Present(value, nil, true)
	if view.CoverageNote != "release metadata available; no changelog evidence" || view.Analysis != nil {
		t.Fatalf("release invented evidence: %#v", view)
	}
}

func TestUT154WithdrawnReleaseRetainsSourceStatusAndAnalysis(t *testing.T) {
	value := releaseFixture(1)
	withdrawn := value.PublishedAt.Add(time.Hour)
	value.State, value.WithdrawnAt = StateWithdrawn, &withdrawn
	value, err := NewIntelligence(value)
	if err != nil {
		t.Fatal(err)
	}
	run := analysis.Run{ID: 9, State: analysis.StateSucceeded}
	view := Present(value, &run, true)
	if view.Release.State != StateWithdrawn || view.Release.WithdrawnAt == nil ||
		view.Analysis == nil || view.Analysis.State != analysis.StateSucceeded {
		t.Fatalf("withdrawn history changed: %#v", view)
	}
}

func TestReleaseHistoryRejectsDuplicatesAndSortsNewestFirst(t *testing.T) {
	old := releaseFixture(1)
	newer := releaseFixture(2)
	newer.Tag, newer.ExternalID, newer.URL = "v2", "release-2", "https://example.test/releases/v2"
	newer.PublishedAt = old.PublishedAt.Add(time.Hour)
	values, err := MergeHistory([]Intelligence{old}, []Intelligence{newer})
	if err != nil || len(values) != 2 || values[0].ID != newer.ID {
		t.Fatalf("history = %#v, err=%v", values, err)
	}
	if _, err := MergeHistory([]Intelligence{old}, []Intelligence{old}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("duplicate error = %v", err)
	}
}
