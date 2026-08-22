---
status: completed
title: Publish contracts and establish the generated platform foundation
type: infra
complexity: critical
---

<!-- markdownlint-disable MD013 MD025 -->

# Task 1: Publish contracts and establish the generated platform foundation

## Overview

Turn the accepted workflow specification into repository-owned, reviewable contracts and establish
the infrastructure, identifier, persistence, generation, configuration, and recovery foundations
that every later vertical slice depends on.

<critical>
- ALWAYS READ `_spec.md` and its catalogs (`_user_stories.md`, `_dx.md`, `_uiux.md` when present, `_tests.md`) before starting
- REFERENCE `_spec.md` Part II for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — implement every test case assigned in ## Tests
</critical>

<requirements>

- Publish the complete product specification set under specs/ without weakening or reinterpreting
  the accepted workflow contracts.
- Record the repository ADRs required before adopting PostgreSQL 18/pgvector, NATS JetStream,
  S3-compatible storage, OpenAPI generation, sqlc, Snowflake node leasing, the expanded test
  strategy, and Valkey.
- Make the committed OpenAPI 3.1 contract the source for strict Go server types and the TypeScript
  client, with pinned reproducible generation and drift detection.
- Establish base migrations, database-leased Snowflake identifiers serialized as decimal strings,
  explicit SQL generation boundaries, infrastructure ownership, configuration validation,
  readiness classification, telemetry propagation, and backup/restore reconciliation.
- Keep PostgreSQL authoritative. Broker, cache, object storage, and generated artifacts must not
  introduce a competing source of truth.
- Preserve the existing modular-monolith layout, independent API/worker binaries, and repository
  ports.

</requirements>

## Subtasks

- [ ] Promote the accepted product, API/DX, UI/UX, story, test, security, deployment, event, metric,
      analysis, observability, and evaluation contracts into versioned repository specifications.
- [ ] Add or supersede repository ADRs for every material decision that the workflow ADRs require.
- [ ] Define and validate the complete OpenAPI contract, generation commands, pinned tool versions,
      generated output locations, and clean-tree drift checks.
- [ ] Establish PostgreSQL 18/pgvector migrations and sqlc contracts for base platform records,
      migration history, Snowflake node leases, and generated boundary types.
- [ ] Extend Compose and deployment documentation with NATS JetStream, Valkey, and S3-compatible
      storage while preserving the assigned ports and repository-local project name.
- [ ] Establish safe immutable configuration, dependency readiness, redaction, telemetry, and
      backup/restore contracts.
- [ ] Implement every assigned unit and real-boundary integration test.

## Implementation Details

### Relevant Files

- /workspace/docs/opensource_project_intelligence.md
- \_spec.md and all companion catalogs in this task directory
- adrs/adr-002.md, adr-019.md, adr-026.md, adr-027.md, adr-029.md, and adr-035.md
- specs/README.md and specs/adrs/0001-record-architecture-decisions.md through 0007-design-system-and-icon-set.md
- go.mod, compose.yaml, Makefile, scripts/migrate.sh, and migrations/README.md
- cmd/api/main.go, cmd/worker/main.go, and internal/platform/
- apps/web/package.json and apps/web/src/config.ts

### Dependent Files

- All later task files depend on the published contracts, generated types, identifiers, migrations,
  infrastructure services, configuration, and verification commands produced here.

### Related ADRs

- [Workflow ADR 002](adrs/adr-002.md), [ADR 019](adrs/adr-019.md),
  [ADR 026](adrs/adr-026.md), [ADR 027](adrs/adr-027.md),
  [ADR 029](adrs/adr-029.md), and [ADR 035](adrs/adr-035.md).
- Repository ADRs 0001 through 0005, plus the repository ADRs required by workflow ADR 027 and
  workflow ADR 035.

## Deliverables

- Repository-owned complete specification suite and accepted ADR chain.
- Reproducible OpenAPI/sqlc generation with committed outputs and drift verification.
- Base PostgreSQL/pgvector schema, database-leased Snowflake identity service, and safe migration
  lifecycle.
- Repository-local Compose services and ownership contracts for PostgreSQL, JetStream, Valkey, and
  S3-compatible storage.
- Validated configuration, redacted readiness/telemetry behavior, and proven backup/restore path.
- Passing evidence for all assigned tests and repository verification commands.

## Tests

Implement these normative cases from \_tests.md exactly once:

- Unit: UT-225, UT-226, UT-227, UT-228, UT-269, UT-271
- Integration: IT-097, IT-098, IT-099, IT-135, IT-136, IT-138

## Success Criteria

- [ ] Repository specifications and ADRs are sufficient to implement every later task without
      consulting an undocumented decision.
- [ ] Generated HTTP and SQL boundaries reproduce cleanly and CI detects drift.
- [ ] Migrations round-trip on PostgreSQL 18/pgvector and leased Snowflake IDs remain unique,
      ordered, opaque, and JavaScript-safe.
- [ ] Required and optional dependencies are classified correctly without exposing secrets.
- [ ] Backup/restore preserves canonical references and evidence checksums.
- [ ] All 12 assigned tests pass with fresh evidence.
