# ADR 0012: Localized Application Runtime

## Status

Accepted

## Context

ADR 0006 deliberately left routing, remote-state ownership, form management, localization, and
accessible data interaction out of the foundation phase. The first product capability now requires
stable English and Brazilian Portuguese routes, a session gate, server-backed catalog and
administration surfaces, conditional forms, responsive tables, and accessible visualizations.

Building each of those mechanisms locally would duplicate mature browser behavior and make route,
cache, validation, and accessibility ownership ambiguous. The visual language and shared controls
remain governed by ADR 0007 and the repository design system.

## Decision

Use React Router 7 in Data Mode for route matching and navigation, TanStack Query v5 for remote
server state, React Hook Form with Zod for forms and validation, and `react-i18next` with ICU message
formatting for fixed product language. Use React Aria Components behind local adapters when a
design-system primitive needs headless interaction behavior. Use TanStack Table for server-driven
tables and Apache ECharts only as a supplementary visualization with an equivalent semantic table
or list.

One typed manifest owns localized paths. URL parameters own shareable search and pagination state;
TanStack Query owns fetched server state; local component state owns transient interaction. The web
application calls the backend only through the generated TypeScript client.

Continue using the tokenized inline-style convention and interaction classes accepted by ADR 0007.
This repository-specific decision replaces the CSS Modules implementation note in workflow ADR 031
without changing its accessibility or design-token constraints.

This ADR supersedes only ADR 0006's foundation-phase decision to omit these runtime dependencies.

## Consequences

- Localized paths and language switching are deterministic and testable.
- Session and server state have one cache owner and can retain safe stale data during failures.
- Forms have explicit validation and conditional-write behavior.
- Accessibility remains visible through semantic equivalents and repository-local wrappers.
- The pinned frontend dependency surface grows and must be maintained through the lockfile and CI.
