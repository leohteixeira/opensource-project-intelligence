# ADR 0004 — Testing strategy

- **Status:** accepted
- **Date:** 2026-08-20

## Context

The specification requires table-driven tests for deterministic rules and metrics, and running the
race detector on concurrent components. CLAUDE.md requires `go test ./...` and `go test -race ./...`,
plus the pnpm scripts for the web application.

## Decision

- Tests with the **standard library `testing` package**, in table-driven style. No `testify` and no
  mocking framework: test doubles are hand-written as implementations of the small interfaces
  declared by the consuming packages.
- External test package (`package foo_test`) by default, to exercise the public API. Internal
  package only when the target is an unexported function, such as `redact`.
- `make test-race` runs the race detector; `make check` bundles formatting, vet, race and build,
  and is what `pre-push` and CI execute.
- The web application uses **Vitest 4**, run through `pnpm test`.
- Integration tests against a real PostgreSQL will be added together with the first concrete
  repository, using the database from `compose.yaml`.

### GOTMPDIR

The `Makefile` sets `GOTMPDIR ?= $(HOME)/.cache/go-tmp`. Some hardened hosts mount `/tmp` with
`noexec`, and `go test` compiles the test binaries into the temporary directory before running
them — without this, every test fails with `fork/exec ...: permission denied`. The variable can be
overridden and the default value also works on CI runners.

## Consequences

More verbose failure messages than with an assertion library, and more lines per test double. In
exchange, zero test dependencies in the module and explicit port contracts.

## Reassessment trigger

Adopt `testcontainers-go` once there is a first integration suite that needs to start and stop
PostgreSQL per test.
