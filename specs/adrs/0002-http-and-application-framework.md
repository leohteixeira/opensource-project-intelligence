# ADR 0002 — HTTP router and binary layout

- **Status:** accepted
- **Date:** 2026-08-20

## Context

The product specification says the API will use HTTP and JSON and explicitly requires the decision
about the HTTP router to be recorded in an ADR. It also mandates using the standard library
whenever it solves the problem adequately, and that frameworks must not dictate how the domain is
organized.

## Decision

- The API uses the **standard library `net/http` and `http.ServeMux`**. No chi, gin or echo.
- Since Go 1.22 `ServeMux` routes by method and by path wildcard (`GET /api/v1/projects/{id}`),
  which is what the product needs.
- The server is built in `internal/platform/httpx` with explicit timeouts (`ReadHeaderTimeout`,
  `ReadTimeout`, `WriteTimeout`, `IdleTimeout`), because the zero value of `http.Server` has no
  timeout at all.
- **Two binaries**, `cmd/api` and `cmd/worker`, sharing the same packages from `internal/`.
- Both shut down gracefully on `SIGINT`/`SIGTERM`, with a deadline controlled by
  `SHUTDOWN_TIMEOUT`.
- Configuration through environment variables, read once at startup and passed through
  constructors. No mutable global state.
- Packages organized by business capability, with small interfaces declared by the consuming
  package. There is no `utils` package.

## Consequences

Middleware (authentication, rate limiting, request id) will be written by hand instead of coming
ready-made from a router ecosystem. In exchange, there is no framework dependency on the request
path and routing behaves exactly as the standard library documentation describes.

## Reassessment trigger

Adopt a third-party router if a need arises for a feature `ServeMux` does not cover — for example
route groups with middleware nested three or more levels deep.
