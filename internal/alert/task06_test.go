package alert_test

import (
	"errors"
	"testing"
	"time"

	"github.com/leohteixeira/opensource-project-intelligence/internal/access"
	"github.com/leohteixeira/opensource-project-intelligence/internal/alert"
)

var now = time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)

func rule() alert.Rule {
	return alert.Rule{ID: 1, Version: 1, Name: "Low release cadence", Signal: "metric.release_frequency",
		Operator: alert.LessThan, Threshold: 2, Scope: "project", ProjectID: 9,
		Severity: alert.SeverityWarning, Cooldown: 24 * time.Hour, Deduplication: 7 * 24 * time.Hour,
		Enabled: true, UpdatedAt: now}
}

func fact() alert.Fact {
	value := 1.0
	return alert.Fact{ProjectID: 9, Signal: "metric.release_frequency", Version: "v1", Value: &value,
		EvidenceID: 77, WindowFrom: now.Add(-30 * 24 * time.Hour), WindowTo: now, DetectedAt: now, Complete: true}
}

func principal(role access.Role, actor int64) access.Principal {
	return access.Principal{ActorID: actor, Workspace: 1, Status: access.StatusActive, Role: role, Kind: access.ActorMember}
}

func TestUT183InvalidRulesRejected(t *testing.T) {
	values := []alert.Rule{rule(), rule(), rule()}
	values[0].Signal = "arbitrary"
	values[1].Threshold = 0
	values[1].Operator = "arbitrary"
	values[2].Cooldown = -time.Second
	for index, value := range values {
		if err := value.Validate(); err == nil {
			t.Fatalf("invalid rule %d accepted", index)
		}
	}
}

func TestUT184MissingEvidenceCannotTrigger(t *testing.T) {
	value := fact()
	value.Complete, value.EvidenceID = false, 0
	result, err := alert.Evaluate(rule(), value, nil)
	if err != nil || result != nil {
		t.Fatalf("result=%#v err=%v", result, err)
	}
}

func TestUT186ViewerOnlyChangesPersonalReadState(t *testing.T) {
	viewer := principal(access.RoleViewer, 10)
	if _, err := alert.MarkRead(viewer, 1, now); err != nil {
		t.Fatal(err)
	}
	occurrence, _ := alert.Evaluate(rule(), fact(), nil)
	if _, err := alert.Transition(viewer, *occurrence, alert.StateAcknowledged, "investigating", 1); !errors.Is(err, access.ErrPermissionDenied) {
		t.Fatalf("viewer transitioned shared state: %v", err)
	}
}

func TestUT185WorkspaceAlertVolumeIsBounded(t *testing.T) {
	if err := alert.DefaultLimits.ValidateVolume(500, 10_000); err != nil {
		t.Fatal(err)
	}
	if err := alert.DefaultLimits.ValidateVolume(501, 1); err == nil {
		t.Fatal("rule quota was not enforced")
	}
	if err := alert.DefaultLimits.ValidateVolume(1, 10_001); err == nil {
		t.Fatal("occurrence quota was not enforced")
	}
}

func TestUT187AndUT267RedeliveryDeduplicatesAndReadStateIsSeparate(t *testing.T) {
	first, err := alert.Evaluate(rule(), fact(), nil)
	if err != nil {
		t.Fatal(err)
	}
	replayed := fact()
	replayed.DetectedAt = now.Add(time.Hour)
	second, err := alert.Evaluate(rule(), replayed, first)
	member, readErr := alert.MarkRead(principal(access.RoleViewer, 10), 99, now.Add(2*time.Hour))
	if err != nil || readErr != nil || second.SuppressionCount != 1 || second.State != alert.StateOpen || member.MemberID != 10 {
		t.Fatalf("second=%#v member=%#v err=%v readErr=%v", second, member, err, readErr)
	}
}

func TestUT188LifecycleOrderingAndExplicitReopen(t *testing.T) {
	occurrence, _ := alert.Evaluate(rule(), fact(), nil)
	analyst := principal(access.RoleAnalyst, 11)
	resolved, err := alert.Transition(analyst, *occurrence, alert.StateResolved, "fixed", 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := alert.Transition(analyst, resolved, alert.StateAcknowledged, "invalid ordering", 2); err == nil {
		t.Fatal("resolved occurrence acknowledged")
	}
	if _, err := alert.Transition(analyst, resolved, alert.StateOpen, "condition recurred", 2); err != nil {
		t.Fatal(err)
	}
}

func TestUT189ArchivedProjectsDoNotCreateNewAlerts(t *testing.T) {
	value := fact()
	value.Archived = true
	if result, err := alert.Evaluate(rule(), value, nil); err != nil || result != nil {
		t.Fatalf("result=%#v err=%v", result, err)
	}
}
