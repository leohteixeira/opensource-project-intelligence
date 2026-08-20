# ADR 0003 — PostgreSQL driver, query generation and migrations

- **Status:** accepted
- **Date:** 2026-08-20

## Context

The specification requires explicit SQL, versioned migrations, transactions at use case
boundaries, dates in UTC and uniqueness constraints on external identifiers. It also requires the
decisions about the driver, query generation and migration system to be recorded in an ADR.

## Decision

- Driver **`github.com/jackc/pgx/v5`** with `pgxpool`, wrapped in `internal/platform/database`. No
  business package imports pgx directly.
- **No ORM and no query generator.** SQL is written by hand. `sqlc` was evaluated and rejected in
  this phase: there is no schema yet, and the benefit shows up once there are many stable queries.
- Migrations are **versioned SQL files** in `migrations/`, applied in lexicographic order by
  `scripts/migrate.sh`, which tracks what has already run in the `schema_migrations` table.
- `goose` was evaluated as a _tool dependency_ and rejected: it tripled the module's dependency
  tree (from 23 to 68 indirect requires and from 85 to 290 lines in `go.sum`), because it carries
  drivers for ClickHouse, SQLite and other databases this project does not use. The shell runner
  meets the same requirement and keeps the module lean and consistent with the sibling repositories
  in the portfolio.
- Connection errors go through `redact` before being logged: pgx includes the connection string in
  some failures, and it carries credentials.
- Raw payloads needed for audit and reprocessing live in `JSONB` columns, with the option to move
  to object storage without changing the domain contracts.

## Consequences

Every query needs manual row-to-struct mapping, which costs more code. In exchange, the SQL that
produces a metric stays visible and reviewable — a precondition for the auditability the product
requires. The shell runner has no automatic rollback migration; a rollback is written as a new
migration.

## Reassessment trigger

Adopt `sqlc` once the number of hand-written queries passes roughly 30, or once manual mapping
starts producing recurring defects. Adopt `goose` or `golang-migrate` if the shell runner stops
being enough, for example because complex transactional migrations or automated rollback are
needed.
