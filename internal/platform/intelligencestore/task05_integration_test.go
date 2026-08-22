//go:build integration

package intelligencestore_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/pgvector/pgvector-go"

	"github.com/leohteixeira/opensource-project-intelligence/internal/access"
	"github.com/leohteixeira/opensource-project-intelligence/internal/analysis"
	"github.com/leohteixeira/opensource-project-intelligence/internal/collector"
	"github.com/leohteixeira/opensource-project-intelligence/internal/knowledge"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/intelligencestore"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/sourceadapter"
	"github.com/leohteixeira/opensource-project-intelligence/internal/topic"
)

const (
	task05RepositoryID int64 = 7_400_000_000_000_050_100
	task05GitSourceID  int64 = 7_400_000_000_000_050_110
	task05NPMSourceID  int64 = 7_400_000_000_000_050_111
	task05AdvSourceID  int64 = 7_400_000_000_000_050_112
	task05DocsSourceID int64 = 7_400_000_000_000_050_113
	task05SnapshotID   int64 = 7_400_000_000_000_055_001
	task05ChunkID      int64 = 7_400_000_000_000_056_001
	task05TopicID      int64 = 7_400_000_000_000_057_002
	task05ReleaseID    int64 = 7_400_000_000_000_058_001
	task05RunID        int64 = 7_400_000_000_000_059_001
	task05SeriesID     int64 = 7_400_000_000_000_059_000
)

