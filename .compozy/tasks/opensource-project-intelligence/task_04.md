---
status: completed
title: Materialize deterministic metrics, health, contributors, and comparisons
type: fullstack
complexity: high
---

<!-- markdownlint-disable MD013 MD025 -->

# Task 4: Materialize deterministic metrics, health, contributors, and comparisons

## Overview

Build the deterministic intelligence layer over canonical evidence: versioned metric cohorts,
seven auditable health dimensions, the secondary overall score whenever calculable, contributor
identity and concentration, and immutable same-window comparisons.

<critical>
- ALWAYS READ `_spec.md` and its catalogs (`_user_stories.md`, `_dx.md`, `_uiux.md` when present, `_tests.md`) before starting
- REFERENCE `_spec.md` Part II for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — implement every test case assigned in ## Tests
</critical>

<requirements>

- Implement US-013, US-014, and US-016 in full, including every accepted edge case.
- Implement the closed metric catalog with half-open UTC windows, immutable cutoffs, explicit units,
  cohort/eligibility/coverage/missing-data rules, versions, formula factors, and evidence.
- Keep zero distinct from unknown, not applicable, insufficient data, incomparable, stale, and
  unavailable.
- Materialize all seven equal-weight health dimensions without redistributing a missing dimension;
  always show the secondary overall score when and only when its evidence requirements are met.
- Resolve contributor identities only from verified or Analyst-confirmed evidence and publish
  resolution coverage and concentration.
- Compare two to five Projects at one common window, cutoff, definition set, and Project aggregation
  boundary; preserve missing and incomparable states.

</requirements>

## Visual Contract

| Surface           | Viewport/state contract                                                                                                                                  | Reference                              | Required evidence                |
| ----------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------- | -------------------------------- |
| S9 metrics/health | Narrow 320px: seven dimensions, calculable/non-calculable overall score, zero/unknown/not-applicable/insufficient/stale, version/window/evidence failure | ProjectDetailScreen and OverviewScreen | artifacts/task_04/ui/s9-narrow/  |
| S9 metrics/health | Wide: the same states with metric drawer, formula/factors, coverage, repositories, version comparison, and evidence pagination                           | ProjectDetailScreen and OverviewScreen | artifacts/task_04/ui/s9-wide/    |
| S10 contributors  | Narrow 320px: empty/one/many, unresolved/corrected/conflicting identity, concentration warning, unknown retention, bots, pagination, and locale          | ContributorsScreen                     | artifacts/task_04/ui/s10-narrow/ |
| S10 contributors  | Wide: the same states with activity/cohorts/maintainers, top-one/top-three shares, evidence, correction, and resolution coverage                         | ContributorsScreen                     | artifacts/task_04/ui/s10-wide/   |
| S12 comparisons   | Narrow 320px: two/three/five, invalid counts/duplicates, preset/custom, coverage gaps, zero/missing/incomparable, version/archive/error states           | CompareScreen                          | artifacts/task_04/ui/s12-narrow/ |
| S12 comparisons   | Wide: the same states with accessible matrix, evidence drill-down, exact cutoff, saved URL, and row-detail alternative                                   | CompareScreen                          | artifacts/task_04/ui/s12-wide/   |

## Subtasks

- [ ] Implement versioned metric definitions, temporal cohorts, materialization keys, immutable
      snapshots, evidence factors, and coverage/status algebra.
- [ ] Implement release, contributor, issue, pull-request, backlog, concentration, and health
      calculations exactly as frozen in the specification and ADRs.
- [ ] Implement contributor association evidence/corrections and targeted recalculation.
- [ ] Implement common-cutoff comparison materialization and the assigned metric/health/
      contributor/comparison HTTP operations.
- [ ] Replace S9, S10, and S12 fixtures with generated-client data and complete accessible table/
      chart/detail alternatives.
- [ ] Implement every assigned deterministic unit, PostgreSQL integration, and browser test.

## Implementation Details

### Relevant Files

- \_spec.md metric catalog, deterministic calculations, snapshots, and API sections
- \_user_stories.md US-013, US-014, and US-016
- \_dx.md intelligence and comparison routes
- \_uiux.md S9, S10, and S12
- adrs/adr-006.md, adr-014.md, adr-025.md, and adr-032.md
- internal/metric/, internal/contributor/, internal/comparison/, internal/issue/,
  internal/pullrequest/, and internal/release/
- apps/web/src/design-system/intelligence/ and relevant project/workspace kit screens

### Dependent Files

- Tasks 05 and 06 consume versioned facts, common cutoffs, evidence factors, contributor resolution,
  health dimensions, and missing-data semantics from this task.

### Related ADRs

- [Workflow ADR 006](adrs/adr-006.md), [ADR 014](adrs/adr-014.md),
  [ADR 025](adrs/adr-025.md), and [ADR 032](adrs/adr-032.md).
- Repository ADRs governing explicit SQL, testing, observability, frontend dependencies, and the
  imported design system.

## Deliverables

- Closed, versioned metric and health definition catalogs with reproducible materialization.
- Auditable contributor identity, sustainability, concentration, and coverage intelligence.
- Immutable two-to-five Project comparisons at one exact window and cutoff.
- Production S9, S10, and S12 browser journeys with accessible evidence alternatives.
- Passing evidence for all 61 assigned tests and every visual-contract row.

## Tests

Implement these normative cases from \_tests.md exactly once:

- Unit: UT-085, UT-086, UT-087, UT-088, UT-089, UT-090, UT-091, UT-092, UT-093, UT-094, UT-095, UT-096, UT-097, UT-098, UT-106, UT-107, UT-108, UT-109, UT-110, UT-111, UT-112, UT-229, UT-230, UT-231, UT-232, UT-233, UT-234, UT-235, UT-236, UT-237, UT-238
- Integration: IT-037, IT-038, IT-039, IT-040, IT-041, IT-042, IT-046, IT-047, IT-048, IT-120, IT-121, IT-122, IT-221, IT-222, IT-223, IT-224, IT-225, IT-226, IT-227, IT-228, IT-233, IT-234, IT-235, IT-236
- End-to-end: E2E-013, E2E-014, E2E-016, E2E-041, E2E-042, E2E-044

## Success Criteria

- [ ] Identical facts and versions reproduce identical values, factors, status, and comparison.
- [ ] Missing evidence never becomes zero and unavailable health weights are never redistributed.
- [ ] Contributor linkage and correction preserve source provenance and publish resolution coverage.
- [ ] Every numeric presentation includes its unit, window, cutoff, and definition version.
- [ ] All 61 assigned tests pass with fresh evidence.
