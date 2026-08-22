CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE snowflake_node_leases (
    node_id smallint PRIMARY KEY CHECK (node_id BETWEEN 0 AND 1023),
    holder_id text NOT NULL UNIQUE CHECK (holder_id <> ''),
    lease_expires_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX snowflake_node_leases_expiry_idx
    ON snowflake_node_leases (lease_expires_at);

CREATE TABLE jobs (
    id bigint PRIMARY KEY CHECK (id > 0),
    kind text NOT NULL CHECK (kind <> ''),
    state text NOT NULL CHECK (state IN ('queued', 'running', 'succeeded', 'failed', 'cancelled')),
    checkpoint jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE object_references (
    id bigint PRIMARY KEY CHECK (id > 0),
    object_key text NOT NULL UNIQUE CHECK (object_key <> ''),
    sha256 bytea NOT NULL CHECK (octet_length(sha256) = 32),
    size_bytes bigint NOT NULL CHECK (size_bytes >= 0),
    media_type text NOT NULL CHECK (media_type <> ''),
    retention_state text NOT NULL DEFAULT 'retained'
        CHECK (retention_state IN ('retained', 'purging', 'purged')),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE evidence_vectors (
    id bigint PRIMARY KEY CHECK (id > 0),
    object_reference_id bigint REFERENCES object_references (id) ON DELETE CASCADE,
    embedding vector(3) NOT NULL,
    model text NOT NULL CHECK (model <> ''),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE audit_events (
    id bigint PRIMARY KEY CHECK (id > 0),
    occurred_at timestamptz NOT NULL DEFAULT now(),
    actor_id bigint,
    action text NOT NULL CHECK (action <> ''),
    resource_type text NOT NULL CHECK (resource_type <> ''),
    resource_id bigint,
    details jsonb NOT NULL DEFAULT '{}'::jsonb
);

CREATE INDEX audit_events_occurred_at_idx ON audit_events (occurred_at DESC, id DESC);

CREATE TABLE outbox_events (
    id bigint PRIMARY KEY CHECK (id > 0),
    aggregate_type text NOT NULL CHECK (aggregate_type <> ''),
    aggregate_id bigint NOT NULL CHECK (aggregate_id > 0),
    event_type text NOT NULL CHECK (event_type <> ''),
    schema_version integer NOT NULL CHECK (schema_version > 0),
    payload jsonb NOT NULL,
    occurred_at timestamptz NOT NULL DEFAULT now(),
    published_at timestamptz
);

CREATE INDEX outbox_events_unpublished_idx
    ON outbox_events (occurred_at, id)
    WHERE published_at IS NULL;
