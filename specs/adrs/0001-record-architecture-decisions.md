# ADR 0001 — Registrar decisões de arquitetura

- **Status:** aceito
- **Data:** 2026-08-20

## Contexto

O documento de produto deixa deliberadamente em aberto escolhas materiais de implementação
(framework HTTP, driver de banco, geração de queries, sistema de migrations) e exige que elas
sejam registradas. Sem um registro, a motivação de cada escolha se perde e revisitá-la vira
arqueologia de código.

## Decisão

Toda decisão material de arquitetura será registrada como um ADR numerado neste diretório,
usando um template MADR reduzido: contexto, decisão, consequências e gatilho de reavaliação.

- Numeração sequencial de quatro dígitos, nunca reaproveitada.
- Um ADR nunca é editado depois de aceito. Para mudar a decisão, escreva um novo ADR e marque
  o anterior como `substituído por ADR NNNN`.
- Status possíveis: `proposto`, `aceito`, `substituído`, `descartado`.

## Consequências

Mudanças estruturais passam a exigir um documento curto antes do código, o que adiciona um passo
ao fluxo. Em troca, cada escolha carrega sua justificativa e a condição sob a qual deve ser
revista.

## Gatilho de reavaliação

Nenhum. Este ADR descreve o processo em si.
