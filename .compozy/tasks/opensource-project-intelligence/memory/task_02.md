# Task 02 Memory: Identity, Access Governance, Audit, and Application Shell

## Scope

Implement only task_02: local identity and authorization, browser and bearer access, membership and
service-account governance, immutable redacted audit, operations visibility, and the generated-client
backed bilingual application shell for stories US-001 through US-004, US-031, and US-032.

## Normative tests

- Unit: UT-001 through UT-028, UT-211 through UT-224, UT-247 through UT-250, and UT-272.
- Integration: IT-001 through IT-012, IT-091 through IT-096, IT-100 through IT-102, and IT-147
  through IT-174.
- End-to-end: E2E-001 through E2E-004, E2E-031 through E2E-035, and E2E-055.

## Baseline

- Task 01 is complete and its uncommitted generated-output and workflow-tracking changes must be
  preserved.
- The worktree is on `main`, ahead of `origin/main`, with existing task_01 changes in generated Go
  and TypeScript boundaries, checksums, shared memory, and task tracking.
- The managed session does not expose canonical `compozy__skill_view`; execution follows the required
  memory-first, task-execution, and fresh-verification workflow from the task sources without reading
  managed skill files through prohibited fallback surfaces.

## Progress

- 2026-08-22: Read workspace and repository instructions, loaded mandatory local Go guidance, read
  shared workflow memory and task_02, and initialized task-local memory before source edits.
- 2026-08-22: Read the specification and task-owned story, DX, UI, test, and ADR contracts. Frozen
  implementation order: access persistence/domain, authentication and generated HTTP, localized
  generated-client UI, then fresh repository verification.
- 2026-08-22: Audited the inherited task implementation across the access domain, PostgreSQL
  adapter and migration, OIDC/session HTTP boundary, generated client, localized router and shells,
  task-owned unit/integration/browser tests, and all eight required visual-evidence directories.
- 2026-08-22: Corrected the platform migration round-trip integration test to roll back every
  applied version. The prior one-step assertion became invalid when task_02 added the reviewed
  identity/access migration, while the migration command intentionally reverts one version at a
  time.
- 2026-08-22: Made task browser navigation wait for DOM readiness and application assertions rather
  than the nondeterministic full-load event blocked by optional external web fonts. The complete
  ten-journey suite then passed in one run and refreshed the account/member visual evidence.
- 2026-08-22: Fresh verification passed generated-contract drift, Go vet/race/build, all Go unit
  packages, all PostgreSQL integration packages, frontend lint/typecheck, 87 frontend tests, the
  production build, and all ten Playwright/axe journeys. PostgreSQL 18/pgvector, JetStream, Valkey,
  and MinIO were healthy during environment-backed verification.

## Follow-up boundaries

- Keep task_03 and later project, ingestion, metrics, AI, recommendations, alert, export, and final
  hardening capabilities outside this task unless a shared identity or shell contract explicitly
  requires a stable seam.
- The Vite development shell still emits React Router's non-fatal missing `HydrateFallback` warning,
  and the production build reports the already-documented initial-chunk size warning. Route-level
  code splitting and hydration fallback cleanup remain later frontend-hardening work.
