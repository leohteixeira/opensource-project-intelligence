package radar_test

import (
	"errors"
	"testing"
	"time"

	"github.com/leohteixeira/opensource-project-intelligence/internal/access"
	"github.com/leohteixeira/opensource-project-intelligence/internal/policy"
	"github.com/leohteixeira/opensource-project-intelligence/internal/radar"
)

var now = time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)

func evaluation(outcome policy.Outcome) policy.Evaluation {
	return policy.Evaluation{ProjectID: 1, PolicyID: 2, PolicyVersion: 3, Outcome: outcome}
}

func mapping() map[policy.Outcome]string {
	return map[policy.Outcome]string{policy.Recommended: "adopt", policy.Conditional: "trial",
		policy.NotRecommended: "hold", policy.InsufficientData: "unplaced"}
}

func principal(role access.Role) access.Principal {
	return access.Principal{ActorID: 7, Workspace: 1, Status: access.StatusActive, Role: role, Kind: access.ActorMember}
}

func TestUT134InvalidOverrideRejected(t *testing.T) {
	for _, test := range []struct {
		ring   radar.Ring
		reason string
		review time.Time
	}{
		{"unknown", "reason", now.Add(time.Hour)}, {radar.RingAssess, "", now.Add(time.Hour)},
		{radar.RingAssess, "reason", now.Add(-time.Hour)}} {
		if _, err := radar.NewOverride(principal(access.RoleAnalyst), test.ring, test.reason, "owner", test.review, now); err == nil {
			t.Fatal("invalid override accepted")
		}
	}
}

func TestUT135InsufficientDataNeedsMapping(t *testing.T) {
	values := mapping()
	delete(values, policy.InsufficientData)
	if _, err := radar.Derive(evaluation(policy.InsufficientData), values); err == nil {
		t.Fatal("implicit insufficient-data mapping accepted")
	}
}

func TestUT137ViewerCannotOverride(t *testing.T) {
	if _, err := radar.NewOverride(principal(access.RoleViewer), radar.RingAssess, "context", "owner", now.Add(time.Hour), now); !errors.Is(err, access.ErrPermissionDenied) {
		t.Fatalf("viewer override error=%v", err)
	}
}

func TestUT136LargeRadarFiltersWithoutHidingCounts(t *testing.T) {
	values := make([]radar.Placement, 0, 300)
	for index := range 300 {
		ring := radar.RingAdopt
		if index%2 == 0 {
			ring = radar.RingAssess
		}
		values = append(values, radar.Placement{ProjectID: int64(index + 1), Effective: ring})
	}
	page, counts, err := radar.FilterAndCount(values, radar.RingAssess, 20, 0)
	if err != nil || len(page) != 20 || counts[0].Count != 150 || counts[2].Count != 150 {
		t.Fatalf("page=%d counts=%#v err=%v", len(page), counts, err)
	}
}

func TestUT138RepeatedResolutionIsStable(t *testing.T) {
	override, _ := radar.NewOverride(principal(access.RoleAnalyst), radar.RingAssess, "context", "owner", now.Add(time.Hour), now)
	first, _ := radar.Resolve(1, "active", evaluation(policy.Recommended), mapping(), &override, now)
	second, _ := radar.Resolve(1, "active", evaluation(policy.Recommended), mapping(), &override, now)
	if first.Effective != second.Effective || first.Effective != radar.RingAssess {
		t.Fatalf("first=%#v second=%#v", first, second)
	}
}

func TestUT139RecommendationRequired(t *testing.T) {
	if _, err := radar.Resolve(1, "active", policy.Evaluation{}, mapping(), nil, now); err == nil {
		t.Fatal("placement without recommendation accepted")
	}
}

func TestUT140ArchivedPlacementIsHistorical(t *testing.T) {
	result, err := radar.Resolve(1, "archived", evaluation(policy.Recommended), mapping(), nil, now)
	if err != nil || !result.Historical {
		t.Fatalf("result=%#v err=%v", result, err)
	}
}

func TestUT246ExpiredOrRemovedOverrideRestoresSuggestion(t *testing.T) {
	override, _ := radar.NewOverride(principal(access.RoleAnalyst), radar.RingAssess, "context", "owner", now.Add(time.Hour), now)
	removed, err := radar.RemoveOverride(override, 1, now.Add(30*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	result, err := radar.Resolve(1, "active", evaluation(policy.Recommended), mapping(), &removed, now.Add(31*time.Minute))
	if err != nil || result.Effective != radar.RingAdopt || result.Override == nil {
		t.Fatalf("result=%#v err=%v", result, err)
	}
}
