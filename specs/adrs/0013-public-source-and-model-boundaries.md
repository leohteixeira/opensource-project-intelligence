# ADR 0013 — Public source and model boundaries

- **Status:** accepted
- **Date:** 2026-08-22

## Context

Adoption, security, release, discussion, and knowledge intelligence consume heterogeneous public
providers. Generated analysis also depends on model capabilities that may be absent or unhealthy.
Provider request and response types must not become domain contracts, and provider failure must not
erase deterministic facts.

## Decision

Implement provider clients as infrastructure adapters that translate GitLab, Gitea, registry,
advisory, discussion, changelog/feed, documentation, website, embedding, and language-model payloads
to canonical application models. Collection is public-only even when operator-owned credentials are
used for quota. Every canonical fact retains source identity, observed time, source unit, population
context, raw-object provenance, and parser or adapter version.

Model ports accept provider-neutral requests and return provider-neutral structured candidates.
Schema and evidence validation occur after the adapter returns and before an immutable successful
analysis is published. Deterministic release metadata and lexical retrieval do not depend on a
model provider.

## Consequences

Adapters contain unavoidable provider drift, while application behavior remains replayable and can
degrade without turning missing data into a zero or a safety claim.

## Reassessment trigger

Reassess when a provider requires private-content authority or when a model capability cannot be
represented without exposing provider-owned types.
