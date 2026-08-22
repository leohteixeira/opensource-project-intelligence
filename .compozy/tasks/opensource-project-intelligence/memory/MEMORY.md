# Workflow Memory: Open Source Project Intelligence

## Repository invariants

- The repository is an independent Go 1.26 and React/pnpm monorepo under
  `/workspace/repos/opensource-project-intelligence`; Git commands and verification run from that
  repository only.
- PostgreSQL 18 with pgvector is the only canonical data authority. JetStream is delivery,
  S3-compatible storage is immutable object content addressed by PostgreSQL records, and Valkey is
  disposable acceleration state.
- API and worker remain separate binaries in one modular monolith. Business packages do not import
  provider or generated infrastructure types.
- OpenAPI 3.1 and committed SQL are reviewed sources. Generated Go, TypeScript, and sqlc outputs are
  reproducible, pinned, committed, and checked for drift.
- Persisted Snowflake IDs use PostgreSQL `bigint`; HTTP and JSON expose opaque decimal strings.
- Configuration is immutable after startup, validates all applicable fields together, and never
  includes secret values in errors, logs, readiness, metrics, or traces.
- No `.env` or `.env.*` file may be read, searched, or modified. `env.example` is the checked-in
  secret-free template.
- Preserve existing repository history and unrelated changes. Task execution remains uncommitted.

## Workflow state

- Task 01 completed with fresh unit, race, build, frontend, generation-drift, Compose-health, and
  PostgreSQL 18/pgvector integration evidence.
- Later tasks depend on the published specifications, generated boundaries, base schema,
  infrastructure ownership, and verification commands delivered here.
- The accepted workflow bundle under `.compozy/tasks/opensource-project-intelligence` remains the
  upstream workflow source; `specs/` becomes the repository-owned implementation contract.
- Task 01 establishes generator and service ownership only. Later vertical tasks still own domain
  schemas/queries, application handlers, identity integration, broker consumers, object adapters,
  cache adapters, and user-facing behavior described by the published contracts.
