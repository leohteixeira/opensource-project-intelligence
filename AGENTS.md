# Open Source Project Intelligence — instruções para agentes

Extensão específica deste repositório. Leia primeiro `/workspace/CLAUDE.md`; o que estiver aqui
prevalece apenas dentro de `repos/opensource-project-intelligence`.

## Estado

Fundação. Existem estrutura de packages, configuração, telemetria, pool de banco, um scheduler com
concorrência limitada e endpoints de health. Não existem entidades, collectors do GitHub, métricas
nem endpoints de negócio.

## Regras de arquitetura

- Packages de `internal/` são organizados por capacidade de negócio, com nomes curtos e sem ciclos.
  **Nunca crie um package `utils`.**
- Interfaces são pequenas e declaradas pelo package **consumidor**, não pelo que as implementa.
- Nenhum tipo vindo da API do GitHub, de um SDK ou de uma resposta HTTP externa entra no domínio:
  converta para o modelo canônico dentro de `internal/platform/github`.
- Propague `context.Context` em toda operação de I/O. Envolva erros com contexto usando `%w`.
- Limite a concorrência explicitamente — use `collector.RunBounded`, nunca uma goroutine por item.
- Toda etapa da ingestão é idempotente e observável. Identificadores externos têm constraint de
  unicidade.
- Nenhuma métrica e nenhuma decisão de sincronização pode depender de uma LLM. A indisponibilidade
  do provider degrada apenas as análises.
- Datas em UTC (`timestamptz`). Payloads brutos para auditoria vão em colunas `JSONB`.

## Convenções

- SQL explícito via `pgx`. Sem ORM e sem gerador de queries.
- Migrations são SQL versionado em `migrations/`, aplicadas por `scripts/migrate.sh`.
- Configuração por variáveis de ambiente, lida uma vez e passada por construtor. Sem estado global
  mutável.
- Logging com `log/slog`. Nunca registre em log connection string, `GITHUB_TOKEN` ou chave de
  provider — `internal/platform/database` redige credenciais antes de propagar erros.
- Testes table-driven; componentes concorrentes passam pelo detector de race.

## Comandos

```bash
make check                  # gofmt, go vet, go test -race, go build
make test-race
pnpm run lint && pnpm run typecheck && pnpm run test && pnpm run build
lefthook run pre-commit --all-files
```

Rode Git sempre a partir deste diretório, nunca de `/workspace`. Não faça commit automaticamente.
