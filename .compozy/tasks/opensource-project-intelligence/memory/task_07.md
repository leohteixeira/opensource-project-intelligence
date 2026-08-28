# Task 07 Workflow Memory

## Scope

- Implement only task `task_07`: bounded ADK/HITL actions, authorized asynchronous exports,
  immutable audit completion, model-provider operations/degradation, and final operational
  hardening.
- Complete US-025 and US-028 through US-030 together with every assigned unit, integration,
  end-to-end, API-contract, accessibility, localization, operational, and visual-contract case.
- Preserve all unrelated and inherited task 01 through task 06 worktree changes; leave this task
  uncommitted.

## Inherited Boundary

- Build on the canonical identity, authorization, project, evidence, job, outbox, object-storage,
  analysis, audit, generated HTTP, and frontend-shell foundations delivered by tasks 01 through 06.
- Keep Google ADK Go v2 inside `internal/analysis/agent` behind consumer-owned ports and typed,
  allowlisted tools; it receives no database, filesystem, broker, object-store, provider-SDK, or
  arbitrary HTTP capability.
- Do not widen into redesigning deterministic metrics, trends, policies, radar, alerts, ingestion,
  or other completed verticals except where task 07 requires integration or release hardening.

## Frozen Invariants

- One assistant proposal represents exactly one typed, non-destructive Analyst action. It displays
  operation, resources, values, effect, quota, and a ten-minute expiry before confirmation.
- Confirmation is action-bound, single-use, idempotent, and revalidates the current identity, role,
  scope, resource version, lifecycle, and quota. Membership, credentials, policy authoring,
  archive, deletion, Admin-only, destructive, multi-action, and untyped requests are refused.
- Exports are durable Jobs over one explicit authorization scope and UTC cutoff. CSV machine fields
  remain stable, supported human labels are localized, evidence JSON retains interpretation
  context, artifacts are SHA-256 bound, and successful files expire after exactly 24 hours or
  earlier with owned Project deletion.
- Audit events are immutable, redacted, stably ordered, bounded, Admin-only, and retain only opaque
  actor/Project tombstones after deletion. Retries and failures remain attributable without
  fabricating duplicate success or state change.
- Provider configuration validates before activation and is frozen per run. Missing, disabled,
  unhealthy, interrupted, or quota-limited providers degrade only AI-dependent work; deterministic
  collection and intelligence remain ready.
- All I/O propagates `context.Context`; assistant tools, model calls, export generation, audit
  pagination, jobs, retry loops, shutdown, storage, and reconciliation are explicitly bounded and
  observable without secrets.

## Required Verification

- Run every Validation, Test Plan, and Testing command named by `task_07`, the repository contracts,
  and the final-verification discovery: generated drift, Go formatting/vet/unit/race/build, all
  real-service integration suites, frontend lint/typecheck/unit/build, assigned Playwright/axe
  journeys, Compose health, backup/restore, operational/security gates, and pre-commit checks.
- Refresh S19 and S22 narrow/wide evidence under `artifacts/task_07/ui/` and verify English,
  Brazilian Portuguese, keyboard, focus, accessible-name/status, reduced-motion, and responsive
  behavior.
- Use fresh command output immediately before changing `task_07.md` frontmatter to
  `status: completed`.

## Progress Notes

- 2026-08-22: Read workspace/repository instructions, task/spec companions, relevant user stories,
  UI/DX/test contracts, shared workflow memory, and inherited task 06 memory. Confirmed the only
  pre-existing worktree changes belong to task 06 and must be preserved.
- 2026-08-22: The managed harness exposes no callable `compozy__tool_search`,
  `compozy__tool_info`, or `compozy__skill_view`; the required managed skill bodies cannot be
  loaded without violating the runtime's no-CLI/no-direct-read rule. Continuing from the embedded
  workflow requirements and repository-mandated Go guidance.
- 2026-08-28: Completed the bounded assistant boundary in `internal/analysis/agent`, including
  finite step, deadline, output, cost, model-concurrency, and typed-tool-concurrency limits. The
  only state-changing tool remains the allowlisted non-destructive repository-add action;
  forbidden actions fail before HITL, while confirmation is actor/action/version bound,
  single-use, ten-minute limited, and revalidated at execution.
- 2026-08-28: Completed requester-authorized durable exports with normalized immutable requests,
  one cutoff, stable localized CSV and evidence JSON, SHA-256 metadata, 24-hour expiry, and Project
  purge ownership. Added durable request-to-Project references and corrected purge SQL typing so
  deletion reconciliation expires succeeded artifacts without crossing requester boundaries.
- 2026-08-28: Completed Admin audit filters and the redacted model-provider operations projection,
  made AI optional for deterministic readiness, added finite runtime configuration, and delivered
  generated-client S19/S22 English and Brazilian Portuguese routes with narrow/wide axe and visual
  evidence. Updated the production runbook for readiness, backup/restore, export retention,
  graceful shutdown, incident redaction, and release gates.
- 2026-08-28: Verified all 57 assigned identifiers exist exactly once. Fresh final gates passed:
  generated drift; Go vet, race tests, and build; the complete real-PostgreSQL integration suite;
  frontend lint, typecheck, 93 unit tests, and production build; all six Task 07 Playwright/axe
  journeys; and every pre-commit hook including formatting, lint, and secret scanning.
- 2026-08-28: Rehearsed all production Compose image builds, API health/readiness with optional AI
  disabled, API/worker graceful `SIGTERM` exits, and PostgreSQL plus object-manifest backup/restore
  checksums against disposable databases. Task 07 is complete and remains uncommitted.
