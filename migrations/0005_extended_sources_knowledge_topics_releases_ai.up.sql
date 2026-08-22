CREATE TABLE registry_adoption_snapshots (
    id bigint PRIMARY KEY CHECK (id > 0),
    project_id bigint NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    source_id bigint NOT NULL REFERENCES sources (id) ON DELETE CASCADE,
    package text NOT NULL CHECK (btrim(package) <> ''),
    registry text NOT NULL CHECK (btrim(registry) <> ''),
    unit text NOT NULL CHECK (btrim(unit) <> ''),
    population_context text NOT NULL CHECK (btrim(population_context) <> ''),
    numeric_value double precision CHECK (numeric_value >= 0),
    status text NOT NULL CHECK (status IN ('available','unknown','incomparable','stale')),
    window_from timestamptz NOT NULL,
    window_to timestamptz NOT NULL,
    observed_at timestamptz NOT NULL,
    raw_object_id bigint NOT NULL REFERENCES raw_objects (id) ON DELETE RESTRICT,
    stale_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (source_id, package, unit, population_context, observed_at),
    CHECK (window_from < window_to AND window_to <= observed_at),
    CHECK ((status = 'available') = (numeric_value IS NOT NULL)),
    CHECK (stale_at IS NULL OR stale_at > observed_at)
);

CREATE INDEX registry_adoption_project_history_idx
    ON registry_adoption_snapshots (project_id, observed_at DESC, id DESC);

CREATE TABLE public_advisories (
    id bigint PRIMARY KEY CHECK (id > 0),
    project_id bigint NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    source_id bigint NOT NULL REFERENCES sources (id) ON DELETE CASCADE,
    external_id text NOT NULL CHECK (btrim(external_id) <> ''),
    severity text NOT NULL DEFAULT '',
    summary text NOT NULL DEFAULT '',
    state text NOT NULL CHECK (state IN ('published','withdrawn')),
    published_at timestamptz NOT NULL,
    withdrawn_at timestamptz,
    raw_object_id bigint NOT NULL REFERENCES raw_objects (id) ON DELETE RESTRICT,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (source_id, external_id, raw_object_id),
    CHECK ((state = 'withdrawn') = (withdrawn_at IS NOT NULL)),
    CHECK (withdrawn_at IS NULL OR withdrawn_at >= published_at)
);

CREATE INDEX public_advisories_project_history_idx
    ON public_advisories (project_id, published_at DESC, id DESC);

CREATE TABLE document_snapshots (
    id bigint PRIMARY KEY CHECK (id > 0),
    project_id bigint NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    source_id bigint NOT NULL REFERENCES sources (id) ON DELETE CASCADE,
    canonical_url text NOT NULL CHECK (canonical_url ~ '^https://'),
    observed_at timestamptz NOT NULL,
    digest bytea NOT NULL CHECK (octet_length(digest) = 32),
    media_type text NOT NULL CHECK (btrim(media_type) <> ''),
    language text NOT NULL DEFAULT 'und',
    object_reference_id bigint REFERENCES object_references (id) ON DELETE RESTRICT,
    raw_object_id bigint REFERENCES raw_objects (id) ON DELETE RESTRICT,
    parser_version text NOT NULL CHECK (btrim(parser_version) <> ''),
    current boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (source_id, canonical_url, digest),
    CHECK (object_reference_id IS NOT NULL OR raw_object_id IS NOT NULL)
);

CREATE UNIQUE INDEX document_snapshots_one_current_idx
    ON document_snapshots (source_id, canonical_url) WHERE current;
CREATE INDEX document_snapshots_project_cutoff_idx
    ON document_snapshots (project_id, current, observed_at DESC, id DESC);

