CREATE TABLE trend_signals (
    id bigint PRIMARY KEY CHECK (id > 0),
    project_id bigint NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    metric_name text NOT NULL,
    metric_version text NOT NULL,
    kind text NOT NULL CHECK (kind IN ('observed','forecast')),
    status text NOT NULL CHECK (status IN ('increase','decrease','stable','insufficient_data')),
    method_version text NOT NULL CHECK (btrim(method_version) <> ''),
    selected_model text NOT NULL DEFAULT '',
    window_from timestamptz NOT NULL,
    window_to timestamptz NOT NULL,
    baseline_from timestamptz,
    baseline_to timestamptz,
    cutoff timestamptz NOT NULL,
    magnitude double precision,
    horizon_days integer CHECK (horizon_days > 0),
    predicted_value double precision,
    interval_low double precision,
    interval_high double precision,
    confidence double precision CHECK (confidence BETWEEN 0 AND 1),
    backtest_error double precision CHECK (backtest_error >= 0),
    eligible_count integer NOT NULL CHECK (eligible_count >= 0),
    observed_count integer NOT NULL CHECK (observed_count >= 0 AND observed_count <= eligible_count),
    coverage_note text NOT NULL DEFAULT '',
    input_digest text NOT NULL CHECK (btrim(input_digest) <> ''),
    outcome_status text NOT NULL DEFAULT '',
    superseded_by bigint REFERENCES trend_signals (id) ON DELETE RESTRICT,
    created_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (metric_name, metric_version) REFERENCES metric_definitions (name, version),
    UNIQUE (project_id, metric_name, metric_version, kind, method_version, window_from, window_to, cutoff, input_digest),
    CHECK (window_from < window_to AND window_to <= cutoff),
    CHECK ((kind = 'forecast') = (horizon_days IS NOT NULL)),
    CHECK (kind <> 'forecast' OR (baseline_from IS NULL AND baseline_to IS NULL)),
    CHECK (kind <> 'observed' OR (baseline_from IS NOT NULL AND baseline_to IS NOT NULL AND baseline_from < baseline_to AND baseline_to <= window_from)),
    CHECK (predicted_value IS NULL OR interval_low IS NOT NULL AND interval_high IS NOT NULL AND interval_low <= predicted_value AND predicted_value <= interval_high)
);

CREATE INDEX trend_signals_history_idx
    ON trend_signals (project_id, kind, cutoff DESC, id DESC);

CREATE TABLE trend_signal_evidence (
    signal_id bigint NOT NULL REFERENCES trend_signals (id) ON DELETE CASCADE,
    ordinal integer NOT NULL CHECK (ordinal >= 0),
    evidence_id bigint NOT NULL CHECK (evidence_id > 0),
    PRIMARY KEY (signal_id, ordinal),
    UNIQUE (signal_id, evidence_id)
);

CREATE TABLE policy_families (
    id bigint PRIMARY KEY CHECK (id > 0),
    workspace_id bigint NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
    name text NOT NULL CHECK (btrim(name) <> ''),
    description text NOT NULL DEFAULT '',
    owner text NOT NULL CHECK (btrim(owner) <> ''),
    active_version integer,
    created_by bigint NOT NULL CHECK (created_by > 0),
    request_id text NOT NULL CHECK (btrim(request_id) <> ''),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (workspace_id, name),
    UNIQUE (workspace_id, request_id)
);

CREATE TABLE policy_versions (
    id bigint PRIMARY KEY CHECK (id > 0),
    policy_id bigint NOT NULL REFERENCES policy_families (id) ON DELETE CASCADE,
    version integer NOT NULL CHECK (version > 0),
    state text NOT NULL CHECK (state IN ('draft','active','superseded','retired')),
    revision bigint NOT NULL DEFAULT 1 CHECK (revision > 0),
    created_by bigint NOT NULL CHECK (created_by > 0),
    request_id text NOT NULL CHECK (btrim(request_id) <> ''),
    created_at timestamptz NOT NULL DEFAULT now(),
    activated_at timestamptz,
    retired_at timestamptz,
    UNIQUE (policy_id, version),
    UNIQUE (policy_id, request_id),
    CHECK ((state = 'active') = (activated_at IS NOT NULL AND retired_at IS NULL) OR state <> 'active'),
    CHECK (retired_at IS NULL OR activated_at IS NOT NULL AND activated_at <= retired_at)
);

ALTER TABLE policy_families ADD CONSTRAINT policy_families_active_version_fk
    FOREIGN KEY (id, active_version) REFERENCES policy_versions (policy_id, version)
    DEFERRABLE INITIALLY DEFERRED;

CREATE TABLE policy_rules (
    policy_version_id bigint NOT NULL REFERENCES policy_versions (id) ON DELETE CASCADE,
    ordinal integer NOT NULL CHECK (ordinal BETWEEN 0 AND 99),
    metric_name text NOT NULL,
    metric_version text NOT NULL,
    operator text NOT NULL CHECK (operator IN ('gt','gte','lt','lte','eq')),
    threshold double precision NOT NULL,
    weight double precision NOT NULL CHECK (weight > 0 AND weight <= 1),
    required boolean NOT NULL,
    required_evidence text NOT NULL DEFAULT '',
    on_failure text NOT NULL CHECK (on_failure IN ('conditional','not_recommended')),
    label text NOT NULL CHECK (btrim(label) <> ''),
    PRIMARY KEY (policy_version_id, ordinal),
    FOREIGN KEY (metric_name, metric_version) REFERENCES metric_definitions (name, version),
    CHECK (NOT required OR btrim(required_evidence) <> '')
);

