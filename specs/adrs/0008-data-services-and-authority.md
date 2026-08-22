# ADR 0008 — Data services and explicit authority

- **Status:** accepted
- **Date:** 2026-08-21

## Context

The complete product requires durable work delivery, large immutable evidence, vector search, and
transient acceleration. The original MVP deliberately used PostgreSQL alone.

## Decision

Use PostgreSQL 18 with pgvector as the sole transactional source of truth. Use NATS JetStream for
at-least-once delivery from a PostgreSQL transactional outbox, S3-compatible storage for
content-addressed bytes owned by PostgreSQL metadata, and Valkey only for disposable caches, rate
limits, short deduplication, and ephemeral fanout. Broker/cache state is rebuilt after restore;
object digests are reconciled against the canonical database manifest.

This ADR supersedes the PostgreSQL-only scope statements in ADRs 0003 and 0004 and repository
documentation, but preserves pgx/v5, explicit SQL, transaction ownership, and redaction.

## Consequences

The deployment operates four data services, with one explicit owner for every datum. Loss or replay
of JetStream/Valkey cannot grant authorization, change terminal Jobs, or erase canonical evidence.

## Reassessment trigger

Reassess topology, not ownership, when a single-node VPS no longer meets measured availability or
throughput needs.
