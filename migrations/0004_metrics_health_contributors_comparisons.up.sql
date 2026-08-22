CREATE TABLE metric_definitions (
    name text NOT NULL CHECK (name ~ '^[a-z][a-z0-9_]*$'),
    version text NOT NULL CHECK (version ~ '^v[1-9][0-9]*$'),
    unit text NOT NULL CHECK (btrim(unit) <> ''),
    default_window_days integer NOT NULL CHECK (default_window_days IN (30, 90, 180, 365)),
    formula text NOT NULL CHECK (btrim(formula) <> ''),
    eligibility text NOT NULL CHECK (btrim(eligibility) <> ''),
    missing_data_rule text NOT NULL CHECK (btrim(missing_data_rule) <> ''),
    valid_from timestamptz NOT NULL,
    valid_to timestamptz,
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'superseded')),
    PRIMARY KEY (name, version),
    CHECK (valid_to IS NULL OR valid_from < valid_to)
);

INSERT INTO metric_definitions
    (name, version, unit, default_window_days, formula, eligibility, missing_data_rule, valid_from)
VALUES
    ('release_frequency','v1','releases',90,'count(stable releases published in [from,to))','published and not draft or prerelease','insufficient_data when release coverage is incomplete','2026-08-21T00:00:00Z'),
    ('active_contributors','v1','people',30,'count(distinct eligible contributor identities)','human non-merge default-branch commits','insufficient_data when commit evidence is absent','2026-08-21T00:00:00Z'),
    ('issues_opened_closed','v1','issues',30,'count(opened), count(closed) in [from,to)','canonical issues','insufficient_data when issue coverage is incomplete','2026-08-21T00:00:00Z'),
    ('median_issue_first_response','v1','hours',30,'median(first qualifying response - opened_at)','issues opened in window and qualifying public member response','unanswered issues are censored and reduce coverage','2026-08-21T00:00:00Z'),
    ('median_pr_merge_time','v1','hours',30,'median(merged_at - ready_for_review)','pull requests merged in window','created_at fallback is counted in coverage','2026-08-21T00:00:00Z'),
    ('backlog_change','v1','issues',30,'open_at_end - open_at_start','issue state events including reopen','insufficient_data without boundary history','2026-08-21T00:00:00Z'),
    ('top_three_author_share','v1','ratio',90,'commits by top three authors / eligible commits','human non-merge default-branch commits','unresolved accounts remain distinct and reduce resolution coverage','2026-08-21T00:00:00Z');

CREATE TABLE metric_snapshots (
    id bigint PRIMARY KEY CHECK (id > 0),
    project_id bigint NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    definition_name text NOT NULL,
    definition_version text NOT NULL,
    window_from timestamptz NOT NULL,
    window_to timestamptz NOT NULL,
    cutoff timestamptz NOT NULL,
    status text NOT NULL CHECK (status IN ('available','unknown','not_applicable','insufficient_data','incomparable','stale','unavailable')),
    numeric_value double precision,
    eligible_count bigint NOT NULL DEFAULT 0 CHECK (eligible_count >= 0),
    observed_count bigint NOT NULL DEFAULT 0 CHECK (observed_count >= 0 AND observed_count <= eligible_count),
    coverage_note text NOT NULL DEFAULT '',
    repository_ids bigint[] NOT NULL DEFAULT '{}',
    stale_reason text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (definition_name, definition_version)
        REFERENCES metric_definitions (name, version),
    UNIQUE (project_id, definition_name, definition_version, window_from, window_to, cutoff),
    CHECK (window_from < window_to AND window_to <= cutoff),
    CHECK ((status = 'available') = (numeric_value IS NOT NULL))
);

CREATE INDEX metric_snapshots_latest_idx
    ON metric_snapshots (project_id, definition_name, cutoff DESC, id DESC);

CREATE TABLE metric_factors (
    snapshot_id bigint NOT NULL REFERENCES metric_snapshots (id) ON DELETE CASCADE,
    ordinal integer NOT NULL CHECK (ordinal >= 0),
    name text NOT NULL CHECK (btrim(name) <> ''),
    numeric_value double precision,
    unit text NOT NULL CHECK (btrim(unit) <> ''),
    evidence_id bigint REFERENCES raw_objects (id) ON DELETE RESTRICT,
    PRIMARY KEY (snapshot_id, ordinal)
);

CREATE TABLE health_definitions (
    version text PRIMARY KEY CHECK (version ~ '^v[1-9][0-9]*$'),
    valid_from timestamptz NOT NULL,
    valid_to timestamptz,
    mandatory_dimensions text[] NOT NULL,
    weight double precision NOT NULL CHECK (weight = 1.0 / 7.0),
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'superseded')),
    CHECK (cardinality(mandatory_dimensions) = 7),
    CHECK (valid_to IS NULL OR valid_from < valid_to)
);

