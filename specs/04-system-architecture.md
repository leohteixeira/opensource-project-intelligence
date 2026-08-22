# System Architecture

The system is a modular monolith built as `cmd/api` and `cmd/worker`. Business capabilities live in
short `internal/` packages. Interfaces are small and declared by consumers; domain/application code
does not import infrastructure or generated provider types.

The API owns versioned HTTP orchestration through `net/http`. The worker owns scheduling,
collection, normalization, deterministic computation, outbox publication, and asynchronous
analysis. Both propagate `context.Context`, bound concurrency, preserve error causes internally,
and shut down gracefully.

Infrastructure ownership is strict:

- PostgreSQL 18/pgvector: transactions, canonical state, Jobs/checkpoints, provenance, indexes,
  object metadata, audit, leases, and outbox;
- NATS JetStream: at-least-once delivery from the transactional outbox only;
- S3-compatible storage: content-addressed immutable bytes referenced and owned by PostgreSQL;
- Valkey: disposable caches, rate limits, short deduplication, and ephemeral fanout;
- shared Keycloak: external authentication only; product membership/authorization remains local.

OpenAPI and SQL are reviewed sources. Generated Go/TypeScript/sqlc code remains at transport and
database boundaries. PostgreSQL is the sole authority after broker replay, cache loss, or restore.
