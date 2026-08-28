CREATE TABLE assistant_proposals (
    id bigint PRIMARY KEY CHECK (id > 0),
    workspace_id bigint NOT NULL REFERENCES workspaces (id),
    actor_id bigint NOT NULL CHECK (actor_id > 0),
    actor_version bigint NOT NULL CHECK (actor_version > 0),
    status text NOT NULL CHECK (status IN (
        'awaiting_confirmation', 'executing', 'executed', 'failed', 'expired'
    )),
    action text NOT NULL CHECK (action = 'repository.add'),
    inputs jsonb NOT NULL,
    resources jsonb NOT NULL CHECK (jsonb_typeof(resources) = 'array'),
    effect text NOT NULL CHECK (btrim(effect) <> ''),
    quota jsonb NOT NULL,
    confirmation_digest bytea NOT NULL CHECK (octet_length(confirmation_digest) = 32),
    confirmation_key text,
    result jsonb NOT NULL DEFAULT '{}'::jsonb,
    expires_at timestamptz NOT NULL,
    consumed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (expires_at = created_at + interval '10 minutes')
);

CREATE INDEX assistant_proposals_owner_idx
    ON assistant_proposals (workspace_id, actor_id, created_at DESC, id DESC);
CREATE INDEX assistant_proposals_expiry_idx
    ON assistant_proposals (expires_at, id)
    WHERE status = 'awaiting_confirmation';

CREATE TABLE export_requests (
    id bigint PRIMARY KEY CHECK (id > 0),
    workspace_id bigint NOT NULL REFERENCES workspaces (id),
    actor_id bigint NOT NULL CHECK (actor_id > 0),
    job_id bigint NOT NULL UNIQUE REFERENCES jobs (id) ON DELETE CASCADE,
    request jsonb NOT NULL,
    request_id text NOT NULL DEFAULT '',
    object_reference_id bigint UNIQUE REFERENCES object_references (id) ON DELETE RESTRICT,
    row_count bigint NOT NULL DEFAULT 0 CHECK (row_count >= 0),
    state text NOT NULL DEFAULT 'queued' CHECK (state IN (
        'queued', 'running', 'succeeded', 'failed', 'cancelled', 'expired'
    )),
    failure_code text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    completed_at timestamptz,
    expires_at timestamptz,
    CHECK (expires_at IS NULL OR (completed_at IS NOT NULL AND expires_at = completed_at + interval '24 hours'))
);

CREATE INDEX export_requests_owner_idx
    ON export_requests (workspace_id, actor_id, created_at DESC, id DESC);
CREATE UNIQUE INDEX export_requests_idempotency_idx
    ON export_requests (workspace_id, actor_id, request_id);
CREATE INDEX export_requests_expiry_idx
    ON export_requests (expires_at, id)
    WHERE state = 'succeeded';

CREATE TABLE export_request_projects (
    export_id bigint NOT NULL REFERENCES export_requests (id) ON DELETE CASCADE,
    project_id bigint NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    PRIMARY KEY (export_id, project_id)
);

CREATE INDEX export_request_projects_project_idx
    ON export_request_projects (project_id, export_id);

CREATE TABLE model_usage_events (
    id bigint PRIMARY KEY CHECK (id > 0),
    workspace_id bigint NOT NULL REFERENCES workspaces (id),
    provider text NOT NULL CHECK (btrim(provider) <> ''),
    model text NOT NULL CHECK (btrim(model) <> ''),
    operation text NOT NULL CHECK (btrim(operation) <> ''),
    input_tokens bigint NOT NULL DEFAULT 0 CHECK (input_tokens >= 0),
    output_tokens bigint NOT NULL DEFAULT 0 CHECK (output_tokens >= 0),
    estimated_cost_micros bigint NOT NULL DEFAULT 0 CHECK (estimated_cost_micros >= 0),
    outcome text NOT NULL CHECK (outcome IN ('succeeded', 'failed', 'refused', 'interrupted')),
    request_id text NOT NULL DEFAULT '',
    occurred_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX model_usage_events_operations_idx
    ON model_usage_events (workspace_id, occurred_at DESC, id DESC);
