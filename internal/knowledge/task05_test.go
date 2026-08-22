package knowledge

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/leohteixeira/opensource-project-intelligence/internal/access"
)

func limitsFixture() Limits {
	return Limits{MaxDomains: 2, MaxDepth: 2, MaxPages: 3, MaxPageBytes: 100,
		MaxTotalBytes: 200, RequestsPerMinute: 60, MediaTypes: []string{"text/html", "text/plain"}}
}

func chunkFixture(id, projectID, sourceID int64, observed time.Time) Chunk {
	return Chunk{ID: id, ProjectID: projectID, SourceID: sourceID, SnapshotID: id + 100,
		Text: "upgrade documentation", StartOffset: 0, EndOffset: 21, ParserVersion: "parser-v1",
		ObservedAt: observed, Current: true}
}

func TestUT156NoIndexedDocumentationReturnsNoEvidence(t *testing.T) {
	if _, err := Cite(nil); !errors.Is(err, ErrNoEvidence) {
		t.Fatalf("error = %v, want no evidence", err)
	}
}

func TestUT157CrawlLimitsStopAtPredictableBoundary(t *testing.T) {
	budget := Budget{Limits: limitsFixture()}
	if err := budget.Accept("docs.example", 0, 80, "text/html"); err != nil {
		t.Fatal(err)
	}
	if err := budget.Accept("docs.example", 1, 80, "text/plain"); err != nil {
		t.Fatal(err)
	}
	if err := budget.Accept("docs.example", 2, 41, "text/plain"); !errors.Is(err, ErrLimitExceeded) {
		t.Fatalf("error = %v, want total byte limit", err)
	}
	if budget.Pages != 3 || budget.Bytes != 201 {
		t.Fatalf("failed page was not accounted: %#v", budget)
	}
}

func TestUT158AnalystsConfigureCrawlsWhileApprovedViewersSearch(t *testing.T) {
	viewer := access.Principal{ActorID: 1, Kind: access.ActorMember, Role: access.RoleViewer,
		Status: access.StatusActive}
	if err := access.Authorize(viewer, access.ActionIntelligenceRead); err != nil {
		t.Fatalf("viewer search denied: %v", err)
	}
	if !errors.Is(access.Authorize(viewer, access.ActionProjectWrite), access.ErrPermissionDenied) {
		t.Fatal("viewer received crawl configuration access")
	}
}

func TestUT159UnchangedContentHasStableSnapshotDigest(t *testing.T) {
	value := Snapshot{ID: 1, ProjectID: 2, SourceID: 3, URL: "https://docs.example/guide",
		ObservedAt: time.Now(), MediaType: "text/plain"}
	left, err := NewSnapshot(value, []byte("same immutable body"))
	if err != nil {
		t.Fatal(err)
	}
	value.ID++
	right, err := NewSnapshot(value, []byte("same immutable body"))
	if err != nil || left.Digest != right.Digest {
		t.Fatalf("digest changed: %x %x err=%v", left.Digest, right.Digest, err)
	}
}

func TestUT160SearchIndexesOnlyValidatedCurrentSnapshots(t *testing.T) {
	now := time.Now().UTC()
	valid := chunkFixture(1, 7, 8, now)
	invalid := chunkFixture(2, 7, 8, now)
	invalid.Current = false
	values, err := Fuse([]Candidate{{Chunk: invalid, LexicalRank: 1}, {Chunk: valid, LexicalRank: 2}},
		Filter{ProjectID: 7, Cutoff: now.Add(time.Second)}, 10, 60)
	if err != nil || len(values) != 1 || values[0].Chunk.ID != valid.ID {
		t.Fatalf("results = %#v, err=%v", values, err)
	}
}

func TestUT161RemovedURLsLeaveCurrentSearchButRetainHistory(t *testing.T) {
	now := time.Now().UTC()
	historical := chunkFixture(1, 7, 8, now)
	historical.Current = false
	values, err := Fuse([]Candidate{{Chunk: historical, LexicalRank: 1}},
		Filter{ProjectID: 7, Cutoff: now.Add(time.Second)}, 10, 60)
	if err != nil || len(values) != 0 || historical.Text == "" || historical.SnapshotID == 0 {
		t.Fatalf("historical/current semantics changed: results=%#v chunk=%#v err=%v", values, historical, err)
	}
}

func TestUT242ReciprocalRankFusionIsDeterministicWithStableTies(t *testing.T) {
	now := time.Now().UTC()
	values := []Candidate{
		{Chunk: chunkFixture(2, 7, 9, now), LexicalRank: 1, VectorRank: 3},
		{Chunk: chunkFixture(1, 7, 8, now), LexicalRank: 3, VectorRank: 1},
		{Chunk: chunkFixture(3, 99, 8, now), LexicalRank: 1, VectorRank: 1},
	}
	filter := Filter{ProjectID: 7, Cutoff: now.Add(time.Second)}
	first, err := Fuse(values, filter, 10, 60)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Fuse([]Candidate{values[2], values[1], values[0]}, filter, 10, 60)
	if err != nil || !reflect.DeepEqual(first, second) || len(first) != 2 || first[0].Chunk.ID != 1 {
		t.Fatalf("unstable RRF: first=%#v second=%#v err=%v", first, second, err)
	}
}

func TestChunkTextIsUTF8SafeAndVersioned(t *testing.T) {
	now := time.Now().UTC()
	snapshot := Snapshot{ID: 1, ProjectID: 2, SourceID: 3, Language: "pt-BR",
		ObservedAt: now, Current: true}
	chunks, err := ChunkText(snapshot, "# Atualização\nVersão estável com migração segura e documentação detalhada.",
		"parser-v1", 64)
	if err != nil || len(chunks) == 0 || chunks[0].ParserVersion != "parser-v1" || chunks[0].Heading != "Atualização" {
		t.Fatalf("chunks = %#v, err=%v", chunks, err)
	}
}