func TestTask05PostgreSQLContracts(t *testing.T) {
	h := newTask04Harness(t)
	seedTask05(t, h)
	succeeded := seedTask05SuccessfulRun(t, h)

	t.Run("IT-043 IT-044 IT-045 adoption cutoffs failures and portfolio scale preserve source context", func(t *testing.T) {
		values, err := h.store.Adoption(h.ctx, h.principal, task04ProjectOne,
			h.window.From, h.window.To, 50, 0)
		if err != nil || len(values) != 2 || values[0].WindowTo != values[1].WindowTo {
			t.Fatalf("adoption snapshot mixed cutoffs: values=%#v err=%v", values, err)
		}
		if _, err := h.pool.Unwrap().Exec(h.ctx, `UPDATE sources SET state='unavailable',
			failure_code='controlled_failure' WHERE id=$1`, task05AdvSourceID); err != nil {
			t.Fatal(err)
		}
		stillVisible, err := h.store.Adoption(h.ctx, h.principal, task04ProjectOne,
			h.window.From, h.window.To, 50, 0)
		if err != nil || len(stillVisible) != 2 {
			t.Fatalf("independent evidence disappeared: count=%d err=%v", len(stillVisible), err)
		}
		if _, err := h.pool.Unwrap().Exec(h.ctx, `WITH values AS (SELECT generate_series(1,500)::bigint n)
			INSERT INTO registry_adoption_snapshots
			(id,project_id,source_id,package,registry,unit,population_context,numeric_value,status,
			 window_from,window_to,observed_at,raw_object_id)
			SELECT 7400000000000060000+n,$1,$2,'package-'||n,'npm','weekly_downloads','npm_public',n,
			'available',$3,$4,$4,7400000000000050201 FROM values`, task04ProjectOne,
			task05NPMSourceID, h.window.From, h.window.To); err != nil {
			t.Fatal(err)
		}
		page, err := h.store.Adoption(h.ctx, h.principal, task04ProjectOne,
			h.window.From, h.window.To, 50, 0)
		if err != nil || len(page) != 50 {
			t.Fatalf("scaled adoption page lost context: count=%d err=%v", len(page), err)
		}
		if page[0].Unit == "" || page[0].Population == "" {
			t.Fatalf("scaled adoption page lost source context: first=%#v", page[0])
		}
	})

	t.Run("IT-061 IT-062 IT-063 topic correction concurrency preserves prior bounded version", func(t *testing.T) {
		start := make(chan struct{})
		results := make(chan error, 2)
		var group sync.WaitGroup
		for index := range 2 {
			group.Add(1)
			go func(index int) {
				defer group.Done()
				<-start
				_, err := h.store.CorrectTopic(h.ctx, h.principal, topic.Correction{
					ProjectID: task04ProjectOne, TopicID: task05TopicID, Action: topic.ActionRename,
					Label: fmt.Sprintf("Canonical %d", index), Reason: "concurrent review",
					RequestID: fmt.Sprintf("topic-concurrent-%d", index), Version: 1,
				})
				results <- err
			}(index)
		}
		close(start)
		group.Wait()
		close(results)
		succeededCount, conflicts := 0, 0
		for err := range results {
			switch {
			case err == nil:
				succeededCount++
			case errors.Is(err, access.ErrVersionConflict):
				conflicts++
			default:
				t.Fatalf("unexpected correction outcome: %v", err)
			}
		}
		values, err := h.store.Topics(h.ctx, h.principal, task04ProjectOne, 10, 0)
		if succeededCount != 1 || conflicts != 1 || err != nil || len(values) != 1 ||
			values[0].Candidate.AlgorithmVersion != "mutual-knn-v1" {
			t.Fatalf("topic version outcomes success=%d conflict=%d values=%#v err=%v",
				succeededCount, conflicts, values, err)
		}
		if _, err := topic.Candidates(task04ProjectOne, nil, 101, "mutual-knn-v1"); !errors.Is(err, topic.ErrInvalid) {
			t.Fatalf("unbounded clustering accepted: %v", err)
		}
	})

	t.Run("IT-064 IT-065 IT-066 reruns remain distinct and failed analysis does not hide deterministic releases", func(t *testing.T) {
		if _, err := h.store.SaveRun(h.ctx, succeeded); err != nil {
			t.Fatalf("exact immutable replay failed: %v", err)
		}
		collision := succeeded
		collision.Model = "different-model"
		if _, err := h.store.SaveRun(h.ctx, collision); !errors.Is(err, analysis.ErrInvalidRun) {
			t.Fatalf("differing immutable replay was accepted: %v", err)
		}
		for index := int64(1); index <= 2; index++ {
			parent := succeeded.ID
			value, err := analysis.NewRun(analysis.Run{ID: task05RunID + index,
				SeriesID: succeeded.SeriesID, ProjectID: succeeded.ProjectID, ParentRunID: &parent,
				Kind: succeeded.Kind, PromptVersion: succeeded.PromptVersion,
				SchemaVersion: succeeded.SchemaVersion, RetrievalVersion: succeeded.RetrievalVersion,
				Provider: "fixture", Model: "rerun", Language: "en", RequestedBy: 201,
				Cutoff:    succeeded.Cutoff,
				CreatedAt: succeeded.CreatedAt.Add(time.Duration(index) * time.Minute)})
			if err != nil {
				t.Fatal(err)
			}
			if index == 2 {
				value, err = value.Start(value.CreatedAt.Add(time.Second))
				if err == nil {
					value, err = value.Fail("provider_timeout", value.CreatedAt.Add(2*time.Second))
				}
			}
			if err == nil {
				_, err = h.store.SaveRun(h.ctx, value)
			}
			if err != nil {
				t.Fatal(err)
			}
		}
		releases, err := h.store.Releases(h.ctx, h.principal, task04ProjectOne, 1, 0)
		if err != nil || len(releases) != 1 || releases[0].ID != task05ReleaseID {
			t.Fatalf("release metadata unavailable after failed run: %#v err=%v", releases, err)
		}
		if _, err := h.pool.Unwrap().Exec(h.ctx, `WITH values AS (SELECT generate_series(1,1001)::bigint n)
			INSERT INTO canonical_releases(id,project_id,repository_id,source_id,external_id,tag,draft,
				prerelease,published_at,raw_object_id,title,body,language,canonical_url,state)
			SELECT 7400000000000061000+n,$1,$2,$3,'historical-'||n,'v0.'||n,false,false,
				$4::timestamptz-(n*interval '1 minute'),7400000000000050206,'Historical release','','en',
				'https://github.com/task05/project/releases/tag/v0.'||n,'published' FROM values`,
			task04ProjectOne, task05RepositoryID, task05GitSourceID, h.window.From); err != nil {
			t.Fatal(err)
		}
		oldestID := int64(7_400_000_000_000_062_001)
		oldest, err := h.store.Release(h.ctx, h.principal, task04ProjectOne, oldestID)
		if err != nil || oldest.ID != oldestID {
			t.Fatalf("release beyond first 1000 rows was not addressable: release=%#v err=%v", oldest, err)
		}
		var runCount int
		if err := h.pool.Unwrap().QueryRow(h.ctx, `SELECT count(*) FROM analysis_runs WHERE series_id=$1`,
			succeeded.SeriesID).Scan(&runCount); err != nil || runCount != 3 {
			t.Fatalf("run count=%d err=%v", runCount, err)
		}
	})

	t.Run("IT-067 IT-068 IT-069 crawl replay snapshot identity and search scale remain bounded", func(t *testing.T) {
		path := fmt.Sprintf("/api/v1/projects/%d/crawls", task04ProjectOne)
		body := map[string]any{"source_ids": []string{fmt.Sprint(task05DocsSourceID)}, "max_depth": 2}
		first := h.task05Request(t, http.MethodPost, path, body, true,
			map[string]string{"Idempotency-Key": "same-crawl"})
		second := h.task05Request(t, http.MethodPost, path, body, true,
			map[string]string{"Idempotency-Key": "same-crawl"})
		if first.Code != http.StatusAccepted || second.Code != http.StatusAccepted ||
			!bytes.Equal(first.Body.Bytes(), second.Body.Bytes()) {
			t.Fatalf("crawl replay differs: first=%d %s second=%d %s", first.Code, first.Body.String(), second.Code, second.Body.String())
		}
		if _, err := h.pool.Unwrap().Exec(h.ctx, `INSERT INTO document_snapshots
			(id,project_id,source_id,canonical_url,observed_at,digest,media_type,raw_object_id,parser_version)
			VALUES(7400000000000055009,$1,$2,'https://docs.example.test/upgrade',$3,
			decode(repeat('04',32),'hex'),'text/html',7400000000000050204,'parser-v1')`,
			task04ProjectOne, task05DocsSourceID, h.window.To); err == nil {
			t.Fatal("duplicate immutable snapshot unexpectedly created")
		}
		results, err := h.store.Search(h.ctx, h.principal, task04ProjectOne,
			intelligencestore.SearchRequest{Query: "upgrade", Cutoff: h.window.To, Limit: 1})
		if err != nil || len(results) != 1 {
			t.Fatalf("bounded search results=%#v err=%v", results, err)
		}
	})

	t.Run("IT-070 IT-071 IT-072 query cutoffs cancellation and broad scopes remain bounded", func(t *testing.T) {
		results, err := h.store.Search(h.ctx, h.principal, task04ProjectOne,
			intelligencestore.SearchRequest{Query: "future-only", Cutoff: h.window.To, Limit: 10})
		if err != nil || len(results) != 0 {
			t.Fatalf("cutoff mixed future evidence: %#v err=%v", results, err)
		}
		running, err := task05QueuedRun(t, task05RunID+10, task05SeriesID+10).Start(h.window.To.Add(time.Minute))
		if err != nil {
			t.Fatal(err)
		}
		cancelled, err := running.Cancel(h.window.To.Add(2 * time.Minute))
		if err != nil || cancelled.State != analysis.StateCancelled || len(cancelled.Output) != 0 {
			t.Fatalf("cancelled run=%#v err=%v", cancelled, err)
		}
		query := analysis.Query{ProjectID: task04ProjectOne, Question: "everything",
			Cutoff: h.window.To, MaxResults: 10, MaxOutputBytes: 1_024, SourceIDs: make([]int64, 33)}
		if !errors.Is(query.Validate(), analysis.ErrInvalidRun) {
			t.Fatal("workspace-wide query accepted")
		}
	})

	t.Run("IT-076 IT-077 IT-078 selection conflicts and interrupted histories stay discoverable", func(t *testing.T) {
		start := make(chan struct{})
		results := make(chan error, 2)
		var group sync.WaitGroup
		for index := range 2 {
			group.Add(1)
			go func(index int) {
				defer group.Done()
				<-start
				_, err := h.store.SelectRun(h.ctx, h.principal, analysis.Selection{
					SeriesID: succeeded.SeriesID, RunID: succeeded.ID,
					RequestID: fmt.Sprintf("selection-concurrent-%d", index), Version: 1})
				results <- err
			}(index)
		}
		close(start)
		group.Wait()
		close(results)
		successes, conflicts := 0, 0
		for err := range results {
			if err == nil {
				successes++
			} else if errors.Is(err, access.ErrVersionConflict) {
				conflicts++
			} else {
				t.Fatal(err)
			}
		}
		cancelled := task05QueuedRun(t, task05RunID+20, task05SeriesID+20)
		cancelled, _ = cancelled.Start(cancelled.CreatedAt.Add(time.Second))
		cancelled, _ = cancelled.Cancel(cancelled.CreatedAt.Add(2 * time.Second))
		if _, err := h.store.SaveRun(h.ctx, cancelled); err != nil {
			t.Fatal(err)
		}
		loaded, err := h.store.Run(h.ctx, h.principal, cancelled.ID)
		if successes != 1 || conflicts != 1 || err != nil || loaded.State != analysis.StateCancelled {
			t.Fatalf("selection/cancel outcomes success=%d conflicts=%d loaded=%#v err=%v", successes, conflicts, loaded, err)
		}
	})

	t.Run("IT-115 provider mappings preserve canonical invariants and distinct provenance", func(t *testing.T) {
		observed := "2026-08-22T00:00:00Z"
		gitlab, err := sourceadapter.GitLabIssues([]byte(`[{"id":42,"iid":7,"title":"Same","description":"Body","web_url":"https://gitlab.example/repo/-/issues/7","created_at":"`+observed+`"}]`), []int64{1})
		if err != nil {
			t.Fatal(err)
		}
		gitea, err := sourceadapter.GiteaIssues([]byte(`[{"id":42,"number":7,"title":"Same","body":"Body","html_url":"https://gitea.example/repo/issues/7","created_at":"`+observed+`"}]`), []int64{2})
		if err != nil || gitlab[0].ExternalID != gitea[0].ExternalID || gitlab[0].Title != gitea[0].Title ||
			gitlab[0].EvidenceID == gitea[0].EvidenceID {
			t.Fatalf("provider mappings differ: gitlab=%#v gitea=%#v err=%v", gitlab, gitea, err)
		}
	})

	t.Run("IT-117 IT-118 crawler revalidates DNS and records hard budget boundaries", func(t *testing.T) {
		resolver := &task05SwitchingResolver{answers: [][]netip.Addr{
			{netip.MustParseAddr("203.0.113.10")}, {netip.MustParseAddr("127.0.0.1")},
		}}
		policy := collector.PublicURLPolicy{Resolver: resolver}
		if _, err := policy.Validate(h.ctx, "https://docs.example.test/start"); err != nil {
			t.Fatal(err)
		}
		if _, err := policy.Validate(h.ctx, "https://docs.example.test/redirect"); !errors.Is(err, collector.ErrUnsafeSource) {
			t.Fatalf("rebound address accepted: %v", err)
		}
		budget := knowledge.Budget{Limits: knowledge.Limits{MaxDomains: 1, MaxDepth: 1,
			MaxPages: 1, MaxPageBytes: 4, MaxTotalBytes: 4, RequestsPerMinute: 1,
			MediaTypes: []string{"text/plain"}}}
		if err := budget.Accept("docs.example.test", 0, 4, "text/plain"); err != nil {
			t.Fatal(err)
		}
		if err := budget.Accept("docs.example.test", 1, 1, "text/plain"); !errors.Is(err, knowledge.ErrLimitExceeded) {
			t.Fatalf("page boundary accepted: %v", err)
		}
	})

	t.Run("IT-123 hybrid retrieval applies project filters before deterministic RRF", func(t *testing.T) {
		near, far := make([]float32, 1_536), make([]float32, 1_536)
		for index := range near {
			near[index] = .01
		}
		for index := range far {
			far[index] = .1
		}
		if _, err := h.pool.Unwrap().Exec(h.ctx, `UPDATE knowledge_chunks SET embedding=$1,
			embedding_model='fixture-v1' WHERE id=$2`, pgvector.NewVector(near), task05ChunkID); err != nil {
			t.Fatal(err)
		}
		if _, err := h.pool.Unwrap().Exec(h.ctx, `UPDATE knowledge_chunks SET embedding=$1,
			embedding_model='fixture-v1' WHERE id=$2`, pgvector.NewVector(far), task05ChunkID+1); err != nil {
			t.Fatal(err)
		}
		request := intelligencestore.SearchRequest{Query: "upgrade", Cutoff: h.window.To,
			Embedding: near, Limit: 10}
		first, err := h.store.Search(h.ctx, h.principal, task04ProjectOne, request)
		if err != nil {
			t.Fatal(err)
		}
		second, err := h.store.Search(h.ctx, h.principal, task04ProjectOne, request)
		if err != nil || !reflect.DeepEqual(first, second) || len(first) == 0 ||
			first[0].Chunk.ProjectID != task04ProjectOne || first[0].Chunk.SnapshotID == 0 {
			t.Fatalf("hybrid results unstable: first=%#v second=%#v err=%v", first, second, err)
		}
	})

	t.Run("IT-124 persisted topic candidates keep generated labels corrections and evidence separate", func(t *testing.T) {
		values, err := h.store.Topics(h.ctx, h.principal, task04ProjectOne, 10, 0)
		if err != nil || len(values) != 1 || len(values[0].Members) != 2 ||
			values[0].Candidate.GeneratedLabel == values[0].Label || len(values[0].History) != 1 {
			t.Fatalf("topic pipeline = %#v, err=%v", values, err)
		}
	})

	t.Run("IT-125 model degradation leaves deterministic releases and lexical retrieval available", func(t *testing.T) {
		releases, releaseErr := h.store.Releases(h.ctx, h.principal, task04ProjectOne, 10, 0)
		results, searchErr := h.store.Search(h.ctx, h.principal, task04ProjectOne,
			intelligencestore.SearchRequest{Query: "upgrade", Cutoff: h.window.To, Limit: 10})
		if releaseErr != nil || searchErr != nil || len(releases) == 0 || len(results) == 0 {
			t.Fatalf("deterministic degradation failed: releases=%d results=%d errors=%v/%v",
				len(releases), len(results), releaseErr, searchErr)
		}
	})

	t.Run("IT-126 only schema-valid fully evidenced structured output succeeds", func(t *testing.T) {
		run := task05QueuedRun(t, task05RunID+30, task05SeriesID+30)
		run, _ = run.Start(run.CreatedAt.Add(time.Second))
		before := run
		if _, err := run.Succeed(json.RawMessage(`{"summary":`), nil, analysis.UsageRecord{},
			run.CreatedAt.Add(2*time.Second)); !errors.Is(err, analysis.ErrSchema) {
			t.Fatalf("invalid schema error=%v", err)
		}
		citation := knowledge.Citation{SnapshotID: task05SnapshotID, ChunkID: task05ChunkID,
			StartOffset: 0, EndOffset: 48}
		raw := json.RawMessage(fmt.Sprintf(`{"summary":"Supported","claims":[{"text":"Upgrade","citations":[{"snapshot_id":"%d","chunk_id":"%d","start_offset":0,"end_offset":48}]}]}`,
			task05SnapshotID, task05ChunkID))
		valid, err := run.Succeed(raw, map[int64]knowledge.Citation{task05ChunkID: citation},
			analysis.UsageRecord{}, run.CreatedAt.Add(2*time.Second))
		if err != nil || valid.State != analysis.StateSucceeded || before.State != analysis.StateRunning {
			t.Fatalf("structured result valid=%#v before=%#v err=%v", valid, before, err)
		}
		withoutEvidence := valid
		withoutEvidence.ID += 100
		withoutEvidence.SeriesID += 100
		withoutEvidence.Evidence = nil
		if _, err := h.store.SaveRun(h.ctx, withoutEvidence); !errors.Is(err, analysis.ErrEvidence) {
			t.Fatalf("succeeded run without evidence error=%v, want evidence rejection", err)
		}
		wrongProjectEvidence := valid
		wrongProjectEvidence.ID += 101
		wrongProjectEvidence.SeriesID += 101
		wrongProjectEvidence.ProjectID = task04ProjectTwo
		if _, err := h.store.SaveRun(h.ctx, wrongProjectEvidence); !errors.Is(err, analysis.ErrEvidence) {
			t.Fatalf("cross-project evidence error=%v, want evidence rejection", err)
		}
	})

	t.Run("feedback idempotency preserves the original immutable payload", func(t *testing.T) {
		value := analysis.Feedback{RunID: succeeded.ID, Rating: "incorrect", Note: "first review",
			RequestID: "immutable-feedback-replay"}
		first, err := h.store.SaveFeedback(h.ctx, h.principal, value)
		if err != nil {
			t.Fatal(err)
		}
		value.Note = "changed review"
		if _, err := h.store.SaveFeedback(h.ctx, h.principal, value); !errors.Is(err, analysis.ErrInvalidRun) {
			t.Fatalf("changed feedback replay error=%v, want immutable replay rejection", err)
		}
		replayed, err := h.store.SaveFeedback(h.ctx, h.principal, analysis.Feedback{
			RunID: succeeded.ID, Rating: first.Rating, Note: first.Note, RequestID: first.RequestID,
		})
		if err != nil || replayed.ID != first.ID || replayed.CreatedAt != first.CreatedAt {
			t.Fatalf("exact feedback replay changed identity: first=%#v replayed=%#v err=%v", first, replayed, err)
		}
	})

	t.Run("IT-229 IT-231 IT-257 IT-259 IT-261 IT-263 IT-265 IT-267 IT-269 IT-271 IT-273 IT-275 IT-277 HTTP operations return typed task contracts", func(t *testing.T) {
		cases := []struct {
			method, path string
			body         any
			status       int
			headers      map[string]string
		}{
			{http.MethodGet, fmt.Sprintf("/api/v1/projects/%d/adoption?window=90d&limit=10", task04ProjectOne), nil, http.StatusOK, nil},
			{http.MethodGet, fmt.Sprintf("/api/v1/projects/%d/security?window=365d&limit=10", task04ProjectOne), nil, http.StatusOK, nil},
			{http.MethodGet, fmt.Sprintf("/api/v1/projects/%d/topics?window=90d&limit=10", task04ProjectOne), nil, http.StatusOK, nil},
			{http.MethodGet, fmt.Sprintf("/api/v1/projects/%d/releases?limit=10", task04ProjectOne), nil, http.StatusOK, nil},
			{http.MethodGet, fmt.Sprintf("/api/v1/projects/%d/releases/%d", task04ProjectOne, task05ReleaseID), nil, http.StatusOK, nil},
			{http.MethodPost, fmt.Sprintf("/api/v1/projects/%d/crawls", task04ProjectOne), map[string]any{"source_ids": []string{fmt.Sprint(task05DocsSourceID)}, "max_depth": 3}, http.StatusAccepted, nil},
			{http.MethodPost, fmt.Sprintf("/api/v1/projects/%d/knowledge/search", task04ProjectOne), map[string]any{"query": "How are upgrades handled?", "language": "en", "limit": 10}, http.StatusOK, nil},
			{http.MethodPost, fmt.Sprintf("/api/v1/projects/%d/queries", task04ProjectOne), map[string]any{"question": "What changed in maintenance risk?", "window": "90d", "language": "en"}, http.StatusAccepted, nil},
			{http.MethodGet, fmt.Sprintf("/api/v1/analysis-runs/%d", task05RunID), nil, http.StatusOK, nil},
			{http.MethodPost, fmt.Sprintf("/api/v1/analysis-runs/%d/reruns", task05RunID), map[string]any{"language": "pt-BR", "reason": "Review corrected topic"}, http.StatusAccepted, nil},
			{http.MethodPost, fmt.Sprintf("/api/v1/analysis-runs/%d/feedback", task05RunID), map[string]any{"rating": "incorrect", "comment": "The cited issue belongs to the SDK."}, http.StatusCreated, nil},
			{http.MethodPost, fmt.Sprintf("/api/v1/analysis-series/%d/selection", task05SeriesID), map[string]any{"run_id": fmt.Sprint(task05RunID)}, http.StatusOK, map[string]string{"If-Match": `"v1"`}},
		}
		for _, item := range cases {
			response := h.task05Request(t, item.method, item.path, item.body, true, item.headers)
			if response.Code != item.status {
				t.Fatalf("%s %s returned %d: %s", item.method, item.path, response.Code, response.Body.String())
			}
		}
		var topicVersion int64
		if err := h.pool.Unwrap().QueryRow(h.ctx, `SELECT COALESCE(max(version),0) FROM topic_corrections WHERE topic_id=$1`, task05TopicID).Scan(&topicVersion); err != nil {
			t.Fatal(err)
		}
		response := h.task05Request(t, http.MethodPost,
			fmt.Sprintf("/api/v1/projects/%d/topics/%d/corrections", task04ProjectOne, task05TopicID),
			map[string]any{"action": "rename", "label": "API label", "reason": "reviewed"}, true,
			map[string]string{"If-Match": fmt.Sprintf(`"v%d"`, topicVersion)})
		if response.Code != http.StatusAccepted || response.Header().Get("ETag") == "" {
			t.Fatalf("topic correction returned %d: %s", response.Code, response.Body.String())
		}
	})

	t.Run("IT-230 IT-232 IT-258 IT-260 IT-262 IT-264 IT-266 IT-268 IT-270 IT-272 IT-274 IT-276 IT-278 rejected HTTP calls disclose no protected intelligence", func(t *testing.T) {
		cases := []struct {
			method, path string
			body         any
		}{
			{http.MethodGet, fmt.Sprintf("/api/v1/projects/%d/adoption?window=90d", task04ProjectOne), nil},
			{http.MethodGet, fmt.Sprintf("/api/v1/projects/%d/security?window=365d", task04ProjectOne), nil},
			{http.MethodGet, fmt.Sprintf("/api/v1/projects/%d/topics?window=90d", task04ProjectOne), nil},
			{http.MethodPost, fmt.Sprintf("/api/v1/projects/%d/topics/%d/corrections", task04ProjectOne, task05TopicID), map[string]any{"action": "rename", "label": "x", "reason": "x"}},
			{http.MethodGet, fmt.Sprintf("/api/v1/projects/%d/releases", task04ProjectOne), nil},
			{http.MethodGet, fmt.Sprintf("/api/v1/projects/%d/releases/%d", task04ProjectOne, task05ReleaseID), nil},
			{http.MethodPost, fmt.Sprintf("/api/v1/projects/%d/crawls", task04ProjectOne), map[string]any{"source_ids": []string{fmt.Sprint(task05DocsSourceID)}, "max_depth": 3}},
			{http.MethodPost, fmt.Sprintf("/api/v1/projects/%d/knowledge/search", task04ProjectOne), map[string]any{"query": "upgrade", "limit": 10}},
			{http.MethodPost, fmt.Sprintf("/api/v1/projects/%d/queries", task04ProjectOne), map[string]any{"question": "risk"}},
			{http.MethodGet, fmt.Sprintf("/api/v1/analysis-runs/%d", task05RunID), nil},
			{http.MethodPost, fmt.Sprintf("/api/v1/analysis-runs/%d/reruns", task05RunID), map[string]any{}},
			{http.MethodPost, fmt.Sprintf("/api/v1/analysis-runs/%d/feedback", task05RunID), map[string]any{"rating": "incorrect", "comment": "reason"}},
			{http.MethodPost, fmt.Sprintf("/api/v1/analysis-series/%d/selection", task05SeriesID), map[string]any{"run_id": fmt.Sprint(task05RunID)}},
		}
		for _, item := range cases {
			response := h.task05Request(t, item.method, item.path, item.body, false,
				map[string]string{"If-Match": `"v0"`})
			if response.Code != http.StatusUnauthorized {
				t.Fatalf("unauthorized %s %s returned %d: %s", item.method, item.path, response.Code, response.Body.String())
			}
		}
	})

	t.Run("public analysis requests cannot select model or inject embeddings", func(t *testing.T) {
		cases := []struct {
			path string
			body map[string]any
		}{
			{fmt.Sprintf("/api/v1/projects/%d/knowledge/search", task04ProjectOne),
				map[string]any{"query": "upgrade", "language": "en", "limit": 10,
					"embedding": []float64{0.1}}},
			{fmt.Sprintf("/api/v1/projects/%d/queries", task04ProjectOne),
				map[string]any{"question": "risk", "window": "90d", "language": "en",
					"provider": "user-selected", "model": "user-selected"}},
			{fmt.Sprintf("/api/v1/analysis-runs/%d/reruns", task05RunID),
				map[string]any{"language": "en", "reason": "review",
					"provider": "user-selected", "model": "user-selected"}},
		}
		for _, item := range cases {
			response := h.task05Request(t, http.MethodPost, item.path, item.body, true, nil)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("POST %s accepted client model controls: %d %s",
					item.path, response.Code, response.Body.String())
			}
		}
	})

	t.Run("extended intelligence rejects writes after project deletion starts", func(t *testing.T) {
		if _, err := h.pool.Unwrap().Exec(h.ctx, `UPDATE projects SET state='deleting',
			unavailable_at=$2,updated_at=$2 WHERE id=$1`, task04ProjectTwo, h.window.To); err != nil {
			t.Fatal(err)
		}
		if _, err := h.pool.Unwrap().Exec(h.ctx, `INSERT INTO analysis_series
			(id,project_id,subject_kind,subject_id) VALUES(7400000000000059999,$1,'release',1)`,
			task04ProjectTwo); err == nil {
			t.Fatal("analysis series write succeeded after project deletion started")
		}
	})
}

