# Task 06 Workflow Memory

## Scope

- Implement only task `task_06`: observed trends, predictive early warnings, deterministic adoption
  policies and recommendations, policy-derived radar governance, and shared alerts.
- Complete US-017 through US-020 and US-027 together with every assigned unit, integration,
  end-to-end, API-contract, accessibility, localization, and visual-contract case.
- Preserve all unrelated and inherited task 01 through task 05 worktree changes; leave this task
  uncommitted.

## Inherited Boundary

- Build on the canonical project, metric snapshot/factor, evidence, job, authorization, generated
  HTTP, and frontend-shell foundations implemented by tasks 01 through 05.
- Keep typed business models in consuming capability packages; provider, PostgreSQL, generated
  transport, and UI-client types remain in their adapters.
- Do not widen into exports, audit/deletion completion, assistant actions, notification delivery,
  or final release work assigned to task 07.

## Frozen Invariants

- Observed trends use versioned deterministic Theil-Sen and Mann-Kendall rules over one immutable
  half-open UTC snapshot. Forecasts are a separate signal class and disclose horizon, interval,
  rolling-backtest error, coverage, and selected versioned baseline.
- Sparse or stale evidence produces explicit insufficient/stale results, never a neutral value,
  fabricated direction, or zero. Failed prediction or explanation cannot suppress deterministic
  observations or recommendations.
- Policy definitions contain only normalized, typed rules over the closed metric catalog. Drafts
  validate before immutable publication and activation. Evaluations retain the exact selected
  policy, metric versions, cutoff, inputs, factors, evidence, and missing-data disposition.
- Recommendation outcomes are exactly `recommended`, `conditional`, `not_recommended`, or
  `insufficient_data`; LLM output cannot select or alter them.
- Radar suggestions derive from the exact policy version. Manual placement retains suggestion,
  reason, actor, owner, review date, version, expiry/removal history, and never rewrites the source
  recommendation.
- Alert rules and occurrences are versioned and deduplicated by rule/fact/window. Shared lifecycle
  is independent from each member's read state; redelivery and evaluation failure cannot duplicate
  or silently close an existing occurrence.
- All I/O propagates `context.Context`; work, pages, evidence displays, detectors, backtests,
  evaluation queues, radar recalculation, and alert volume are explicitly bounded.

## Required Verification

- Run generated-contract drift, Go formatting/vet/unit/race/build checks, all real-PostgreSQL
  integration tests, frontend lint/typecheck/unit/build gates, and the assigned Playwright/axe
  journeys.
- Refresh the required S13, S14, S15, and S21 narrow/wide visual artifacts and validate localization,
  accessibility, reduced-motion/chart alternatives, role controls, and responsive states.
- Run formatting, Markdown, and whitespace checks and use fresh command output immediately before
  changing `task_06.md` status to `completed`.

## Progress Notes

- 2026-08-22: Read workspace/repository instructions, task/spec companions, relevant user stories,
  UX/DX/test contracts, workflow ADRs, shared memory, and task 05 memory. Confirmed a deliberately
  dirty worktree containing inherited task 02 through task 05 work.
- 2026-08-22: The managed harness exposes no callable `compozy__tool_search`,
  `compozy__tool_info`, or `compozy__skill_view`; the requested managed skill bodies cannot be
  loaded without violating the no-CLI/no-direct-read runtime rule. Proceeding from the supplied
  workflow contracts and repository-mandated Go guidance as the safe fallback.
