DROP TRIGGER IF EXISTS source_events_project_write_guard ON source_events;
DROP TRIGGER IF EXISTS canonical_releases_project_write_guard ON canonical_releases;
DROP TRIGGER IF EXISTS canonical_pull_requests_project_write_guard ON canonical_pull_requests;
DROP TRIGGER IF EXISTS canonical_issues_project_write_guard ON canonical_issues;
DROP TRIGGER IF EXISTS canonical_commits_project_write_guard ON canonical_commits;
DROP TRIGGER IF EXISTS raw_objects_project_write_guard ON raw_objects;
DROP TRIGGER IF EXISTS sources_project_write_guard ON sources;
DROP TRIGGER IF EXISTS repositories_project_write_guard ON repositories;
DROP FUNCTION IF EXISTS reject_deleted_project_write();
DROP TABLE IF EXISTS project_tombstones;
DROP TABLE IF EXISTS purge_manifest_objects;
DROP TABLE IF EXISTS purge_manifests;
ALTER TABLE outbox_events DROP COLUMN IF EXISTS last_error, DROP COLUMN IF EXISTS available_at,
    DROP COLUMN IF EXISTS attempts, DROP COLUMN IF EXISTS causation_id, DROP COLUMN IF EXISTS correlation_id,
    DROP COLUMN IF EXISTS job_id;
DROP TABLE IF EXISTS source_events;
DROP TABLE IF EXISTS canonical_releases;
DROP TABLE IF EXISTS canonical_pull_requests;
DROP TABLE IF EXISTS canonical_issues;
DROP TABLE IF EXISTS canonical_commits;
DROP TABLE IF EXISTS raw_objects;
DROP INDEX IF EXISTS object_references_project_idx;
ALTER TABLE object_references DROP COLUMN IF EXISTS verified_at, DROP COLUMN IF EXISTS provenance,
    DROP COLUMN IF EXISTS project_id;
DROP TABLE IF EXISTS sync_checkpoints;
DROP TABLE IF EXISTS job_events;
DROP TABLE IF EXISTS job_attempts;
ALTER TABLE idempotency_records DROP CONSTRAINT IF EXISTS idempotency_records_job_fk;
DROP INDEX IF EXISTS jobs_project_idx;
DROP INDEX IF EXISTS jobs_claim_idx;
DROP INDEX IF EXISTS jobs_active_coalescing_idx;
ALTER TABLE jobs DROP COLUMN IF EXISTS failure_detail, DROP COLUMN IF EXISTS failure_code,
    DROP COLUMN IF EXISTS max_attempts, DROP COLUMN IF EXISTS heartbeat_at,
    DROP COLUMN IF EXISTS lease_expires_at, DROP COLUMN IF EXISTS lease_holder,
    DROP COLUMN IF EXISTS finished_at, DROP COLUMN IF EXISTS started_at,
    DROP COLUMN IF EXISTS available_at, DROP COLUMN IF EXISTS causation_id,
    DROP COLUMN IF EXISTS correlation_id, DROP COLUMN IF EXISTS request_id,
    DROP COLUMN IF EXISTS requested_by, DROP COLUMN IF EXISTS cancellable,
    DROP COLUMN IF EXISTS coalesced_requests, DROP COLUMN IF EXISTS coalescing_key,
    DROP COLUMN IF EXISTS progress, DROP COLUMN IF EXISTS version,
    DROP COLUMN IF EXISTS project_id, DROP COLUMN IF EXISTS workspace_id;
DROP TABLE IF EXISTS idempotency_records;
DROP TABLE IF EXISTS identity_corrections;
DROP TABLE IF EXISTS source_associations;
DROP TABLE IF EXISTS sources;
DROP INDEX IF EXISTS repositories_provider_external_idx;
DROP TABLE IF EXISTS repositories;
DROP TABLE IF EXISTS projects;
