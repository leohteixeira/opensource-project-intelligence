# syntax=docker/dockerfile:1
FROM node:24.19.0-alpine AS build
WORKDIR /source

RUN corepack enable && corepack prepare pnpm@11.22.0 --activate

COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
COPY apps/web/package.json ./apps/web/
RUN pnpm install --frozen-lockfile

COPY apps/web/ ./apps/web/
RUN pnpm --filter "@opensource-project-intelligence/web" build

FROM nginx:1.29-alpine AS runtime

COPY --from=build /source/apps/web/dist /usr/share/nginx/html
COPY docker/web-nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 3100
