---
status: completed
title: Implement Projects, durable work, evidence ownership, and core source ingestion
type: fullstack
complexity: critical
---

<!-- markdownlint-disable MD013 MD025 -->

# Task 3: Implement Projects, durable work, evidence ownership, and core source ingestion

## Overview

Deliver the transactional core of the product: Projects distinct from typed Repositories, governed
sources and associations, durable Jobs and events, raw/canonical evidence ownership, GitHub and
restricted Git ingestion, resumable synchronization/history, lifecycle and purge semantics, and the
platform primitives downstream intelligence uses.

<critical>
- ALWAYS READ `_spec.md` and its catalogs (`_user_stories.md`, `_dx.md`, `_uiux.md` when present, `_tests.md`) before starting
- REFERENCE `_spec.md` Part II for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — implement every test case assigned in ## Tests
</critical>

<requirements>

- Implement US-005 through US-012 in full, including every accepted edge case.
- Keep Project, Repository, Source, association, raw evidence, canonical facts, checkpoints,
  snapshots, Jobs, attempts, outbox messages, and object ownership explicit and auditable.
- Commit aggregate changes, Jobs, idempotency records, and outbox messages atomically in PostgreSQL;
  use JetStream for delivery, Valkey only for acceleration, and S3 only for owned immutable bytes.
- Bound, lease, retry, checkpoint, coalesce, rate-limit, cancel, and gracefully stop worker activity.
- Accept only public eligible sources; harden URL resolution, redirects, DNS rebinding, Git
  transports, filesystem isolation, resource limits, and credential redaction.
- Enforce Admin-only Project lifecycle and unavailable-first resumable deletion while preserving the
  minimal audit tombstone.
- Deliver the Portfolio, Projects, Project header/lifecycle, repositories/sources/associations, and
  Jobs/history browser journeys using real generated client data.

</requirements>

## Visual Contract

| Surface      | Viewport/state contract                                                                                                                                 | Reference                             | Required evidence               |
| ------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------- | ------------------------------- |
| S4 portfolio | Narrow 320px: populated, no Projects, filtered empty, mixed freshness/status, partial panel error, refresh, archive removal, and scale                  | PortfolioScreen                       | artifacts/task_03/ui/s4-narrow/ |
| S4 portfolio | Wide: the same independent panels with evidence links, bounded attention queue, URL filters, and retained valid panels                                  | PortfolioScreen                       | artifacts/task_03/ui/s4-wide/   |
| S5 Projects  | Narrow 320px: list/empty/search/page, valid preview, unsafe/duplicate/quota/conflict, accepted sync, recoverable failure, and backfill load             | ProjectsScreen                        | artifacts/task_03/ui/s5-narrow/ |
| S5 Projects  | Wide: the same states with registration and identity editing, ETags, role-appropriate actions, and stable page history                                  | ProjectsScreen                        | artifacts/task_03/ui/s5-wide/   |
| S6 lifecycle | Narrow 320px: active, pause/archive/restore, conflicts, exact deletion, purge/tombstone, repeat action, and forbidden roles                             | LifecycleScreen                       | artifacts/task_03/ui/s6-narrow/ |
| S6 lifecycle | Wide: the same states with effect summaries, irreversible confirmation, Job progress, focus management, and read-only archive                           | LifecycleScreen                       | artifacts/task_03/ui/s6-wide/   |
| S7 sources   | Narrow 320px: repository roles/primary replacement, source/capability states, association confidence/correction, hostile input, pagination, and archive | SourcesScreen                         | artifacts/task_03/ui/s7-narrow/ |
| S7 sources   | Wide: the same states with accessible tables, evidence detail, redaction, stale derived data, and bounded correction workflows                          | SourcesScreen                         | artifacts/task_03/ui/s7-wide/   |
| S8 Jobs      | Narrow 320px: SSE connected/reconnecting/resumed/fallback and every queued/running/succeeded/partial/failed/cancelled/coalesced/quota/checkpoint state  | ProjectDetailScreen and SourcesScreen | artifacts/task_03/ui/s8-narrow/ |
| S8 Jobs      | Wide: the same states with source freshness, factual progress, requested/actual coverage, history requests, retry, and pagination                       | ProjectDetailScreen and SourcesScreen | artifacts/task_03/ui/s8-wide/   |

## Subtasks

- [x] Implement Project, Repository, Source, association, lifecycle, ownership, Job, attempt,
      checkpoint, idempotency, outbox, and purge aggregates and persistence.
- [x] Implement transactional commands and every assigned portfolio/project/repository/source/
      association/sync/history/Job HTTP operation.
