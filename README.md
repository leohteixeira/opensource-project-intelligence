# Open Source Project Intelligence

Platform for collecting, tracking, comparing and analyzing open source projects. The goal is not a
GitHub dashboard, but an intelligence layer that turns commits, issues, pull requests, releases and
contributors into answers about health, sustainability, momentum and abandonment risk.

The product specification lives at `/workspace/docs/opensource_project_intelligence.md`.

> Current state: **foundation**. Structure, tooling and processes are in place; there are no
> entities, no collectors, no metrics and no business endpoints. The web application carries the
> full interface — design system, four shells and every specified surface — against illustrative
> fixtures, waiting for the HTTP contract.

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
