# ADR 0006 — Frontend toolchain and TypeScript pin

- **Status:** accepted
- **Date:** 2026-08-20

## Context

CLAUDE.md pins React, TypeScript and Vite for the frontend and pnpm workspaces for shared
dependencies, without defining versions. As of August 2026 the npm `latest` for TypeScript is
7.0.2, the native release rewritten in Go.

## Decision

- React 19.2.8, Vite 8.2.2, `@vitejs/plugin-react` 6.1.0, Vitest 4.1.11.
- **TypeScript pinned at 5.9.3.**
- ESLint 10.8.1 with `typescript-eslint` 8.67.0 in flat config.
- No Tailwind, no router and no data fetching library in this phase.
- Prettier 3.8.1 and markdownlint-cli2 0.22.1, exactly the versions installed in the Dev Container
  image, so that hooks, CI and container agree.
- The development server binds to `0.0.0.0` so that Dev Container port forwarding works.

The TypeScript pin is not generic conservatism: `typescript-eslint` 8.67.0 declares the peer range
`typescript >=4.8.4 <6.1.0`, which excludes 7.x; and `@nestjs/cli` 11.0.24 — used in the sibling
Real Estate repository — bundles `typescript@5.9.3`. Pinning 5.9.3 across the three repositories
keeps the portfolio on a single version supported by the whole toolchain.

## Consequences

The portfolio stays one line behind the native compiler, giving up the 7.x speed gain. In exchange,
lint, build and editor agree with each other and across the three repositories.

## Reassessment trigger

Move to TypeScript 7 once `typescript-eslint` publishes a stable release that declares it as a
supported peer.