- [x] Implement JetStream relay/consumer semantics, worker leases, retries, dead-letter advisories,
      cancellation, SSE replay, polling fallback, quotas, and graceful shutdown.
- [x] Implement S3 atomic visibility/checksum ownership and Valkey degradation without authority.
- [x] Implement GitHub and restricted Git collectors, canonical mapping, initial backfill,
      incremental synchronization, event ordering, and public-only transport safety.
- [x] Implement Project identity resolution and reversible Analyst corrections with targeted
      invalidation.
- [x] Replace S4–S8 fixtures with generated-client queries/mutations while preserving the imported
      design system and frozen responsive states.
- [x] Implement every assigned unit, real-service integration, race, and browser test.

## Implementation Details

### Relevant Files

- \_spec.md Part II domain, persistence, worker, security, API, and implementation-order sections
- \_user_stories.md US-005 through US-012
- \_dx.md portfolio, Projects, repositories, sources, Jobs, concurrency, and lifecycle routes
- \_uiux.md S4 through S8
- adrs/adr-005.md, adr-008.md, adr-013.md, adr-016.md, adr-020.md, adr-022.md,
  adr-023.md, adr-026.md, adr-030.md, and adr-035.md
- internal/project/, internal/repository/, internal/collector/, internal/issue/,
  internal/pullrequest/, internal/release/, and internal/platform/
- cmd/api/main.go, cmd/worker/main.go, migrations/, compose.yaml
- apps/web/src/kits/workspace/ and apps/web/src/kits/project-evidence/

### Dependent Files

- Tasks 04–07 consume the canonical facts, evidence references, Job/outbox pipeline, source
  capabilities, snapshots, lifecycle guards, audit hooks, and real Project UI established here.

### Related ADRs

- [Workflow ADR 005](adrs/adr-005.md), [ADR 008](adrs/adr-008.md),
  [ADR 013](adrs/adr-013.md), [ADR 016](adrs/adr-016.md),
  [ADR 020](adrs/adr-020.md), [ADR 022](adrs/adr-022.md),
  [ADR 023](adrs/adr-023.md), [ADR 026](adrs/adr-026.md),
  [ADR 030](adrs/adr-030.md), and [ADR 035](adrs/adr-035.md).
- Repository ADRs 0002 through 0005 and the infrastructure/generation ADRs added by Task 01.

## Deliverables

- Complete Project/Repository/Source/association lifecycle and APIs.
- Durable PostgreSQL-backed Job/outbox/worker system with resumable SSE and polling.
- Public-only GitHub/restricted-Git ingestion, canonical facts, raw evidence, provenance, and
  checkpoints.
- Safe S3 ownership, Valkey degradation, JetStream delivery, quotas, and resumable purge.
- Production S4–S8 browser journeys with complete responsive/accessibility states.
- Passing evidence for all 184 assigned tests and every visual-contract row.

## Tests

Implement these normative cases from \_tests.md exactly once:

- Unit: UT-029, UT-030, UT-031, UT-032, UT-033, UT-034, UT-035, UT-036, UT-037, UT-038, UT-039, UT-040, UT-041, UT-042, UT-043, UT-044, UT-045, UT-046, UT-047, UT-048, UT-049, UT-050, UT-051, UT-052, UT-053, UT-054, UT-055, UT-056, UT-057, UT-058, UT-059, UT-060, UT-061, UT-062, UT-063, UT-064, UT-065, UT-066, UT-067, UT-068, UT-069, UT-070, UT-071, UT-072, UT-073, UT-074, UT-075, UT-076, UT-077, UT-078, UT-079, UT-080, UT-081, UT-082, UT-083, UT-084, UT-251, UT-252, UT-253, UT-254, UT-255, UT-256, UT-257, UT-258, UT-259, UT-260, UT-263, UT-264, UT-265, UT-268, UT-270, UT-274
- Integration: IT-013, IT-014, IT-015, IT-016, IT-017, IT-018, IT-019, IT-020, IT-021, IT-022, IT-023, IT-024, IT-025, IT-026, IT-027, IT-028, IT-029, IT-030, IT-031, IT-032, IT-033, IT-034, IT-035, IT-036, IT-103, IT-104, IT-105, IT-106, IT-107, IT-108, IT-109, IT-110, IT-111, IT-112, IT-113, IT-114, IT-116, IT-119, IT-127, IT-128, IT-129, IT-131, IT-132, IT-133, IT-134, IT-139, IT-140, IT-141, IT-142, IT-143, IT-144, IT-145, IT-146, IT-175, IT-176, IT-177, IT-178, IT-179, IT-180, IT-181, IT-182, IT-183, IT-184, IT-185, IT-186, IT-187, IT-188, IT-189, IT-190, IT-191, IT-192, IT-193, IT-194, IT-195, IT-196, IT-197, IT-198, IT-199, IT-200, IT-201, IT-202, IT-203, IT-204, IT-205, IT-206, IT-207, IT-208, IT-209, IT-210, IT-211, IT-212, IT-213, IT-214, IT-215, IT-216, IT-217, IT-218, IT-219, IT-220
- End-to-end: E2E-005, E2E-006, E2E-007, E2E-008, E2E-009, E2E-010, E2E-011, E2E-012, E2E-036, E2E-037, E2E-038, E2E-039, E2E-040

