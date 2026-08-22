#!/usr/bin/env bash
# Regenerates every committed adapter boundary from reviewed sources.
set -Eeuo pipefail

readonly REPOSITORY_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${REPOSITORY_ROOT}"

go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.8.0 \
  --config api/oapi-codegen.yaml api/openapi.yaml
go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1 generate
pnpm --filter '@opensource-project-intelligence/web' exec openapi-ts \
  --input ../../api/openapi.yaml \
  --output ./src/api/generated
pnpm exec prettier --write apps/web/src/api/generated >/dev/null

mapfile -t generated_typescript < <(
  find apps/web/src/api/generated -type f -name '*.ts' -print | LC_ALL=C sort
)

sha256sum \
  api/openapi.yaml \
  sqlc.yaml \
  migrations/0001_platform.up.sql \
  internal/platform/database/query/platform.sql \
  internal/platform/httpapi/openapi.gen.go \
  internal/platform/database/sqlc/*.go \
  "${generated_typescript[@]}" \
  > generated.sha256
