# Task 06 Governance and Monitoring Implementation Plan

> **For agentic workers:** Execute this plan inline in the task-owned packages and preserve all
> work from tasks 01 through 05. Do not commit; the orchestration surface owns commits and pull
> requests.

**Goal:** Deliver reproducible trends and forecasts, immutable deterministic adoption governance,
policy-derived radar placement, deduplicated shared alerts, and the localized accessible browser
journeys assigned to task 06.

**Architecture:** Keep deterministic calculations in the `trend`, `policy`, `radar`, and `alert`
capability packages. Persist normalized immutable results in PostgreSQL through
`intelligencestore`, expose them through the generated HTTP contract in `intelligenceapi`, and let
the React application consume only the generated client. Forecasts and generated explanations are
strictly downstream of deterministic facts and cannot alter observed signals or policy outcomes.

**Tech Stack:** Go 1.26, PostgreSQL 18/pgvector with pgx, OpenAPI-generated Go and TypeScript,
React 19, React Router, i18next, Vitest, Playwright, and axe.

---

## Task 1: Reproducible observed trends and forecasts

**Files:**

- Implement: `internal/trend/trend.go`
- Persist: `internal/platform/intelligencestore/governance.go`
- Migrate: `migrations/0006_trends_policies_radar_alerts.up.sql`
- Test: `internal/trend/task06_test.go`
- Integration test: `internal/platform/intelligencestore/task06_integration_test.go`

- [x] Define separate observed and forecast result types with immutable method/model versions,
      half-open UTC windows, coverage, evidence, and supersession metadata.
- [x] Calculate observed direction with bounded Theil-Sen and Mann-Kendall rules; report sparse or
      noisy histories explicitly instead of fabricating a direction.
- [x] Select exponential or seasonal forecasts by bounded rolling backtest and publish horizon,
      interval, confidence, error, model version, and outcome-evaluation state.
- [x] Prove UT-113–UT-119 and UT-239–UT-241 plus IT-049–IT-051 and IT-237–IT-238.

## Task 2: Immutable policies and four-state recommendations

**Files:**

- Implement: `internal/policy/policy.go`
- Persist: `internal/platform/intelligencestore/governance.go`
- Expose: `internal/platform/intelligenceapi/governance.go`
- Contract: `api/openapi.yaml`
- Test: `internal/policy/task06_test.go`
- Integration test: `internal/platform/intelligencestore/task06_integration_test.go`

- [x] Validate policy rules exclusively against the closed versioned metric catalog, including
      typed operators, finite thresholds, normalized weights, required evidence, and explicit radar
      mappings.
- [x] Keep drafts editable, publication/activation atomic, published versions immutable, and
      retired/superseded versions available to historical evaluations.
- [x] Evaluate exact policy and metric versions into only `recommended`, `conditional`,
      `not_recommended`, or `insufficient_data`, preserving factors, decisive evidence, stale and
      missing inputs, cutoff, and window.
- [x] Prove UT-120–UT-133 and UT-245 plus IT-052–IT-057 and IT-239–IT-250.

## Task 3: Policy-derived radar governance

**Files:**

- Implement: `internal/radar/radar.go`
- Persist: `internal/platform/intelligencestore/governance.go`
- Expose: `internal/platform/intelligenceapi/governance.go`
- Test: `internal/radar/task06_test.go`
- Integration test: `internal/platform/intelligencestore/task06_integration_test.go`

- [x] Derive the suggested ring from the exact policy evaluation and keep `insufficient_data`
      placement dependent on an explicit versioned mapping.
- [x] Store attributed overrides with reason, actor, owner, review date, revision, expiry/removal
      history, and optimistic concurrency while retaining the original recommendation.
- [x] Exclude archived projects from active radar pages without deleting historical placements and
      keep pages, groups, and counts bounded.
- [x] Prove UT-134–UT-140 and UT-246 plus IT-058–IT-060 and IT-251–IT-256.

## Task 4: Deduplicated shared alerts and personal read state

**Files:**

- Implement: `internal/alert/alert.go`
- Persist: `internal/platform/intelligencestore/governance.go`
- Expose: `internal/platform/intelligenceapi/governance.go`
- Test: `internal/alert/task06_test.go`
- Integration test: `internal/platform/intelligencestore/task06_integration_test.go`

- [x] Validate typed, bounded, versioned rules and require complete versioned evidence before an
      occurrence can open.
- [x] Deduplicate redelivery by rule, fact, and window; retain severity, occurrence time, evidence,
      cooldown, and suppression counts without silently closing on evaluation failure.
- [x] Keep shared open/acknowledged/resolved/dismissed transitions independent from each member's
      read timestamp, with role checks and optimistic concurrency.
- [x] Prove UT-183–UT-189 and UT-267 plus IT-079–IT-081, IT-130, and IT-279–IT-288.

## Task 5: Generated-client browser journeys and final verification

**Files:**

- Implement: `apps/web/src/application/screens/GovernanceScreen.tsx`
- Route/localize: `apps/web/src/application/router.tsx`, `apps/web/src/application/routes.ts`,
  `apps/web/src/application/i18n.ts`
- Test: `apps/web/e2e/task06.spec.ts`
- Evidence: `artifacts/task_06/ui/`

- [x] Render observed and forecast claims separately with method disclosure, evidence, coverage,
      chart alternatives, and reduced-motion-compatible semantics.
- [x] Render four recommendation states, immutable policy governance, suggestion/effective radar
      placement, and shared versus personal alert state in English and Brazilian Portuguese.
- [x] Exercise keyboard-only interactions, 320-pixel and wide viewports, axe, generated-client
      mutations, and refresh all S13/S14/S15/S21 visual artifacts.
- [x] Run `make check`, the real-PostgreSQL integration suite, frontend lint/typecheck/unit/build,
      assigned Playwright journeys, pre-commit formatting/Markdown checks, and a final whitespace
      check before setting `task_06.md` to `status: completed`.
