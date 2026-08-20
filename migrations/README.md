# Migrations

Migrations são arquivos SQL versionados, aplicados em ordem lexicográfica por
`scripts/migrate.sh`. A decisão de usar SQL explícito, sem ORM e sem framework de migrations, está
registrada em
[`../specs/adrs/0003-persistence-driver-and-migrations.md`](../specs/adrs/0003-persistence-driver-and-migrations.md).

## Convenção de nome

```text
NNNN_descricao_em_snake_case.sql
```

`NNNN` é um contador sequencial de quatro dígitos, nunca reaproveitado.

## Regras

- Datas e timestamps são armazenados em UTC (`timestamptz`).
- Identificadores externos (por exemplo o `id` numérico do GitHub) recebem constraint de
  unicidade, para que a sincronização incremental seja idempotente.
- Payloads brutos necessários para auditoria e reprocessamento ficam em colunas `JSONB`.
- Nenhuma migration de negócio existe ainda: o repositório está na fase de fundação.
