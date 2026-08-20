# ADR 0001 — Record architecture decisions

- **Status:** accepted
- **Date:** 2026-08-20

## Context

The product document deliberately leaves material implementation choices open (HTTP framework,
database driver, query generation, migration system) and requires them to be recorded. Without a
record, the motivation behind each choice is lost and revisiting it turns into code archaeology.

## Decision

Every material architecture decision is recorded as a numbered ADR in this directory, using a
reduced MADR template: context, decision, consequences and reassessment trigger.

- Sequential four-digit numbering, never reused.
- An ADR is never edited after being accepted. To change the decision, write a new ADR and mark the
  previous one as `superseded by ADR NNNN`.
- Possible statuses: `proposed`, `accepted`, `superseded`, `rejected`.

## Consequences

Structural changes now require a short document before the code, which adds a step to the flow. In
exchange, every choice carries its rationale and the condition under which it should be revisited.

## Reassessment trigger

None. This ADR describes the process itself.
