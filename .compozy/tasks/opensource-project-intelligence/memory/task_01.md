# Task 01 Memory: Generated Platform Foundation

## Scope

Publish the accepted contracts and establish only the reusable platform foundation required by
later vertical slices: ADRs/specifications, OpenAPI and sqlc generation, base migrations,
database-leased Snowflake IDs, dependency/configuration/readiness/telemetry ownership, Compose
services, and backup/restore reconciliation.

## Normative tests

- Unit: UT-225, UT-226, UT-227, UT-228, UT-269, UT-271.
- Integration: IT-097, IT-098, IT-099, IT-135, IT-136, IT-138.

## Decisions carried into implementation

- Use a signed-positive 64-bit Snowflake layout with a documented custom epoch, database-leased
  node number, per-millisecond sequence, bounded clock-regression wait, and fail-closed issuance.
- Keep SQL explicit and committed. sqlc-generated records stop inside the database adapter.
- Liveness touches no dependency. Readiness distinguishes required-unavailable from
  optional-degraded dependencies without exposing endpoints or credentials.
- Backup/restore is PostgreSQL-first and reconciles referenced object digests; broker and cache are
  rebuilt from canonical state.
- The full HTTP surface belongs in the committed OpenAPI contract even when later tasks implement
  handlers incrementally.

## Baseline

- Initial worktree: clean, branch `main` ahead of `origin/main` by eight existing commits.
- Existing foundation has PostgreSQL-only Compose, hand-written HTTP DTOs, no migrations, no sqlc,
  and repository ADRs 0001 through 0007.
- The managed session did not expose the canonical Compozy skill-view tool. Execution follows the
  required memory-first, task-execution, and fresh-verification workflows directly from the task
  sources without reading managed skill files through prohibited fallback surfaces.

## Progress

- 2026-08-21: Read workspace/repository instructions, source proposal, task specification and
  companion catalogs, related workflow ADRs, repository ADR chain, and current platform files.
- 2026-08-21: Initialized shared and task-local workflow memory before source edits.
- 2026-08-21: Published the complete accepted product, story, DX/API, UI/UX, and test catalogs plus
  focused numbered specifications and repository ADRs 0008 through 0011.
- 2026-08-21: Added the full frozen route inventory as OpenAPI 3.1, pinned oapi-codegen/sqlc/Hey API
  generation, committed strict Go/sqlc/TypeScript output, checksums, Make targets, and CI drift
  reproduction.
- 2026-08-21: Added reversible PostgreSQL 18/pgvector platform migrations, generated vector/NULL/
  bigint access, renewable database-leased Snowflake IDs, half-open windows, safe problem details,
  readiness classification, recovery reconciliation, and backup/restore scripts.
- 2026-08-21: Added pinned healthy Compose services for PostgreSQL, JetStream, Valkey, MinIO, and a
  deterministic one-shot bucket initializer without consuming unassigned host ports.
- 2026-08-21: All six assigned unit cases passed. All six assigned integration cases passed against
  the real PostgreSQL 18/pgvector container, including migration round trip, generated vector scan,
  lease failover, readiness matrix, isolated drift failure, and database/object-manifest recovery.

## Follow-up boundaries

- Later tasks must replace the existing `/api/v1/` fallback incrementally with generated strict
  handlers and operation-specific DTO schemas as each vertical slice lands; they must not create a
  parallel hand-written route contract.
- Later durable-work and evidence tasks own the NATS outbox relay, S3 adapter, Valkey fallback, and
  whole-system recovery drill. Task 01 proves service lifecycle, canonical ownership, database dump,
  object manifest, and digest reconciliation without implementing those later business adapters.
