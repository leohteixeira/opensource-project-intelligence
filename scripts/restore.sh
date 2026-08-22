#!/usr/bin/env bash
# Restores PostgreSQL and verifies the canonical object manifest checksum.
set -Eeuo pipefail

readonly DATABASE_URL="${DATABASE_URL:?DATABASE_URL is required}"
readonly BACKUP_DIRECTORY="${BACKUP_DIRECTORY:?BACKUP_DIRECTORY must name the backup directory}"
readonly POSTGRES_CONTAINER="${POSTGRES_CONTAINER:-}"
readonly POSTGRES_DATABASE="${POSTGRES_DATABASE:-}"
readonly POSTGRES_USER="${POSTGRES_USER:-}"

(
  cd "${BACKUP_DIRECTORY}"
  sha256sum --check --strict backup.sha256
)

if [[ -n "${POSTGRES_CONTAINER}" ]]; then
  : "${POSTGRES_DATABASE:?POSTGRES_DATABASE is required with POSTGRES_CONTAINER}"
  : "${POSTGRES_USER:?POSTGRES_USER is required with POSTGRES_CONTAINER}"
  readonly CONTAINER_DUMP="/tmp/opi-platform-$(printf '%s' "${BACKUP_DIRECTORY}" | sha256sum | cut -c1-16).dump"
  docker cp "${BACKUP_DIRECTORY}/postgres.dump" "${POSTGRES_CONTAINER}:${CONTAINER_DUMP}"
  docker exec "${POSTGRES_CONTAINER}" pg_restore \
    --username="${POSTGRES_USER}" \
    --clean --if-exists --no-owner --no-privileges \
    --dbname="${POSTGRES_DATABASE}" \
    "${CONTAINER_DUMP}"
else
  pg_restore --clean --if-exists --no-owner --no-privileges \
    --dbname="${DATABASE_URL}" \
    "${BACKUP_DIRECTORY}/postgres.dump"
fi

echo "PostgreSQL restored. Restore S3 bytes, verify each SHA-256 from object-manifest.csv, then rebuild JetStream and Valkey from PostgreSQL."