CREATE TABLE policy_radar_mappings (
    policy_version_id bigint NOT NULL REFERENCES policy_versions (id) ON DELETE CASCADE,
    outcome text NOT NULL CHECK (outcome IN ('recommended','conditional','not_recommended','insufficient_data')),
    ring text NOT NULL CHECK (ring IN ('adopt','trial','assess','hold','unplaced')),
    PRIMARY KEY (policy_version_id, outcome)
);

CREATE INDEX policy_versions_history_idx ON policy_versions (policy_id, version DESC);

CREATE TABLE recommendation_evaluations (
    id bigint PRIMARY KEY CHECK (id > 0),
    project_id bigint NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    policy_id bigint NOT NULL REFERENCES policy_families (id) ON DELETE RESTRICT,
    policy_version_id bigint NOT NULL REFERENCES policy_versions (id) ON DELETE RESTRICT,
    window_from timestamptz NOT NULL,
    window_to timestamptz NOT NULL,
    cutoff timestamptz NOT NULL,
    outcome text NOT NULL CHECK (outcome IN ('recommended','conditional','not_recommended','insufficient_data')),
    created_at timestamptz NOT NULL DEFAULT now(),
    input_digest text NOT NULL CHECK (btrim(input_digest) <> ''),
    explanation text NOT NULL DEFAULT '',
    UNIQUE (project_id, policy_version_id, window_from, window_to, cutoff, input_digest),
    CHECK (window_from < window_to AND window_to <= cutoff)
);

CREATE INDEX recommendation_evaluations_latest_idx
    ON recommendation_evaluations (project_id, policy_id, cutoff DESC, id DESC);

CREATE TABLE recommendation_factors (
    evaluation_id bigint NOT NULL REFERENCES recommendation_evaluations (id) ON DELETE CASCADE,
    ordinal integer NOT NULL CHECK (ordinal >= 0),
    rule_ordinal integer NOT NULL CHECK (rule_ordinal >= 0),
    metric_name text NOT NULL,
    numeric_value double precision NOT NULL,
    threshold double precision NOT NULL,
    weight double precision NOT NULL,
    matched boolean NOT NULL,
    label text NOT NULL CHECK (btrim(label) <> ''),
    PRIMARY KEY (evaluation_id, ordinal)
);

CREATE TABLE recommendation_evidence (
    evaluation_id bigint NOT NULL REFERENCES recommendation_evaluations (id) ON DELETE CASCADE,
    evidence_id bigint NOT NULL CHECK (evidence_id > 0),
    decisive boolean NOT NULL DEFAULT false,
    PRIMARY KEY (evaluation_id, evidence_id)
);

CREATE TABLE recommendation_gaps (
    evaluation_id bigint NOT NULL REFERENCES recommendation_evaluations (id) ON DELETE CASCADE,
    kind text NOT NULL CHECK (kind IN ('missing','stale','condition','decisive')),
    ordinal integer NOT NULL CHECK (ordinal >= 0),
    value text NOT NULL CHECK (btrim(value) <> ''),
    PRIMARY KEY (evaluation_id, kind, ordinal)
);

CREATE TABLE radar_selections (
    project_id bigint PRIMARY KEY REFERENCES projects (id) ON DELETE CASCADE,
    evaluation_id bigint NOT NULL REFERENCES recommendation_evaluations (id) ON DELETE RESTRICT,
    selected_by bigint NOT NULL CHECK (selected_by > 0),
    selected_at timestamptz NOT NULL DEFAULT now(),
    revision bigint NOT NULL DEFAULT 1 CHECK (revision > 0)
);

CREATE TABLE radar_overrides (
    id bigint PRIMARY KEY CHECK (id > 0),
    project_id bigint NOT NULL REFERENCES radar_selections (project_id) ON DELETE CASCADE,
    ring text NOT NULL CHECK (ring IN ('adopt','trial','assess','hold')),
    reason text NOT NULL CHECK (btrim(reason) <> ''),
    owner text NOT NULL CHECK (btrim(owner) <> ''),
    actor_id bigint NOT NULL CHECK (actor_id > 0),
    review_on date NOT NULL,
    revision bigint NOT NULL DEFAULT 1 CHECK (revision > 0),
    created_at timestamptz NOT NULL,
    removed_at timestamptz,
    request_id text NOT NULL CHECK (btrim(request_id) <> ''),
    UNIQUE (project_id, request_id),
    CHECK (removed_at IS NULL OR removed_at >= created_at)
);

CREATE UNIQUE INDEX radar_overrides_active_idx ON radar_overrides (project_id) WHERE removed_at IS NULL;

