# Observability

API requests, Jobs, collectors, outbox/consumer work, dependency probes, model/tool calls, and
reconciliation carry W3C trace context plus stable request/correlation/causation identifiers. Logs
use structured `slog`; metrics and traces use OpenTelemetry with bounded-cardinality attributes.

Required signals include request rate/latency/status, Job queue/lease/retry/terminal state, outbox
age and publish failures, consumer redelivery, checkpoint freshness, provider rate limits, object
checksum/reconciliation failures, model tokens/cost/outcomes, readiness classification, and backup
restore results.

Secrets, authorization headers, connection URIs, raw payloads, evidence bytes, model prompts with
sensitive content, and unbounded resource identifiers never enter logs, metrics, traces, readiness,
or public problem details. Internal causes stay available for attributed server logs while HTTP
serialization exposes only frozen safe codes and request IDs.
