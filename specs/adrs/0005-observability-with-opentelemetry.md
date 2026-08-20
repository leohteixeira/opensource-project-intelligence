# ADR 0005 — Observabilidade com OpenTelemetry

- **Status:** aceito
- **Data:** 2026-08-20

## Contexto

A especificação exige instrumentação com OpenTelemetry de collection jobs, chamadas de API, erros
de fonte, latência de ingestão, enrichment, execuções de agente, custo de LLM e jobs de analytics,
com logs, métricas e traces correlacionáveis. Um critério de conclusão do MVP é que API e jobs
tenham logs, métricas e traces correlacionáveis.

## Decisão

- OpenTelemetry Go 1.45.0 configurado em `internal/platform/telemetry`, consumido pelos dois
  binários.
- Propagação `TraceContext` + `Baggage` sempre instalada.
- Exportação OTLP ativada **somente** quando `OTEL_EXPORTER_OTLP_ENDPOINT` está definido, para que
  o processo funcione sem um collector local.
- Instrumentação HTTP do servidor via `otelhttp`.
- Logging com **`log/slog`** da standard library, handler JSON. Sem zap e sem zerolog.
- Connection strings, tokens do GitHub e chaves de provider nunca entram em log, span ou métrica.
  O package `database` redige credenciais antes de propagar um erro.

## Consequências

Um único ponto de configuração para telemetria. A correlação entre log e trace ainda não é
automática: quando os handlers de negócio existirem, o `trace_id` precisará ser adicionado ao
`slog.Logger` do request.

## Gatilho de reavaliação

Substituir `log/slog` apenas se for necessário um sink que ele não cubra. Adicionar um handler
`slog` que injete `trace_id`/`span_id` assim que existir o primeiro endpoint de negócio.
