---
status: pending
title: Add extended sources, knowledge retrieval, topics, releases, and immutable AI analyses
type: fullstack
complexity: high
---

<!-- markdownlint-disable MD013 MD025 -->

# Task 5: Add extended sources, knowledge retrieval, topics, releases, and immutable AI analyses

## Overview

Extend the canonical evidence platform beyond core GitHub ingestion to public provider, registry,
advisory, discussion, documentation, and website sources; deliver adoption/security evidence,
hybrid retrieval, deterministic topic candidates, release intelligence, cited analysis, and
immutable AI run governance.

<critical>
- ALWAYS READ `_spec.md` and its catalogs (`_user_stories.md`, `_dx.md`, `_uiux.md` when present, `_tests.md`) before starting
- REFERENCE `_spec.md` Part II for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — implement every test case assigned in ## Tests
</critical>

<requirements>

- Implement US-015, US-021 through US-024, and US-026 in full, including every accepted edge case.
- Add GitLab, Gitea, registry, advisory, discussion, changelog/feed, documentation, and website
  adapters without leaking provider DTOs into domain/application contracts.
- Preserve source-specific adoption units and context; never turn missing public security evidence
  into a safety claim.
- Harden every crawl hop and bound domains, depth, bytes, pages, types, rate, robots behavior,
  snapshots, parsers, chunks, and indexes.
- Combine PostgreSQL full-text and pgvector retrieval through deterministic versioned RRF with
  immutable citations and Project/source/time authorization filters.
- Build deterministic topic candidates and preserve Analyst corrections as canonical constraints;
  keep generated labels and all AI outputs immutable, evidenced, versioned, attributable, and
  degradable.
- Retain deterministic release metadata and lexical retrieval when model capabilities are absent.

</requirements>

## Visual Contract

| Surface               | Viewport/state contract                                                                                                                               | Reference              | Required evidence                |
| --------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------- | -------------------------------- |
| S11 adoption/security | Narrow 320px: one/many/no registries, source units, incomparable/unavailable, no-advisory unknown, releases/conflicts/withdrawal/stale/scale          | AdoptionSecurityScreen | artifacts/task_05/ui/s11-narrow/ |
| S11 adoption/security | Wide: the same states with separate registry panels, public-evidence timeline, coverage, source context, and non-scanner disclosure                   | AdoptionSecurityScreen | artifacts/task_05/ui/s11-wide/   |
| S16 topics            | Narrow 320px: known/emerging/empty/low-confidence/sparse, corrections/conflicts/reanalysis states, taxonomy scale, original language                  | ProjectDetailScreen    | artifacts/task_05/ui/s16-narrow/ |
| S16 topics            | Wide: the same states with prevalence/change, representative evidence, confidence/version, correction detail, and pagination                          | ProjectDetailScreen    | artifacts/task_05/ui/s16-wide/   |
| S17 releases          | Narrow 320px: analyzed/pending/stale/failed, no changelog/claims, sparse/conflicting evidence, prerelease/duplicate/withdrawn, provider unavailable   | ReleasesScreen         | artifacts/task_05/ui/s17-narrow/ |
| S17 releases          | Wide: the same states with categorized claim citations, run metadata, histories, truncation, and original/generated translation                       | ReleasesScreen         | artifacts/task_05/ui/s17-wide/   |
| S18 knowledge         | Narrow 320px: no docs, crawl lifecycle, robots/unsafe/bounds, empty/no result/cited/stale/conflicting snapshots, lexical fallback, bilingual evidence | KnowledgeScreen        | artifacts/task_05/ui/s18-narrow/ |
| S18 knowledge         | Wide: the same states with explicit source scope, snapshot result pages, original viewer, coverage/truncation, and authorized filters                 | KnowledgeScreen        | artifacts/task_05/ui/s18-wide/   |
| S20 AI runs           | Narrow 320px: queued/running/succeeded/failed/stale, no success/many versions/older selection, feedback/rerun/provider/schema/output states           | AiRunsScreen           | artifacts/task_05/ui/s20-narrow/ |
| S20 AI runs           | Wide: the same states with immutable metadata/evidence, side-by-side versions, usage/cost, selection history, and role visibility                     | AiRunsScreen           | artifacts/task_05/ui/s20-wide/   |

## Subtasks

- [ ] Implement the extended public-source adapters and canonical normalization/provenance contracts.
- [ ] Implement registry adoption snapshots and qualified public security evidence.
- [ ] Implement hardened crawling, immutable snapshots/chunks, full-text/pgvector indexing,
      deterministic RRF, citation filtering, and evidence deletion behavior.
