CREATE TABLE projects (
    id bigint PRIMARY KEY CHECK (id > 0),
    workspace_id bigint NOT NULL REFERENCES workspaces (id),
    name text NOT NULL CHECK (btrim(name) <> ''),
    slug text NOT NULL CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$'),
    description text NOT NULL DEFAULT '',
    state text NOT NULL DEFAULT 'active'
        CHECK (state IN ('active', 'paused', 'archived', 'deleting', 'deleted')),
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    unavailable_at timestamptz,
    deleted_at timestamptz,
    deletion_actor_id bigint,
    deletion_reason text NOT NULL DEFAULT '',
    UNIQUE (workspace_id, slug),
    CHECK ((state IN ('deleting', 'deleted')) = (unavailable_at IS NOT NULL)),
    CHECK ((state = 'deleted') = (deleted_at IS NOT NULL))
);

CREATE INDEX projects_portfolio_idx ON projects (workspace_id, state, updated_at DESC, id DESC);

CREATE TABLE repositories (
    id bigint PRIMARY KEY CHECK (id > 0),
    project_id bigint NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    provider text NOT NULL CHECK (provider IN ('github', 'gitlab', 'gitea', 'git')),
    external_id text,
    canonical_url text NOT NULL CHECK (canonical_url ~ '^https://'),
    role text NOT NULL
        CHECK (role IN ('primary', 'core', 'documentation', 'examples', 'sdk', 'other')),
    default_branch text NOT NULL DEFAULT '',
    public boolean NOT NULL DEFAULT true CHECK (public),
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (canonical_url)
);

CREATE UNIQUE INDEX repositories_one_primary_idx
    ON repositories (project_id) WHERE role = 'primary';
CREATE UNIQUE INDEX repositories_provider_external_idx
    ON repositories (provider, external_id) WHERE external_id IS NOT NULL;
CREATE INDEX repositories_project_idx ON repositories (project_id, role, id);

CREATE TABLE sources (
    id bigint PRIMARY KEY CHECK (id > 0),
    project_id bigint NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    repository_id bigint REFERENCES repositories (id) ON DELETE CASCADE,
    kind text NOT NULL CHECK (kind IN (
        'github', 'gitlab', 'gitea', 'git', 'npm', 'nuget', 'pypi', 'docs', 'website',
        'changelog', 'rss', 'discussion', 'advisory'
    )),
    canonical_url text NOT NULL CHECK (canonical_url ~ '^https://'),
    state text NOT NULL DEFAULT 'available'
        CHECK (state IN ('available', 'unavailable', 'paused', 'removed')),
    public boolean NOT NULL DEFAULT true,
    collection_limits jsonb NOT NULL DEFAULT '{}'::jsonb,
    coverage_from timestamptz,
    coverage_to timestamptz,
    last_attempt_at timestamptz,
    last_success_at timestamptz,
    next_run_at timestamptz,
    failure_code text NOT NULL DEFAULT '',
    quota jsonb NOT NULL DEFAULT '{}'::jsonb,
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (project_id, kind, canonical_url),
    CHECK (coverage_from IS NULL OR coverage_to IS NULL OR coverage_from <= coverage_to),
    CHECK (state <> 'available' OR public)
);

CREATE INDEX sources_project_state_idx ON sources (project_id, state, kind, id);

CREATE TABLE source_associations (
    id bigint PRIMARY KEY CHECK (id > 0),
    source_id bigint NOT NULL REFERENCES sources (id) ON DELETE CASCADE,
    project_id bigint NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    method text NOT NULL CHECK (btrim(method) <> ''),
    confidence double precision NOT NULL CHECK (confidence BETWEEN 0 AND 1),
    evidence_ids bigint[] NOT NULL DEFAULT '{}',
    decision_version text NOT NULL CHECK (btrim(decision_version) <> ''),
    status text NOT NULL CHECK (status IN ('linked', 'unresolved', 'corrected')),
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (source_id)
);

CREATE TABLE identity_corrections (
    id bigint PRIMARY KEY CHECK (id > 0),
    association_id bigint NOT NULL REFERENCES source_associations (id) ON DELETE CASCADE,
    action text NOT NULL CHECK (action IN ('split', 'reassign', 'confirm')),
    from_project_id bigint REFERENCES projects (id) ON DELETE SET NULL,
    to_project_id bigint REFERENCES projects (id) ON DELETE SET NULL,
    reason text NOT NULL CHECK (btrim(reason) <> ''),
    actor_id bigint NOT NULL CHECK (actor_id > 0),
    request_id text NOT NULL CHECK (btrim(request_id) <> ''),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE NULLS NOT DISTINCT (association_id, action, to_project_id)
);

