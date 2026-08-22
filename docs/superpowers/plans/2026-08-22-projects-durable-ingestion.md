# Projects, Durable Work, and Core Ingestion Implementation Plan

> **For agentic workers:** Execute this plan inline for task_03 only. The conductor explicitly
> requires an uninterrupted implementation and an uncommitted handoff, so commit steps and a
> separate execution-choice pause are intentionally omitted.

**Goal:** Deliver the task_03 Project/source lifecycle, durable work and evidence core, public-only
GitHub/restricted-Git ingestion, and production S4-S8 browser journeys.

**Architecture:** Keep provider-neutral rules in consuming business packages, explicit pgx/sqlc
adapters under `internal/platform`, and transactional use-case boundaries in PostgreSQL. Publish
only committed outbox work to JetStream, treat Valkey as disposable acceleration, and expose the
same authoritative Job representation through reads and resumable SSE.

**Tech Stack:** Go 1.26, `net/http`, pgx/v5, sqlc, PostgreSQL 18/pgvector, NATS JetStream,
S3-compatible storage, Valkey, React 19, React Router, TanStack Query, generated Hey API client,
Vitest/Testing Library, and Playwright/axe.

---

## Task 1: Add provider-neutral Project, Source, Job, and evidence rules

**Files:**

- Create: `internal/project/project.go`
- Create: `internal/project/project_test.go`
- Create: `internal/job/job.go`
- Create: `internal/job/job_test.go`
- Create: `internal/evidence/evidence.go`
- Create: `internal/evidence/evidence_test.go`
- Modify: `internal/repository/doc.go`

- [ ] Define typed lifecycle, repository roles, source kinds/states, association decisions,
      coverage/checkpoints, Job states/transitions, purge manifests, and checksum ownership with
      constructors that reject unsupported values and terminal-state regression.
- [ ] Add table-driven tests named with UT-029 through UT-084 and UT-251 through UT-274 wherever
      their behavior is owned by these domain rules.
- [ ] Run `go test ./internal/project ./internal/job ./internal/evidence` and require every package
      to pass before persistence work.

## Task 2: Add public-source and restricted-Git collection boundaries

**Files:**

- Create: `internal/collector/source.go`
- Create: `internal/collector/source_test.go`
- Create: `internal/platform/github/client.go`
- Create: `internal/platform/github/client_test.go`
- Create: `internal/platform/git/repository.go`
- Create: `internal/platform/git/repository_test.go`

- [ ] Implement URL canonicalization and per-hop DNS/IP validation for HTTPS public endpoints;
      reject credentials, unsupported ports, loopback/private/link-local/multicast addresses, and
      source visibility other than public.
- [ ] Implement Git subprocess construction with `exec.CommandContext`, an argument vector,
      disabled credential helpers/hooks/submodules/file protocol, an environment allowlist, output and
      duration bounds, and a task-owned isolated mirror root.
- [ ] Map GitHub fixtures immediately into provider-neutral Repository, issue, pull-request,
      release, and temporal-event inputs; keep raw bytes only behind evidence references.
- [ ] Run collector and adapter unit tests, including UT-036, UT-078-084, UT-254-255 and IT-114,
      IT-116, IT-119-120 controlled-boundary cases.

## Task 3: Add the transactional PostgreSQL ownership model

**Files:**

- Create: `migrations/0003_projects_jobs_ingestion.up.sql`
- Create: `migrations/0003_projects_jobs_ingestion.down.sql`
- Create: `internal/platform/database/query/projects.sql`
- Regenerate: `internal/platform/database/sqlc/`
- Create: `internal/platform/projectstore/store.go`
- Create: `internal/platform/projectstore/store_integration_test.go`

- [ ] Add Project, Repository, Source, association/correction, idempotency, Job/attempt/event,
      outbox, checkpoint, raw/canonical fact, object ownership, purge manifest, and tombstone tables
      with scoped unique/check/state/one-primary constraints and unavailable-first deletion guards.
- [ ] Implement project registration as one transaction that creates exactly one Project, primary
      Repository, initial-sync Job, idempotency result, audit entry, and outbox row or commits none.
- [ ] Implement conditional Project/repository/source mutations, targeted association correction,
      compatible sync/history coalescing, lease/heartbeat/checkpoint updates, cancellation, page commits,
      and resumable purge using parameterized context-aware SQL.
- [ ] Run generation drift and PostgreSQL integration cases IT-013-036, IT-103-114, IT-119,
      IT-129, IT-133, IT-139-144 and the task's route pairs IT-175-220.

