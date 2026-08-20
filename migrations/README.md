# Migrations

Migrations are versioned SQL files, applied in lexicographic order by `scripts/migrate.sh`. The
decision to use explicit SQL, with no ORM and no migration framework, is recorded in
[`../specs/adrs/0003-persistence-driver-and-migrations.md`](../specs/adrs/0003-persistence-driver-and-migrations.md).

## Naming convention

```text
NNNN_description_in_snake_case.sql
```

`NNNN` is a four-digit sequential counter, never reused.

## Rules

- Dates and timestamps are stored in UTC (`timestamptz`).
- External identifiers (for example the numeric GitHub `id`) carry a uniqueness constraint, so that
  incremental synchronization is idempotent.
- Raw payloads needed for audit and reprocessing live in `JSONB` columns.
- No business migration exists yet: the repository is in the foundation phase.
