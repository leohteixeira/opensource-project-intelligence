CREATE TABLE workspaces (
    id bigint PRIMARY KEY CHECK (id > 0),
    name text NOT NULL CHECK (btrim(name) <> ''),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE external_identities (
    id bigint PRIMARY KEY CHECK (id > 0),
    issuer text NOT NULL CHECK (btrim(issuer) <> ''),
    subject text NOT NULL CHECK (btrim(subject) <> ''),
    display_name text NOT NULL DEFAULT '',
    email text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (issuer, subject)
);

CREATE TABLE memberships (
    id bigint PRIMARY KEY CHECK (id > 0),
    workspace_id bigint NOT NULL REFERENCES workspaces (id),
    identity_id bigint NOT NULL REFERENCES external_identities (id),
    role text CHECK (role IN ('viewer', 'analyst', 'admin')),
    status text NOT NULL CHECK (status IN ('pending', 'active', 'rejected', 'suspended', 'deleted')),
    locale text NOT NULL DEFAULT 'en' CHECK (locale IN ('en', 'pt-BR')),
    timezone text NOT NULL DEFAULT 'UTC' CHECK (btrim(timezone) <> ''),
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    requested_at timestamptz NOT NULL DEFAULT now(),
    decided_at timestamptz,
    deleted_at timestamptz,
    UNIQUE (workspace_id, identity_id),
    CHECK ((status = 'active' AND role IS NOT NULL) OR status <> 'active')
);

CREATE INDEX memberships_review_idx
    ON memberships (workspace_id, status, requested_at, id);

CREATE TABLE service_accounts (
    id bigint PRIMARY KEY CHECK (id > 0),
    workspace_id bigint NOT NULL REFERENCES workspaces (id),
    issuer text NOT NULL CHECK (btrim(issuer) <> ''),
    external_subject text NOT NULL CHECK (btrim(external_subject) <> ''),
    name text NOT NULL CHECK (btrim(name) <> ''),
    role text NOT NULL CHECK (role IN ('viewer', 'analyst')),
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'suspended', 'deleted')),
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (workspace_id, issuer, external_subject)
);

CREATE TABLE service_account_scopes (
    service_account_id bigint NOT NULL REFERENCES service_accounts (id) ON DELETE CASCADE,
    scope text NOT NULL CHECK (btrim(scope) <> ''),
    PRIMARY KEY (service_account_id, scope)
);

CREATE TABLE browser_sessions (
    id bigint PRIMARY KEY CHECK (id > 0),
    membership_id bigint NOT NULL REFERENCES memberships (id) ON DELETE CASCADE,
    verifier_hash bytea NOT NULL CHECK (octet_length(verifier_hash) = 32),
    csrf_hash bytea NOT NULL CHECK (octet_length(csrf_hash) = 32),
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL,
    last_seen_at timestamptz NOT NULL DEFAULT now(),
    revoked_at timestamptz,
    CHECK (expires_at > created_at)
);

CREATE INDEX browser_sessions_membership_active_idx
    ON browser_sessions (membership_id, expires_at)
    WHERE revoked_at IS NULL;

CREATE TABLE oidc_login_flows (
    state_hash bytea PRIMARY KEY CHECK (octet_length(state_hash) = 32),
    nonce_hash bytea NOT NULL CHECK (octet_length(nonce_hash) = 32),
    verifier_hash bytea NOT NULL CHECK (octet_length(verifier_hash) = 32),
    return_to text NOT NULL CHECK (return_to LIKE '/%'),
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL,
    consumed_at timestamptz,
    CHECK (expires_at > created_at)
);

CREATE TABLE public_catalog_projects (
    id bigint PRIMARY KEY CHECK (id > 0),
    name text NOT NULL CHECK (btrim(name) <> ''),
    slug text NOT NULL UNIQUE CHECK (btrim(slug) <> ''),
    description text NOT NULL DEFAULT '',
    source_links jsonb NOT NULL DEFAULT '[]'::jsonb,
    state text NOT NULL DEFAULT 'active' CHECK (state IN ('active', 'paused', 'archived', 'deleted')),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (jsonb_typeof(source_links) = 'array')
);

CREATE INDEX public_catalog_projects_visible_idx
    ON public_catalog_projects (lower(name), id)
    WHERE state IN ('active', 'paused');

ALTER TABLE audit_events
    ADD COLUMN actor_kind text NOT NULL DEFAULT 'system'
        CHECK (actor_kind IN ('member', 'service_account', 'system', 'deleted_actor')),
    ADD COLUMN outcome text NOT NULL DEFAULT 'succeeded'
        CHECK (outcome IN ('succeeded', 'denied', 'failed', 'stale')),
    ADD COLUMN request_id text NOT NULL DEFAULT '',
    ADD COLUMN changes jsonb NOT NULL DEFAULT '{}'::jsonb;

CREATE OR REPLACE FUNCTION reject_audit_event_mutation()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    RAISE EXCEPTION 'audit events are immutable';
END;
$$;

CREATE TRIGGER audit_events_immutable_update
BEFORE UPDATE OR DELETE ON audit_events
FOR EACH ROW EXECUTE FUNCTION reject_audit_event_mutation();

