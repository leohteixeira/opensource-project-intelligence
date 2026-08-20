#!/usr/bin/env bash
# Applies the versioned SQL migrations in lexicographic order.
#
# Requires `psql` (already present in the Dev Container image) and a
# DATABASE_URL pointing at the target database.
set -Eeuo pipefail

readonly MIGRATIONS_DIR="${MIGRATIONS_DIR:-"$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/migrations"}"
readonly DATABASE_URL="${DATABASE_URL:?DATABASE_URL is required, for example postgresql://user:password@localhost:5432/db}"

run_sql() {
  psql "${DATABASE_URL}" --set ON_ERROR_STOP=on --quiet --no-psqlrc "$@"
}

ensure_control_table() {
  run_sql --command "
    CREATE TABLE IF NOT EXISTS schema_migrations (
      version     text        PRIMARY KEY,
      applied_at  timestamptz NOT NULL DEFAULT now()
    );
  "
}

is_applied() {
  local version="$1"
  local found

  found="$(run_sql --tuples-only --no-align --command \
    "SELECT 1 FROM schema_migrations WHERE version = '${version}';")"
  [[ -n "${found}" ]]
}

apply_migration() {
  local file="$1"
  local version
  version="$(basename "${file}" .sql)"

  if is_applied "${version}"; then
    printf 'skip   %s\n' "${version}"
    return 0
  fi

  printf 'apply  %s\n' "${version}"
  run_sql --single-transaction \
    --file "${file}" \
    --command "INSERT INTO schema_migrations (version) VALUES ('${version}');"
}

main() {
  ensure_control_table

  shopt -s nullglob
  local -a migrations=("${MIGRATIONS_DIR}"/*.sql)
  shopt -u nullglob

  if (( ${#migrations[@]} == 0 )); then
    echo "No migrations found in ${MIGRATIONS_DIR}."
    return 0
  fi

  local migration
  for migration in "${migrations[@]}"; do
    apply_migration "${migration}"
  done

  echo "Migrations completed."
}

main "$@"