## Success Criteria

- [x] Commands never publish partial aggregate/Job/outbox state and redelivery never duplicates
      canonical facts.
- [x] Workers remain bounded, leased, resumable, observable, cancellation-aware, and race-free.
- [x] Unsafe/private sources cannot cause protected HTTP, local Git execution, secret exposure, or
      unowned object visibility.
- [x] Project lifecycle, purge, archive, corrections, coverage, quotas, Jobs, and UI states satisfy
      their frozen contracts.
- [x] All 184 assigned tests pass with fresh evidence.

## Recovery Continuation Evidence — 2026-08-22

The recovery continuation preserved the prior uncommitted Task 2 and Task 3 work and completed these
additional Task 3 slices:

- Concurrent registration now serializes by idempotency scope and canonical repository identity;
  every duplicate request resolves the same Project and initial Job without creating partial state.
- GitHub ingestion now owns independently checkpointed issue, pull-request, release, and commit
  pages, retaining raw DTO bytes and committing canonical facts, coverage, and checkpoints together.
- Job event delivery now provides a resumable live SSE stream with durable replay, terminal close,
  heartbeats, and the existing JSON polling fallback.
- Real JetStream verification corrected NATS header negotiation, header-bearing response parsing,
  and request subscription lifecycle; stable message identity now deduplicates broker retries.
- A real MinIO contract verifies staging, atomic promotion, checksum-preserving reads, and purge.

Fresh passing evidence gathered after the implementation changes:

- `make check`: generated-source drift, Go formatting, `go vet`, `go test -race ./...`, and
  `go build ./...` passed.
- `go test -tags=integration ./... -count=1` passed against the running PostgreSQL 18, NATS
  JetStream, and MinIO services with explicit integration endpoints.
- `pnpm lint`, `pnpm typecheck`, `pnpm test`, and `pnpm build` passed; Vitest reported 87 tests.
- `pnpm --dir apps/web exec playwright test e2e/task03.spec.ts` passed 13 Chromium tests including
  the current axe scan and refreshed narrow/wide screenshots.

The final recovery closed every previously recorded blocker:

- The worker now consumes broker-selected Job IDs through an explicit-ack durable JetStream pull
  consumer while PostgreSQL remains authoritative. Real broker verification found and fixed the
  HMSG subscription-SID parsing rule and status-frame handling; retries and dead-letter advisories
  are committed through the outbox before acknowledgement.
- Valkey now provides disposable publish/subscribe wake-ups with a real-service degradation
  contract; failures never replace PostgreSQL polling or committed Job state.
- `task03.real.spec.ts` exercises the current generated client against the Go E2E backend and
  verifies registration persistence after reload, Job visibility, and Portfolio visibility.
  `task03.spec.ts` covers the frozen S4-S8 responsive state matrix and Axe, refreshing all ten
  required visual evidence directories.
- The assignment audit found all 184 normative identifiers in concrete Go, Vitest, or Playwright
  tests.

Final fresh evidence:

- `make check` passed generation drift, Go formatting, `go vet ./...`, `go test -race ./...`, and
  `go build ./...`.
- `go test -race -count=1 ./...` passed every Go package without test-result cache reuse.
- `go test -tags=integration ./... -count=1` passed against PostgreSQL 18, NATS JetStream, MinIO,
  and Valkey with explicit local integration endpoints.
- `pnpm lint`, `pnpm typecheck`, `pnpm test`, and `pnpm build` passed; Vitest reported 93 tests.
- `pnpm format:check` and `pnpm lint:md` passed.
- `pnpm --dir apps/web exec playwright test e2e/task03.spec.ts e2e/task03.real.spec.ts` passed all
  14 Chromium journeys and the Axe scan, with current narrow/wide artifacts under
  `artifacts/task_03/ui/`.
