# ADR 0011 — Isolated real-boundary verification

- **Status:** accepted
- **Date:** 2026-08-21

## Context

PostgreSQL constraints, pgvector types, JetStream delivery, Valkey loss, S3 checksums, generated
drift, concurrent workers, and browser accessibility cannot be accepted through unit fakes alone.

## Decision

Retain standard-library table-driven Go unit tests and race detection. Add isolated real-service
integration fixtures for PostgreSQL 18/pgvector, NATS, Valkey, and S3-compatible storage; controlled
provider servers; generated drift/migration gates; and production-build Playwright/axe browser
tests. Pin images and isolate each test namespace.

This ADR supersedes ADR 0004's shared-Compose first-integration clause. Its testing package, external
test-package preference, race detector, and hardened `GOTMPDIR` decisions remain accepted.

## Consequences

The highest-risk semantics are verified at their actual boundaries, with slower Docker-dependent
suites and explicit fixture lifecycle costs.

## Reassessment trigger

Split fast and exhaustive CI lanes only after both remain mandatory protected checks and preserve
the same stable test IDs.
