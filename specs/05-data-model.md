# Data Model and Recovery Ownership

PostgreSQL stores signed-positive `bigint` Snowflake IDs, UTC `timestamptz`, explicit lifecycle
states, immutable versions, checksums, provenance, and uniqueness constraints for idempotency.
Public IDs are decimal strings and clients treat them as opaque.

Base platform tables establish migration history, exclusive Snowflake node leases, Jobs, audit
records, and object references. Later capability migrations add canonical provider-neutral tables;
committed queries use sqlc with pgx/v5, while transactions remain application-owned.

Object bytes use content-addressed keys and SHA-256 digests. PostgreSQL records project ownership,
media type, size, digest, retention, and provenance. A backup is only complete when the database
manifest and referenced object set agree. Restore loads PostgreSQL first, restores/reconciles object
digests, then rebuilds JetStream delivery and Valkey state from canonical records.

Migrations are lexicographically ordered, transactional where PostgreSQL permits, checksum-verified,
and exercised from an empty PostgreSQL 18/pgvector database. Destructive rollback is supported only
where a reviewed down migration exists; production recovery prefers forward correction.