- [ ] Implement discussion/topic evidence, mutual-kNN candidates, corrections, versioning, and
      reanalysis invalidation.
- [ ] Implement deterministic release records and evidence-gated structured analysis runs,
      translations, reruns, feedback, and selected-version history.
- [ ] Implement assigned adoption/security/topic/release/crawl/search/query/run HTTP operations.
- [ ] Replace S11 and S16–S18/S20 fixtures with generated-client data and all required states.
- [ ] Implement deterministic fixtures/evaluation data and every assigned test without requiring a
      live model provider.

## Implementation Details

### Relevant Files

- \_spec.md source adapters, adoption/security, retrieval, topics, releases, analysis, and AI sections
- \_user_stories.md US-015, US-021 through US-024, and US-026
- \_dx.md intelligence, topics, releases, documentation, and AI routes
- \_uiux.md S11 and S16 through S18 and S20
- adrs/adr-008.md, adr-010.md, adr-016.md, adr-030.md, adr-033.md, and adr-034.md
- internal/analysis/, internal/issue/, internal/release/, internal/collector/, and internal/platform/
- apps/web/src/kits/project-evidence/ and project detail topic states

### Dependent Files

- Task 06 consumes topic/adoption/security/release evidence where policy, trends, radar, and alerts
  reference it. Task 07 consumes immutable analysis runs, retrieval tools, provider ports, and
  evidence gates for bounded agent/HITL flows.

### Related ADRs

- [Workflow ADR 008](adrs/adr-008.md), [ADR 010](adrs/adr-010.md),
  [ADR 016](adrs/adr-016.md), [ADR 030](adrs/adr-030.md),
  [ADR 033](adrs/adr-033.md), and [ADR 034](adrs/adr-034.md).
- Repository ADRs for generated contracts, source/model dependencies, retrieval, and evaluation
  assets added before their implementations.

## Deliverables

- Public extended-source adapters and canonical evidence/provenance.
- Source-contextual adoption and qualified public security intelligence.
- Hardened crawl/snapshot/index pipeline and deterministic hybrid retrieval.
- Correctable topic intelligence and cited release/natural-language analysis.
- Immutable run, feedback, rerun, selection, usage, and degradation behavior.
- Production S11, S16, S17, S18, and S20 browser journeys.
- Passing evidence for all 111 assigned tests and every visual-contract row.

## Tests

Implement these normative cases from \_tests.md exactly once:

- Unit: UT-099, UT-100, UT-101, UT-102, UT-103, UT-104, UT-105, UT-141, UT-142, UT-143, UT-144, UT-145, UT-146, UT-147, UT-148, UT-149, UT-150, UT-151, UT-152, UT-153, UT-154, UT-155, UT-156, UT-157, UT-158, UT-159, UT-160, UT-161, UT-162, UT-163, UT-164, UT-165, UT-166, UT-167, UT-168, UT-176, UT-177, UT-178, UT-179, UT-180, UT-181, UT-182, UT-242, UT-243, UT-244, UT-261, UT-262, UT-266, UT-273
- Integration: IT-043, IT-044, IT-045, IT-061, IT-062, IT-063, IT-064, IT-065, IT-066, IT-067, IT-068, IT-069, IT-070, IT-071, IT-072, IT-076, IT-077, IT-078, IT-115, IT-117, IT-118, IT-123, IT-124, IT-125, IT-126, IT-229, IT-230, IT-231, IT-232, IT-257, IT-258, IT-259, IT-260, IT-261, IT-262, IT-263, IT-264, IT-265, IT-266, IT-267, IT-268, IT-269, IT-270, IT-271, IT-272, IT-273, IT-274, IT-275, IT-276, IT-277, IT-278
- End-to-end: E2E-015, E2E-021, E2E-022, E2E-023, E2E-024, E2E-026, E2E-043, E2E-048, E2E-049, E2E-050, E2E-052

## Success Criteria

- [ ] Provider and crawler inputs cannot escape the public-only, bounded, provenance-bearing
      collection contract.
- [ ] Search, topic membership, and every deterministic result reproduce from versioned inputs.
- [ ] Every generated claim is schema-valid and backed by accessible immutable evidence.
- [ ] Model failure leaves deterministic collection, release metadata, lexical search, metrics, and
      prior successful versions usable.
- [ ] All 111 assigned tests pass with fresh evidence.
