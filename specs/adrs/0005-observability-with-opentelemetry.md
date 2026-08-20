# ADR 0005 — Observability with OpenTelemetry

- **Status:** accepted
- **Date:** 2026-08-20

## Context

The specification requires OpenTelemetry instrumentation of collection jobs, API calls, source
errors, ingestion latency, enrichment, agent runs, LLM cost and analytics jobs, with correlatable
logs, metrics and traces. One of the MVP completion criteria is that the API and the jobs have
correlatable logs, metrics and traces.

## Decision

- OpenTelemetry Go 1.45.0 configured in `internal/platform/telemetry`, consumed by both binaries.
- `TraceContext` + `Baggage` propagation always installed.
- OTLP export enabled **only** when `OTEL_EXPORTER_OTLP_ENDPOINT` is set, so that the process runs
  without a local collector.
- Server-side HTTP instrumentation through `otelhttp`.
- Logging with the standard library **`log/slog`**, JSON handler. No zap and no zerolog.
- Connection strings, GitHub tokens and provider keys never reach a log, a span or a metric. The
  `database` package redacts credentials before propagating an error.

## Consequences

A single configuration point for telemetry. Correlation between log and trace is not automatic yet:
once the business handlers exist, the `trace_id` will have to be added to the request's
`slog.Logger`.

## Reassessment trigger

Replace `log/slog` only if a sink is needed that it does not cover. Add a `slog` handler injecting
`trace_id`/`span_id` as soon as the first business endpoint exists.
