DROP TRIGGER IF EXISTS analysis_runs_project_write_guard ON analysis_runs;
DROP TRIGGER IF EXISTS analysis_series_project_write_guard ON analysis_series;
DROP TRIGGER IF EXISTS topic_corrections_project_write_guard ON topic_corrections;
DROP TRIGGER IF EXISTS topics_project_write_guard ON topics;
DROP TRIGGER IF EXISTS topic_candidate_sets_project_write_guard ON topic_candidate_sets;
DROP TRIGGER IF EXISTS knowledge_chunks_project_write_guard ON knowledge_chunks;
DROP TRIGGER IF EXISTS document_snapshots_project_write_guard ON document_snapshots;
DROP TRIGGER IF EXISTS public_advisories_project_write_guard ON public_advisories;
DROP TRIGGER IF EXISTS registry_adoption_snapshots_project_write_guard ON registry_adoption_snapshots;
DROP TABLE IF EXISTS analysis_selections;
DROP TABLE IF EXISTS analysis_feedback;
DROP TABLE IF EXISTS analysis_run_citations;
ALTER TABLE IF EXISTS topics DROP CONSTRAINT IF EXISTS topics_label_run_fk;
DROP TABLE IF EXISTS analysis_runs;
DROP TABLE IF EXISTS analysis_series;
DROP INDEX IF EXISTS canonical_releases_project_history_extended_idx;
ALTER TABLE IF EXISTS canonical_releases
    DROP CONSTRAINT IF EXISTS canonical_releases_withdrawn_time_check,
    DROP CONSTRAINT IF EXISTS canonical_releases_withdrawn_check,
    DROP CONSTRAINT IF EXISTS canonical_releases_state_check,
    DROP CONSTRAINT IF EXISTS canonical_releases_url_check,
    DROP COLUMN IF EXISTS changelog_snapshot_id,
    DROP COLUMN IF EXISTS withdrawn_at,
    DROP COLUMN IF EXISTS state,
    DROP COLUMN IF EXISTS canonical_url,
    DROP COLUMN IF EXISTS language,
    DROP COLUMN IF EXISTS body,
    DROP COLUMN IF EXISTS title;
DROP TABLE IF EXISTS topic_corrections;
DROP TABLE IF EXISTS topic_members;
DROP TABLE IF EXISTS topics;
DROP TABLE IF EXISTS topic_candidate_sets;
DROP TABLE IF EXISTS knowledge_chunks;
DROP TABLE IF EXISTS document_snapshots;
DROP TABLE IF EXISTS public_advisories;
DROP TABLE IF EXISTS registry_adoption_snapshots;
