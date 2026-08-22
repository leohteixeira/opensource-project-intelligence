# Migrations

Migrations are versioned SQL files, applied in lexicographic order by `scripts/migrate.sh`. The
decisions to use explicit SQL, no ORM, reviewed up/down migrations, pgvector, and generated sqlc
adapters are recorded in [ADR 0003](../specs/adrs/0003-persistence-driver-and-migrations.md),
[ADR 0008](../specs/adrs/0008-data-services-and-authority.md), and
[ADR 0009](../specs/adrs/0009-generated-http-and-sql-boundaries.md).

## Naming convention

```text
NNNN_description_in_snake_case.up.sql
NNNN_description_in_snake_case.down.sql
```

`NNNN` is a four-digit sequential counter, never reused.

## Rules

- Dates and timestamps are stored in UTC (`timestamptz`).
- External identifiers (for example the numeric GitHub `id`) carry a uniqueness constraint, so that
  incremental synchronization is idempotent.
- Raw payloads needed for audit and reprocessing live in `JSONB` columns.
- Every applied migration records the reviewed up-file checksum in `schema_migrations`; changing an
  applied file fails instead of silently rewriting history.
- Every supported rollback has a reviewed `.down.sql` file. `scripts/migrate.sh down` reverts only
  the latest applied version.
- Migrations run with `ON_ERROR_STOP` and one transaction per version.

## Commands

```bash
DATABASE_URL=postgres://opensource:opensource@localhost:5433/opensource_project_intelligence \
  make migrate
DATABASE_URL=postgres://opensource:opensource@localhost:5433/opensource_project_intelligence \
  make migrate-down
```

Run the `integration` build-tag suite against PostgreSQL 18 with the pgvector extension; an older or
in-memory SQL substitute is not acceptable evidence for migration and generated-type behavior.
