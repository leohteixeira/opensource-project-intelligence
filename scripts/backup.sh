#!/usr/bin/env bash
# Captures PostgreSQL and the canonical object-reference manifest.
set -Eeuo pipefail

readonly DATABASE_URL="${DATABASE_URL:?DATABASE_URL is required}"
readonly BACKUP_DIRECTORY="${BACKUP_DIRECTORY:?BACKUP_DIRECTORY must name a dedicated backup directory}"
readonly POSTGRES_CONTAINER="${POSTGRES_CONTAINER:-}"
readonly POSTGRES_DATABASE="${POSTGRES_DATABASE:-}"
readonly POSTGRES_USER="${POSTGRES_USER:-}"

mkdir -p "${BACKUP_DIRECTORY}"

if [[ -n "${POSTGRES_CONTAINER}" ]]; then
  : "${POSTGRES_DATABASE:?POSTGRES_DATABASE is required with POSTGRES_CONTAINER}"
  : "${POSTGRES_USER:?POSTGRES_USER is required with POSTGRES_CONTAINER}"
  readonly CONTAINER_DUMP="/tmp/opi-platform-$(printf '%s' "${BACKUP_DIRECTORY}" | sha256sum | cut -c1-16).dump"
  docker exec "${POSTGRES_CONTAINER}" pg_dump \
    --username="${POSTGRES_USER}" \
    --format=custom --no-owner --no-privileges \
    --file="${CONTAINER_DUMP}" \
    "${POSTGRES_DATABASE}"
  docker cp "${POSTGRES_CONTAINER}:${CONTAINER_DUMP}" "${BACKUP_DIRECTORY}/postgres.dump"
  docker exec "${POSTGRES_CONTAINER}" psql \
    --username="${POSTGRES_USER}" \
    --dbname="${POSTGRES_DATABASE}" \
    --no-psqlrc --quiet \
    --command="COPY (
      SELECT object_key, encode(sha256, 'hex') AS sha256, size_bytes, media_type
      FROM object_references
      WHERE retention_state <> 'purged'
      ORDER BY object_key
    ) TO STDOUT WITH CSV HEADER;" \
    > "${BACKUP_DIRECTORY}/object-manifest.csv"
else
  pg_dump --format=custom --no-owner --no-privileges \
    --file="${BACKUP_DIRECTORY}/postgres.dump" \
    "${DATABASE_URL}"

  psql "${DATABASE_URL}" --no-psqlrc --quiet \
    --output="${BACKUP_DIRECTORY}/object-manifest.csv" \
    --command="COPY (
      SELECT object_key, encode(sha256, 'hex') AS sha256, size_bytes, media_type
      FROM object_references
      WHERE retention_state <> 'purged'
      ORDER BY object_key
    ) TO STDOUT WITH CSV HEADER;"
fi

(
  cd "${BACKUP_DIRECTORY}"
  sha256sum postgres.dump object-manifest.csv > backup.sha256
)

echo "Database backup and canonical object manifest completed."
echo "Back up the manifest-listed object bytes with the deployment S3 tool before declaring the backup complete."