CREATE TABLE alert_rules (
    id bigint NOT NULL CHECK (id > 0),
    version bigint NOT NULL CHECK (version > 0),
    workspace_id bigint NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
    name text NOT NULL CHECK (btrim(name) <> ''),
    signal text NOT NULL CHECK (btrim(signal) <> ''),
    operator text NOT NULL CHECK (operator IN ('gt','lt','eq')),
    threshold double precision NOT NULL,
    scope text NOT NULL CHECK (scope IN ('workspace','project')),
    project_id bigint REFERENCES projects (id) ON DELETE CASCADE,
    severity text NOT NULL CHECK (severity IN ('info','warning','critical')),
    cooldown_seconds bigint NOT NULL CHECK (cooldown_seconds >= 0),
    deduplication_seconds bigint NOT NULL CHECK (deduplication_seconds > 0),
    enabled boolean NOT NULL,
    created_by bigint NOT NULL CHECK (created_by > 0),
    updated_at timestamptz NOT NULL,
    PRIMARY KEY (id, version),
    CHECK ((scope = 'project') = (project_id IS NOT NULL))
);

CREATE INDEX alert_rules_current_idx ON alert_rules (id, version DESC);

CREATE TABLE alert_occurrences (
    id bigint PRIMARY KEY CHECK (id > 0),
    rule_id bigint NOT NULL,
    rule_version bigint NOT NULL,
    project_id bigint NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    signal_version text NOT NULL CHECK (btrim(signal_version) <> ''),
    severity text NOT NULL CHECK (severity IN ('info','warning','critical')),
    state text NOT NULL CHECK (state IN ('open','acknowledged','resolved','dismissed')),
    window_from timestamptz NOT NULL,
    window_to timestamptz NOT NULL,
    first_detected_at timestamptz NOT NULL,
    last_detected_at timestamptz NOT NULL,
    suppression_count bigint NOT NULL DEFAULT 0 CHECK (suppression_count >= 0),
    transition_reason text NOT NULL DEFAULT '',
    transitioned_by bigint,
    revision bigint NOT NULL DEFAULT 1 CHECK (revision > 0),
    FOREIGN KEY (rule_id, rule_version) REFERENCES alert_rules (id, version) ON DELETE RESTRICT,
    CHECK (window_from < window_to),
    CHECK (first_detected_at <= last_detected_at)
);

CREATE INDEX alert_occurrences_inbox_idx
    ON alert_occurrences (state, last_detected_at DESC, id DESC);
CREATE INDEX alert_occurrences_dedup_idx
    ON alert_occurrences (rule_id, rule_version, project_id, first_detected_at DESC);

CREATE TABLE alert_occurrence_evidence (
    occurrence_id bigint NOT NULL REFERENCES alert_occurrences (id) ON DELETE CASCADE,
    evidence_id bigint NOT NULL CHECK (evidence_id > 0),
    PRIMARY KEY (occurrence_id, evidence_id)
);

CREATE TABLE alert_member_state (
    occurrence_id bigint NOT NULL REFERENCES alert_occurrences (id) ON DELETE CASCADE,
    member_id bigint NOT NULL CHECK (member_id > 0),
    read_at timestamptz NOT NULL,
    PRIMARY KEY (occurrence_id, member_id)
);

CREATE TABLE alert_transitions (
    occurrence_id bigint NOT NULL REFERENCES alert_occurrences (id) ON DELETE CASCADE,
    revision bigint NOT NULL CHECK (revision > 1),
    from_state text NOT NULL CHECK (from_state IN ('open','acknowledged','resolved','dismissed')),
    to_state text NOT NULL CHECK (to_state IN ('open','acknowledged','resolved','dismissed')),
    reason text NOT NULL CHECK (btrim(reason) <> ''),
    actor_id bigint NOT NULL CHECK (actor_id > 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (occurrence_id, revision)
);

CREATE TRIGGER trend_signals_project_write_guard
BEFORE INSERT OR UPDATE ON trend_signals
FOR EACH ROW EXECUTE FUNCTION reject_deleted_project_write();
CREATE TRIGGER recommendation_evaluations_project_write_guard
BEFORE INSERT OR UPDATE ON recommendation_evaluations
FOR EACH ROW EXECUTE FUNCTION reject_deleted_project_write();
CREATE TRIGGER radar_selections_project_write_guard
BEFORE INSERT OR UPDATE ON radar_selections
FOR EACH ROW EXECUTE FUNCTION reject_deleted_project_write();
CREATE TRIGGER radar_overrides_project_write_guard
BEFORE INSERT OR UPDATE ON radar_overrides
FOR EACH ROW EXECUTE FUNCTION reject_deleted_project_write();
CREATE TRIGGER alert_rules_project_write_guard
BEFORE INSERT OR UPDATE ON alert_rules
FOR EACH ROW WHEN (NEW.project_id IS NOT NULL) EXECUTE FUNCTION reject_deleted_project_write();
CREATE TRIGGER alert_occurrences_project_write_guard
BEFORE INSERT OR UPDATE ON alert_occurrences
FOR EACH ROW EXECUTE FUNCTION reject_deleted_project_write();
