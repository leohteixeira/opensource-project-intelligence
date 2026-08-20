# ADR 0003 — Driver PostgreSQL, geração de queries e migrations

- **Status:** aceito
- **Data:** 2026-08-20

## Contexto

A especificação exige SQL explícito, migrations versionadas, transações nos limites dos casos de
uso, datas em UTC e constraints de unicidade em identificadores externos. Também exige que as
decisões sobre driver, geração de queries e sistema de migrations sejam registradas em ADR.

## Decisão

- Driver **`github.com/jackc/pgx/v5`** com `pgxpool`, encapsulado em
  `internal/platform/database`. Nenhum package de negócio importa pgx diretamente.
- **Sem ORM e sem gerador de queries.** O SQL é escrito à mão. `sqlc` foi avaliado e descartado
  nesta fase: ainda não existe schema, e o benefício aparece quando há muitas queries estáveis.
- Migrations são **arquivos SQL versionados** em `migrations/`, aplicados em ordem lexicográfica
  por `scripts/migrate.sh`, que controla o que já rodou na tabela `schema_migrations`.
- `goose` foi avaliado como _tool dependency_ e descartado: ele triplicava a árvore de dependências
  do módulo (de 23 para 68 requires indiretos e de 85 para 290 linhas em `go.sum`), porque carrega
  drivers de ClickHouse, SQLite e outros bancos que este projeto não usa. O runner em shell atende
  ao mesmo requisito e mantém o módulo enxuto e igual ao dos repositórios irmãos do portfólio.
- Erros de conexão passam por `redact` antes de virarem log: o pgx inclui a connection string em
  algumas falhas, e ela carrega credenciais.
- Payloads brutos necessários para auditoria e reprocessamento ficam em colunas `JSONB`, com a
  opção de migrar para object storage sem alterar os contratos do domínio.

## Consequências

Cada query precisa de mapeamento manual de linha para struct, o que custa mais código. Em troca, o
SQL que produz uma métrica fica visível e revisável — condição para a auditabilidade que o produto
exige. O runner em shell não tem migration de rollback automática; o rollback é escrito como uma
migration nova.

## Gatilho de reavaliação

Adotar `sqlc` quando o número de queries escritas à mão passar de algo em torno de 30, ou quando o
mapeamento manual começar a produzir defeitos recorrentes. Adotar `goose` ou `golang-migrate` se o
runner em shell deixar de atender, por exemplo por precisar de migrations transacionais complexas
ou de rollback automatizado.
