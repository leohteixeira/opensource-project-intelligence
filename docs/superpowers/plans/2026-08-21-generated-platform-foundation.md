# Generated Platform Foundation Implementation Plan

> Scope: task 01 only. Keep every later business capability behind the contracts produced here.

## 1. Publish accepted contracts and decisions

- Promote the workflow product, story, DX/API, UI/UX, and test catalogs into the numbered `specs/`
  suite without weakening normative language.
- Add focused repository specifications for domain, architecture, data, HTTP, events, AI, metrics,
  evaluation, observability, security, and deployment, cross-linking the complete accepted bundle.
- Add repository ADRs for PostgreSQL 18/pgvector and service ownership, generated HTTP and SQL
  boundaries, Snowflake node leasing, real-boundary tests, and Valkey authority; supersede only the
  clauses in earlier ADRs that conflict.

## 2. Establish generated HTTP and SQL boundaries

- Add a committed OpenAPI 3.1 source with the frozen health/readiness/problem foundation and all
  versioned operation paths from the accepted DX contract.
- Pin generation tool versions and provide deterministic scripts/Make targets for strict Go server
  types, a TypeScript client, and sqlc pgx/v5 output.
- Commit generated outputs and implement a clean-tree drift check used locally and by CI.
- Keep middleware, authorization, transactions, domain errors, and business behavior outside
  generated files.

## 3. Establish persistence and identifiers

- Add reversible base migrations for pgvector, migration history, Snowflake node leases, canonical
  object references/checksums, jobs, and audit/recovery records.
- Add sqlc schema/query configuration and generated adapter types at the database boundary.
- Implement the Snowflake generator against a small consuming-package lease interface, with a
  controllable clock and decimal string transport representation.
- Add exact tests for lease exclusivity, same-millisecond sequencing, clock regression, migration
  convergence, sqlc scanning, and expired-lease failover.

## 4. Establish configuration, dependency, and recovery behavior

- Extend immutable configuration for required and optional service capabilities, validating all
  applicable safe field names together and never including values.
- Add dependency health classification and problem-details serialization with explicit request IDs
  and no wrapped-cause leakage.
- Extend Compose with PostgreSQL 18/pgvector, NATS JetStream, Valkey, and S3-compatible storage using
  repository-owned ports and health checks.
- Add PostgreSQL-first backup/restore scripts and digest reconciliation that rebuilds delivery/cache
  state from canonical records.
- Add exact tests for readiness classification, backup/restore reconciliation, configuration
  redaction, and error serialization.

## 5. Verify and close the task

- Regenerate contracts and prove a clean generation drift check.
- Run the task integration target against isolated real services when Docker is available.
- Run `make check`, `make test-race`, web lint/typecheck/test/build, and
  `lefthook run pre-commit --all-files` with fresh output.
- Fix failures without weakening checks, update workflow memory with evidence/follow-ups, then set
  `task_01.md` frontmatter to `status: completed` only if every required check is clean.
