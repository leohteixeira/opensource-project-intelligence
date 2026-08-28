# Open Source Project Intelligence

Platform for collecting, tracking, comparing and analyzing open source projects. The goal is not a
GitHub dashboard, but an intelligence layer that turns commits, issues, pull requests, releases and
contributors into answers about health, sustainability, momentum and abandonment risk.

The versioned product specification and its delivery tasks live under
`.compozy/tasks/opensource-project-intelligence/`; material architecture decisions live under
`specs/adrs/`. The modular monolith includes the versioned API, worker pipeline, deterministic
intelligence calculations, governed AI runs, bounded human-approved assistant actions, durable
exports, and the English/Portuguese web surfaces.

## Stack

| Layer        | Choice                                                                  |
| ------------ | ----------------------------------------------------------------------- |
| API          | Go 1.26 with strict generated `net/http` types (port 8100)              |
| Worker       | separate Go binary, same package base                                   |
| Persistence  | PostgreSQL 18/pgvector via `pgx/v5` and generated sqlc adapters         |
| Delivery     | NATS JetStream from a PostgreSQL transactional outbox                   |
| Evidence     | S3-compatible bytes owned by PostgreSQL object references               |
| Acceleration | disposable Valkey caches, limits, and ephemeral fanout                  |
| Migrations   | checksum-verified up/down SQL applied by `scripts/migrate.sh`           |
| Web          | React 19 + Vite 8 + generated TypeScript client (port 3100)             |
| Interface    | vendored design system, tokens and four shells in `apps/web`            |
| Telemetry    | OpenTelemetry Go 1.45 and `log/slog`                                    |
| Tests        | table-driven unit, race, generated-drift, and real-boundary integration |

The decisions are recorded in [`specs/adrs/`](specs/adrs/).

## Layout

```text
cmd/
├── api/         # HTTP server
└── worker/      # scheduler and collectors
internal/
├── project/ repository/ collector/ issue/ pullrequest/
├── release/ contributor/ metric/ comparison/ analysis/
└── platform/
    ├── config/     # configuration from the environment
    ├── database/   # pgx pool, reviewed queries, and generated sqlc adapters
    ├── health/     # required/optional dependency classification
    ├── httpapi/    # generated strict OpenAPI server types
    ├── github/     # GitHub API adapter
    ├── httpx/      # server and JSON responses
    ├── id/         # database-leased Snowflake identifiers
    ├── llm/        # model abstraction
    └── telemetry/  # OpenTelemetry
migrations/      # checksum-verified up/down SQL
api/             # reviewed OpenAPI 3.1 source and generator configuration
apps/web/        # React frontend
specs/           # complete numbered product/technical contracts and ADRs
```

The frontend carries the vendored design system (`src/design-system`) and the four shells
(`src/kits`): public catalog, member workspace, project evidence and administration. See
[`apps/web/README.md`](apps/web/README.md) and
[ADR 0007](specs/adrs/0007-design-system-and-icon-set.md).

Packages are organized by business capability, with short names and no cyclic dependencies. There
is no `utils` package. Interfaces are small and declared by the package that consumes them.

## Getting started

```bash
cp env.example .env          # adjust if needed
pnpm install --frozen-lockfile
go mod download
lefthook install             # installs the git hooks
```

The example file is named `env.example`, without a leading dot, because the workspace root
`.claudeignore` excludes every `.env.*` path.

## Commands

```bash
make help          # lists the targets
make build
make test-race
make generate      # reproduce OpenAPI, sqlc, and TypeScript adapters
make generate-check
make check         # generated drift, gofmt, go vet, race and build — CI parity
make run-api       # http://0.0.0.0:8100
make run-worker

pnpm run lint
pnpm run typecheck
pnpm run test
pnpm run build
pnpm --filter "@opensource-project-intelligence/web" dev   # http://0.0.0.0:3100

docker compose up -d
DATABASE_URL=postgres://opensource:opensource@localhost:5433/opensource_project_intelligence \
  make migrate
```

The `Makefile` sets `GOTMPDIR` outside `/tmp` because some hosts mount `/tmp` with `noexec`, which
prevents `go test` from running the test binaries it compiles.

## Endpoints

| Method | Route     | Description                                             |
| ------ | --------- | ------------------------------------------------------- |
| `GET`  | `/health` | Liveness. Touches no dependency at all.                 |
| `GET`  | `/ready`  | Required dependency readiness and optional degradation. |
| —      | `/api/v1` | Versioned surface defined in `api/openapi.yaml`.        |

## Ports

Web 3100, API 8100, PostgreSQL 5433. JetStream, Valkey, and S3-compatible storage remain internal to
the repository-local Compose network and consume no additional host ports.

## Quality

The lefthook hooks run gitleaks, prettier, markdownlint, hadolint, `gofmt`, `go vet` and ESLint on
`pre-commit`; they validate Conventional Commits on `commit-msg`; and they run the gitleaks history
scan plus `make check` and the web tests on `pre-push`.

