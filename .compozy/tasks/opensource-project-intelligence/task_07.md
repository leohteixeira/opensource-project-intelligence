---
status: pending
title: Add bounded ADK/HITL, exports, and final operational hardening
type: fullstack
complexity: critical
---

<!-- markdownlint-disable MD013 MD025 -->

# Task 7: Add bounded ADK/HITL, exports, and final operational hardening

## Overview

Complete the product with bounded natural-language action orchestration, single-use human approval,
authorized asynchronous exports, complete audit/model-provider operations, deletion/recovery
closure, and full-system verification of the one-delivery product vision.

<critical>
- ALWAYS READ `_spec.md` and its catalogs (`_user_stories.md`, `_dx.md`, `_uiux.md` when present, `_tests.md`) before starting
- REFERENCE `_spec.md` Part II for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — implement every test case assigned in ## Tests
</critical>

<requirements>

- Implement US-025 and US-028 through US-030 in full, including every accepted edge case.
- Contain Google ADK Go v2 behind narrow application ports and allowlisted typed tools; never expose
  SQL, filesystem, infrastructure clients, provider SDKs, or arbitrary HTTP.
- Show one exact non-destructive Analyst action, inputs, resources, effect, quota, and ten-minute
  expiry before single-use confirmation; recheck identity, scope, state, and version at execution.
- Refuse membership, credentials, policy authoring, lifecycle/archive/deletion, or other destructive
  proposals before confirmation.
- Generate authorized cutoff-consistent CSV and evidence JSON through durable Jobs and checksummed
  S3 artifacts; preserve stable machine fields, localized labels, cancellation/retry, and 24-hour
  expiry.
- Complete immutable redacted audit, model provider status/usage/cost/degradation, correlation,
  deletion reconciliation, backup/restore, race/shutdown, security, accessibility, and evaluation
  release gates.
- Run every explicit repository and task-catalog verification command before completion.

</requirements>

## Visual Contract

| Surface            | Viewport/state contract                                                                                                                                        | Reference                          | Required evidence                |
| ------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------- | -------------------------------- |
| S19 assistant/HITL | Narrow 320px full-screen: query/clarification, insufficient/provider/run states, cited result, proposal preview, prohibited/expired/changed/single-use/failure | AssistantPanel and KnowledgeScreen | artifacts/task_07/ui/s19-narrow/ |
| S19 assistant/HITL | Wide side panel: the same states with exact scope/window/cutoff, atomic effect/quota/expiry, audit receipt, focus return, and no hidden action                 | AssistantPanel and KnowledgeScreen | artifacts/task_07/ui/s19-wide/   |
| S22 exports        | Narrow 320px: configuration/validation, every Job state, concurrent cutoff, zero rows, size quota, expiry/revocation/retry/coalescing, download handoff        | ExportsScreen                      | artifacts/task_07/ui/s22-narrow/ |
| S22 exports        | Wide: the same states with resource/filter/window/locale, artifact metadata/checksum/version, Job evidence, and accessible download recovery                   | ExportsScreen                      | artifacts/task_07/ui/s22-wide/   |

## Subtasks

- [ ] Implement bounded ADK run orchestration, typed evidence tools, clarification, durable pause/
      resume, budgets, proposal persistence, and strict allowlists.
- [ ] Implement assistant proposal/confirmation execution with single-use expiry, idempotency,
      current authorization/version checks, safe refusal, audit attribution, and recovery.
- [ ] Implement export specification, authorization, snapshot cutoff, CSV/evidence JSON generation,
      Job lifecycle, object checksum, download, expiry, and purge ownership.
- [ ] Complete model provider configuration/status, aggregate usage/cost, quota degradation, and
      deterministic-feature isolation.
- [ ] Complete audit searches/exports, telemetry correlation, retention/purge reconciliation,
      backup/restore, readiness, graceful shutdown, race, security, evaluation, and operational docs.
