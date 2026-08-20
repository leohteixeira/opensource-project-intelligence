# Especificações — Open Source Project Intelligence

A especificação de produto que originou este repositório vive no meta-workspace, em
`/workspace/docs/`. Este diretório guarda as especificações versionadas **junto ao código** e as
decisões de arquitetura.

## Estrutura prevista

A seção "Specification-Driven Development" do documento de produto prevê a estrutura numerada
abaixo. Os arquivos serão escritos conforme cada capacidade for implementada; hoje o repositório
está na fase de fundação e nenhum deles existe ainda.

```text
specs/
├── 00-brief.md
├── 01-product-requirements.md
├── 02-user-stories.md
├── 03-domain-model.md
├── 04-system-architecture.md
├── 05-data-model.md
├── 06-api-contracts.md
├── 07-events.md
├── 08-ai-system.md
├── ...
└── adrs/
```

## ADRs

`adrs/` contém as decisões de arquitetura já tomadas. Toda decisão material sobre framework,
persistência, telemetria ou toolchain precisa de um ADR antes de virar código.

As especificações e a documentação voltada ao usuário podem ser escritas em português. Código,
identificadores, comentários, nomes de branch e mensagens de commit permanecem em inglês.