## Task 4: Wire generated HTTP operations and resumable Job presentation

**Files:**

- Create: `internal/platform/projectapi/api.go`
- Create: `internal/platform/projectapi/api_test.go`
- Modify: `internal/platform/accessapi/api.go`
- Modify: `cmd/api/main.go`

- [ ] Reuse the access middleware's committed local Principal, enforce Viewer/Analyst/Admin
      capabilities server-side, bind cursors to route/filter context, require idempotency keys and
      `If-Match`, and serialize frozen safe problems without wrapped causes.
- [ ] Implement every generated task_03 route: portfolio; Project list/create/get/update/lifecycle;
      repositories; sources; associations/corrections; sync/history; Project Jobs; Job read/SSE/cancel.
- [ ] Emit full Job representations in monotonically versioned SSE events, honor `Last-Event-ID`,
      close on terminal state, return `Retry-After` on active Job reads, and preserve PostgreSQL polling
      when Valkey is absent.
- [ ] Run the generated transport contract tests IT-175-220 plus UT-251-253 and UT-271.

## Task 5: Wire the bounded worker and infrastructure adapters

**Files:**

- Create: `internal/platform/work/dispatcher.go`
- Create: `internal/platform/work/dispatcher_test.go`
- Create: `internal/platform/objectstore/store.go`
- Create: `internal/platform/objectstore/store_test.go`
- Modify: `cmd/worker/main.go`

- [ ] Relay committed outbox rows with their identity as the JetStream deduplication ID, consume
      bounded pull batches with explicit acknowledgement, and make handler redelivery observe
      PostgreSQL checkpoint/idempotency truth.
- [ ] Claim Jobs through expiring leases, record attempts and heartbeats, stop starting pages after
      cancellation, resume rate-limited work with bounded backoff/jitter, dead-letter exhausted work,
      and drain owned goroutines during graceful shutdown.
- [ ] Upload immutable content-addressed bytes before committing visible ownership references;
      verify digest/size on reads and resume manifest-driven deletion/reconciliation.
- [ ] Run IT-103-113, IT-127-134, IT-137 and `go test -race ./...`.

## Task 6: Replace S4-S8 fixtures with generated-client browser features

**Files:**

- Create: `apps/web/src/application/projects.ts`
- Create: `apps/web/src/application/screens/PortfolioScreen.tsx`
- Create: `apps/web/src/application/screens/ProjectsScreen.tsx`
- Create: `apps/web/src/application/screens/ProjectSourcesJobsScreen.tsx`
- Create: `apps/web/src/application/task03.test.tsx`
- Create: `apps/web/src/application/task03.integration.test.tsx`
- Create: `apps/web/e2e/task03.spec.ts`
- Modify: `apps/web/src/application/router.tsx`
- Modify: `apps/web/src/application/i18n.ts`
- Modify: `apps/web/src/application/api.ts`

- [ ] Add generated-client query/mutation adapters for portfolio, Project identity/lifecycle,
      repositories/sources/associations, sync/history and Jobs; retain ETags, create idempotency keys,
      preserve cursor chains in URL state, replace state from full SSE events, and poll after stream
      gaps/failure.
- [ ] Compose the existing design-system and kit references into localized S4-S8 routes with
      independent panel failure, empty/stale/unknown states, role-hidden actions, exact destructive
      confirmation, narrow row disclosure, keyboard focus restoration, and non-color status text.
- [ ] Run web lint/typecheck/unit/build and E2E-005-012 plus E2E-036-040 in en/pt-BR at narrow/wide
      viewports with keyboard and axe checks; capture the ten required visual evidence directories.

## Task 7: Fresh verification and workflow closure

**Files:**

- Modify: `.compozy/tasks/opensource-project-intelligence/memory/task_03.md`
- Modify: `.compozy/tasks/opensource-project-intelligence/memory/MEMORY.md`
- Modify: `.compozy/tasks/opensource-project-intelligence/task_03.md`

- [ ] Run `make generate-check`, `make check`, the real-service integration target, root pnpm
      lint/typecheck/test/build, Playwright/axe, markdown/format checks, and every task body Validation,
      Test Plan, or Testing command with fresh output.
- [ ] Confirm no `.env` file was read or modified, task_02 dirty files remain preserved, generated
      outputs are clean, no assigned test is skipped, and all ten visual-contract directories contain
      current evidence.
- [ ] Record exact commands/results and bounded follow-ups in task/shared memory, then change only
      task_03 frontmatter from `status: pending` to `status: completed` if every required check passed.
