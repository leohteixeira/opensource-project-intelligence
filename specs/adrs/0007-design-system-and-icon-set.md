# ADR 0007 — Design system, tokens and icon set

- **Status:** accepted
- **Date:** 2026-08-21

## Context

The web application was a single static `<main>` with no navigation, no component inventory and no
design tokens. The product specification is unusually strict about presentation: missing data has
three distinct words and is never a zero, health is seven independent dimensions rather than one
verdict, observed values and forecasts and generated text are three separately labelled things, and
every number carries a unit, a window, a cutoff and a definition version. Those rules are only
enforceable if they live in components rather than in each screen.

A versioned design system for the product exists outside the repository. It defines the token set,
37 components with their props contracts, and four UI kits that recreate the specified surfaces
against illustrative fixtures. It ships as React JSX loaded from a CDN with Babel in the browser,
which is a prototyping format, not a shippable one.

The design system substitutes Lucide 0.474.0 for the product's icon set, because the source
specifications name glyph _meanings_ but ship no icon font, sprite or SVG.

## Decision

- The design system is **ported into this repository** as TypeScript under
  `apps/web/src/design-system`: CSS custom properties in `styles/`, components grouped by
  `core/`, `forms/`, `navigation/`, `overlays/`, `data/` and `intelligence/`, and one barrel at
  `index.ts`. Screens import from the barrel, never from a component file.
- **Styling stays CSS custom properties plus inline `style` objects**, exactly as the design system
  defines them. Interaction states (`hover`, `focus`, `active`, `aria-current`) need real CSS and
  live in `styles/base.css` behind the `opi-*` classes. No CSS Modules, no CSS-in-JS runtime and no
  utility framework.
- **`lucide-react` 1.33.0** provides the substituted glyphs, and `design-system/core/icons.ts` is
  the only file that imports it. The glyph vocabulary is frozen there as a map, and `IconName` is
  its `keyof`, so a screen cannot name a glyph outside the vocabulary.
- The **frozen data is separated from the components that render it** — `core/status.ts` holds the
  status vocabulary, `core/icons.ts` the glyph map, `intelligence/ranking.ts` the comparison
  ranking rule — so the invariants are unit-testable without a DOM.
- The four UI kits are ported to `apps/web/src/kits` against fixture modules. Fixture values are
  illustrative and were not derived from any repository; each kit's fixtures live in one file so
  they can be replaced by the HTTP contract in one place.

## Consequences

The interface now enforces the presentation rules structurally: `StatusBadge` cannot render a
status without a glyph and a word, `Table` cannot silently print a blank cell for a missing value,
and `bestCellIndex` cannot rank a cell that carries a status instead of a number. Those three are
covered by table-driven tests.

One runtime dependency is added to the web application. It is pinned, tree-shaken to the glyphs
actually used, and reachable only through `Icon`, so replacing Lucide with the project's own set
changes one file. The bundle is a single chunk of roughly 760 kB (200 kB gzipped), dominated by
React DOM; code splitting is deferred until routing exists.

Screens hold their own state and there is no router, so the four shells are reachable through a
selector in `App.tsx` rather than through localized `/en` and `/pt-br` routes.

## Reassessment trigger

Replace `lucide-react` when the project's own icon set exists — only `core/icons.ts` changes. Split
`design-system` into a published package if a second frontend in this repository needs it. Revisit
the single-chunk bundle when the router lands, since routes are the natural split points.
