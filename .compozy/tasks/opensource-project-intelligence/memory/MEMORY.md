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
- Task 02 completed with fresh Go unit/race/build, PostgreSQL integration, frontend
  lint/typecheck/test/build, Playwright/axe, localized narrow/wide visual, and infrastructure-health
  evidence. Local identity, access governance, audit/operations, generated-client routing, and the
  bilingual application shell are now the stable seams for later tasks.
- Later tasks depend on the published specifications, generated boundaries, base schema,
  infrastructure ownership, verification commands, principal/session contract, authorization
  middleware, audit append port, and localized shell delivered by tasks 01 and 02.
- The accepted workflow bundle under `.compozy/tasks/opensource-project-intelligence` remains the
  upstream workflow source; `specs/` becomes the repository-owned implementation contract.
- Tasks 01 and 02 establish generator/service ownership plus identity and shell governance. Later
  vertical tasks still own project and ingestion schemas, broker consumers, object adapters, cache
  adapters, metrics, analysis, policy, alert, export, and their assigned user-facing behavior.
- Task 03 is complete with Project/Repository/Source lifecycle and persistence, durable
  PostgreSQL Jobs/outbox/checkpoints, an explicit-ack JetStream consumer, Valkey disposable
  wake-ups, immutable S3 ownership, four-scope GitHub and restricted-Git ingestion, resumable Job
  SSE, generated-client S4-S8 surfaces, and unavailable-first resumable purge.
- Task 03 completion has fresh evidence from the full race/build gate, all-package real-service
  integration tests against PostgreSQL/NATS/MinIO/Valkey, 93 frontend unit tests plus
  lint/typecheck/build, formatting and Markdown checks, and 14 Playwright/axe journeys including a
  real API persistence path. All 184 assigned identifiers map to concrete tests and all ten visual
  evidence directories were refreshed. A final uncached `go test -race -count=1 ./...` also passed.
- Task 04 is complete with immutable versioned metric/health definitions and normalized factors,
  deterministic half-open-window cohort calculations, verified/Analyst-confirmed contributor
  identities and immutable corrections, exact-replay-safe PostgreSQL materialization, and coherent
  two-to-five Project comparisons. Collection and targeted recalculation jobs now freeze the
  supported preset windows from canonical evidence; metric, health, contributor, and comparison
  HTTP reads use the generated contract and bounded pagination.
- Task 04 completion has fresh evidence from `make check`, every real-PostgreSQL integration test,
  frontend lint/typecheck/93 unit tests/build, formatting and Markdown checks, and all six assigned
  Playwright/axe journeys. Required S9, S10, and S12 narrow/wide visual artifacts are present under
  `artifacts/task_04/ui/`.
- Task 05 is complete with canonical extended public-source adapters, source-contextual adoption
  and qualified security intelligence, public-only bounded crawling, immutable snapshots/chunks,
  deterministic versioned hybrid retrieval, correctable mutual-kNN topics, deterministic release
  records, and immutable evidenced analysis runs with feedback/rerun/selection governance.
- Task 05 persistence rejects schema-invalid or unevidenced success, cross-project or post-cutoff
  citations, changed immutable replays, and writes after project deletion starts. Direct release
  reads remain addressable independently of list pagination, and deterministic data remains usable
  when model capabilities fail.
- Task 05 completion has fresh evidence from `make check`, the full real-PostgreSQL integration
  suite, frontend lint/typecheck/93 unit tests/build, formatting and Markdown checks, and all 11
  assigned Playwright/axe journeys. Required S11, S16, S17, S18, and S20 narrow/wide artifacts are
  present under `artifacts/task_05/ui/`.
