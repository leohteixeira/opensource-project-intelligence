# Deployment and Recovery

The repository Compose project is `opensource-project-intelligence`. It publishes web 3100, API
8100, and PostgreSQL 5433. NATS, Valkey, and S3-compatible services are repository-internal by
default so this product does not consume unassigned host ports. Keycloak and production secret
management remain external shared deployment surfaces.

Liveness reports process execution and touches no dependency. Readiness requires PostgreSQL and a
valid Snowflake lease for write-capable processes. Enabled durable delivery/object capabilities are
required; disabled optional capabilities are reported unavailable, while enabled non-authoritative
accelerators such as Valkey may report degraded without claiming canonical loss.

Services use pinned images, health checks, persistent volumes where state ownership requires it,
bounded startup/shutdown, and no production credentials in Compose. Backups capture PostgreSQL plus
a manifest of referenced object digests. The deployment S3 tool must export every manifest-listed
byte alongside the database backup; a database dump alone is explicitly incomplete. Restore
verifies the backup files, imports PostgreSQL, restores those S3 bytes into a clean bucket, and runs
digest reconciliation before serving traffic. JetStream and Valkey are then reconstructed from
PostgreSQL. The runbook records versions, checksums, cutoff, result, and unresolved references
without secret values.

`scripts/backup.sh` and `scripts/restore.sh` use matching local PostgreSQL clients by default. For
the pinned Compose database, set `POSTGRES_CONTAINER`, `POSTGRES_DATABASE`, and `POSTGRES_USER` so
the scripts execute the PostgreSQL 18 client in that container. This avoids unsupported older-client
dump formats without publishing another host port.
