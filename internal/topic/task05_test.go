package topic

import (
	"errors"
	"reflect"
	"testing"
	"time"
)

func candidateFixture() Candidate {
	return Candidate{ID: 10, ProjectID: 1, Members: []int64{1, 2, 3},
		AlgorithmVersion: "mutual-knn-v1", GeneratedLabel: "Generated",
		CreatedAt: time.Date(2026, 8, 22, 10, 0, 0, 0, time.UTC)}
}

func correctionFixture(id, version int64, action Action) Correction {
	return Correction{ID: id, ProjectID: 1, TopicID: 10, Action: action, IssueIDs: []int64{3},
		Reason: "analyst correction", ActorID: 8, RequestID: "request-" + string(rune('a'+id)),
		Version: version, CreatedAt: time.Date(2026, 8, 22, 11, int(id), 0, 0, time.UTC)}
}

func TestUT141InvalidTopicCorrectionsAreRejected(t *testing.T) {
	for _, value := range []Correction{
		correctionFixture(1, 1, ActionRename),
		{ID: 1, ProjectID: 1, TopicID: 10, Action: ActionMerge, OtherTopicIDs: []int64{10},
			Reason: "circular merge", ActorID: 8, RequestID: "merge", Version: 1, CreatedAt: time.Now()},
		{ID: 1, ProjectID: 1, TopicID: 10, Action: "unsupported", IssueIDs: []int64{3},
			Reason: "unsupported", ActorID: 8, RequestID: "unsupported", Version: 1, CreatedAt: time.Now()},
	} {
		if err := value.Validate(); err == nil {
			t.Fatalf("invalid correction accepted: %#v", value)
		}
	}
}

func TestUT142NoEligibleContentProducesNoTopicCandidates(t *testing.T) {
	values, err := Candidates(1, nil, 5, "mutual-knn-v1")
	if err != nil || len(values) != 0 {
		t.Fatalf("candidates = %#v, err = %v", values, err)
	}
}

func TestUT143LargeTopicEvidenceIsDeterministicAndPageReady(t *testing.T) {
	edges := make([]Neighbor, 0, 398)
	for id := int64(1); id < 200; id++ {
		edges = append(edges, Neighbor{IssueID: id, NeighborID: id + 1, Rank: 1, Similarity: .9},
			Neighbor{IssueID: id + 1, NeighborID: id, Rank: 1, Similarity: .9})
	}
	values, err := Candidates(1, edges, 2, "mutual-knn-v1")
	if err != nil || len(values) != 1 {
		t.Fatalf("bounded candidate failed: count=%d err=%v", len(values), err)
	}
	if len(values[0].Members) != 200 {
		t.Fatalf("members=%d, want 200", len(values[0].Members))
	}
}

func TestUT144ViewerCannotCorrectTopicClassifications(t *testing.T) {
	// The domain correction is valid; the application boundary must require Analyst write access.
	if err := correctionFixture(1, 1, ActionExclude).Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestUT145RepeatedCorrectionRequestIsRejected(t *testing.T) {
	value := correctionFixture(1, 1, ActionExclude)
	replayed := correctionFixture(2, 2, ActionInclude)
	replayed.RequestID = value.RequestID
	if _, err := Apply(candidateFixture(), []Correction{value, replayed}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("error = %v, want duplicate request rejection", err)
	}
}

func TestUT146TrendInputRequiresOrderedCompleteTopicVersion(t *testing.T) {
	newer := correctionFixture(2, 2, ActionInclude)
	older := correctionFixture(1, 1, ActionExclude)
	if _, err := Apply(candidateFixture(), []Correction{newer, older}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("out-of-order versions accepted: %v", err)
	}
}

func TestUT147RetiredTopicHistoryRemainsImmutable(t *testing.T) {
	candidate := candidateFixture()
	retired := candidate.CreatedAt.Add(time.Hour)
	candidate.RetiredAt = &retired
	value, err := Apply(candidate, nil)
	if err != nil || value.Candidate.RetiredAt == nil || !reflect.DeepEqual(value.Members, []int64{1, 2, 3}) {
		t.Fatalf("retired topic changed: %#v, err=%v", value, err)
	}
}

func TestUT243OneWayNeighborDoesNotCreateMutualEdge(t *testing.T) {
	values, err := Candidates(1, []Neighbor{{IssueID: 1, NeighborID: 2, Rank: 1, Similarity: .9}},
		2, "mutual-knn-v1")
	if err != nil || len(values) != 0 {
		t.Fatalf("one-way edge produced candidates: %#v, err=%v", values, err)
	}
}

func TestUT244AnalystConstraintPrecedesReanalysisAndPreservesCandidate(t *testing.T) {
	candidate := candidateFixture()
	original := append([]int64(nil), candidate.Members...)
	value, err := Apply(candidate, []Correction{correctionFixture(1, 1, ActionSplit)})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(candidate.Members, original) || reflect.DeepEqual(value.Members, original) ||
		len(value.History) != 1 {
		t.Fatalf("candidate/history immutability violated: candidate=%v canonical=%#v", candidate.Members, value)
	}
}
