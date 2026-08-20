# ADR 0006 — Toolchain de frontend e pin do TypeScript

- **Status:** aceito
- **Data:** 2026-08-20

## Contexto

O CLAUDE.md fixa React, TypeScript e Vite para o frontend e pnpm workspaces para as dependências
compartilhadas, sem definir versões. Em agosto de 2026 o `latest` do npm para TypeScript é 7.0.2,
a versão nativa reescrita em Go.

## Decisão

- React 19.2.8, Vite 8.2.2, `@vitejs/plugin-react` 6.1.0, Vitest 4.1.11.
- **TypeScript fixado em 5.9.3.**
- ESLint 10.8.1 com `typescript-eslint` 8.67.0 em flat config.
- Sem Tailwind, sem router e sem biblioteca de data fetching nesta fase.
- Prettier 3.8.1 e markdownlint-cli2 0.22.1, exatamente as versões instaladas na imagem do Dev
  Container, para que hooks, CI e container concordem.
- O servidor de desenvolvimento faz bind em `0.0.0.0` para que o encaminhamento de portas do Dev
  Container funcione.

O pin do TypeScript não é conservadorismo genérico: `typescript-eslint` 8.67.0 declara peer
`typescript >=4.8.4 <6.1.0`, o que exclui a 7.x; e `@nestjs/cli` 11.0.24 — usado no repositório
irmão de Real Estate — embarca `typescript@5.9.3`. Fixar 5.9.3 nos três repositórios mantém o
portfólio sobre uma única versão suportada por todo o ferramental.

## Consequências

O portfólio fica uma linha atrás do compilador nativo, perdendo o ganho de velocidade da 7.x. Em
troca, lint, build e editor concordam entre si e entre os três repositórios.

## Gatilho de reavaliação

Subir para TypeScript 7 quando `typescript-eslint` publicar uma versão estável que o declare como
peer suportado.