- [ ] Replace S19 and S22 fixtures with generated-client data and every required responsive state.
- [ ] Run the full unit, integration, race, generated-contract, Playwright/axe, lint, typecheck,
      build, Compose, backup/restore, and pre-commit verification matrix.

## Implementation Details

### Relevant Files

- \_spec.md assistant, HITL, exports, audit, provider, security, deletion, observability, evaluation,
  deployment, and completion sections
- \_user_stories.md US-025 and US-028 through US-030
- \_dx.md assistant proposals, alerts/exports, audit/operations, errors, and deployment
- \_uiux.md S19, S22, and the cross-surface accessibility/localization constraints
- adrs/adr-010.md, adr-012.md, adr-013.md, adr-018.md, adr-020.md, adr-024.md,
  adr-026.md, adr-028.md, adr-034.md, and adr-035.md
- internal/analysis/, internal/platform/, cmd/api/, cmd/worker/, migrations/, and compose.yaml
- apps/web/src/kits/workspace/AssistantPanel.tsx
- apps/web/src/kits/project-evidence/ExportsScreen.tsx and AiRunsScreen.tsx
- apps/web/src/kits/administration/AuditScreen.tsx and OperationsScreen.tsx

### Dependent Files

- This is the terminal delivery task. It consumes every prior contract and must leave no incomplete
  story, assigned test, generated artifact, migration, operational procedure, or visual state.

### Related ADRs

- [Workflow ADR 010](adrs/adr-010.md), [ADR 012](adrs/adr-012.md),
  [ADR 013](adrs/adr-013.md), [ADR 018](adrs/adr-018.md),
  [ADR 020](adrs/adr-020.md), [ADR 024](adrs/adr-024.md),
  [ADR 026](adrs/adr-026.md), [ADR 028](adrs/adr-028.md),
  [ADR 034](adrs/adr-034.md), and [ADR 035](adrs/adr-035.md).
- All accepted repository ADRs, including those added or superseded by Tasks 01–06.

## Deliverables

- Bounded evidence-only ADK orchestration and safe single-action HITL.
- Authorized asynchronous CSV/evidence exports with 24-hour artifacts.
- Complete audit/model operations, degradation, correlation, purge, recovery, and operator guidance.
- Production S19 and S22 browser journeys.
- Fresh evidence for all 57 assigned tests and the complete 627-case repository matrix.

## Tests

Implement these normative cases from \_tests.md exactly once:

- Unit: UT-169, UT-170, UT-171, UT-172, UT-173, UT-174, UT-175, UT-190, UT-191, UT-192, UT-193, UT-194, UT-195, UT-196, UT-197, UT-198, UT-199, UT-200, UT-201, UT-202, UT-203, UT-204, UT-205, UT-206, UT-207, UT-208, UT-209, UT-210
- Integration: IT-073, IT-074, IT-075, IT-082, IT-083, IT-084, IT-085, IT-086, IT-087, IT-088, IT-089, IT-090, IT-137, IT-289, IT-290, IT-291, IT-292, IT-293, IT-294, IT-295, IT-296, IT-297, IT-298
- End-to-end: E2E-025, E2E-028, E2E-029, E2E-030, E2E-051, E2E-054

## Success Criteria

- [ ] Agentic execution cannot escape typed allowlisted application capabilities or bypass approval.
- [ ] Confirmation is action-bound, single-use, expiring, idempotent, and revalidated against current
      identity, scope, resource version, lifecycle, and quota.
- [ ] Exports are authorized at one cutoff, checksummed, reproducible, localized where appropriate,
      purged with owners, and inaccessible after 24 hours.
- [ ] Provider/model failure never blocks deterministic collection or intelligence.
- [ ] Every explicit test and repository verification command passes from a clean supported
      environment, including race, Playwright/axe, generated drift, and backup/restore.
- [ ] All 57 assigned tests pass with fresh evidence.
