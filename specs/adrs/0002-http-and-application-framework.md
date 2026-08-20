# ADR 0002 — Router HTTP e organização dos binários

- **Status:** aceito
- **Data:** 2026-08-20

## Contexto

A especificação de produto diz que a API usará HTTP e JSON e exige explicitamente que a decisão
sobre o router HTTP seja registrada em um ADR. Também determina que se use a standard library
sempre que ela resolva o problema de forma adequada e que frameworks não determinem a organização
do domínio.

## Decisão

- A API usa **`net/http` e `http.ServeMux` da standard library**. Sem chi, gin ou echo.
- Desde o Go 1.22 o `ServeMux` roteia por método e por wildcard de path (`GET /api/v1/projects/{id}`),
  que é o que o produto precisa.
- Servidor construído em `internal/platform/httpx` com timeouts explícitos
  (`ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout`, `IdleTimeout`), porque o zero value de
  `http.Server` não tem timeout algum.
- **Dois binários**, `cmd/api` e `cmd/worker`, compartilhando os mesmos packages de `internal/`.
- Ambos fazem graceful shutdown em `SIGINT`/`SIGTERM`, com prazo controlado por `SHUTDOWN_TIMEOUT`.
- Configuração por variáveis de ambiente, lida uma vez na inicialização e passada por construtor.
  Sem estado global mutável.
- Packages organizados por capacidade de negócio, com interfaces pequenas declaradas pelo package
  consumidor. Não existe package `utils`.

## Consequências

Middleware (autenticação, rate limit, request id) será escrito à mão em vez de vir pronto de um
ecossistema de router. Em troca, não há dependência de framework no caminho da requisição e o
roteamento é o mesmo que a documentação da standard library descreve.

## Gatilho de reavaliação

Adotar um router de terceiros se surgir necessidade de recurso que o `ServeMux` não cobre —
por exemplo grupos de rota com middleware aninhado em três ou mais níveis.
