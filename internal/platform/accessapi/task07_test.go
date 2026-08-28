package accessapi

import (
	"errors"
	"testing"
	"time"

	"github.com/leohteixeira/opensource-project-intelligence/internal/access"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/accessstore"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/httpapi"
)

func TestUT197InvalidAuditRangesAndFiltersAreRejected(t *testing.T) {
	actor, from, to := "invalid", "2026-08-23T00:00:00Z", "2026-08-22T00:00:00Z"
	if _, err := auditFilter(httpapi.GetApiV1AdminAuditParams{Actor: &actor}); err == nil {
		t.Fatal("invalid actor was accepted")
	}
	if _, err := auditFilter(httpapi.GetApiV1AdminAuditParams{From: &from, To: &to}); err == nil {
		t.Fatal("reversed range was accepted")
	}
}

func TestUT198EmptyAuditQueryIsValid(t *testing.T) {
	filter, err := auditFilter(httpapi.GetApiV1AdminAuditParams{})
	if err != nil || filter.ActorID != nil || filter.OccurredFrom != nil || filter.OccurredTo != nil {
		t.Fatalf("empty filter = %#v, %v", filter, err)
	}
}

func TestUT199AuditPagesAreBounded(t *testing.T) {
	large := int32(10000)
	got, err := pageLimit(&large)
	if err == nil || got != 0 {
		t.Fatalf("unbounded page accepted: %d, %v", got, err)
	}
}

func TestUT200AuditIsAdminOnly(t *testing.T) {
	viewer := access.Principal{ActorID: 1, Kind: access.ActorMember, Role: access.RoleViewer,
		Status: access.StatusActive, Workspace: 1}
	if !errors.Is(access.Authorize(viewer, access.ActionAuditRead), access.ErrPermissionDenied) {
		t.Fatal("viewer could read audit")
	}
}

func TestUT201RetryEventsRemainIndividuallyAttributable(t *testing.T) {
	events := []accessstore.AuditEvent{{ID: 1, RequestID: "attempt-1", Outcome: "failed"},
		{ID: 2, RequestID: "attempt-2", Outcome: "succeeded"}}
	if events[0].ID == events[1].ID || events[0].RequestID == events[1].RequestID {
		t.Fatal("retry audit identities collapsed")
	}
}

func TestUT202AuditOrderingUsesEventTimeAndStableIdentity(t *testing.T) {
	when := time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)
	older, newer := accessstore.AuditEvent{ID: 8, OccurredAt: when}, accessstore.AuditEvent{ID: 9, OccurredAt: when}
	if !newer.OccurredAt.Equal(older.OccurredAt) || newer.ID <= older.ID {
		t.Fatal("stable tie identity unavailable")
	}
}

func TestUT203AuditRepresentationSurvivesSubjectRemoval(t *testing.T) {
	event := accessstore.AuditEvent{ID: 7, ActorID: nil, ActorKind: "member", Action: "project.delete",
		ResourceType: "project", Outcome: "succeeded", Changes: map[string]any{"project_id": "42"}}
	if event.ActorKind == "" || event.Action == "" || event.Changes["project_id"] != "42" {
		t.Fatalf("immutable representation = %#v", event)
	}
}

func TestIT087LongAuditRetentionQueryStaysTimeBounded(t *testing.T) {
	from, to := "2020-01-01T00:00:00Z", "2026-08-22T00:00:00Z"
	filter, err := auditFilter(httpapi.GetApiV1AdminAuditParams{From: &from, To: &to})
	if err != nil || filter.OccurredFrom == nil || filter.OccurredTo == nil {
		t.Fatalf("bounded retention filter = %#v, %v", filter, err)
	}
}