CREATE TABLE idempotency_records (
    workspace_id bigint NOT NULL REFERENCES workspaces (id),
    actor_id bigint NOT NULL CHECK (actor_id > 0),
    route text NOT NULL CHECK (btrim(route) <> ''),
    idempotency_key text NOT NULL CHECK (btrim(idempotency_key) <> ''),
    request_digest bytea NOT NULL CHECK (octet_length(request_digest) = 32),
    response_status integer,
    response_headers jsonb NOT NULL DEFAULT '{}'::jsonb,
    response_body jsonb,
    resource_id bigint,
    job_id bigint,
    created_at timestamptz NOT NULL DEFAULT now(),
    completed_at timestamptz,
    PRIMARY KEY (workspace_id, actor_id, route, idempotency_key)
);

ALTER TABLE jobs
    ADD COLUMN workspace_id bigint REFERENCES workspaces (id),
    ADD COLUMN project_id bigint REFERENCES projects (id) ON DELETE SET NULL,
    ADD COLUMN version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    ADD COLUMN progress jsonb NOT NULL DEFAULT '{"completed":0,"total_status":"unknown","unit":"items"}'::jsonb,
    ADD COLUMN requested_from timestamptz,
    ADD COLUMN requested_to timestamptz,
    ADD COLUMN coalescing_key text,
    ADD COLUMN coalesced_requests bigint NOT NULL DEFAULT 0 CHECK (coalesced_requests >= 0),
    ADD COLUMN cancellable boolean NOT NULL DEFAULT true,
    ADD COLUMN requested_by bigint,
    ADD COLUMN request_id text NOT NULL DEFAULT '',
    ADD COLUMN correlation_id text NOT NULL DEFAULT '',
    ADD COLUMN causation_id text NOT NULL DEFAULT '',
    ADD COLUMN available_at timestamptz NOT NULL DEFAULT now(),
    ADD COLUMN started_at timestamptz,
    ADD COLUMN finished_at timestamptz,
    ADD COLUMN lease_holder text,
    ADD COLUMN lease_expires_at timestamptz,
    ADD COLUMN heartbeat_at timestamptz,
    ADD COLUMN max_attempts integer NOT NULL DEFAULT 5 CHECK (max_attempts > 0),
    ADD COLUMN failure_code text NOT NULL DEFAULT '',
    ADD COLUMN failure_detail text NOT NULL DEFAULT '';

ALTER TABLE jobs
    ADD CONSTRAINT jobs_requested_range_ordered
    CHECK (requested_from IS NULL OR requested_to IS NULL OR requested_from < requested_to);

CREATE UNIQUE INDEX jobs_active_coalescing_idx
    ON jobs (workspace_id, coalescing_key)
    WHERE coalescing_key IS NOT NULL AND state IN ('queued', 'running');
CREATE INDEX jobs_claim_idx ON jobs (available_at, created_at, id)
    WHERE state = 'queued';
CREATE INDEX jobs_project_idx ON jobs (project_id, created_at DESC, id DESC);

ALTER TABLE idempotency_records
    ADD CONSTRAINT idempotency_records_job_fk FOREIGN KEY (job_id) REFERENCES jobs (id);

CREATE TABLE job_attempts (
    id bigint PRIMARY KEY CHECK (id > 0),
    job_id bigint NOT NULL REFERENCES jobs (id) ON DELETE CASCADE,
    attempt integer NOT NULL CHECK (attempt > 0),
    worker_id text NOT NULL CHECK (btrim(worker_id) <> ''),
    state text NOT NULL CHECK (state IN ('running', 'succeeded', 'failed', 'cancelled')),
    started_at timestamptz NOT NULL DEFAULT now(),
    finished_at timestamptz,
    failure_code text NOT NULL DEFAULT '',
    UNIQUE (job_id, attempt)
);

CREATE TABLE job_events (
    id bigint PRIMARY KEY CHECK (id > 0),
    job_id bigint NOT NULL REFERENCES jobs (id) ON DELETE CASCADE,
    job_version bigint NOT NULL CHECK (job_version > 0),
    representation jsonb NOT NULL,
    occurred_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (job_id, job_version)
);

CREATE INDEX job_events_resume_idx ON job_events (job_id, id);

