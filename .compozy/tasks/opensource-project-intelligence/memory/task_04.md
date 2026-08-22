# Task 04 Workflow Memory

## Scope

- Implement only task_04: deterministic metric, health, contributor, and comparison
  materialization plus the S9, S10, and S12 browser journeys.
- Treat the task file, unified specification, companion catalogs, workflow ADRs 006, 014, 025,
  and 032, and repository instructions as the source of truth.
- Leave all work uncommitted and preserve the existing task_02/task_03 worktree changes.

## Inherited Boundary

- Task 03 already owns Project/Repository/Source, lifecycle, Jobs/outbox/checkpoints, canonical
  GitHub and restricted-Git ingestion, evidence storage, generated HTTP clients, and the S4-S8
  browser paths. Task 04 extends those seams additively.
- PostgreSQL remains authoritative; generated OpenAPI/SQL outputs must stay drift-free.

## Frozen Task Invariants

- Windows are immutable half-open UTC intervals `[from, to)` with a frozen cutoff and definition
  version. Every numeric result carries a unit, window, cutoff, version, coverage, factors, and
  evidence references.
- `available` zero is distinct from `unknown`, `not_applicable`, `insufficient_data`,
  `incomparable`, `stale`, and `unavailable`.
- Health exposes Activity, Community, Maintenance, Concentration, Stability, Security, and Adoption
  independently. Each carries exactly one-seventh weight; missing weight is never redistributed and
  the secondary overall score exists only when every mandatory dimension meets coverage.
- Contributor aggregation combines accounts only with provider-verified or Analyst-confirmed
  evidence. Corrections are attributable and preserve source provenance; resolution coverage is
  always published.
- Comparisons accept two to five distinct resolved Projects and freeze one Project boundary,
  interval, cutoff, and compatible definition set. Missing or incompatible items remain explicit.

## Required Verification

- Implement and freshly execute all 61 identifiers assigned by `task_04.md`, including the real
  PostgreSQL route/materialization cases and the S9/S10/S12 Playwright/axe journeys.
- Run the repository's generated-contract, Go check/race, frontend lint/typecheck/unit/build,
  integration, formatting, markdown, and browser gates before marking the task complete.
- The managed Compozy skill-view capability is not projected into this worker session. The required
  workflow skills cannot be loaded through their permitted native surface, so this execution uses
  the task/spec/memory requirements directly and does not bypass policy with the operator CLI or
  managed skill files.

## Progress Notes

- 2026-08-22: Read workspace and repository instructions; the task/spec/story/DX/UI/test contracts;
  workflow ADRs 006, 014, 025, and 032; inherited workflow memory; and mandatory repository Go
  guidance. No source code was edited before this task memory was established.
- 2026-08-22: Added provider-neutral release, contributor, issue, pull-request, metric, health, and
  comparison models. The metric engine freezes half-open windows and cutoffs, preserves all
  missing-data states, emits explicit cohort/factor/coverage provenance, and calculates the seven
  initial metric families deterministically. Health preserves seven independently inspectable
  one-seventh dimensions and suppresses the overall score until all mandatory results are
  available.
- 2026-08-22: Added migration 0004 with normalized immutable definition, snapshot, factor,
  contributor identity/correction, comparison, issue-response/state, and pull-request-readiness
  records. Store publication is transactional; exact computation replays converge while
  same-key/different-content collisions fail; failed candidates leave completed snapshots intact.
- 2026-08-22: Wired canonical fact materialization into completed collection and targeted
  recalculation jobs. The production store reads one repeatable PostgreSQL snapshot and freezes all
  30/90/180/365-day presets. Contributor responses are bounded and cursor-paged without leaking the
  full detail through the aggregate summary.
- 2026-08-22: Added authenticated metric/detail/health/contributor and immutable comparison HTTP
  operations, generated clients, locale-aware S9/S10/S12 routes, custom/preset windows, explicit
  unknown/incomparable states, accessible wide matrices and narrow row-detail alternatives, and the
  required visual artifacts.
- 2026-08-22: Fresh verification passed: `make check` (generated drift, format, vet, full Go race,
  build), `make test-integration` against PostgreSQL, frontend lint/typecheck/93 unit tests/build,
  Prettier and Markdown checks, the task-specific PostgreSQL suite (all 24 assigned integration
  identifiers plus production materializer replay), and all six assigned Playwright journeys with
  axe. The task-created Go cache directories were removed after verification; the Compose
  PostgreSQL service remains available for the surrounding workflow.
