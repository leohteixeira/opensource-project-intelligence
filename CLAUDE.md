# Open Source Project Intelligence — agent instructions

Repository-specific extension. Read `/workspace/CLAUDE.md` first; what is written here prevails
only inside `repos/opensource-project-intelligence`.

## State

Foundation. Package structure, configuration, telemetry, a database pool, a bounded-concurrency
scheduler and health endpoints exist. There are no entities, no GitHub collectors, no metrics and
no business endpoints.

The web application carries the vendored design system (`apps/web/src/design-system`) and the four
shells (`apps/web/src/kits`) against illustrative fixtures. No screen calls the API yet: each kit
reads one `fixtures.ts`, which is the file the HTTP contract replaces.

## Architecture rules

- Packages under `internal/` are organized by business capability, with short names and no cycles.
  **Never create a `utils` package.**
- Interfaces are small and declared by the **consuming** package, not by the one implementing them.
- No type coming from the GitHub API, an SDK or an external HTTP response enters the domain:
  convert it to the canonical model inside `internal/platform/github`.
- Propagate `context.Context` through every I/O operation. Wrap errors with context using `%w`.
- Bound concurrency explicitly — use `collector.RunBounded`, never one goroutine per item.
- Every ingestion step is idempotent and observable. External identifiers carry a uniqueness
  constraint.
- No metric and no synchronization decision may depend on an LLM. Provider unavailability degrades
  the analyses only.
- Dates in UTC (`timestamptz`). Raw payloads kept for audit go into `JSONB` columns.

## Conventions

- Explicit SQL through `pgx`. No ORM and no query generator.
- Migrations are versioned SQL in `migrations/`, applied by `scripts/migrate.sh`.
- Configuration through environment variables, read once and passed through constructors. No
  mutable global state.
- Logging with `log/slog`. Never log a connection string, `GITHUB_TOKEN` or provider key —
  `internal/platform/database` redacts credentials before propagating errors.
- Table-driven tests; concurrent components go through the race detector.

## Frontend conventions

- Screens import from the `design-system` barrel, never from a component file.
- Static styling is inline `style` with `var(--token)` values; interaction states live in
  `design-system/styles/base.css` behind the `opi-*` classes. No CSS Modules and no CSS-in-JS.
- Icons go through `Icon` only; the vocabulary is frozen in `design-system/core/icons.ts`.
- **Missing data is never zero.** `Unknown`, `Not applicable` and `Insufficient data` are three
  distinct states, each with a glyph and a word, and none of them is a blank cell or a `0`.
- Every number carries a unit, a window, a cutoff and a definition version. Colour is never the
  only cue. See `apps/web/README.md` and ADR 0007.

## Language

Everything in this repository is written in English: code, identifiers, comments, branch names,
commit messages, specifications, READMEs and any other documentation.

## Commands

```bash
make check                  # gofmt, go vet, go test -race, go build
make test-race
pnpm run lint && pnpm run typecheck && pnpm run test && pnpm run build
lefthook run pre-commit --all-files
```

Always run Git from this directory, never from `/workspace`. Do not commit automatically.

Before any Go coding, review, debugging, troubleshooting, or setup task,
load the `samber/cc-skills-golang@golang-how-to` skill first — it routes to whichever other Go
skills the task needs.

## Required Go skills

The following Go skills from `samber/cc-skills-golang` MUST always be applied when working on
this project. Load them at the start of every Go-related task, regardless of whether the user
explicitly mentions them.

- `samber/cc-skills-golang@golang-code-style`
- `samber/cc-skills-golang@golang-concurrency`
- `samber/cc-skills-golang@golang-context`
- `samber/cc-skills-golang@golang-continuous-integration`
- `samber/cc-skills-golang@golang-data-structures`
- `samber/cc-skills-golang@golang-database`
- `samber/cc-skills-golang@golang-design-patterns`
- `samber/cc-skills-golang@golang-documentation`
- `samber/cc-skills-golang@golang-error-handling`
- `samber/cc-skills-golang@golang-modernize`
- `samber/cc-skills-golang@golang-naming`
- `samber/cc-skills-golang@golang-safety`
- `samber/cc-skills-golang@golang-security`
- `samber/cc-skills-golang@golang-testing`
- `samber/cc-skills-golang@golang-troubleshooting`
