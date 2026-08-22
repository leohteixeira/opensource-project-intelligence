package adoption_test

import (
	"errors"
	"testing"
	"time"

	"github.com/leohteixeira/opensource-project-intelligence/internal/access"
	"github.com/leohteixeira/opensource-project-intelligence/internal/adoption"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/sourceadapter"
)

func adoptionSnapshot(id int64, registry, unit, population string, observed time.Time) adoption.Snapshot {
	value := float64(id)
	return adoption.Snapshot{ID: id, ProjectID: 10, SourceID: 20, Registry: registry,
		Package: "canonical-package", Unit: unit, Population: population, Value: &value,
		WindowFrom: observed.Add(-24 * time.Hour), WindowTo: observed,
		ObservedAt: observed, EvidenceID: id + 100}
}

func TestUT099IncomparableRegistryUnitsCannotBecomeAUniversalRank(t *testing.T) {
	now := time.Date(2026, 8, 22, 10, 0, 0, 0, time.UTC)
	values := []adoption.Snapshot{
		adoptionSnapshot(1, "npm", "weekly_downloads", "npm_public", now),
		adoptionSnapshot(2, "docker", "pulls", "docker_hub", now),
	}
	if status := adoption.ComparisonStatus(values); status != adoption.StatusIncomparable {
		t.Fatalf("status = %q, want incomparable", status)
	}
}

func TestUT100MissingRegistryAndAdvisoryEvidenceIsUnknown(t *testing.T) {
	if status := adoption.ComparisonStatus(nil); status != adoption.StatusUnknown {
		t.Fatalf("registry status = %q", status)
	}
	security := adoption.SummarizeSecurity(false, false, nil)
	if security.Status != adoption.StatusUnknown || security.CoverageNote == "" {
		t.Fatalf("security = %#v", security)
	}
}

func TestUT101LargeHistoriesRemainWindowedAndPageable(t *testing.T) {
	cutoff := time.Date(2026, 8, 22, 10, 0, 0, 0, time.UTC)
	values := make([]adoption.Snapshot, 250)
	for index := range values {
		values[index] = adoptionSnapshot(int64(index+1), "npm", "weekly_downloads", "npm_public",
			cutoff.Add(-time.Duration(index)*time.Hour))
	}
	merged, err := adoption.MergeHistory(values[:125], values[125:])
	if err != nil {
		t.Fatal(err)
	}
	const pageSize = 50
	if len(merged) != 250 || len(merged[:pageSize]) != pageSize ||
		merged[0].ObservedAt.Before(merged[pageSize].ObservedAt) {
		t.Fatalf("history is not stably page-ready: %d", len(merged))
	}
}

func TestUT102ProtectedIntelligenceRequiresApprovedMembership(t *testing.T) {
	for _, principal := range []access.Principal{
		{},
		{ActorID: 1, Role: access.RoleViewer, Status: access.StatusPending},
		{ActorID: 1, Role: access.RoleViewer, Status: access.StatusSuspended},
	} {
		if err := access.Authorize(principal, access.ActionIntelligenceRead); err == nil {
			t.Fatalf("principal unexpectedly authorized: %#v", principal)
		}
	}
	approved := access.Principal{ActorID: 1, Kind: access.ActorMember, Role: access.RoleViewer,
		Status: access.StatusActive}
	if err := access.Authorize(approved, access.ActionIntelligenceRead); err != nil {
		t.Fatalf("approved viewer denied: %v", err)
	}
}

func TestUT103DuplicateRegistrySampleIsRejected(t *testing.T) {
	now := time.Date(2026, 8, 22, 10, 0, 0, 0, time.UTC)
	value := adoptionSnapshot(1, "npm", "weekly_downloads", "npm_public", now)
	if _, err := adoption.MergeHistory([]adoption.Snapshot{value}, []adoption.Snapshot{value}); !errors.Is(err, adoption.ErrInvalid) {
		t.Fatalf("error = %v, want invalid duplicate", err)
	}
}

func TestUT104NormalizationRequiresPopulationContext(t *testing.T) {
	_, err := sourceadapter.Registry(sourceadapter.RegistryValue{Package: "example",
		URL: "https://registry.example/example", Unit: "downloads", Value: 12,
		ObservedAt: time.Now(), EvidenceID: 1})
	if !errors.Is(err, sourceadapter.ErrMalformed) {
		t.Fatalf("error = %v, want malformed", err)
	}
}

func TestUT105WithdrawnAdvisoryRetainsProvenanceAndState(t *testing.T) {
	now := time.Date(2026, 8, 22, 10, 0, 0, 0, time.UTC)
	withdrawn := now.Add(time.Hour)
	item := adoption.Advisory{ID: 1, ProjectID: 2, SourceID: 3, ExternalID: "ADV-1",
		State: adoption.AdvisoryWithdrawn, PublishedAt: now, WithdrawnAt: &withdrawn, EvidenceID: 4}
	result := adoption.SummarizeSecurity(true, true, []adoption.Advisory{item})
	if len(result.Advisories) != 1 || result.Advisories[0].State != adoption.AdvisoryWithdrawn ||
		result.Advisories[0].EvidenceID != 4 || result.Advisories[0].WithdrawnAt == nil {
		t.Fatalf("withdrawn evidence changed: %#v", result)
	}
}
