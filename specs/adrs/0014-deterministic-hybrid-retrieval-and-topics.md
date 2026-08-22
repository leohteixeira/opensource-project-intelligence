# ADR 0014 — Deterministic hybrid retrieval and topic candidates

- **Status:** accepted
- **Date:** 2026-08-22

## Context

Knowledge search needs multilingual lexical recall and semantic recall without sacrificing
authorization, cutoff reproducibility, or explainability. Topic discovery must not depend on an
opaque clustering service.

## Decision

Store deterministic heading-and-offset chunks in PostgreSQL with `tsvector` and optional pgvector
embeddings. Apply workspace, project, source, current-state, and cutoff filters before ranking.
Fuse lexical and vector ranks with versioned reciprocal-rank fusion; use evidence identity as the
final stable tie-breaker. If embeddings are unavailable, lexical ranking remains valid and declares
the degraded mode. Search citations point to immutable snapshots and chunk offsets.

Build topic candidates from a versioned mutual-k-nearest-neighbour graph followed by deterministic
connected components. One-way neighbour relationships never create an edge. Analyst merge, split,
include, exclude, and rename corrections are versioned canonical constraints; generated candidates
and labels remain immutable history.

## Consequences

Given the same authorized input set and algorithm versions, search order and topic membership are
reproducible. PostgreSQL indexing and candidate materialization add storage and integration-test
cost.

## Reassessment trigger

Reassess only when an alternative preserves pre-ranking authorization, stable replay, lexical-only
degradation, and inspectable topic edges.
