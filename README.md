# Open Source Project Intelligence

Plataforma para coletar, acompanhar, comparar e analisar projetos open source. O objetivo não é um
dashboard sobre o GitHub, e sim uma camada de inteligência que transforma commits, issues, pull
requests, releases e contributors em respostas sobre saúde, sustentabilidade, momentum e risco de
abandono.

A especificação de produto vive em `/workspace/docs/opensource_project_intelligence.md`.

> Estado atual: **fundação**. A estrutura, o ferramental e os processos estão prontos; não existem
> entidades, collectors, métricas nem endpoints de negócio.

## Stack

| Camada       | Escolha                                                         |
| ------------ | --------------------------------------------------------------- |
| API          | Go 1.26 com `net/http` e `http.ServeMux` da stdlib (porta 8100) |
| Worker       | binário Go separado, mesma base de packages                     |
| Persistência | PostgreSQL 18 via `pgx/v5`, SQL explícito, sem ORM              |
| Migrations   | SQL versionado aplicado por `scripts/migrate.sh`                |
| Web          | React 19 + Vite 8 + TypeScript 5.9.3 (porta 3100)               |
| Telemetria   | OpenTelemetry Go 1.45 e `log/slog`                              |
| Testes       | `testing` da stdlib, table-driven, com detector de race         |

As decisões estão registradas em [`specs/adrs/`](specs/adrs/).

## Layout

```text
cmd/
├── api/         # servidor HTTP
└── worker/      # scheduler e collectors
internal/
├── project/ repository/ collector/ issue/ pullrequest/
├── release/ contributor/ metric/ comparison/ analysis/
└── platform/
    ├── config/     # configuração por ambiente
    ├── database/   # pool pgx
    ├── github/     # adapter da API do GitHub
    ├── httpx/      # servidor e respostas JSON
    ├── llm/        # abstração de modelos
    └── telemetry/  # OpenTelemetry
migrations/      # SQL versionado
apps/web/        # frontend React
specs/adrs/      # decisões de arquitetura
```

Packages são organizados por capacidade de negócio, com nomes curtos e sem dependências cíclicas.
Não existe package `utils`. Interfaces são pequenas e declaradas pelo package que as consome.

## Como começar

```bash
cp env.example .env          # ajuste se precisar
pnpm install --frozen-lockfile
go mod download
lefthook install             # instala os git hooks
```

O arquivo de exemplo se chama `env.example`, sem ponto inicial, porque o `.claudeignore` da raiz do
workspace exclui todo caminho `.env.*`.

## Comandos

```bash
make help          # lista os alvos
make build
make test-race
make check         # gofmt, go vet, race e build — o mesmo que a CI roda
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

O `Makefile` define `GOTMPDIR` fora de `/tmp` porque alguns hosts montam `/tmp` com `noexec`, o que
impede o `go test` de executar os binários de teste que ele compila.

## Endpoints

| Método | Rota      | Descrição                                               |
| ------ | --------- | ------------------------------------------------------- |
| `GET`  | `/health` | Liveness. Não toca em dependência alguma.               |
| `GET`  | `/ready`  | Readiness. Responde 503 quando o PostgreSQL não atende. |
| —      | `/api/v1` | Superfície versionada do contrato, ainda vazia.         |

## Portas

Web 3100, API 8100, PostgreSQL 5433. Este produto não usa Valkey, broker externo nem object
storage: eles estão explicitamente fora do escopo do MVP.

## Qualidade

Os hooks do lefthook rodam gitleaks, prettier, markdownlint, hadolint, `gofmt`, `go vet` e ESLint
no `pre-commit`; validam Conventional Commits no `commit-msg`; e rodam a varredura de histórico do
gitleaks mais `make check` e os testes do web no `pre-push`.

```bash
lefthook run pre-commit --all-files
gitleaks git --redact --no-banner
```

Commits seguem Conventional Commits em inglês. Especificações e documentação voltada ao usuário
podem ser escritas em português; código, identificadores e comentários permanecem em inglês.
