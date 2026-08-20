# Specifications — Open Source Project Intelligence

The product specification this repository originated from lives in the meta-workspace, under
`/workspace/docs/`. This directory holds the versioned specifications kept **next to the code**
along with the architecture decisions.

## Planned structure

The "Specification-Driven Development" section of the product document describes the numbered
structure below. The files are written as each capability is implemented; today the repository is
in the foundation phase and none of them exists yet.

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

`adrs/` holds the architecture decisions already taken. Every material decision about a framework,
persistence, telemetry or toolchain needs an ADR before it becomes code.

Specifications and user-facing documentation are written in English, like the rest of the
repository: code, identifiers, comments, branch names and commit messages.