CREATE TABLE knowledge_chunks (
    id bigint PRIMARY KEY CHECK (id > 0),
    project_id bigint NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    source_id bigint NOT NULL REFERENCES sources (id) ON DELETE CASCADE,
    snapshot_id bigint NOT NULL REFERENCES document_snapshots (id) ON DELETE CASCADE,
    ordinal integer NOT NULL CHECK (ordinal >= 0),
    heading text NOT NULL DEFAULT '',
    content text NOT NULL CHECK (btrim(content) <> ''),
    language text NOT NULL DEFAULT 'und',
    start_offset integer NOT NULL CHECK (start_offset >= 0),
    end_offset integer NOT NULL CHECK (end_offset > start_offset),
    parser_version text NOT NULL CHECK (btrim(parser_version) <> ''),
    embedding_model text NOT NULL DEFAULT '',
    embedding vector(1536),
    content_tsv tsvector GENERATED ALWAYS AS
        (to_tsvector('simple', coalesce(heading, '') || ' ' || content)) STORED,
    observed_at timestamptz NOT NULL,
    current boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (snapshot_id, ordinal),
    UNIQUE (snapshot_id, start_offset, end_offset)
);

CREATE INDEX knowledge_chunks_fts_idx ON knowledge_chunks USING gin (content_tsv);
CREATE INDEX knowledge_chunks_vector_idx ON knowledge_chunks
    USING hnsw (embedding vector_cosine_ops) WHERE embedding IS NOT NULL AND current;
CREATE INDEX knowledge_chunks_authorized_idx
    ON knowledge_chunks (project_id, source_id, current, observed_at DESC, id DESC);

CREATE TABLE topic_candidate_sets (
    id bigint PRIMARY KEY CHECK (id > 0),
    project_id bigint NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    window_from timestamptz NOT NULL,
    window_to timestamptz NOT NULL,
    cutoff timestamptz NOT NULL,
    algorithm_version text NOT NULL CHECK (btrim(algorithm_version) <> ''),
    embedding_model text NOT NULL DEFAULT '',
    neighbor_k integer NOT NULL CHECK (neighbor_k BETWEEN 1 AND 100),
    sampling jsonb NOT NULL DEFAULT '{}'::jsonb,
    state text NOT NULL CHECK (state IN ('current','superseded','failed')),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (project_id, window_from, window_to, cutoff, algorithm_version),
    CHECK (window_from < window_to AND window_to <= cutoff)
);

CREATE UNIQUE INDEX topic_candidate_sets_one_current_idx
    ON topic_candidate_sets (project_id) WHERE state = 'current';

CREATE TABLE topics (
    id bigint PRIMARY KEY CHECK (id > 0),
    candidate_set_id bigint NOT NULL REFERENCES topic_candidate_sets (id) ON DELETE CASCADE,
    project_id bigint NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    generated_label text NOT NULL DEFAULT '',
    generated_language text NOT NULL DEFAULT 'und',
    label_run_id bigint,
    confidence double precision CHECK (confidence BETWEEN 0 AND 1),
    created_at timestamptz NOT NULL DEFAULT now(),
    retired_at timestamptz,
    UNIQUE (candidate_set_id, id),
    CHECK (retired_at IS NULL OR retired_at >= created_at)
);

CREATE INDEX topics_project_current_idx ON topics (project_id, retired_at, id);

CREATE TABLE topic_members (
    topic_id bigint NOT NULL REFERENCES topics (id) ON DELETE CASCADE,
    issue_id bigint NOT NULL REFERENCES canonical_issues (id) ON DELETE CASCADE,
    ordinal integer NOT NULL CHECK (ordinal >= 0),
    similarity double precision CHECK (similarity BETWEEN 0 AND 1),
    representative boolean NOT NULL DEFAULT false,
    PRIMARY KEY (topic_id, issue_id),
    UNIQUE (topic_id, ordinal)
);

