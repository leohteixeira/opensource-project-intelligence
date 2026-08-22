package intelligencestore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pgvector/pgvector-go"

	"github.com/leohteixeira/opensource-project-intelligence/internal/access"
	"github.com/leohteixeira/opensource-project-intelligence/internal/adoption"
	"github.com/leohteixeira/opensource-project-intelligence/internal/analysis"
	"github.com/leohteixeira/opensource-project-intelligence/internal/job"
	"github.com/leohteixeira/opensource-project-intelligence/internal/knowledge"
	"github.com/leohteixeira/opensource-project-intelligence/internal/release"
	"github.com/leohteixeira/opensource-project-intelligence/internal/topic"
)

// QueueCrawl persists a bounded crawl request before delivery. Its immutable limit and root inputs
// live in the checkpoint payload so a worker never has to reconstruct policy from UI state.
func (s *Store) QueueCrawl(ctx context.Context, principal access.Principal, projectID int64,
	sourceIDs []int64, limits knowledge.Limits, requestID string, now time.Time) (job.Job, error) {
	if err := access.Authorize(principal, access.ActionProjectWrite); err != nil {
		return job.Job{}, err
	}
	if err := s.authorizeProject(ctx, principal, projectID); err != nil {
		return job.Job{}, err
	}
	if limits.Validate() != nil || len(sourceIDs) == 0 || strings.TrimSpace(requestID) == "" {
		return job.Job{}, knowledge.ErrInvalid
	}
	rows, err := s.pool.Query(ctx, `SELECT canonical_url FROM sources WHERE project_id=$1
		AND id=ANY($2) AND public AND state='available' AND kind IN ('docs','website','changelog','rss')
		ORDER BY id`, projectID, sourceIDs)
	if err != nil {
		return job.Job{}, fmt.Errorf("load crawl sources: %w", err)
	}
	roots, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return job.Job{}, fmt.Errorf("scan crawl sources: %w", err)
	}
	if len(roots) != len(sourceIDs) {
		return job.Job{}, access.ErrNotFound
	}
	id, err := s.ids.Next(ctx)
	if err != nil {
		return job.Job{}, fmt.Errorf("crawl job ID: %w", err)
	}
	value, err := job.New(id, projectID, "documentation_crawl", "pages", nil, true, now)
	if err != nil {
		return job.Job{}, err
	}
	checkpoint, err := json.Marshal(map[string]any{"roots": roots, "limits": limits})
	if err != nil {
		return job.Job{}, fmt.Errorf("encode crawl request: %w", err)
	}
	progress, err := json.Marshal(value.Progress)
	if err != nil {
		return job.Job{}, fmt.Errorf("encode crawl progress: %w", err)
	}
	eventID, err := s.ids.Next(ctx)
	if err != nil {
		return job.Job{}, fmt.Errorf("crawl event ID: %w", err)
	}
	representation, err := json.Marshal(value)
	if err != nil {
		return job.Job{}, fmt.Errorf("encode crawl event: %w", err)
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return job.Job{}, fmt.Errorf("begin crawl job: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	_, err = tx.Exec(ctx, `INSERT INTO jobs
		(id,kind,state,checkpoint,workspace_id,project_id,version,progress,cancellable,requested_by,
		 request_id,correlation_id,available_at,created_at,updated_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$11,$12,$12,$12)`, value.ID, value.Kind,
		value.State, checkpoint, principal.Workspace, projectID, value.Version, progress,
		value.Cancellable, principal.ActorID, requestID, value.CreatedAt)
	if err != nil {
		return job.Job{}, fmt.Errorf("insert crawl job: %w", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO job_events(id,job_id,job_version,representation,occurred_at)
		VALUES($1,$2,$3,$4,$5)`, eventID, value.ID, value.Version, representation, value.CreatedAt); err != nil {
		return job.Job{}, fmt.Errorf("insert crawl event: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return job.Job{}, fmt.Errorf("commit crawl job: %w", err)
	}
	return value, nil
}

func (s *Store) Adoption(ctx context.Context, principal access.Principal, projectID int64,
	from, to time.Time, limit, offset int) ([]adoption.Snapshot, error) {
	if err := s.authorizeProject(ctx, principal, projectID); err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `SELECT id,project_id,source_id,registry,package,unit,
		population_context,numeric_value,status,window_from,window_to,observed_at,raw_object_id,stale_at
		FROM registry_adoption_snapshots WHERE project_id=$1 AND window_to>$2 AND window_from<$3
		ORDER BY observed_at DESC,id DESC LIMIT $4 OFFSET $5`, projectID, from, to, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("load adoption snapshots: %w", err)
	}
	defer rows.Close()
	values := make([]adoption.Snapshot, 0)
	for rows.Next() {
		var value adoption.Snapshot
		if err := rows.Scan(&value.ID, &value.ProjectID, &value.SourceID, &value.Registry,
			&value.Package, &value.Unit, &value.Population, &value.Value, &value.Status,
			&value.WindowFrom, &value.WindowTo, &value.ObservedAt, &value.EvidenceID, &value.StaleAt); err != nil {
			return nil, fmt.Errorf("scan adoption snapshot: %w", err)
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate adoption snapshots: %w", err)
	}
	return values, nil
}

type SecurityResult struct {
	Observed bool
	Complete bool
	Items    []adoption.Advisory
}

func (s *Store) Security(ctx context.Context, principal access.Principal, projectID int64,
	from, to time.Time, limit, offset int) (SecurityResult, error) {
	if err := s.authorizeProject(ctx, principal, projectID); err != nil {
		return SecurityResult{}, err
	}
	result := SecurityResult{Items: make([]adoption.Advisory, 0)}
	err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM sources WHERE project_id=$1 AND kind='advisory'),
		COALESCE(bool_and(state='available' AND coverage_from IS NOT NULL AND coverage_from<=$2
		 AND coverage_to IS NOT NULL AND coverage_to>=$3),false)
		FROM sources WHERE project_id=$1 AND kind='advisory'`, projectID, from, to).
		Scan(&result.Observed, &result.Complete)
	if err != nil {
		return SecurityResult{}, fmt.Errorf("load advisory coverage: %w", err)
	}
	rows, err := s.pool.Query(ctx, `SELECT id,project_id,source_id,external_id,severity,summary,state,
		published_at,withdrawn_at,raw_object_id FROM public_advisories
		WHERE project_id=$1 AND published_at >= $2 AND published_at < $3
		ORDER BY published_at DESC,id DESC LIMIT $4 OFFSET $5`, projectID, from, to, limit, offset)
	if err != nil {
		return SecurityResult{}, fmt.Errorf("load public advisories: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var value adoption.Advisory
		if err := rows.Scan(&value.ID, &value.ProjectID, &value.SourceID, &value.ExternalID,
			&value.Severity, &value.Summary, &value.State, &value.PublishedAt,
			&value.WithdrawnAt, &value.EvidenceID); err != nil {
			return SecurityResult{}, fmt.Errorf("scan public advisory: %w", err)
		}
		result.Items = append(result.Items, value)
	}
	if err := rows.Err(); err != nil {
		return SecurityResult{}, fmt.Errorf("iterate public advisories: %w", err)
	}
	return result, nil
}

type SearchRequest struct {
	Query     string
	SourceIDs []int64
	Cutoff    time.Time
	Embedding []float32
	Limit     int
}

// Search filters the authorized corpus before independently ranking lexical and vector candidates.
func (s *Store) Search(ctx context.Context, principal access.Principal, projectID int64,
	request SearchRequest) ([]knowledge.Result, error) {
	if err := s.authorizeProject(ctx, principal, projectID); err != nil {
		return nil, err
	}
	if strings.TrimSpace(request.Query) == "" || request.Cutoff.IsZero() || request.Limit < 1 || request.Limit > 100 {
		return nil, knowledge.ErrInvalid
	}
	sourceIDs := request.SourceIDs
	if sourceIDs == nil {
		sourceIDs = []int64{}
	}
	byID := make(map[int64]knowledge.Candidate)
	rows, err := s.pool.Query(ctx, `SELECT id,source_id,snapshot_id,heading,content,language,
		start_offset,end_offset,parser_version,observed_at,current,
		row_number() OVER (ORDER BY ts_rank_cd(content_tsv,websearch_to_tsquery('simple',$2)) DESC,id)
		FROM knowledge_chunks WHERE project_id=$1 AND current AND observed_at<=$3
		AND (cardinality($4::bigint[])=0 OR source_id=ANY($4))
		AND content_tsv @@ websearch_to_tsquery('simple',$2)
		ORDER BY ts_rank_cd(content_tsv,websearch_to_tsquery('simple',$2)) DESC,id LIMIT 200`,
		projectID, request.Query, request.Cutoff, sourceIDs)
	if err != nil {
		return nil, fmt.Errorf("lexical knowledge search: %w", err)
	}
	for rows.Next() {
		chunk, rank, scanErr := scanSearchChunk(rows, projectID)
		if scanErr != nil {
			rows.Close()
			return nil, scanErr
		}
		candidate := byID[chunk.ID]
		candidate.Chunk, candidate.LexicalRank = chunk, rank
		byID[chunk.ID] = candidate
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("iterate lexical search: %w", err)
	}
	rows.Close()
	if len(request.Embedding) > 0 {
		if len(request.Embedding) != 1536 {
			return nil, knowledge.ErrInvalid
		}
		rows, err = s.pool.Query(ctx, `SELECT id,source_id,snapshot_id,heading,content,language,
			start_offset,end_offset,parser_version,observed_at,current,
			row_number() OVER (ORDER BY embedding <=> $2,id)
			FROM knowledge_chunks WHERE project_id=$1 AND current AND observed_at<=$3
			AND (cardinality($4::bigint[])=0 OR source_id=ANY($4)) AND embedding IS NOT NULL
			ORDER BY embedding <=> $2,id LIMIT 200`, projectID, pgvector.NewVector(request.Embedding),
			request.Cutoff, sourceIDs)
		if err != nil {
			return nil, fmt.Errorf("vector knowledge search: %w", err)
		}
		for rows.Next() {
			chunk, rank, scanErr := scanSearchChunk(rows, projectID)
			if scanErr != nil {
				rows.Close()
				return nil, scanErr
			}
			candidate := byID[chunk.ID]
			candidate.Chunk, candidate.VectorRank = chunk, rank
			byID[chunk.ID] = candidate
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, fmt.Errorf("iterate vector search: %w", err)
		}
		rows.Close()
	}
	candidates := make([]knowledge.Candidate, 0, len(byID))
	for _, candidate := range byID {
		candidates = append(candidates, candidate)
	}
	return knowledge.Fuse(candidates, knowledge.Filter{ProjectID: projectID,
		SourceIDs: sourceIDs, Cutoff: request.Cutoff}, request.Limit, 60)
}

func scanSearchChunk(row interface{ Scan(...any) error }, projectID int64) (knowledge.Chunk, int, error) {
	var chunk knowledge.Chunk
	var rank int
	err := row.Scan(&chunk.ID, &chunk.SourceID, &chunk.SnapshotID, &chunk.Heading, &chunk.Text,
		&chunk.Language, &chunk.StartOffset, &chunk.EndOffset, &chunk.ParserVersion,
		&chunk.ObservedAt, &chunk.Current, &rank)
	if err != nil {
		return knowledge.Chunk{}, 0, fmt.Errorf("scan knowledge result: %w", err)
	}
	chunk.ProjectID = projectID
	return chunk, rank, nil
}

func (s *Store) Topics(ctx context.Context, principal access.Principal, projectID int64,
	limit, offset int) ([]topic.Canonical, error) {
	if err := s.authorizeProject(ctx, principal, projectID); err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `SELECT t.id,t.candidate_set_id,c.algorithm_version,t.generated_label,
		t.label_run_id,t.created_at,t.retired_at FROM topics t JOIN topic_candidate_sets c ON c.id=t.candidate_set_id
		WHERE t.project_id=$1 ORDER BY (t.retired_at IS NULL) DESC,t.id LIMIT $2 OFFSET $3`, projectID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("load topics: %w", err)
	}
	defer rows.Close()
	type raw struct{ candidate topic.Candidate }
	rawValues := make([]raw, 0)
	for rows.Next() {
		var value raw
		if err := rows.Scan(&value.candidate.ID, new(int64), &value.candidate.AlgorithmVersion,
			&value.candidate.GeneratedLabel, &value.candidate.LabelRunID, &value.candidate.CreatedAt,
			&value.candidate.RetiredAt); err != nil {
			return nil, fmt.Errorf("scan topic: %w", err)
		}
		value.candidate.ProjectID = projectID
		rawValues = append(rawValues, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate topics: %w", err)
	}
	rows.Close()
	values := make([]topic.Canonical, 0, len(rawValues))
	for _, rawValue := range rawValues {
		candidate := rawValue.candidate
		memberRows, err := s.pool.Query(ctx, `SELECT issue_id FROM topic_members WHERE topic_id=$1 ORDER BY ordinal`, candidate.ID)
		if err != nil {
			return nil, fmt.Errorf("load topic members: %w", err)
		}
		candidate.Members, err = pgx.CollectRows(memberRows, pgx.RowTo[int64])
		if err != nil {
			return nil, fmt.Errorf("scan topic members: %w", err)
		}
		corrections, err := s.topicCorrections(ctx, candidate.ID, projectID)
		if err != nil {
			return nil, err
		}
		value, err := topic.Apply(candidate, corrections)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

func (s *Store) topicCorrections(ctx context.Context, topicID, projectID int64) ([]topic.Correction, error) {
	rows, err := s.pool.Query(ctx, `SELECT id,project_id,topic_id,action,issue_ids,other_topic_ids,label,
		reason,actor_id,request_id,version,created_at FROM topic_corrections
		WHERE topic_id=$1 ORDER BY version`, topicID)
	if err != nil {
		return nil, fmt.Errorf("load topic corrections: %w", err)
	}
	defer rows.Close()
	values := make([]topic.Correction, 0)
	for rows.Next() {
		var value topic.Correction
		if err := rows.Scan(&value.ID, &value.ProjectID, &value.TopicID, &value.Action,
			&value.IssueIDs, &value.OtherTopicIDs, &value.Label, &value.Reason, &value.ActorID,
			&value.RequestID, &value.Version, &value.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan topic correction: %w", err)
		}
		if value.ProjectID != projectID {
			return nil, access.ErrNotFound
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate topic corrections: %w", err)
	}
	return values, nil
}

func (s *Store) CorrectTopic(ctx context.Context, principal access.Principal,
	value topic.Correction) (topic.Correction, error) {
	if err := access.Authorize(principal, access.ActionProjectWrite); err != nil {
		return topic.Correction{}, err
	}
	if err := s.authorizeProject(ctx, principal, value.ProjectID); err != nil {
		return topic.Correction{}, err
	}
	if value.ID == 0 {
		id, err := s.ids.Next(ctx)
		if err != nil {
			return topic.Correction{}, err
		}
		value.ID = id
	}
	value.ActorID, value.CreatedAt = principal.ActorID, time.Now().UTC()
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return topic.Correction{}, fmt.Errorf("begin topic correction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, value.TopicID); err != nil {
		return topic.Correction{}, fmt.Errorf("lock topic correction version: %w", err)
	}
	var actualProjectID int64
	if err := tx.QueryRow(ctx, `SELECT project_id FROM topics WHERE id=$1`, value.TopicID).Scan(&actualProjectID); errors.Is(err, pgx.ErrNoRows) {
		return topic.Correction{}, access.ErrNotFound
	} else if err != nil {
		return topic.Correction{}, fmt.Errorf("load corrected topic: %w", err)
	} else if actualProjectID != value.ProjectID {
		return topic.Correction{}, access.ErrNotFound
	}
	var nextVersion int64
	if err := tx.QueryRow(ctx, `SELECT COALESCE(max(version),0)+1 FROM topic_corrections WHERE topic_id=$1`, value.TopicID).Scan(&nextVersion); err != nil {
		return topic.Correction{}, fmt.Errorf("read topic correction version: %w", err)
	}
	if value.Version == 0 {
		value.Version = nextVersion
	} else if value.Version != nextVersion {
		return topic.Correction{}, access.ErrVersionConflict
	}
	if err := value.Validate(); err != nil {
		return topic.Correction{}, err
	}
	if value.IssueIDs == nil {
		value.IssueIDs = []int64{}
	}
	if value.OtherTopicIDs == nil {
		value.OtherTopicIDs = []int64{}
	}
	_, err = tx.Exec(ctx, `INSERT INTO topic_corrections
		(id,project_id,topic_id,action,issue_ids,other_topic_ids,label,reason,actor_id,request_id,version,created_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`, value.ID, value.ProjectID,
		value.TopicID, value.Action, value.IssueIDs, value.OtherTopicIDs, value.Label, value.Reason,
		value.ActorID, value.RequestID, value.Version, value.CreatedAt)
	if err != nil {
		return topic.Correction{}, fmt.Errorf("insert topic correction: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return topic.Correction{}, fmt.Errorf("commit topic correction: %w", err)
	}
	return value, nil
}

func (s *Store) Releases(ctx context.Context, principal access.Principal, projectID int64,
	limit, offset int) ([]release.Intelligence, error) {
	if err := s.authorizeProject(ctx, principal, projectID); err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `SELECT id,project_id,repository_id,source_id,external_id,tag,title,
		body,language,canonical_url,published_at,prerelease,state,withdrawn_at,raw_object_id,
		changelog_snapshot_id FROM canonical_releases WHERE project_id=$1
		ORDER BY published_at DESC,id DESC LIMIT $2 OFFSET $3`, projectID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("load releases: %w", err)
	}
	defer rows.Close()
	values := make([]release.Intelligence, 0)
	for rows.Next() {
		var value release.Intelligence
		if err := rows.Scan(&value.ID, &value.ProjectID, &value.RepositoryID, &value.SourceID,
			&value.ExternalID, &value.Tag, &value.Title, &value.Body, &value.Language, &value.URL,
			&value.PublishedAt, &value.Prerelease, &value.State, &value.WithdrawnAt,
			&value.EvidenceID, &value.ChangelogID); err != nil {
			return nil, fmt.Errorf("scan release: %w", err)
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate releases: %w", err)
	}
	return values, nil
}

func (s *Store) Release(ctx context.Context, principal access.Principal,
	projectID, releaseID int64) (release.Intelligence, error) {
	if err := s.authorizeProject(ctx, principal, projectID); err != nil {
		return release.Intelligence{}, err
	}
	var value release.Intelligence
	err := s.pool.QueryRow(ctx, `SELECT id,project_id,repository_id,source_id,external_id,tag,title,
		body,language,canonical_url,published_at,prerelease,state,withdrawn_at,raw_object_id,
		changelog_snapshot_id FROM canonical_releases WHERE project_id=$1 AND id=$2`,
		projectID, releaseID).Scan(&value.ID, &value.ProjectID, &value.RepositoryID, &value.SourceID,
		&value.ExternalID, &value.Tag, &value.Title, &value.Body, &value.Language, &value.URL,
		&value.PublishedAt, &value.Prerelease, &value.State, &value.WithdrawnAt,
		&value.EvidenceID, &value.ChangelogID)
	if errors.Is(err, pgx.ErrNoRows) {
		return release.Intelligence{}, access.ErrNotFound
	}
	if err != nil {
		return release.Intelligence{}, fmt.Errorf("load release: %w", err)
	}
	return value, nil
}

func (s *Store) SaveRun(ctx context.Context, run analysis.Run) (analysis.Run, error) {
	if err := run.ValidateForPersistence(); err != nil {
		return analysis.Run{}, err
	}
	output := any(nil)
	if len(run.Output) > 0 {
		output = run.Output
	}
	usage, err := json.Marshal(run.Usage)
	if err != nil {
		return analysis.Run{}, fmt.Errorf("encode analysis usage: %w", err)
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return analysis.Run{}, fmt.Errorf("begin analysis run: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if run.ParentRunID != nil {
		var parentExists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM analysis_runs
			WHERE id=$1 AND project_id=$2 AND series_id=$3)`, *run.ParentRunID, run.ProjectID,
			run.SeriesID).Scan(&parentExists); err != nil {
			return analysis.Run{}, fmt.Errorf("validate analysis parent: %w", err)
		}
		if !parentExists {
			return analysis.Run{}, fmt.Errorf("%w: analysis parent is outside the series", analysis.ErrInvalidRun)
		}
	}
	for _, citation := range run.Evidence {
		var accessible bool
		err := tx.QueryRow(ctx, `SELECT EXISTS(
			SELECT 1 FROM knowledge_chunks chunk
			JOIN document_snapshots snapshot ON snapshot.id=chunk.snapshot_id
			WHERE chunk.id=$1 AND chunk.project_id=$2 AND chunk.snapshot_id=$3
				AND snapshot.project_id=$2 AND snapshot.source_id=chunk.source_id
				AND chunk.observed_at<=$4 AND snapshot.observed_at<=$4
				AND chunk.start_offset<=$5 AND chunk.end_offset>=$6)`, citation.ChunkID,
			run.ProjectID, citation.SnapshotID, run.Cutoff, citation.StartOffset,
			citation.EndOffset).Scan(&accessible)
		if err != nil {
			return analysis.Run{}, fmt.Errorf("validate analysis citation: %w", err)
		}
		if !accessible {
			return analysis.Run{}, fmt.Errorf("%w: citation is outside the project or cutoff", analysis.ErrEvidence)
		}
	}
	var persistedID int64
	err = tx.QueryRow(ctx, `INSERT INTO analysis_series(id,project_id,subject_kind,subject_id)
		VALUES($1,$2,$3,$1) ON CONFLICT(id) DO UPDATE SET id=EXCLUDED.id
		WHERE analysis_series.project_id=EXCLUDED.project_id
			AND analysis_series.subject_kind=EXCLUDED.subject_kind
			AND analysis_series.subject_id=EXCLUDED.subject_id
		RETURNING id`, run.SeriesID, run.ProjectID, run.Kind).Scan(&persistedID)
	if errors.Is(err, pgx.ErrNoRows) {
		return analysis.Run{}, fmt.Errorf("%w: analysis series identity collision", analysis.ErrInvalidRun)
	}
	if err != nil {
		return analysis.Run{}, fmt.Errorf("ensure analysis series: %w", err)
	}
	err = tx.QueryRow(ctx, `INSERT INTO analysis_runs(id,series_id,project_id,parent_run_id,kind,state,
		prompt_version,schema_version,retrieval_version,provider,model,language,requested_by,reason,
		cutoff,output,usage,failure_code,created_at,started_at,finished_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21)
		ON CONFLICT(id) DO UPDATE SET id=EXCLUDED.id
		WHERE analysis_runs.series_id=EXCLUDED.series_id
			AND analysis_runs.project_id=EXCLUDED.project_id
			AND analysis_runs.parent_run_id IS NOT DISTINCT FROM EXCLUDED.parent_run_id
			AND analysis_runs.kind=EXCLUDED.kind
			AND analysis_runs.state=EXCLUDED.state
			AND analysis_runs.prompt_version=EXCLUDED.prompt_version
			AND analysis_runs.schema_version=EXCLUDED.schema_version
			AND analysis_runs.retrieval_version=EXCLUDED.retrieval_version
			AND analysis_runs.provider=EXCLUDED.provider
			AND analysis_runs.model=EXCLUDED.model
			AND analysis_runs.language=EXCLUDED.language
			AND analysis_runs.requested_by=EXCLUDED.requested_by
			AND analysis_runs.reason=EXCLUDED.reason
			AND analysis_runs.cutoff=EXCLUDED.cutoff
			AND analysis_runs.output IS NOT DISTINCT FROM EXCLUDED.output
			AND analysis_runs.usage=EXCLUDED.usage
			AND analysis_runs.failure_code IS NOT DISTINCT FROM EXCLUDED.failure_code
			AND analysis_runs.created_at=EXCLUDED.created_at
			AND analysis_runs.started_at IS NOT DISTINCT FROM EXCLUDED.started_at
			AND analysis_runs.finished_at IS NOT DISTINCT FROM EXCLUDED.finished_at
		RETURNING id`, run.ID, run.SeriesID, run.ProjectID,
		run.ParentRunID, run.Kind, run.State, run.PromptVersion, run.SchemaVersion,
		run.RetrievalVersion, run.Provider, run.Model, run.Language, run.RequestedBy, run.Reason,
		run.Cutoff, output, usage, run.FailureCode, run.CreatedAt, run.StartedAt,
		run.FinishedAt).Scan(&persistedID)
	if errors.Is(err, pgx.ErrNoRows) {
		return analysis.Run{}, fmt.Errorf("%w: immutable analysis run replay differs", analysis.ErrInvalidRun)
	}
	if err != nil {
		return analysis.Run{}, fmt.Errorf("insert analysis run: %w", err)
	}
	for index, citation := range run.Evidence {
		err = tx.QueryRow(ctx, `INSERT INTO analysis_run_citations
			(run_id,ordinal,snapshot_id,chunk_id,start_offset,end_offset) VALUES($1,$2,$3,$4,$5,$6)
			ON CONFLICT(run_id,ordinal) DO UPDATE SET run_id=EXCLUDED.run_id
			WHERE analysis_run_citations.snapshot_id=EXCLUDED.snapshot_id
				AND analysis_run_citations.chunk_id=EXCLUDED.chunk_id
				AND analysis_run_citations.start_offset=EXCLUDED.start_offset
				AND analysis_run_citations.end_offset=EXCLUDED.end_offset
			RETURNING run_id`, run.ID, index, citation.SnapshotID, citation.ChunkID,
			citation.StartOffset, citation.EndOffset).Scan(&persistedID)
		if errors.Is(err, pgx.ErrNoRows) {
			return analysis.Run{}, fmt.Errorf("%w: immutable analysis citation replay differs", analysis.ErrInvalidRun)
		}
		if err != nil {
			return analysis.Run{}, fmt.Errorf("insert analysis citation: %w", err)
		}
	}
	var citationCount int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM analysis_run_citations WHERE run_id=$1`,
		run.ID).Scan(&citationCount); err != nil {
		return analysis.Run{}, fmt.Errorf("count analysis citations: %w", err)
	}
	if citationCount != len(run.Evidence) {
		return analysis.Run{}, fmt.Errorf("%w: immutable analysis citation count differs", analysis.ErrInvalidRun)
	}
	if err := tx.Commit(ctx); err != nil {
		return analysis.Run{}, fmt.Errorf("commit analysis run: %w", err)
	}
	return run, nil
}

// QueueRun authorizes and persists a new immutable queued analysis version.
func (s *Store) QueueRun(ctx context.Context, principal access.Principal,
	run analysis.Run) (analysis.Run, error) {
	if err := access.Authorize(principal, access.ActionProjectWrite); err != nil {
		return analysis.Run{}, err
	}
	if err := s.authorizeProject(ctx, principal, run.ProjectID); err != nil {
		return analysis.Run{}, err
	}
	return s.SaveRun(ctx, run)
}

func (s *Store) Run(ctx context.Context, principal access.Principal, runID int64) (analysis.Run, error) {
	var run analysis.Run
	var output []byte
	var usage []byte
	err := s.pool.QueryRow(ctx, `SELECT id,series_id,project_id,parent_run_id,kind,state,prompt_version,
		schema_version,retrieval_version,provider,model,language,requested_by,reason,
		cutoff,created_at,started_at,finished_at,
		COALESCE(output,'null'::jsonb),usage,failure_code FROM analysis_runs WHERE id=$1`, runID).
		Scan(&run.ID, &run.SeriesID, &run.ProjectID, &run.ParentRunID, &run.Kind, &run.State,
			&run.PromptVersion, &run.SchemaVersion, &run.RetrievalVersion, &run.Provider, &run.Model,
			&run.Language, &run.RequestedBy, &run.Reason, &run.Cutoff, &run.CreatedAt,
			&run.StartedAt, &run.FinishedAt, &output, &usage, &run.FailureCode)
	if errors.Is(err, pgx.ErrNoRows) {
		return analysis.Run{}, access.ErrNotFound
	}
	if err != nil {
		return analysis.Run{}, fmt.Errorf("load analysis run: %w", err)
	}
	if err := s.authorizeProject(ctx, principal, run.ProjectID); err != nil {
		return analysis.Run{}, err
	}
	if string(output) != "null" {
		run.Output = slices.Clone(output)
	}
	if err := json.Unmarshal(usage, &run.Usage); err != nil {
		return analysis.Run{}, fmt.Errorf("decode analysis usage: %w", err)
	}
	rows, err := s.pool.Query(ctx, `SELECT snapshot_id,chunk_id,start_offset,end_offset
		FROM analysis_run_citations WHERE run_id=$1 ORDER BY ordinal`, run.ID)
	if err != nil {
		return analysis.Run{}, fmt.Errorf("load analysis citations: %w", err)
	}
	run.Evidence, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (knowledge.Citation, error) {
		var value knowledge.Citation
		err := row.Scan(&value.SnapshotID, &value.ChunkID, &value.StartOffset, &value.EndOffset)
		return value, err
	})
	if err != nil {
		return analysis.Run{}, fmt.Errorf("scan analysis citations: %w", err)
	}
	return run, nil
}

func (s *Store) SaveFeedback(ctx context.Context, principal access.Principal,
	value analysis.Feedback) (analysis.Feedback, error) {
	run, err := s.Run(ctx, principal, value.RunID)
	if err != nil {
		return analysis.Feedback{}, err
	}
	if err := access.Authorize(principal, access.ActionProjectWrite); err != nil {
		return analysis.Feedback{}, err
	}
	if value.ID == 0 {
		value.ID, err = s.ids.Next(ctx)
		if err != nil {
			return analysis.Feedback{}, err
		}
	}
	value.ActorID, value.CreatedAt = principal.ActorID, time.Now().UTC()
	if err := value.Validate(run); err != nil {
		return analysis.Feedback{}, err
	}
	err = s.pool.QueryRow(ctx, `INSERT INTO analysis_feedback(id,run_id,actor_id,rating,note,request_id,created_at)
		VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT(run_id,actor_id,request_id) DO UPDATE
		SET rating=analysis_feedback.rating
		WHERE analysis_feedback.rating=EXCLUDED.rating AND analysis_feedback.note=EXCLUDED.note
		RETURNING id,rating,note,created_at`, value.ID, value.RunID,
		value.ActorID, value.Rating, value.Note, value.RequestID, value.CreatedAt).
		Scan(&value.ID, &value.Rating, &value.Note, &value.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return analysis.Feedback{}, fmt.Errorf("%w: feedback replay differs", analysis.ErrInvalidRun)
	}
	if err != nil {
		return analysis.Feedback{}, fmt.Errorf("save analysis feedback: %w", err)
	}
	return value, nil
}

func (s *Store) SelectRun(ctx context.Context, principal access.Principal,
	value analysis.Selection) (analysis.Selection, error) {
	run, err := s.Run(ctx, principal, value.RunID)
	if err != nil {
		return analysis.Selection{}, err
	}
	if err := access.Authorize(principal, access.ActionProjectWrite); err != nil {
		return analysis.Selection{}, err
	}
	if run.State != analysis.StateSucceeded || run.SeriesID != value.SeriesID {
		return analysis.Selection{}, analysis.ErrSelection
	}
	if value.ID == 0 {
		value.ID, err = s.ids.Next(ctx)
		if err != nil {
			return analysis.Selection{}, err
		}
	}
	value.ActorID, value.SelectedAt = principal.ActorID, time.Now().UTC()
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return analysis.Selection{}, fmt.Errorf("begin analysis selection: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, value.SeriesID); err != nil {
		return analysis.Selection{}, fmt.Errorf("lock analysis selection: %w", err)
	}
	var nextVersion int64
	if err := tx.QueryRow(ctx, `SELECT COALESCE(max(version),0)+1 FROM analysis_selections
		WHERE series_id=$1`, value.SeriesID).Scan(&nextVersion); err != nil {
		return analysis.Selection{}, fmt.Errorf("read analysis selection version: %w", err)
	}
	if value.Version == 0 {
		value.Version = nextVersion
	} else if value.Version != nextVersion {
		return analysis.Selection{}, access.ErrVersionConflict
	}
	_, err = tx.Exec(ctx, `INSERT INTO analysis_selections
		(id,series_id,run_id,actor_id,version,request_id,selected_at) VALUES($1,$2,$3,$4,$5,$6,$7)`,
		value.ID, value.SeriesID, value.RunID, value.ActorID, value.Version, value.RequestID, value.SelectedAt)
	if err != nil {
		return analysis.Selection{}, fmt.Errorf("insert analysis selection: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return analysis.Selection{}, fmt.Errorf("commit analysis selection: %w", err)
	}
	return value, nil
}
