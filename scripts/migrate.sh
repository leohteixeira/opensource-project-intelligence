#!/usr/bin/env bash
# Applies or rolls back checksum-verified SQL migrations.
set -Eeuo pipefail

readonly REPOSITORY_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
readonly MIGRATIONS_DIR="${MIGRATIONS_DIR:-${REPOSITORY_ROOT}/migrations}"
readonly DATABASE_URL="${DATABASE_URL:?DATABASE_URL is required}"
readonly DIRECTION="${1:-up}"

run_sql() {
  psql "${DATABASE_URL}" --set ON_ERROR_STOP=on --quiet --no-psqlrc "$@"
}

ensure_control_table() {
  run_sql --command "
    CREATE TABLE IF NOT EXISTS schema_migrations (
      version     text        PRIMARY KEY,
      checksum    text        NOT NULL,
      applied_at  timestamptz NOT NULL DEFAULT now()
    );
  "
}

checksum() {
  sha256sum "$1" | awk '{print $1}'
}

apply_migration() {
  local file="$1"
  local version expected recorded
  version="$(basename "${file}" .up.sql)"
  expected="$(checksum "${file}")"
  recorded="$(run_sql --tuples-only --no-align --command \
    "SELECT checksum FROM schema_migrations WHERE version = '${version}';")"

  if [[ -n "${recorded}" ]]; then
    if [[ "${recorded}" != "${expected}" ]]; then
      echo "migration ${version} checksum differs from the applied migration" >&2
      return 1
    fi
    printf 'skip   %s\n' "${version}"
    return 0
  fi

  printf 'apply  %s\n' "${version}"
  run_sql --single-transaction \
    --file "${file}" \
    --command "INSERT INTO schema_migrations (version, checksum) VALUES ('${version}', '${expected}');"
}

rollback_latest() {
  local version down_file
  version="$(run_sql --tuples-only --no-align --command \
    'SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1;')"
  if [[ -z "${version}" ]]; then
    echo "No migration to roll back."
    return 0
  fi

  down_file="${MIGRATIONS_DIR}/${version}.down.sql"
  if [[ ! -f "${down_file}" ]]; then
    echo "migration ${version} has no reviewed down migration" >&2
    return 1
  fi

  printf 'revert %s\n' "${version}"
  run_sql --single-transaction \
    --file "${down_file}" \
    --command "DELETE FROM schema_migrations WHERE version = '${version}';"
}

main() {
  ensure_control_table
  case "${DIRECTION}" in
    up)
      shopt -s nullglob
      local -a migrations=("${MIGRATIONS_DIR}"/*.up.sql)
      shopt -u nullglob
      local migration
      for migration in "${migrations[@]}"; do
        apply_migration "${migration}"
      done
      ;;
    down)
      rollback_latest
      ;;
    *)
      echo "usage: scripts/migrate.sh [up|down]" >&2
      return 2
      ;;
  esac
}

main "$@"
