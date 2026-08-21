# Web application

React 19 + Vite 8 + TypeScript 5.9.3, served on port 3100. The interface talks to the backend only
through the versioned HTTP contract at `/api/v1`; it never reaches a database or a vendor API.

```bash
pnpm --filter "@opensource-project-intelligence/web" dev   # http://0.0.0.0:3100
pnpm run lint && pnpm run typecheck && pnpm run test && pnpm run build
```

## Layout

```text
src/
├── design-system/     # tokens, components and the frozen vocabularies
│   ├── styles/        # CSS custom properties; index.css is the single entry point
│   ├── core/          # Button, StatusBadge, Panel, Icon, Banner, …
│   ├── forms/         # FormField, TextField, Select, RadioGroup, …
│   ├── navigation/    # AppShell, Tabs, Menu, FilterBar, Pagination
│   ├── overlays/      # Dialog, Drawer
│   ├── data/          # Table, DefinitionList
│   ├── intelligence/  # MetricValue, HealthDimensions, Recommendation, …
│   └── index.ts       # the barrel every screen imports from
├── kits/              # the four shells, against illustrative fixtures
│   ├── public-catalog/
│   ├── workspace/
│   ├── project-evidence/
│   └── administration/
├── App.tsx            # shell selector — scaffolding until routes exist
└── config.ts          # API base URL from the environment
```

## The rules the components enforce

These come from the design system and are not stylistic preferences. Breaking one changes what the
product claims about its evidence.

1. **Missing data is never zero.** `Unknown`, `Not applicable` and `Insufficient data` are three
   distinct states, each with its own glyph and word. None of them is a blank cell or a `0`.
2. **Health is seven independent dimensions.** The overall score is a secondary summary, shown only
   when its evidence requirements are met and always labelled with its version.
3. **Observed, forecast, AI-generated and human override are four separate, labelled things.**
4. **Every number carries a unit, a window, a cutoff and a definition version.**
5. **Colour is never the only cue.** Every signal renders a glyph and a word as well as a hue.
6. **Cards are white on a light page**, 12px radius, hairline border, one quiet shadow. Navigation
   is a single 64px top bar; there is no rail, and panels never nest.

## Conventions

- Screens import from `../../design-system`, never from a component file directly.
- Static styling is inline `style` with `var(--token)` values; interaction states live in
  `design-system/styles/base.css` behind the `opi-*` classes.
- Icons go through `Icon` only. The vocabulary is frozen in `design-system/core/icons.ts` and
  `IconName` is its `keyof`, so an unknown glyph is a type error.
- Fixture data lives in one `fixtures.ts` per kit. Values are illustrative and were not derived
  from any repository; replace one file to see a kit against real collections.

## Substitutions and caveats

These come from the design system and are carried over unchanged. Each is a placeholder waiting for
a real asset, not a finished decision.

- **Fonts are substituted.** No binaries were supplied. Libre Franklin and IBM Plex Mono load from
  Google Fonts in `design-system/styles/fonts.css`; both cover Latin Extended, so pt-BR diacritics
  render, and both ship tabular figures. Send the real faces and only that file changes.
- **Icons are substituted.** Lucide at 1.75 stroke rather than its default 2, so glyphs sit calmly
  inside dense evidence tables. `design-system/core/icons.ts` is the only seam.
- **No logo and no imagery.** Nothing was provided, so nothing was drawn. `Wordmark` signs the
  product in type wherever a mark would go. There are no photographs, illustrations, textures or
  hero images anywhere, and inventing them would misrepresent the product.
- **Colour values are proposed, not given.** The specification freezes the semantics — green, amber,
  red, blue, grey, violet, each with text and a glyph — and leaves the exact hues open. Every
  foreground clears 4.5:1 on white, but the values await review.
- **Charts are hand-rolled SVG.** `TrendChart` is a faithful visual contract (solid observed,
  dashed forecast, hidden table) and deliberately not a charting-library wrapper.
- **Fixtures are illustrative.** Values were chosen to make each specified state visible — 0.71
  concentration against a 0.60 threshold, 34 days of coverage against a 90-day requirement, a
  withdrawn advisory a changelog still references. None was derived from a real repository.

See [ADR 0007](../../specs/adrs/0007-design-system-and-icon-set.md) for why the design system is
vendored here and why `lucide-react` is the one runtime dependency.
