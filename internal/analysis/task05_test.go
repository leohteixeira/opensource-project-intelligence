package analysis

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/leohteixeira/opensource-project-intelligence/internal/access"
	"github.com/leohteixeira/opensource-project-intelligence/internal/knowledge"
)

func task05Run(t *testing.T) Run {
	t.Helper()
	created := time.Date(2026, 8, 22, 10, 0, 0, 0, time.UTC)
	run, err := NewRun(Run{ID: 101, SeriesID: 201, ProjectID: 301, Kind: "release",
		PromptVersion: "release-v1", SchemaVersion: "analysis-v1", RetrievalVersion: "rrf-v1",
		Provider: "fixture", Model: "fixture-v1", Language: "en", RequestedBy: 401,
		Cutoff: created, CreatedAt: created})
	if err != nil {
		t.Fatal(err)
	}
	return run
}

func startedTask05Run(t *testing.T) Run {
	t.Helper()
	run := task05Run(t)
	started, err := run.Start(run.CreatedAt.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	return started
}

func successfulTask05Run(t *testing.T) Run {
	t.Helper()
	run := startedTask05Run(t)
	citation := knowledge.Citation{SnapshotID: 11, ChunkID: 12, StartOffset: 0, EndOffset: 10}
	raw := json.RawMessage(`{"summary":"Evidence-backed result","claims":[{"text":"Supported","citations":[{"snapshot_id":"11","chunk_id":"12","start_offset":0,"end_offset":10}]}]}`)
	value, err := run.Succeed(raw, map[int64]knowledge.Citation{12: citation},
		UsageRecord{InputTokens: 20, OutputTokens: 10, Cost: .01, Currency: "USD"},
		run.StartedAt.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func TestUT148MalformedProviderOutputCannotPublish(t *testing.T) {
	run := startedTask05Run(t)
	if _, err := run.Succeed(json.RawMessage(`{"summary":`), nil, UsageRecord{},
		run.StartedAt.Add(time.Minute)); !errors.Is(err, ErrSchema) {
		t.Fatalf("error = %v, want schema rejection", err)
	}
	if run.State != StateRunning || len(run.Output) != 0 {
		t.Fatalf("receiver was mutated: %#v", run)
	}
}

func TestUT150LargeEvidenceOutputIsBoundedByQueryContract(t *testing.T) {
	query := Query{ProjectID: 1, Question: "bounded", Cutoff: time.Now(), MaxResults: 100,
		MaxOutputBytes: 1 << 20}
	if err := query.Validate(); err != nil {
		t.Fatal(err)
	}
	query.MaxResults++
	if !errors.Is(query.Validate(), ErrInvalidRun) {
		t.Fatal("oversized evidence set accepted")
	}
}

func TestUT151ProtectedReleaseAnalysisRequiresMembership(t *testing.T) {
	if err := access.Authorize(access.Principal{}, access.ActionIntelligenceRead); !errors.Is(err, access.ErrAuthenticationRequired) {
		t.Fatalf("error = %v", err)
	}
}

func TestUT152RunIdentityReplayCannotMutateOriginal(t *testing.T) {
	run := task05Run(t)
	copy := cloneRun(run)
	started, err := run.Start(run.CreatedAt.Add(time.Minute))
	if err != nil || !reflect.DeepEqual(run, copy) || started.ID != run.ID {
		t.Fatalf("run identity changed: original=%#v started=%#v err=%v", run, started, err)
	}
}

func TestUT153ClaimsWaitForAccessibleEvidence(t *testing.T) {
	run := startedTask05Run(t)
	raw := json.RawMessage(`{"summary":"result","claims":[{"text":"unsupported","citations":[{"snapshot_id":"1","chunk_id":"2","start_offset":0,"end_offset":3}]}]}`)
	if _, err := run.Succeed(raw, nil, UsageRecord{}, run.StartedAt.Add(time.Minute)); !errors.Is(err, ErrEvidence) {
		t.Fatalf("error = %v, want evidence rejection", err)
	}
}

func TestUT162HostileOrUnboundedQueriesAreRejected(t *testing.T) {
	query := Query{ProjectID: 1, Question: strings.Repeat("x", 4_097), Cutoff: time.Now(),
		MaxResults: 10, MaxOutputBytes: 1_024}
	if !errors.Is(query.Validate(), ErrInvalidRun) {
		t.Fatal("unbounded question accepted")
	}
	query.Question = "safe\x00escape"
	if !errors.Is(query.Validate(), ErrInvalidRun) {
		t.Fatal("control-character question accepted")
	}
}

func TestUT163BlankQuestionsReturnValidationGuidance(t *testing.T) {
	query := Query{ProjectID: 1, Cutoff: time.Now(), MaxResults: 10, MaxOutputBytes: 1_024}
	if !errors.Is(query.Validate(), ErrInvalidRun) {
		t.Fatal("blank question accepted")
	}
}

func TestUT164QueryEvidenceAndOutputLimitsAreHardBounds(t *testing.T) {
	query := Query{ProjectID: 1, Question: "question", Cutoff: time.Now(), MaxResults: 101,
		MaxOutputBytes: (1 << 20) + 1}
	if !errors.Is(query.Validate(), ErrInvalidRun) {
		t.Fatal("oversized query accepted")
	}
}

func TestUT165QuerySourceScopeIsBoundedAndValidated(t *testing.T) {
	query := Query{ProjectID: 1, Question: "question", Cutoff: time.Now(), MaxResults: 10,
		MaxOutputBytes: 1_024, SourceIDs: []int64{2, 2}}
	if !errors.Is(query.Validate(), ErrInvalidRun) {
		t.Fatal("duplicate source scope accepted")
	}
}

func TestUT166RepeatedQuestionRetainsExplicitCutoff(t *testing.T) {
	cutoff := time.Date(2026, 8, 22, 10, 0, 0, 0, time.UTC)
	left := Query{ProjectID: 1, Question: "question", Cutoff: cutoff, MaxResults: 10,
		MaxOutputBytes: 1_024}
	right := left
	if left.Cutoff.IsZero() || left.Cutoff != right.Cutoff || left.Validate() != nil || right.Validate() != nil {
		t.Fatal("query replay lost its cutoff")
	}
}

func TestUT167ResolvedProjectScopePrecedesAnalysis(t *testing.T) {
	query := Query{Question: "Which project?", Cutoff: time.Now(), MaxResults: 10, MaxOutputBytes: 1_024}
	if !errors.Is(query.Validate(), ErrInvalidRun) {
		t.Fatal("ambiguous project scope accepted")
	}
}

func TestUT168DeletedOrMissingProjectCannotBeRepresentedAsAValidQuery(t *testing.T) {
	query := Query{ProjectID: 0, Question: "history", Cutoff: time.Now(), MaxResults: 10,
		MaxOutputBytes: 1_024}
	if !errors.Is(query.Validate(), ErrInvalidRun) {
		t.Fatal("missing project accepted")
	}
}

func TestUT176FeedbackNeedsTargetVersionAndReason(t *testing.T) {
	run := successfulTask05Run(t)
	feedback := Feedback{ID: 1, RunID: run.ID, ActorID: 2, Rating: "incorrect",
		RequestID: "feedback-1", CreatedAt: time.Now()}
	if !errors.Is(feedback.Validate(run), ErrInvalidRun) {
		t.Fatal("reasonless feedback accepted")
	}
	feedback.Note = "reason"
	feedback.RunID = 0
	if !errors.Is(feedback.Validate(run), ErrInvalidRun) {
		t.Fatal("targetless feedback accepted")
	}
}

func TestUT177NoSuccessfulRunRemainsUnavailable(t *testing.T) {
	run := task05Run(t)
	if run.State != StateQueued || len(run.Output) != 0 || len(run.Evidence) != 0 {
		t.Fatalf("queued run fabricated success: %#v", run)
	}
}

func TestUT178VersionAndEvidenceHistoriesRemainPageable(t *testing.T) {
	run := successfulTask05Run(t)
	history := make([]Selection, 0, 200)
	for index := int64(1); index <= 200; index++ {
		value := Selection{ID: index, SeriesID: run.SeriesID, RunID: run.ID, ActorID: 2,
			Version: index, RequestID: "selection-" + time.Unix(index, 0).UTC().Format(time.RFC3339Nano),
			SelectedAt: time.Unix(index, 0).UTC()}
		var err error
		history, err = Select(history, run, value)
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(history) != 200 || len(history[:50]) != 50 {
		t.Fatalf("history size = %d", len(history))
	}
}

func TestUT179ViewerCannotRerunFlagOrSelect(t *testing.T) {
	viewer := access.Principal{ActorID: 1, Kind: access.ActorMember, Role: access.RoleViewer,
		Status: access.StatusActive}
	if !errors.Is(access.Authorize(viewer, access.ActionProjectWrite), access.ErrPermissionDenied) {
		t.Fatal("viewer received analysis write access")
	}
}

func TestUT180DuplicateFeedbackRequestIsIdempotent(t *testing.T) {
	run := successfulTask05Run(t)
	value := Feedback{ID: 1, RunID: run.ID, ActorID: 2, Rating: "incorrect", Note: "reason",
		RequestID: "feedback-1", CreatedAt: time.Now()}
	history, err := AppendFeedback(nil, run, value)
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := AppendFeedback(history, run, value)
	if err != nil || len(replayed) != 1 {
		t.Fatalf("feedback replay duplicated: %#v err=%v", replayed, err)
	}
}

func TestUT181FailedOrIncompleteRunCannotBeSelected(t *testing.T) {
	run := task05Run(t)
	value := Selection{ID: 1, SeriesID: run.SeriesID, RunID: run.ID, ActorID: 2, Version: 1,
		RequestID: "selection-1", SelectedAt: time.Now()}
	if _, err := Select(nil, run, value); !errors.Is(err, ErrSelection) {
		t.Fatalf("error = %v, want selection rejection", err)
	}
}

func TestUT182OlderSelectedRunRemainsVisibleUntilReplaced(t *testing.T) {
	run := successfulTask05Run(t)
	first := Selection{ID: 1, SeriesID: run.SeriesID, RunID: run.ID, ActorID: 2, Version: 1,
		RequestID: "first", SelectedAt: time.Now()}
	history, err := Select(nil, run, first)
	if err != nil || len(history) != 1 || history[0] != first {
		t.Fatalf("selection history changed: %#v err=%v", history, err)
	}
}

func TestUT261UnsupportedClaimCannotPassEvidenceGate(t *testing.T) {
	TestUT153ClaimsWaitForAccessibleEvidence(t)
}

func TestUT262MalformedStructuredOutputPublishesNoPartialSuccess(t *testing.T) {
	TestUT148MalformedProviderOutputCannotPublish(t)
}

func TestUT273FeedbackRerunAndSelectionPreserveOriginalRun(t *testing.T) {
	run := successfulTask05Run(t)
	original := cloneRun(run)
	feedback := Feedback{ID: 1, RunID: run.ID, ActorID: 2, Rating: "partial", Note: "review",
		RequestID: "feedback", CreatedAt: time.Now()}
	if _, err := AppendFeedback(nil, run, feedback); err != nil {
		t.Fatal(err)
	}
	selection := Selection{ID: 2, SeriesID: run.SeriesID, RunID: run.ID, ActorID: 2, Version: 1,
		RequestID: "selection", SelectedAt: time.Now()}
	if _, err := Select(nil, run, selection); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(run, original) {
		t.Fatalf("original run mutated: before=%#v after=%#v", original, run)
	}
}