func seedTask05(t *testing.T, h *task04Harness) {
	t.Helper()
	statements := []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO repositories(id,project_id,provider,canonical_url,role,default_branch)
			VALUES($1,$2,'github','https://github.com/task05/project','primary','main')`, []any{task05RepositoryID, task04ProjectOne}},
		{`INSERT INTO sources(id,project_id,repository_id,kind,canonical_url,coverage_from,coverage_to) VALUES
			($1,$5,$6,'github','https://api.github.com/repos/task05/project',$7,$8),
			($2,$5,NULL,'npm','https://registry.npmjs.org/task05',$7,$8),
			($3,$5,NULL,'advisory','https://advisories.example.test/task05',$7,$8),
			($4,$5,NULL,'docs','https://docs.example.test/upgrade',$7,$8)`,
			[]any{task05GitSourceID, task05NPMSourceID, task05AdvSourceID, task05DocsSourceID,
				task04ProjectOne, task05RepositoryID, h.window.From.Add(-300 * 24 * time.Hour), h.window.To}},
		{`INSERT INTO raw_objects(id,project_id,source_id,external_type,external_id,observed_at,payload,digest) VALUES
			(7400000000000050201,$1,$2,'registry','npm-task05',$6,'{}',decode(repeat('01',32),'hex')),
			(7400000000000050202,$1,$3,'advisory','ADV-1',$6,'{}',decode(repeat('02',32),'hex')),
			(7400000000000050203,$1,$4,'issue','1',$6,'{}',decode(repeat('03',32),'hex')),
			(7400000000000050204,$1,$5,'document','upgrade',$6,'{}',decode(repeat('04',32),'hex')),
			(7400000000000050205,$1,$4,'issue','2',$6,'{}',decode(repeat('05',32),'hex')),
			(7400000000000050206,$1,$4,'release','v1.2.3',$6,'{}',decode(repeat('06',32),'hex'))`,
			[]any{task04ProjectOne, task05NPMSourceID, task05AdvSourceID, task05GitSourceID,
				task05DocsSourceID, h.window.To}},
		{`INSERT INTO registry_adoption_snapshots
			(id,project_id,source_id,package,registry,unit,population_context,numeric_value,status,
			 window_from,window_to,observed_at,raw_object_id) VALUES
			(7400000000000050301,$1,$2,'task05','npm','weekly_downloads','npm_public',120,'available',$3,$4,$4,7400000000000050201),
			(7400000000000050302,$1,$2,'task05','npm','dependents','npm_public',12,'available',$3,$4,$4,7400000000000050201)`,
			[]any{task04ProjectOne, task05NPMSourceID, h.window.From, h.window.To}},
		{`INSERT INTO public_advisories(id,project_id,source_id,external_id,severity,summary,state,
			published_at,withdrawn_at,raw_object_id) VALUES
			(7400000000000050401,$1,$2,'ADV-1','high','historical advisory','withdrawn',$3,$4,7400000000000050202)`,
			[]any{task04ProjectOne, task05AdvSourceID, h.window.From, h.window.From.Add(24 * time.Hour)}},
		{`INSERT INTO document_snapshots(id,project_id,source_id,canonical_url,observed_at,digest,
			media_type,language,raw_object_id,parser_version,current) VALUES
			($1,$2,$3,'https://docs.example.test/upgrade',$4,decode(repeat('04',32),'hex'),
			'text/html','en',7400000000000050204,'parser-v1',true)`,
			[]any{task05SnapshotID, task04ProjectOne, task05DocsSourceID, h.window.To}},
		{`INSERT INTO knowledge_chunks(id,project_id,source_id,snapshot_id,ordinal,heading,content,
			language,start_offset,end_offset,parser_version,observed_at,current) VALUES
			($1,$3,$4,$5,0,'Upgrade','Upgrade safely using the documented migration path.','en',0,48,'parser-v1',$6,true),
			($2,$3,$4,$5,1,'Maintenance','Maintenance upgrade risk is evidence bounded.','en',49,94,'parser-v1',$6,true)`,
			[]any{task05ChunkID, task05ChunkID + 1, task04ProjectOne, task05DocsSourceID, task05SnapshotID, h.window.To}},
		{`INSERT INTO canonical_issues(id,project_id,repository_id,source_id,external_id,number,title,state,
			created_at,updated_at,raw_object_id) VALUES
			(7400000000000050601,$1,$2,$3,'1',1,'Upgrade docs','open',$4,$4,7400000000000050203),
			(7400000000000050602,$1,$2,$3,'2',2,'Migration discussion','open',$4,$4,7400000000000050205)`,
			[]any{task04ProjectOne, task05RepositoryID, task05GitSourceID, h.window.To}},
		{`INSERT INTO topic_candidate_sets(id,project_id,window_from,window_to,cutoff,algorithm_version,
			neighbor_k,state) VALUES(7400000000000050701,$1,$2,$3,$3,'mutual-knn-v1',10,'current')`,
			[]any{task04ProjectOne, h.window.From, h.window.To}},
		{`INSERT INTO topics(id,candidate_set_id,project_id,generated_label,generated_language,confidence,created_at)
			VALUES($1,7400000000000050701,$2,'Generated maintenance','en',0.8,$3)`,
			[]any{task05TopicID, task04ProjectOne, h.window.To}},
		{`INSERT INTO topic_members(topic_id,issue_id,ordinal,similarity,representative) VALUES
			($1,7400000000000050601,0,0.9,true),($1,7400000000000050602,1,0.85,false)`,
			[]any{task05TopicID}},
		{`INSERT INTO canonical_releases(id,project_id,repository_id,source_id,external_id,tag,draft,
			prerelease,published_at,raw_object_id,title,body,language,canonical_url,state,changelog_snapshot_id)
			VALUES($1,$2,$3,$4,'release-v1.2.3','v1.2.3',false,false,$5,7400000000000050206,
			'Stable release','Upgrade and maintenance fixes','en','https://github.com/task05/project/releases/tag/v1.2.3',
			'published',$6)`, []any{task05ReleaseID, task04ProjectOne, task05RepositoryID,
			task05GitSourceID, h.window.To, task05SnapshotID}},
		{`INSERT INTO knowledge_chunks(id,project_id,source_id,snapshot_id,ordinal,heading,content,
			language,start_offset,end_offset,parser_version,observed_at,current) VALUES
			(7400000000000056999,$1,$2,$3,2,'Future','future-only upgrade evidence','en',95,123,
			'parser-v1',$4,true)`, []any{task04ProjectOne, task05DocsSourceID, task05SnapshotID, h.window.To.Add(time.Hour)}},
	}
	for _, statement := range statements {
		if _, err := h.pool.Unwrap().Exec(h.ctx, statement.sql, statement.args...); err != nil {
			t.Fatalf("seed Task 5: %v\nSQL: %s", err, statement.sql)
		}
	}
}

func seedTask05SuccessfulRun(t *testing.T, h *task04Harness) analysis.Run {
	t.Helper()
	run := task05QueuedRun(t, task05RunID, task05SeriesID)
	run, err := run.Start(run.CreatedAt.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	citation := knowledge.Citation{SnapshotID: task05SnapshotID, ChunkID: task05ChunkID,
		StartOffset: 0, EndOffset: 48}
	raw := json.RawMessage(fmt.Sprintf(`{"summary":"Supported release","claims":[{"text":"Upgrade","citations":[{"snapshot_id":"%d","chunk_id":"%d","start_offset":0,"end_offset":48}]}]}`,
		task05SnapshotID, task05ChunkID))
	run, err = run.Succeed(raw, map[int64]knowledge.Citation{task05ChunkID: citation},
		analysis.UsageRecord{InputTokens: 20, OutputTokens: 10, Cost: .01, Currency: "USD"},
		run.CreatedAt.Add(2*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.store.SaveRun(h.ctx, run); err != nil {
		t.Fatal(err)
	}
	return run
}

func task05QueuedRun(t *testing.T, id, seriesID int64) analysis.Run {
	t.Helper()
	created := time.Date(2026, 8, 22, 0, 0, 0, 0, time.UTC)
	value, err := analysis.NewRun(analysis.Run{ID: id, SeriesID: seriesID,
		ProjectID: task04ProjectOne, Kind: "release", PromptVersion: "release-v1",
		SchemaVersion: "analysis-v1", RetrievalVersion: "rrf-v1", Provider: "fixture",
		Model: "fixture-v1", Language: "en", RequestedBy: 201, Cutoff: created,
		CreatedAt: created})
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func (h *task04Harness) task05Request(t *testing.T, method, path string, body any,
	authenticated bool, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var encoded []byte
	if body != nil {
		var err error
		encoded, err = json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
	}
	request := httptest.NewRequest(method, "http://opi.integration.test"+path, bytes.NewReader(encoded))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "task05-"+fmt.Sprint(time.Now().UnixNano()))
	if authenticated {
		request.Header.Set("Authorization", "Bearer task04-analyst")
	}
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	response := httptest.NewRecorder()
	h.handler.ServeHTTP(response, request)
	return response
}

type task05SwitchingResolver struct {
	mu      sync.Mutex
	answers [][]netip.Addr
}

func (resolver *task05SwitchingResolver) LookupNetIP(context.Context, string, string) ([]netip.Addr, error) {
	resolver.mu.Lock()
	defer resolver.mu.Unlock()
	if len(resolver.answers) == 0 {
		return nil, errors.New("no resolver answer")
	}
	answer := resolver.answers[0]
	resolver.answers = resolver.answers[1:]
	return answer, nil
}
