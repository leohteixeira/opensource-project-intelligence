---
status: completed
title: Implement trends, policies, recommendations, radar, and alerts
type: fullstack
complexity: high
---

<!-- markdownlint-disable MD013 MD025 -->

# Task 6: Implement trends, policies, recommendations, radar, and alerts

## Overview

Deliver the governance and monitoring layer: reproducible observed trends, separately labeled
predictive warnings, deterministic versioned adoption policies and four-state recommendations,
policy-derived radar placement with attributed overrides, and deduplicated shared alerts with
per-member read state.

<critical>
- ALWAYS READ `_spec.md` and its catalogs (`_user_stories.md`, `_dx.md`, `_uiux.md` when present, `_tests.md`) before starting
- REFERENCE `_spec.md` Part II for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — implement every test case assigned in ## Tests
</critical>

<requirements>

- Implement US-017 through US-020 and US-027 in full, including every accepted edge case.
- Keep observed trends deterministic and distinct from forecasts; expose windows, magnitude,
  coverage, method/model versions, horizon, confidence, calibration/error, and evidence.
- Evaluate only typed catalogued metric rules. Policy drafts must validate before immutable
  publication/activation, and historical evaluations retain their exact version.
- Produce exactly recommended, conditional, not_recommended, or insufficient_data; generated
  explanations cannot change deterministic outcomes.
- Derive suggested radar rings from the exact policy version while retaining attributed manual
  override, owner, reason, review date, history, and expiry behavior.
- Deduplicate alert occurrences from versioned rules/facts while separating shared lifecycle from
  each member's read state.

</requirements>

## Visual Contract

| Surface                   | Viewport/state contract                                                                                                                    | Reference                          | Required evidence                |
| ------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------ | ---------------------------------- | -------------------------------- |
| S13 trends                | Narrow 320px: increase/decrease/stable, warning/no signal/insufficient/poor coverage/stale/corrected/AI unavailable/overlap/dense timeline | ProjectDetailScreen                | artifacts/task_06/ui/s13-narrow/ |
| S13 trends                | Wide: the same states with separate observed/forecast views, method disclosure, evidence, chart table, and reduced-motion semantics        | ProjectDetailScreen                | artifacts/task_06/ui/s13-wide/   |
| S14 recommendation/policy | Narrow 320px: all four results, conditions/missing/stale/unavailable, draft/version/validation/conflict/history, large keyboard rule tree  | PortfolioScreen and PoliciesScreen | artifacts/task_06/ui/s14-narrow/ |
| S14 recommendation/policy | Wide: the same states with factors/weights/evidence, version history, impact preview, activation, and immutable prior results              | PortfolioScreen and PoliciesScreen | artifacts/task_06/ui/s14-wide/   |
| S15 radar                 | Narrow 320px: four rings/unplaced, policy/override/expiry/review/change/archive/conflict/empty/scale, accessible list and print/export     | RadarScreen                        | artifacts/task_06/ui/s15-narrow/ |
| S15 radar                 | Wide: the same states with list as primary, optional plot, suggestion/effective placement, attribution, and keyboard access                | RadarScreen                        | artifacts/task_06/ui/s15-wide/   |
| S21 alerts                | Narrow 320px: unread/read, all shared states, recurrence/invalid/empty/stale/conflict/recovery/scale, Viewer/Analyst controls, filters     | AlertsScreen                       | artifacts/task_06/ui/s21-narrow/ |
| S21 alerts                | Wide: the same states with rules, evidence, deduplication grouping, suppression counts, personal read state, and shared transitions        | AlertsScreen                       | artifacts/task_06/ui/s21-wide/   |

## Subtasks

- [ ] Implement versioned observed-trend and forecast definitions, Theil-Sen/Mann-Kendall rules,
      bounded backtesting, evidence, coverage, and evaluation metadata.
- [ ] Implement typed policy families/drafts/versions/validation/activation and deterministic
      four-state evaluation with immutable historical attribution.
