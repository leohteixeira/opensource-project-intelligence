# ADR 0015 — Model-independent evaluation assets

- **Status:** accepted
- **Date:** 2026-08-22

## Context

Release claims, topic labels, and natural-language answers need regression evidence without making
CI depend on a live provider or accepting unsupported generated text.

## Decision

Commit deterministic multilingual fixtures containing source snapshots, chunks, retrieval ranks,
topic neighbours, structured model candidates, expected schema failures, and citation outcomes.
Evaluate schema conformance, evidence accessibility, citation completeness, deterministic fallback,
and immutable run transitions locally. Provider smoke tests remain optional and cannot replace the
fixture suite.

## Consequences

CI proves the evidence gate and degradation behavior reproducibly. Fixtures must be versioned with
the parser, retrieval, prompt, schema, and evaluation contracts they exercise.

## Reassessment trigger

Reassess when a provider-neutral recorded-response format can improve coverage without introducing
network dependence or sensitive payloads.
