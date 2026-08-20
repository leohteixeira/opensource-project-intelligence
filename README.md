# Open Source Project Intelligence

Platform for collecting, tracking, comparing and analyzing open source projects. The goal is not a
GitHub dashboard, but an intelligence layer that turns commits, issues, pull requests, releases and
contributors into answers about health, sustainability, momentum and abandonment risk.

The product specification lives at `/workspace/docs/opensource_project_intelligence.md`.

> Current state: **foundation**. Structure, tooling and processes are in place; there are no
> entities, no collectors, no metrics and no business endpoints.

## Stack

| Layer       | Choice                                                         |
| ----------- | -------------------------------------------------------------- |
| API         | Go 1.26 with stdlib `net/http` and `http.ServeMux` (port 8100) |
| Worker      | separate Go binary, same package base                          |
| Persistence | PostgreSQL 18 via `pgx/v5`, explicit SQL, no ORM               |
| Migrations  | versioned SQL applied by `scripts/migrate.sh`                  |
| Web         | React 19 + Vite 8 + TypeScript 5.9.3 (port 3100)               |
| Telemetry   | OpenTelemetry Go 1.45 and `log/slog`                           |
| Tests       | stdlib `testing`, table-driven, with the race detector         |

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
    ├── database/   # pgx pool
    ├── github/     # GitHub API adapter
    ├── httpx/      # server and JSON responses
    ├── llm/        # model abstraction
    └── telemetry/  # OpenTelemetry
migrations/      # versioned SQL
apps/web/        # React frontend
specs/adrs/      # architecture decisions
```

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
make check         # gofmt, go vet, race and build — the same thing CI runs
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

| Method | Route     | Description                                            |
| ------ | --------- | ------------------------------------------------------ |
| `GET`  | `/health` | Liveness. Touches no dependency at all.                |
| `GET`  | `/ready`  | Readiness. Answers 503 when PostgreSQL does not reply. |
| —      | `/api/v1` | Versioned contract surface, still empty.               |

## Ports

Web 3100, API 8100, PostgreSQL 5433. This product uses no Valkey, no external broker and no object
storage: they are explicitly out of the MVP scope.

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
