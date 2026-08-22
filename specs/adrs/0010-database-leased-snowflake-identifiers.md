# ADR 0010 — Database-leased Snowflake identifiers

- **Status:** accepted
- **Date:** 2026-08-21

## Context

Persisted resources need compact sortable identifiers across multiple API/worker replicas, while
JavaScript cannot safely represent arbitrary 64-bit numbers and static node IDs collide silently.

## Decision

Use signed-positive 64-bit Snowflake IDs with a versioned epoch, timestamp, database-leased node,
and per-millisecond sequence. A process issues only while its exclusive renewable PostgreSQL lease
is valid. Small clock regressions wait within tolerance; larger regressions or lease loss fail
closed and make the process unready. PostgreSQL stores `bigint`; HTTP/JSON uses decimal strings and
clients treat them as opaque.

Version 1 uses the UTC epoch `2026-01-01T00:00:00Z`, 41 timestamp bits, 10 leased-node bits, and 12
sequence bits. The default production lease is 30 seconds and renews every 10 seconds; callers may
shorten both together in isolated tests. Exhausting the 12-bit sequence waits for the next
millisecond while the lease remains valid.

## Consequences

Identifiers are ordered and JavaScript-safe on the wire. Writes depend on clock discipline and a
healthy authoritative lease table.

## Reassessment trigger

Reassess the bit allocation before the documented timestamp horizon or if measured replica counts
exceed available leased nodes.