CREATE TABLE sync_checkpoints (
    id bigint PRIMARY KEY CHECK (id > 0),
    project_id bigint NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    source_id bigint NOT NULL REFERENCES sources (id) ON DELETE CASCADE,
    scope text NOT NULL CHECK (btrim(scope) <> ''),
    cursor text NOT NULL DEFAULT '',
    coverage_from timestamptz,
    coverage_to timestamptz,
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (source_id, scope),
    CHECK (coverage_from IS NULL OR coverage_to IS NULL OR coverage_from <= coverage_to)
);

ALTER TABLE object_references
    ADD COLUMN project_id bigint REFERENCES projects (id) ON DELETE CASCADE,
    ADD COLUMN provenance jsonb NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN verified_at timestamptz;

CREATE INDEX object_references_project_idx ON object_references (project_id, retention_state, id);

CREATE TABLE raw_objects (
    id bigint PRIMARY KEY CHECK (id > 0),
    project_id bigint NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    source_id bigint NOT NULL REFERENCES sources (id) ON DELETE CASCADE,
    external_type text NOT NULL CHECK (btrim(external_type) <> ''),
    external_id text NOT NULL CHECK (btrim(external_id) <> ''),
    observed_at timestamptz NOT NULL,
    payload jsonb,
    object_reference_id bigint REFERENCES object_references (id) ON DELETE RESTRICT,
    digest bytea NOT NULL CHECK (octet_length(digest) = 32),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (source_id, external_type, external_id, digest),
    CHECK ((payload IS NULL) <> (object_reference_id IS NULL))
);

CREATE TABLE canonical_commits (
    id bigint PRIMARY KEY CHECK (id > 0),
    project_id bigint NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    repository_id bigint NOT NULL REFERENCES repositories (id) ON DELETE CASCADE,
    source_id bigint NOT NULL REFERENCES sources (id) ON DELETE CASCADE,
    external_id text NOT NULL CHECK (btrim(external_id) <> ''),
    sha text NOT NULL CHECK (btrim(sha) <> ''),
    author_external_id text NOT NULL DEFAULT '',
    committed_at timestamptz NOT NULL,
    default_branch boolean NOT NULL,
    merge_commit boolean NOT NULL,
    raw_object_id bigint NOT NULL REFERENCES raw_objects (id) ON DELETE RESTRICT,
    UNIQUE (repository_id, sha)
);

CREATE TABLE canonical_issues (
    id bigint PRIMARY KEY CHECK (id > 0),
    project_id bigint NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    repository_id bigint REFERENCES repositories (id) ON DELETE CASCADE,
    source_id bigint NOT NULL REFERENCES sources (id) ON DELETE CASCADE,
    external_id text NOT NULL CHECK (btrim(external_id) <> ''),
    number bigint,
    title text NOT NULL DEFAULT '',
    state text NOT NULL CHECK (state IN ('open', 'closed')),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    closed_at timestamptz,
    raw_object_id bigint NOT NULL REFERENCES raw_objects (id) ON DELETE RESTRICT,
    UNIQUE (source_id, external_id)
);

CREATE TABLE canonical_pull_requests (
    id bigint PRIMARY KEY CHECK (id > 0),
    project_id bigint NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    repository_id bigint REFERENCES repositories (id) ON DELETE CASCADE,
    source_id bigint NOT NULL REFERENCES sources (id) ON DELETE CASCADE,
    external_id text NOT NULL CHECK (btrim(external_id) <> ''),
    number bigint,
    title text NOT NULL DEFAULT '',
    state text NOT NULL CHECK (state IN ('open', 'closed', 'merged')),
    created_at timestamptz NOT NULL,
    ready_at timestamptz,
    merged_at timestamptz,
    raw_object_id bigint NOT NULL REFERENCES raw_objects (id) ON DELETE RESTRICT,
    UNIQUE (source_id, external_id)
);

CREATE TABLE canonical_releases (
    id bigint PRIMARY KEY CHECK (id > 0),
    project_id bigint NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    repository_id bigint REFERENCES repositories (id) ON DELETE CASCADE,
    source_id bigint NOT NULL REFERENCES sources (id) ON DELETE CASCADE,
    external_id text NOT NULL CHECK (btrim(external_id) <> ''),
    tag text NOT NULL CHECK (btrim(tag) <> ''),
    draft boolean NOT NULL,
    prerelease boolean NOT NULL,
    published_at timestamptz,
    raw_object_id bigint NOT NULL REFERENCES raw_objects (id) ON DELETE RESTRICT,
    UNIQUE (source_id, external_id)
);