INSERT INTO health_definitions (version, valid_from, mandatory_dimensions, weight)
VALUES ('v1','2026-08-21T00:00:00Z',ARRAY['Activity','Community','Maintenance','Concentration','Stability','Security','Adoption'],1.0/7.0);

CREATE TABLE health_snapshots (
    id bigint PRIMARY KEY CHECK (id > 0),
    project_id bigint NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    definition_version text NOT NULL REFERENCES health_definitions (version),
    window_from timestamptz NOT NULL,
    window_to timestamptz NOT NULL,
    cutoff timestamptz NOT NULL,
    overall_status text NOT NULL CHECK (overall_status IN ('available','insufficient_data','stale','unavailable')),
    overall_score double precision CHECK (overall_score BETWEEN 0 AND 1),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (project_id, definition_version, window_from, window_to, cutoff),
    CHECK (window_from < window_to AND window_to <= cutoff),
    CHECK ((overall_status = 'available') = (overall_score IS NOT NULL))
);

CREATE TABLE health_dimensions (
    health_snapshot_id bigint NOT NULL REFERENCES health_snapshots (id) ON DELETE CASCADE,
    ordinal integer NOT NULL CHECK (ordinal BETWEEN 0 AND 6),
    name text NOT NULL CHECK (name IN ('Activity','Community','Maintenance','Concentration','Stability','Security','Adoption')),
    status text NOT NULL CHECK (status IN ('available','unknown','not_applicable','insufficient_data','stale','unavailable')),
    score double precision CHECK (score BETWEEN 0 AND 1),
    weight double precision NOT NULL CHECK (weight = 1.0 / 7.0),
    metric_snapshot_ids bigint[] NOT NULL DEFAULT '{}',
    PRIMARY KEY (health_snapshot_id, ordinal),
    UNIQUE (health_snapshot_id, name),
    CHECK ((status = 'available') = (score IS NOT NULL))
);

CREATE TABLE health_dimension_factors (
    health_snapshot_id bigint NOT NULL,
    dimension_ordinal integer NOT NULL CHECK (dimension_ordinal BETWEEN 0 AND 6),
    ordinal integer NOT NULL CHECK (ordinal >= 0),
    name text NOT NULL CHECK (btrim(name) <> ''),
    numeric_value double precision,
    unit text NOT NULL CHECK (btrim(unit) <> ''),
    evidence_id bigint REFERENCES raw_objects (id) ON DELETE RESTRICT,
    PRIMARY KEY (health_snapshot_id, dimension_ordinal, ordinal),
    FOREIGN KEY (health_snapshot_id, dimension_ordinal)
        REFERENCES health_dimensions (health_snapshot_id, ordinal) ON DELETE CASCADE
);

CREATE TABLE contributor_identities (
    id bigint PRIMARY KEY CHECK (id > 0),
    project_id bigint NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    public_handle text NOT NULL CHECK (btrim(public_handle) <> ''),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (project_id, public_handle)
);

CREATE TABLE contributor_accounts (
    id bigint PRIMARY KEY CHECK (id > 0),
    project_id bigint NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    source_id bigint NOT NULL REFERENCES sources (id) ON DELETE CASCADE,
    external_id text NOT NULL CHECK (btrim(external_id) <> ''),
    public_handle text NOT NULL DEFAULT '',
    source_status text NOT NULL DEFAULT 'available' CHECK (source_status IN ('available','removed','deleted')),
    bot boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (source_id, external_id)
);

CREATE TABLE contributor_identity_links (
    account_id bigint PRIMARY KEY REFERENCES contributor_accounts (id) ON DELETE CASCADE,
    identity_id bigint REFERENCES contributor_identities (id) ON DELETE CASCADE,
    status text NOT NULL CHECK (status IN ('unresolved','verified','analyst_confirmed')),
    evidence_id bigint REFERENCES raw_objects (id) ON DELETE RESTRICT,
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK ((status = 'unresolved') = (identity_id IS NULL))
);

CREATE TABLE contributor_identity_corrections (
    id bigint PRIMARY KEY CHECK (id > 0),
    account_id bigint NOT NULL REFERENCES contributor_accounts (id) ON DELETE CASCADE,
    from_identity_id bigint REFERENCES contributor_identities (id) ON DELETE SET NULL,
    to_identity_id bigint REFERENCES contributor_identities (id) ON DELETE SET NULL,
    action text NOT NULL CHECK (action IN ('confirm','split')),
    reason text NOT NULL CHECK (btrim(reason) <> ''),
    actor_id bigint NOT NULL CHECK (actor_id > 0),
    request_id text NOT NULL CHECK (btrim(request_id) <> ''),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (account_id, request_id)
);

