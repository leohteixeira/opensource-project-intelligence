# ADR 0004 — Estratégia de testes

- **Status:** aceito
- **Data:** 2026-08-20

## Contexto

A especificação exige testes table-driven para regras e métricas determinísticas e execução do
detector de race conditions nos componentes concorrentes. O CLAUDE.md exige `go test ./...` e
`go test -race ./...`, além dos scripts pnpm para a aplicação web.

## Decisão

- Testes com o package **`testing` da standard library**, no estilo table-driven. Sem `testify`
  e sem framework de mock: dublês de teste são escritos à mão como implementações das interfaces
  pequenas declaradas pelos packages consumidores.
- Testes em package externo (`package foo_test`) por padrão, para exercitar a API pública.
  Package interno apenas quando o alvo for uma função não exportada, como `redact`.
- `make test-race` roda o detector de race conditions; `make check` reúne formatação, vet, race e
  build, e é o que o `pre-push` e a CI executam.
- A aplicação web usa **Vitest 4**, executado por `pnpm test`.
- Testes de integração contra PostgreSQL real serão adicionados junto com o primeiro repositório
  concreto, usando o banco do `compose.yaml`.

### GOTMPDIR

O `Makefile` define `GOTMPDIR ?= $(HOME)/.cache/go-tmp`. Alguns hosts endurecidos montam `/tmp`
com `noexec`, e o `go test` compila os binários de teste no diretório temporário antes de
executá-los — sem isso, todo teste falha com `fork/exec ...: permission denied`. A variável é
sobrescrevível e o valor padrão funciona também em runners de CI.

## Consequências

Mensagens de falha mais verbosas do que com uma biblioteca de asserção, e mais linhas por dublê de
teste. Em troca, zero dependência de teste no módulo e contratos de porta explícitos.

## Gatilho de reavaliação

Adotar `testcontainers-go` quando existir a primeira suíte de integração que precise subir e
derrubar PostgreSQL por teste.