CREATE TABLE source_events (
    id bigint PRIMARY KEY CHECK (id > 0),
    project_id bigint NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    source_id bigint NOT NULL REFERENCES sources (id) ON DELETE CASCADE,
    aggregate_type text NOT NULL CHECK (btrim(aggregate_type) <> ''),
    aggregate_id bigint NOT NULL CHECK (aggregate_id > 0),
    external_id text NOT NULL CHECK (btrim(external_id) <> ''),
    kind text NOT NULL CHECK (btrim(kind) <> ''),
    occurred_at timestamptz NOT NULL,
    raw_object_id bigint NOT NULL REFERENCES raw_objects (id) ON DELETE RESTRICT,
    UNIQUE (source_id, external_id)
);

ALTER TABLE outbox_events
    ADD COLUMN job_id bigint REFERENCES jobs (id) ON DELETE CASCADE,
    ADD COLUMN correlation_id text NOT NULL DEFAULT '',
    ADD COLUMN causation_id text NOT NULL DEFAULT '',
    ADD COLUMN attempts integer NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    ADD COLUMN available_at timestamptz NOT NULL DEFAULT now(),
    ADD COLUMN last_error text NOT NULL DEFAULT '';

CREATE TABLE purge_manifests (
    id bigint PRIMARY KEY CHECK (id > 0),
    project_id bigint NOT NULL UNIQUE REFERENCES projects (id) ON DELETE CASCADE,
    job_id bigint NOT NULL UNIQUE REFERENCES jobs (id) ON DELETE CASCADE,
    state text NOT NULL CHECK (state IN ('pending', 'purging', 'reconciling', 'completed')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    completed_at timestamptz
);

CREATE TABLE purge_manifest_objects (
    manifest_id bigint NOT NULL REFERENCES purge_manifests (id) ON DELETE CASCADE,
    object_reference_id bigint NOT NULL REFERENCES object_references (id) ON DELETE CASCADE,
    deleted_at timestamptz,
    attempts integer NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    last_error text NOT NULL DEFAULT '',
    PRIMARY KEY (manifest_id, object_reference_id)
);

CREATE TABLE project_tombstones (
    id bigint PRIMARY KEY CHECK (id > 0),
    project_id bigint NOT NULL UNIQUE CHECK (project_id > 0),
    workspace_id bigint NOT NULL REFERENCES workspaces (id),
    slug_hash bytea NOT NULL CHECK (octet_length(slug_hash) = 32),
    deletion_actor_id bigint CHECK (deletion_actor_id > 0),
    deletion_reason text NOT NULL DEFAULT '',
    deleted_at timestamptz NOT NULL,
    outcome text NOT NULL DEFAULT 'succeeded' CHECK (outcome IN ('succeeded', 'failed'))
);

CREATE OR REPLACE FUNCTION reject_deleted_project_write()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE current_state text;
BEGIN
    SELECT state INTO current_state FROM projects WHERE id = NEW.project_id;
    IF current_state IN ('deleting', 'deleted') THEN
        RAISE EXCEPTION 'project is unavailable' USING ERRCODE = '55000';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER repositories_project_write_guard
BEFORE INSERT OR UPDATE ON repositories
FOR EACH ROW EXECUTE FUNCTION reject_deleted_project_write();
CREATE TRIGGER sources_project_write_guard
BEFORE INSERT OR UPDATE ON sources
FOR EACH ROW EXECUTE FUNCTION reject_deleted_project_write();
CREATE TRIGGER raw_objects_project_write_guard
BEFORE INSERT OR UPDATE ON raw_objects
FOR EACH ROW EXECUTE FUNCTION reject_deleted_project_write();
CREATE TRIGGER canonical_commits_project_write_guard
BEFORE INSERT OR UPDATE ON canonical_commits
FOR EACH ROW EXECUTE FUNCTION reject_deleted_project_write();
CREATE TRIGGER canonical_issues_project_write_guard
BEFORE INSERT OR UPDATE ON canonical_issues
FOR EACH ROW EXECUTE FUNCTION reject_deleted_project_write();
CREATE TRIGGER canonical_pull_requests_project_write_guard
BEFORE INSERT OR UPDATE ON canonical_pull_requests
FOR EACH ROW EXECUTE FUNCTION reject_deleted_project_write();
CREATE TRIGGER canonical_releases_project_write_guard
BEFORE INSERT OR UPDATE ON canonical_releases
FOR EACH ROW EXECUTE FUNCTION reject_deleted_project_write();
CREATE TRIGGER source_events_project_write_guard
BEFORE INSERT OR UPDATE ON source_events
FOR EACH ROW EXECUTE FUNCTION reject_deleted_project_write();
