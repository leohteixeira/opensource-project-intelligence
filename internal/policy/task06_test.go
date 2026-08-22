package policy_test

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/leohteixeira/opensource-project-intelligence/internal/access"
	"github.com/leohteixeira/opensource-project-intelligence/internal/metric"
	"github.com/leohteixeira/opensource-project-intelligence/internal/policy"
)

var cutoff = time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)

func version(state policy.State) policy.Version {
	return policy.Version{ID: 11, FamilyID: 10, Version: 1, Name: "Production adoption", Owner: "Platform council",
		State: state, Revision: 1, Rules: []policy.Rule{{MetricName: "release_frequency", MetricVersion: "v1",
			Operator: policy.GreaterThanOrEqual, Threshold: 2, Weight: 0.5, Required: true,
			RequiredEvidence: "complete release history", OnFailure: policy.Conditional, Label: "confirm release cadence"},
			{MetricName: "top_three_author_share", MetricVersion: "v1", Operator: policy.LessThanOrEqual,
				Threshold: 0.8, Weight: 0.5, Required: true, RequiredEvidence: "eligible commits",
				OnFailure: policy.NotRecommended, Label: "reduce contributor concentration"}},
		RadarMap: map[policy.Outcome]string{policy.Recommended: "adopt", policy.Conditional: "trial",
			policy.NotRecommended: "hold", policy.InsufficientData: "unplaced"}, CreatedAt: cutoff}
}

func valueWindow(t *testing.T) metric.Window {
	t.Helper()
	value, err := metric.PresetWindow("90d", cutoff)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func facts(releases, concentration float64) []policy.Fact {
	return []policy.Fact{{MetricName: "release_frequency", MetricVersion: "v1", Status: metric.StatusAvailable,
		Value: &releases, EvidenceIDs: []int64{101}, SnapshotID: 1},
		{MetricName: "top_three_author_share", MetricVersion: "v1", Status: metric.StatusAvailable,
			Value: &concentration, EvidenceIDs: []int64{102}, SnapshotID: 2}}
}

func principal(role access.Role) access.Principal {
	return access.Principal{ActorID: 1, Workspace: 1, Role: role, Status: access.StatusActive, Kind: access.ActorMember}
}

func TestUT120InactivePolicyRejected(t *testing.T) {
	if _, err := policy.Evaluate(1, version(policy.StateDraft), valueWindow(t), facts(4, .2)); err == nil {
		t.Fatal("draft policy evaluated")
	}
}

func TestUT121MissingRequiredEvidenceIsInsufficient(t *testing.T) {
	result, err := policy.Evaluate(1, version(policy.StateActive), valueWindow(t), facts(4, .2)[:1])
	if err != nil || result.Outcome != policy.InsufficientData || len(result.Missing) != 1 {
		t.Fatalf("result=%#v err=%v", result, err)
	}
}

func TestUT122DecisiveEvidenceRemainsLinked(t *testing.T) {
	result, err := policy.Evaluate(1, version(policy.StateActive), valueWindow(t), facts(1, .9))
	if err != nil || len(result.Decisive) != 2 || !reflect.DeepEqual(result.EvidenceIDs, []int64{101, 102}) {
		t.Fatalf("result=%#v err=%v", result, err)
	}
}

func TestUT123SelectionPermissions(t *testing.T) {
	if err := policy.CanSelect(principal(access.RoleViewer)); !errors.Is(err, access.ErrPermissionDenied) {
		t.Fatalf("viewer selected policy: %v", err)
	}
	if err := policy.CanSelect(principal(access.RoleAnalyst)); err != nil {
		t.Fatal(err)
	}
}

func TestUT124AndUT245PolicyDeterminism(t *testing.T) {
	first, _ := policy.Evaluate(1, version(policy.StateActive), valueWindow(t), facts(4, .2))
	second, _ := policy.Evaluate(1, version(policy.StateActive), valueWindow(t), facts(4, .2))
	if !reflect.DeepEqual(first, second) || first.Outcome != policy.Recommended {
		t.Fatalf("first=%#v second=%#v", first, second)
	}
}

func TestUT125StalePrerequisitesAreExplicit(t *testing.T) {
	values := facts(4, .2)
	values[0].Status, values[0].Value = metric.StatusStale, nil
	result, err := policy.Evaluate(1, version(policy.StateActive), valueWindow(t), values)
	if err != nil || result.Outcome != policy.InsufficientData || len(result.StaleInputs) != 1 {
		t.Fatalf("result=%#v err=%v", result, err)
	}
}

func TestUT126HistoricalAttributionIsFrozen(t *testing.T) {
	selected := version(policy.StateActive)
	result, err := policy.Evaluate(1, selected, valueWindow(t), facts(4, .2))
	selected.Version = 2
	if err != nil || result.PolicyVersion != 1 || result.PolicyOwner != "Platform council" {
		t.Fatalf("result=%#v err=%v", result, err)
	}
}

func TestUT127AndUT128InvalidRuleTrees(t *testing.T) {
	tests := []policy.Version{version(policy.StateDraft), version(policy.StateDraft), version(policy.StateDraft)}
	tests[0].Rules[0].MetricName = "arbitrary_formula"
	tests[1].Rules[0].Weight = 2
	tests[2].Rules[0].RequiredEvidence = ""
	for index, value := range tests {
		if err := value.Validate(); err == nil {
			t.Fatalf("invalid rule tree %d accepted", index)
		}
	}
}

func TestUT130GovernancePermissions(t *testing.T) {
	if err := policy.CanGovern(principal(access.RoleAnalyst)); !errors.Is(err, access.ErrPermissionDenied) {
		t.Fatalf("analyst governed policy: %v", err)
	}
	if err := policy.CanGovern(principal(access.RoleAdmin)); err != nil {
		t.Fatal(err)
	}
}

func TestUT129PolicyHistoryIsBoundedAndPaginated(t *testing.T) {
	values := make([]policy.Version, 250)
	page, more, err := policy.PageVersions(values, 50, 200)
	if err != nil || len(page) != 50 || more {
		t.Fatalf("page=%d more=%v err=%v", len(page), more, err)
	}
	if _, _, err := policy.PageVersions(values, 201, 0); err == nil {
		t.Fatal("unbounded policy page accepted")
	}
}

func TestUT131PublishedVersionCannotBeActivatedTwice(t *testing.T) {
	active, err := policy.Activate(version(policy.StateDraft), cutoff)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := policy.Activate(active, cutoff.Add(time.Minute)); err == nil {
		t.Fatal("one draft produced a second publication")
	}
}

func TestUT132AndUT133LifecycleValidationAndImmutability(t *testing.T) {
	draft := version(policy.StateDraft)
	draft.Rules[0].RequiredEvidence = ""
	if _, err := policy.Activate(draft, cutoff); err == nil {
		t.Fatal("invalid draft activated")
	}
	active, err := policy.Activate(version(policy.StateDraft), cutoff)
	if err != nil || active.State != policy.StateActive {
		t.Fatal(err)
	}
	retired, err := policy.Retire(active, cutoff.Add(time.Hour))
	if err != nil || retired.State != policy.StateRetired {
		t.Fatal(err)
	}
	if _, err := policy.Activate(retired, cutoff.Add(2*time.Hour)); err == nil {
		t.Fatal("retired version silently reactivated")
	}
}