CREATE TABLE topic_corrections (
    id bigint PRIMARY KEY CHECK (id > 0),
    project_id bigint NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    topic_id bigint NOT NULL REFERENCES topics (id) ON DELETE CASCADE,
    action text NOT NULL CHECK (action IN ('rename','include','exclude','merge','split','reassign')),
    issue_ids bigint[] NOT NULL DEFAULT '{}',
    other_topic_ids bigint[] NOT NULL DEFAULT '{}',
    label text NOT NULL DEFAULT '',
    reason text NOT NULL CHECK (btrim(reason) <> ''),
    actor_id bigint NOT NULL CHECK (actor_id > 0),
    request_id text NOT NULL CHECK (btrim(request_id) <> ''),
    version bigint NOT NULL CHECK (version > 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (topic_id, request_id),
    UNIQUE (topic_id, version)
);

ALTER TABLE canonical_releases
    ADD COLUMN title text NOT NULL DEFAULT '',
    ADD COLUMN body text NOT NULL DEFAULT '',
    ADD COLUMN language text NOT NULL DEFAULT 'und',
    ADD COLUMN canonical_url text NOT NULL DEFAULT '',
    ADD COLUMN state text NOT NULL DEFAULT 'published',
    ADD COLUMN withdrawn_at timestamptz,
    ADD COLUMN changelog_snapshot_id bigint REFERENCES document_snapshots (id) ON DELETE RESTRICT,
    ADD CONSTRAINT canonical_releases_url_check
        CHECK (canonical_url = '' OR canonical_url ~ '^https://'),
    ADD CONSTRAINT canonical_releases_state_check
        CHECK (state IN ('published','withdrawn')),
    ADD CONSTRAINT canonical_releases_withdrawn_check
        CHECK ((state = 'withdrawn') = (withdrawn_at IS NOT NULL)),
    ADD CONSTRAINT canonical_releases_withdrawn_time_check
        CHECK (withdrawn_at IS NULL OR published_at IS NULL OR withdrawn_at >= published_at);

CREATE INDEX canonical_releases_project_history_extended_idx
    ON canonical_releases (project_id, published_at DESC, id DESC);

CREATE TABLE analysis_series (
    id bigint PRIMARY KEY CHECK (id > 0),
    project_id bigint NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    subject_kind text NOT NULL CHECK (btrim(subject_kind) <> ''),
    subject_id bigint,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE NULLS NOT DISTINCT (project_id, subject_kind, subject_id)
);

CREATE TABLE analysis_runs (
    id bigint PRIMARY KEY CHECK (id > 0),
    series_id bigint NOT NULL REFERENCES analysis_series (id) ON DELETE CASCADE,
    project_id bigint NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    parent_run_id bigint REFERENCES analysis_runs (id) ON DELETE RESTRICT,
    kind text NOT NULL CHECK (btrim(kind) <> ''),
    state text NOT NULL CHECK (state IN ('queued','running','succeeded','failed','cancelled')),
    prompt_version text NOT NULL CHECK (btrim(prompt_version) <> ''),
    schema_version text NOT NULL CHECK (btrim(schema_version) <> ''),
    retrieval_version text NOT NULL CHECK (btrim(retrieval_version) <> ''),
    provider text NOT NULL CHECK (btrim(provider) <> ''),
    model text NOT NULL CHECK (btrim(model) <> ''),
    language text NOT NULL CHECK (btrim(language) <> ''),
    requested_by bigint NOT NULL CHECK (requested_by > 0),
    reason text NOT NULL DEFAULT '',
    cutoff timestamptz NOT NULL,
    output jsonb,
    usage jsonb NOT NULL DEFAULT '{}'::jsonb,
    failure_code text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    started_at timestamptz,
    finished_at timestamptz,
    CHECK ((state = 'succeeded') = (output IS NOT NULL)),
    CHECK (finished_at IS NULL OR started_at IS NOT NULL),
    CHECK (started_at IS NULL OR started_at >= created_at),
    CHECK (finished_at IS NULL OR finished_at >= started_at)
);

CREATE INDEX analysis_runs_series_history_idx
    ON analysis_runs (series_id, created_at DESC, id DESC);
CREATE INDEX analysis_runs_project_idx
    ON analysis_runs (project_id, created_at DESC, id DESC);

ALTER TABLE topics
    ADD CONSTRAINT topics_label_run_fk FOREIGN KEY (label_run_id) REFERENCES analysis_runs (id) ON DELETE RESTRICT;

CREATE TABLE analysis_run_citations (
    run_id bigint NOT NULL REFERENCES analysis_runs (id) ON DELETE CASCADE,
    ordinal integer NOT NULL CHECK (ordinal >= 0),
    snapshot_id bigint NOT NULL REFERENCES document_snapshots (id) ON DELETE RESTRICT,
    chunk_id bigint NOT NULL REFERENCES knowledge_chunks (id) ON DELETE RESTRICT,
    start_offset integer NOT NULL CHECK (start_offset >= 0),
    end_offset integer NOT NULL CHECK (end_offset > start_offset),
    PRIMARY KEY (run_id, ordinal),
    UNIQUE (run_id, chunk_id, start_offset, end_offset)
);

CREATE TABLE analysis_feedback (
    id bigint PRIMARY KEY CHECK (id > 0),
    run_id bigint NOT NULL REFERENCES analysis_runs (id) ON DELETE CASCADE,
    actor_id bigint NOT NULL CHECK (actor_id > 0),
    rating text NOT NULL CHECK (rating IN ('faithful','partial','not_faithful','correct','incorrect')),
    note text NOT NULL CHECK (btrim(note) <> ''),
    request_id text NOT NULL CHECK (btrim(request_id) <> ''),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (run_id, actor_id, request_id)
);

CREATE TABLE analysis_selections (
    id bigint PRIMARY KEY CHECK (id > 0),
    series_id bigint NOT NULL REFERENCES analysis_series (id) ON DELETE CASCADE,
    run_id bigint NOT NULL REFERENCES analysis_runs (id) ON DELETE RESTRICT,
    actor_id bigint NOT NULL CHECK (actor_id > 0),
    version bigint NOT NULL CHECK (version > 0),
    request_id text NOT NULL CHECK (btrim(request_id) <> ''),
    selected_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (series_id, version),
    UNIQUE (series_id, request_id)
);

CREATE INDEX analysis_selections_current_idx
    ON analysis_selections (series_id, version DESC, id DESC);

CREATE TRIGGER registry_adoption_snapshots_project_write_guard
BEFORE INSERT OR UPDATE ON registry_adoption_snapshots
FOR EACH ROW EXECUTE FUNCTION reject_deleted_project_write();
CREATE TRIGGER public_advisories_project_write_guard
BEFORE INSERT OR UPDATE ON public_advisories
FOR EACH ROW EXECUTE FUNCTION reject_deleted_project_write();
CREATE TRIGGER document_snapshots_project_write_guard
BEFORE INSERT OR UPDATE ON document_snapshots
FOR EACH ROW EXECUTE FUNCTION reject_deleted_project_write();
CREATE TRIGGER knowledge_chunks_project_write_guard
BEFORE INSERT OR UPDATE ON knowledge_chunks
FOR EACH ROW EXECUTE FUNCTION reject_deleted_project_write();
CREATE TRIGGER topic_candidate_sets_project_write_guard
BEFORE INSERT OR UPDATE ON topic_candidate_sets
FOR EACH ROW EXECUTE FUNCTION reject_deleted_project_write();
CREATE TRIGGER topics_project_write_guard
BEFORE INSERT OR UPDATE ON topics
FOR EACH ROW EXECUTE FUNCTION reject_deleted_project_write();
CREATE TRIGGER topic_corrections_project_write_guard
BEFORE INSERT OR UPDATE ON topic_corrections
FOR EACH ROW EXECUTE FUNCTION reject_deleted_project_write();
CREATE TRIGGER analysis_series_project_write_guard
BEFORE INSERT OR UPDATE ON analysis_series
FOR EACH ROW EXECUTE FUNCTION reject_deleted_project_write();
CREATE TRIGGER analysis_runs_project_write_guard
BEFORE INSERT OR UPDATE ON analysis_runs
FOR EACH ROW EXECUTE FUNCTION reject_deleted_project_write();
