# Task 03 Workflow Memory

## Scope

- Implement only task_03: Projects, typed Repositories, Sources and associations, lifecycle and
  purge, durable Jobs/outbox/checkpoints, evidence ownership, core GitHub/restricted-Git ingestion,
  and the S4-S8 browser journeys.
- Treat `.compozy/tasks/opensource-project-intelligence/` as the workflow source and the numbered
  repository `specs/` as the implementation contract.
- Leave all work uncommitted. Do not alter or revert task_02's existing uncommitted files.

## Inherited Worktree Boundary

The session began with task_02 changes in its workflow memory/task file, account/member browser
evidence and tests, and `internal/platform/database/platform_integration_test.go`. Those paths are
owned by task_02 and must be preserved unless task_03 has an unavoidable additive overlap.

## Frozen Task Invariants

- A Project is distinct from its one-or-more typed Repositories and always has exactly one primary
  Repository.
- PostgreSQL is authoritative. Aggregate changes, idempotency, Jobs, and outbox records commit in
  one transaction. JetStream is at-least-once delivery, Valkey is disposable acceleration, and S3
  contains only PostgreSQL-owned immutable bytes.
- Checkpoints advance only with committed canonical facts and provenance. Redelivery cannot create
  duplicate facts or terminal-state regression.
- Project lifecycle is Admin-only. Permanent deletion makes the Project unavailable first, blocks
  late collector writes, resumes partial purge from an owned manifest, and leaves only the minimal
  audit tombstone.
- Collection accepts public sources only. Every URL/redirect/DNS hop and Git transport is
  fail-closed; credentials are operator-owned and always redacted.
- The S4-S8 UI uses the generated HTTP client, localized URL state, cursor history, ETags,
  idempotency keys, resumable SSE with polling fallback, and complete narrow/wide accessible states.

## Required Verification

- All task-assigned unit, integration, race, generated-contract, web unit/build, and Playwright/axe
  checks must have fresh passing evidence before `task_03.md` can become `status: completed`.
- The managed Compozy skill-view capability was not projected into this worker session, so the
  required `cy-workflow-memory`, `cy-execute-task`, and `cy-final-verify` skill bodies could not be
  loaded through their permitted native surface. The task continues from the embedded workflow
  requirements without bypassing that policy through the operator CLI or direct managed files.

## Progress Notes

- 2026-08-22: Read workspace/repository instructions, the complete workflow specification and
  companion catalogs relevant to task_03, task tracking, workflow memory, and ADRs 005, 008, 013,
  016, 020, 022, 023, 026, 030, and 035.
- 2026-08-22: Loaded the repository's mandatory Go guidance for database, concurrency, context,
  security, testing, errors, naming, modernization, safety, documentation, and code style.
- 2026-08-22: Preserved the inherited worktree and extended Project registration so concurrent
  same-URL requests, including different idempotency keys, resolve one authoritative Project and
  initial Job under PostgreSQL advisory locks.
- 2026-08-22: Extended core GitHub ingestion from issues to independently checkpointed issues,
  pull requests, releases, and commits. Raw evidence retention, canonical upserts, source coverage,
  and each checkpoint advance share one transaction.
- 2026-08-22: Replaced one-shot Job SSE with durable replay plus live polling, heartbeats,
  Last-Event-ID resumption, terminal close, cancellation, and JSON fallback.
- 2026-08-22: Added real JetStream and MinIO integration tests. The JetStream test found and drove
  fixes for NATS header negotiation, HMSG response parsing, unique subscription IDs, and automatic
  request unsubscribe; stable message redelivery now deduplicates on the real broker.
- 2026-08-22: Fresh `make check`, all-package real-service integration, frontend lint/typecheck/
  unit/build, and the 13-case Task 3 Playwright/axe suite passed. Task 3 remains pending because the
  broker consumer, Valkey functional degradation, real-API browser journeys, full visual matrix,
  and 67 explicitly assigned normative test mappings are still absent. The task file contains the
  exact evidence and identifier audit.
- 2026-08-22: Continuation session recovered the existing dirty worktree, loaded the managed
  `cy-workflow-memory`, `cy-execute-task`, and `cy-final-verify` instructions plus the tracking
  checklist, completed an inline corpus survey because child-session capacity is unavailable, and
  confirmed the same four implementation gaps and 67 missing test mappings before further edits.
- 2026-08-22: The first fresh baseline proved generated-source drift clean. Go verification must use
  repository-local ignored `GOCACHE`/`GOTMPDIR` paths because the global cache is read-only and
  `/tmp` is mounted noexec in this session; those environmental failures are not code failures.
- 2026-08-22: Closed the recovery gaps with a PostgreSQL-authoritative durable JetStream consumer,
  explicit ack/retry/heartbeat semantics, Valkey disposable wake-ups, a real generated-client/API
  browser persistence journey, expanded S4-S8 visual evidence, and concrete mappings for all 184
  assigned identifiers.
- 2026-08-22: Real NATS verification exposed that JetStream HMSG deliveries preserve the stream
  subject and identify the inbox subscription by SID. Corrected the parser, decoded 404/408 status
  frames, isolated the integration consumer, and added matching protocol regressions.
- 2026-08-22: Final fresh evidence passed: `make check`; all-package integration tests against
  PostgreSQL, NATS, MinIO, and Valkey; root pnpm lint/typecheck/test/build with 93 Vitest tests;
  Prettier and Markdown checks; and 14 Task 3 Playwright/axe journeys including the real API path.
  An additional `go test -race -count=1 ./...` passed every Go package without cache reuse. Task 3
  is complete and remains uncommitted.