- [ ] Implement radar selection, policy mapping, override/review lifecycle, effective placement, and
      history.
- [ ] Implement alert rules, evaluation, deduplication/cooldown, shared lifecycle, per-member read
      state, evidence, quotas, and redelivery behavior.
- [ ] Implement every assigned trend/recommendation/policy/radar/alert HTTP operation.
- [ ] Replace S13–S15 and S21 fixtures with generated-client data and accessible chart/list/table
      alternatives.
- [ ] Implement every assigned deterministic, integration, and browser test.

## Implementation Details

### Relevant Files

- \_spec.md trend/forecast, policy, recommendation, radar, alert, metric, and worker sections
- \_user_stories.md US-017 through US-020 and US-027
- \_dx.md intelligence, policies, radar, and alerts routes
- \_uiux.md S13, S14, S15, and S21
- adrs/adr-003.md, adr-007.md, adr-009.md, adr-011.md, adr-014.md, and adr-032.md
- internal/metric/, internal/analysis/, internal/project/, and new capability packages
- apps/web/src/design-system/intelligence/ and workspace/administration kit screens

### Dependent Files

- Task 07 consumes alert actions, policy/radar evidence, trend results, and operational events for
  assistant tools, exports, audit, and final release verification.

### Related ADRs

- [Workflow ADR 003](adrs/adr-003.md), [ADR 007](adrs/adr-007.md),
  [ADR 009](adrs/adr-009.md), [ADR 011](adrs/adr-011.md),
  [ADR 014](adrs/adr-014.md), and [ADR 032](adrs/adr-032.md).
- Repository ADRs for any statistical/model dependency and the existing generated-contract,
  persistence, testing, observability, and design-system decisions.

## Deliverables

- Reproducible observed trends and clearly separate calibrated forecasts.
- Versioned deterministic policies and auditable four-state recommendations.
- Policy-derived radar with attributed, reviewable manual overrides.
- Deduplicated shared alerts with independent member read state.
- Production S13, S14, S15, and S21 browser journeys.
- Passing evidence for all 96 assigned tests and every visual-contract row.

## Tests

Implement these normative cases from \_tests.md exactly once:

- Unit: UT-113, UT-114, UT-115, UT-116, UT-117, UT-118, UT-119, UT-120, UT-121, UT-122, UT-123, UT-124, UT-125, UT-126, UT-127, UT-128, UT-129, UT-130, UT-131, UT-132, UT-133, UT-134, UT-135, UT-136, UT-137, UT-138, UT-139, UT-140, UT-183, UT-184, UT-185, UT-186, UT-187, UT-188, UT-189, UT-239, UT-240, UT-241, UT-245, UT-246, UT-267
- Integration: IT-049, IT-050, IT-051, IT-052, IT-053, IT-054, IT-055, IT-056, IT-057, IT-058, IT-059, IT-060, IT-079, IT-080, IT-081, IT-130, IT-237, IT-238, IT-239, IT-240, IT-241, IT-242, IT-243, IT-244, IT-245, IT-246, IT-247, IT-248, IT-249, IT-250, IT-251, IT-252, IT-253, IT-254, IT-255, IT-256, IT-279, IT-280, IT-281, IT-282, IT-283, IT-284, IT-285, IT-286, IT-287, IT-288
- End-to-end: E2E-017, E2E-018, E2E-019, E2E-020, E2E-027, E2E-045, E2E-046, E2E-047, E2E-053

## Success Criteria

- [ ] Observations and predictions are never presented as the same kind of claim.
- [ ] Policy outcomes and radar suggestions reproduce from exact immutable facts and versions.
- [ ] Manual radar context never rewrites the original recommendation.
- [ ] Alert redelivery preserves one occurrence and never conflates personal read state with shared
      resolution.
- [ ] All 96 assigned tests pass with fresh evidence.