CREATE TABLE contributor_snapshots (
    id bigint PRIMARY KEY CHECK (id > 0),
    project_id bigint NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    definition_version text NOT NULL CHECK (btrim(definition_version) <> ''),
    window_from timestamptz NOT NULL,
    window_to timestamptz NOT NULL,
    cutoff timestamptz NOT NULL,
    status text NOT NULL CHECK (status IN ('available','insufficient_data','stale','unavailable')),
    active_count bigint,
    new_count bigint,
    retention_ratio double precision,
    maintainer_count bigint,
    top_one_share double precision,
    top_three_share double precision,
    resolution_coverage double precision NOT NULL CHECK (resolution_coverage BETWEEN 0 AND 1),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (project_id, definition_version, window_from, window_to, cutoff),
    CHECK (window_from < window_to AND window_to <= cutoff),
    CHECK (status <> 'available' OR active_count IS NOT NULL)
);

CREATE TABLE comparisons (
    id bigint PRIMARY KEY CHECK (id > 0),
    workspace_id bigint NOT NULL REFERENCES workspaces (id),
    window_from timestamptz NOT NULL,
    window_to timestamptz NOT NULL,
    cutoff timestamptz NOT NULL,
    definition_set text NOT NULL CHECK (btrim(definition_set) <> ''),
    project_boundary bigint[] NOT NULL,
    created_by bigint NOT NULL CHECK (created_by > 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK (window_from < window_to AND window_to <= cutoff),
    CHECK (cardinality(project_boundary) BETWEEN 2 AND 5)
);

CREATE INDEX comparisons_workspace_idx ON comparisons (workspace_id, created_at DESC, id DESC);

CREATE TABLE comparison_items (
    comparison_id bigint NOT NULL REFERENCES comparisons (id) ON DELETE CASCADE,
    metric_name text NOT NULL,
    project_id bigint NOT NULL REFERENCES projects (id),
    ordinal integer NOT NULL CHECK (ordinal >= 0),
    metric_snapshot_id bigint REFERENCES metric_snapshots (id) ON DELETE RESTRICT,
    status text NOT NULL CHECK (status IN ('available','unknown','not_applicable','insufficient_data','incomparable','stale','unavailable')),
    numeric_value double precision,
    unit text NOT NULL CHECK (btrim(unit) <> ''),
    definition_version text NOT NULL CHECK (btrim(definition_version) <> ''),
    PRIMARY KEY (comparison_id, metric_name, project_id),
    UNIQUE (comparison_id, ordinal, project_id),
    CHECK ((status = 'available') = (numeric_value IS NOT NULL))
);

CREATE TABLE issue_response_events (
    id bigint PRIMARY KEY CHECK (id > 0),
    issue_id bigint NOT NULL REFERENCES canonical_issues (id) ON DELETE CASCADE,
    source_id bigint NOT NULL REFERENCES sources (id) ON DELETE CASCADE,
    actor_external_id text NOT NULL CHECK (btrim(actor_external_id) <> ''),
    public boolean NOT NULL,
    bot boolean NOT NULL,
    recognized_member boolean NOT NULL,
    actor_is_opener boolean NOT NULL,
    occurred_at timestamptz NOT NULL,
    raw_object_id bigint NOT NULL REFERENCES raw_objects (id) ON DELETE RESTRICT,
    UNIQUE (source_id, raw_object_id)
);

CREATE TABLE issue_state_events (
    id bigint PRIMARY KEY CHECK (id > 0),
    issue_id bigint NOT NULL REFERENCES canonical_issues (id) ON DELETE CASCADE,
    source_id bigint NOT NULL REFERENCES sources (id) ON DELETE CASCADE,
    state text NOT NULL CHECK (state IN ('open','closed')),
    occurred_at timestamptz NOT NULL,
    raw_object_id bigint NOT NULL REFERENCES raw_objects (id) ON DELETE RESTRICT,
    UNIQUE (source_id, raw_object_id)
);

CREATE TABLE pull_request_readiness_events (
    id bigint PRIMARY KEY CHECK (id > 0),
    pull_request_id bigint NOT NULL REFERENCES canonical_pull_requests (id) ON DELETE CASCADE,
    source_id bigint NOT NULL REFERENCES sources (id) ON DELETE CASCADE,
    state text NOT NULL CHECK (state IN ('draft','ready_for_review')),
    occurred_at timestamptz NOT NULL,
    raw_object_id bigint NOT NULL REFERENCES raw_objects (id) ON DELETE RESTRICT,
    UNIQUE (source_id, raw_object_id)
);
