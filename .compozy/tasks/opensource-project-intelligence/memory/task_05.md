# Task 05 Workflow Memory

## Scope

- Implement only task `task_05`: extended public sources, bounded knowledge crawling and retrieval,
  topics, releases, and immutable AI analyses.
- Complete US-015, US-021 through US-024, and US-026 together with the assigned unit,
  integration, end-to-end, API-contract, and UI tests.
- Preserve all unrelated and inherited task 01 through task 04 worktree changes; leave this task
  uncommitted.

## Inherited Boundary

- Build on the canonical project, snapshot, run, authorization, provenance, event, metric,
  comparison, and contributor foundations implemented by tasks 01 through 04.
- Do not widen into saved views, exports, notifications, administration, or agent approval flows
  assigned to later tasks.
- Provider DTOs remain inside adapters. Business packages expose canonical models and
  consumer-owned interfaces only.

## Frozen Invariants

- Adoption values retain source unit and population context; incomparable units remain separate,
  and missing public advisory or registry evidence is `unknown`, never zero.
- Every crawl DNS resolution and redirect hop is validated. Domain, depth, byte, page, media-type,
  rate, and robots limits are hard bounds. Raw snapshots and deterministic parser/chunk versions
  remain immutable and attributable.
- Hybrid retrieval combines PostgreSQL FTS and pgvector with deterministic, versioned reciprocal
  rank fusion. Authorization, project/source filters, and cutoff time are enforced before ranking;
  citations are immutable evidence references.
- Topic candidates use mutual-kNN edges and deterministic connected components. Analyst
  corrections are canonical without mutating generated history; generated labels and AI outputs
  are immutable run artifacts.
- Structured AI output is accepted only after schema and evidence validation. Failed attempts do
  not replace a prior successful result or enter success-only series selection.
- Release metadata and lexical search continue to work when embedding or model providers are
  unavailable.
- All I/O propagates `context.Context`; concurrency and retry behavior are bounded; error causes
  are wrapped and preserved.

## Required Verification

- Run every Validation, Test Plan, and Testing command named by `task_05.md`, including the full Go
  unit/integration/race gates and the repository pnpm lint, test, type-check, and build gates.
- Produce the task's required UI visual artifacts and validate the OpenAPI/generated-contract
  boundary.
- Use fresh command output immediately before changing task status to `completed`.

## Progress Notes

- 2026-08-22: Read workspace/repository instructions, task/spec companions, workflow ADRs, shared
  memory, and task 04 memory. Confirmed a deliberately dirty worktree containing inherited task 02
  through task 04 work.
- 2026-08-22: The managed harness exposes no callable `compozy__tool_search`,
  `compozy__tool_info`, or `compozy__skill_view`; the requested managed skill bodies cannot be
  loaded without violating the no-CLI/no-direct-read runtime rule. Proceeding with the task
  artifacts and repository-mandated local Go guidance as the safe fallback.
- 2026-08-22: Implemented canonical GitLab/Gitea/registry/advisory/discussion/feed/document
  adapters; contextual adoption and qualified public-security reads; bounded public crawling;
  immutable snapshots/chunks; deterministic FTS/pgvector RRF; mutual-kNN topics and correction
  history; deterministic releases; and immutable evidenced analysis runs, feedback, reruns, usage,
  and selection history.
- 2026-08-22: Published generated-contract HTTP routes and generated-client S11, S16, S17, S18,
  and S20 journeys with English/Portuguese copy, narrow/wide responsive behavior, axe coverage,
  and ten required visual artifacts under `artifacts/task_05/ui/`.
- 2026-08-22: Final persistence audit added store-boundary aggregate validation, direct
  project-scoped release lookup beyond any list page, immutable exact feedback replay, citation
  project/snapshot/cutoff/offset validation before a run write, parent-run series validation,
  contextual row-iteration errors, empty security collections, and deleted-project write guards
  for every Task 5 table carrying `project_id`.
- 2026-08-22: Added real-PostgreSQL regressions for releases beyond the first 1,000 records,
  successful runs without evidence, cross-project citations, changed-payload feedback replays,
  exact feedback replays, and writes after project deletion starts.

## Final Verification

- `GOCACHE=/workspace/.task05-go-build GOTMPDIR=/workspace/.task05-go-tmp make check` passed:
  generated-contract drift, Go formatting, vet, race tests for all packages, and all-binary build.
- `make test-integration` with `OPI_INTEGRATION_DATABASE_URL` targeting the repository PostgreSQL
  service passed against fresh isolated databases, including all Task 5 store contracts.
- `pnpm run typecheck`, `pnpm run lint`, `pnpm run test`, and `pnpm run build` passed; Vitest
  reported 8 files and 93 tests passing. Vite emitted only its advisory chunk-size warning.
- `pnpm --filter @opensource-project-intelligence/web exec playwright test e2e/task05.spec.ts`
  passed all 11 assigned journeys and refreshed all required narrow/wide screenshots.
- `pnpm run format:check`, `pnpm run lint:md`, and `git diff --check` passed.

## Follow-up Boundaries

- Task 5 validates analysis citation and parent provenance at the store boundary and guards every
  directly project-owned table during deletion. A future schema-hardening task may add redundant
  composite project foreign keys for all cross-table provenance relationships; that broader
  mechanical migration is not required to widen this task's tested application contract.
- The existing Task 5 PostgreSQL contract uses one isolated database for its ordered subtests.
  Future test-maintenance work may split those fixtures per subtest if parallel execution becomes
  necessary; production behavior and current deterministic coverage are unaffected.