```bash
lefthook run pre-commit --all-files
gitleaks git --redact --no-banner
```

Everything in this repository is written in English: code, identifiers, comments, branch names,
commit messages, specifications and documentation. Commits follow Conventional Commits.

## Operations and recovery

`GET /health` proves only that the API process is alive. `GET /ready` checks PostgreSQL and reports
JetStream and object storage as required when enabled; Valkey and the model provider are optional.
A missing or degraded model provider therefore does not make deterministic collection, metrics,
health, policies, radar, comparisons, or trends unavailable. AI endpoints return a stable safe
problem response while the Admin Operations screen continues to expose redacted health and
aggregate usage.

The runtime enforces finite concurrency and time, output, step, and cost ceilings. The checked-in
defaults are deliberately conservative:

| Variable               |  Default | Purpose                                                |
| ---------------------- | -------: | ------------------------------------------------------ |
| `AI_CONCURRENCY`       |      `4` | Concurrent model runs across the process               |
| `ADK_MAX_STEPS`        |     `12` | Maximum bounded agent steps                            |
| `ADK_TIMEOUT`          |     `2m` | End-to-end assistant deadline; never above ten minutes |
| `ADK_MAX_OUTPUT_BYTES` |  `65536` | Maximum typed planner output                           |
| `ADK_MAX_COST_MICROS`  | `100000` | Per-run cost ceiling in micros                         |
| `ADK_TOOL_CONCURRENCY` |      `1` | Concurrent typed assistant actions                     |
| `EXPORT_CONCURRENCY`   |      `2` | Concurrent export artifact generation                  |
| `SHUTDOWN_TIMEOUT`     |    `15s` | API and worker graceful-shutdown budget                |

Set `AI_PROVIDER` and `AI_MODEL` together to advertise AI capabilities. Provider credentials remain
deployment secrets and must never be committed, logged, returned by status endpoints, or included
in audit metadata. Assistant confirmations are single-use, action-bound, actor-bound, and expire
after ten minutes. The execution path rechecks authorization, resource version, quota, and the
allowlisted non-destructive action before changing state.

Exports are asynchronous jobs. A request freezes one scope, window, locale, and cutoff; successful
artifacts expose a SHA-256 checksum and expire after 24 hours or earlier when a referenced project
is purged. Downloads are authorized against the original requester. CSV keeps stable machine fields
and localized human labels; evidence JSON preserves formulas, versions, policy context, analysis
references, coverage, and provenance.

### Backup

PostgreSQL is the system of record. A complete backup contains the custom-format database dump,
the checksummed object manifest, and every S3 byte named by that manifest. JetStream and Valkey are
rebuildable and are not backup authorities. Use a dedicated directory outside the repository:

Load `DATABASE_URL` from the deployment secret manager, then run:

```bash
BACKUP_DIRECTORY='/var/backups/opi/2026-08-27' make backup

cd /var/backups/opi/2026-08-27
sha256sum --check --strict backup.sha256
```

For the repository Compose deployment, set `POSTGRES_CONTAINER`, `POSTGRES_DATABASE`, and
`POSTGRES_USER`; the script then runs `pg_dump` inside that container. After `make backup`, copy the
manifest-listed object bytes with the deployment's S3 tooling before declaring the backup complete.

### Restore rehearsal

Restore is destructive to the target database. Rehearse it only against an empty disposable
database, never against the sole production copy:

Load `DATABASE_URL` for the disposable target from the deployment secret manager, then run:

```bash
BACKUP_DIRECTORY='/var/backups/opi/2026-08-27' make restore
```

Then restore the S3 bytes, verify each object against `object-manifest.csv`, run `make migrate`, and
rebuild JetStream and Valkey from PostgreSQL. Start the API and worker, require `GET /ready` to return
ready, request and verify one evidence export checksum, and exercise one collector checkpoint before
promoting the restored environment.

### Shutdown and incident checks

Send `SIGTERM` to the API and worker and allow the configured shutdown timeout. The API stops
accepting traffic and drains its HTTP server. Workers stop taking new jobs, persist checkpoints,
and convert interrupted work to a retryable or terminal state rather than leaving an ambiguous
running record. During an incident, inspect correlated OpenTelemetry request/job/run IDs and the
append-only Admin audit log; do not copy raw authorization headers, prompts, credentials, or source
payloads into logs or tickets.

Before a release, run the same checks as CI plus the real-boundary and browser suites:

```bash
make generate-check
make check
OPI_INTEGRATION_DATABASE_URL="$DATABASE_URL" make test-integration
pnpm run lint
pnpm run typecheck
pnpm run test
pnpm run build
pnpm --filter '@opensource-project-intelligence/web' run test:e2e
lefthook run pre-commit --all-files
```
