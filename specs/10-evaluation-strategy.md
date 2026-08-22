# Evaluation and Verification Strategy

Unit tests use deterministic clocks/IDs and table-driven standard-library Go tests. Integration
tests isolate real PostgreSQL 18/pgvector, NATS JetStream, Valkey, and S3-compatible boundaries.
Provider tests use controlled protocol servers. Browser tests use a production build, real API,
wide/narrow viewports, en/pt-BR, keyboard paths, and automated accessibility checks.

Contract gates regenerate OpenAPI-derived Go/TypeScript and sqlc output and fail on any diff.
Migration tests converge from empty state and exercise supported rollback/reapply. Concurrent code
passes the race detector and shutdown/leak checks. AI datasets version inputs and expected fidelity,
citation, classification, retrieval, tool-selection, and refusal behavior.

The stable normative test IDs, environments, isolation rules, and exit criteria live in
`15-test-contract.md`; no mock may replace the boundary semantics a test claims to verify.
