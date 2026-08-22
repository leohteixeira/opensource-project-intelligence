# Specifications — Open Source Project Intelligence

This directory is the repository-owned implementation contract. The accepted workflow source under
`.compozy/tasks/opensource-project-intelligence/` is promoted here without narrowing the complete
product delivery.

## Numbered contract

| File                         | Authority                                                   |
| ---------------------------- | ----------------------------------------------------------- |
| `00-brief.md`                | product intent, actors, scope, and non-goals                |
| `01-product-requirements.md` | complete normative product and technical specification      |
| `02-user-stories.md`         | user stories, acceptance criteria, and approved edge cases  |
| `03-domain-model.md`         | canonical aggregates, ownership, and invariants             |
| `04-system-architecture.md`  | modular-monolith and infrastructure boundaries              |
| `05-data-model.md`           | persistence, identifiers, retention, and recovery ownership |
| `06-api-contracts.md`        | frozen HTTP/DX contract and transport rules                 |
| `07-events.md`               | Job, outbox, JetStream, and event-envelope contracts        |
| `08-ai-system.md`            | model/tool boundaries, evidence, and run governance         |
| `09-metrics-model.md`        | deterministic windows, cohorts, coverage, and versions      |
| `10-evaluation-strategy.md`  | deterministic, AI, contract, and boundary gates             |
| `11-observability.md`        | logs, metrics, traces, correlation, and redaction           |
| `12-security.md`             | identity, authorization, secrets, SSRF, and audit rules     |
| `13-deployment.md`           | services, ports, readiness, backup, and restore             |
| `14-ui-ux.md`                | normative UI surface and accessibility map                  |
| `15-test-contract.md`        | stable test IDs and release-gate matrix                     |

`01`, `02`, `06`, `14`, and `15` preserve the accepted workflow catalogs in full. The focused files
make their cross-cutting decisions discoverable; when a focused summary and a normative catalog
appear to conflict, the complete requirement and test catalogs win.

Architecture decisions live under `adrs/`. Accepted ADRs are historical records; changes are made by
new ADRs that identify exactly what they supersede.
