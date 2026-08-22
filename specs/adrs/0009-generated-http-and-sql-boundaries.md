# ADR 0009 — Generated HTTP and SQL boundaries

- **Status:** accepted
- **Date:** 2026-08-21

## Context

The frozen HTTP surface and persistence model exceed safe hand-maintained DTO and row-scanning
limits. ADR 0003 anticipated adopting sqlc after this threshold.

## Decision

Commit an OpenAPI 3.1 contract and generate strict `net/http` Go types with oapi-codegen 2.8.0 plus
a TypeScript client with `@hey-api/openapi-ts` 0.98.1. Commit explicit SQL and generate pgx/v5
adapters with sqlc 1.31.1. Versions and outputs are pinned; CI regenerates and fails on diff.
Generated types stop at transport/database adapters. Middleware, authorization, transactions,
validation, errors, and business logic remain hand-written.

This ADR supersedes only ADR 0003's rejection of a query generator. Its pgx/v5, explicit SQL,
manual migration, redaction, and transaction decisions remain accepted.

## Consequences

Reviewed sources reproduce every boundary and remove parallel DTO drift, at the cost of generator
tooling in development and CI.

## Reassessment trigger

Replace a generator only after an alternative reproduces the same reviewed contracts and drift
gate without moving domain logic into generated code.
