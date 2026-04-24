#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
MIGRATIONS_DIR="${MIGRATIONS_DIR:-${ROOT_DIR}/db/migrations}"
POSTGRES_TOOLS_IMAGE="${POSTGRES_TOOLS_IMAGE:-postgres:17-alpine}"
POSTGRES_TOOLS_MODE="${POSTGRES_TOOLS_MODE:-auto}"
command_name="${1:-up}"

if [[ -z "${DATABASE_URL:-}" ]]; then
  printf 'DATABASE_URL is required\n' >&2
  exit 1
fi

psql_cmd() {
  if [[ "${POSTGRES_TOOLS_MODE}" != "docker" ]] && command -v psql >/dev/null 2>&1; then
    psql "$@"
    return
  fi

  docker run --rm -i --network host "${POSTGRES_TOOLS_IMAGE}" psql "$@"
}

psql_query() {
  psql_cmd "${DATABASE_URL}" -v ON_ERROR_STOP=1 -Atqc "$1" < /dev/null
}

ensure_migrations_table() {
psql_cmd "${DATABASE_URL}" -v ON_ERROR_STOP=1 -q <<'SQL'
SET client_min_messages TO warning;
CREATE TABLE IF NOT EXISTS schema_migrations (
  version TEXT PRIMARY KEY,
  checksum TEXT NOT NULL,
  applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
RESET client_min_messages;
SQL
}

validate_version() {
  local version="$1"
  if [[ ! "${version}" =~ ^[0-9A-Za-z_.-]+\.sql$ ]]; then
    printf 'invalid migration filename: %s\n' "${version}" >&2
    exit 1
  fi
}

list_migrations() {
  shopt -s nullglob
  local migrations=("${MIGRATIONS_DIR}"/*.sql)
  if [[ "${#migrations[@]}" -eq 0 ]]; then
    printf 'no migration files found in %s\n' "${MIGRATIONS_DIR}" >&2
    exit 1
  fi

  printf '%s\n' "${migrations[@]}"
}

apply_migrations() {
  ensure_migrations_table

  while IFS= read -r migration; do
    local version checksum applied_checksum
    version="$(basename "${migration}")"
    validate_version "${version}"
    checksum="$(sha256sum "${migration}" | cut -d ' ' -f1)"
    applied_checksum="$(psql_query "SELECT checksum FROM schema_migrations WHERE version = '${version}'")"

    if [[ -n "${applied_checksum}" ]]; then
      if [[ "${applied_checksum}" != "${checksum}" ]]; then
        printf 'migration checksum mismatch for %s\n' "${version}" >&2
        exit 1
      fi

      printf 'skipping already applied migration %s\n' "${version}"
      continue
    fi

    printf 'applying migration %s\n' "${version}"
    {
      printf 'BEGIN;\n'
      cat "${migration}"
      printf '\nINSERT INTO schema_migrations (version, checksum) VALUES ('"'"'%s'"'"', '"'"'%s'"'"');\n' "${version}" "${checksum}"
      printf 'COMMIT;\n'
    } | psql_cmd "${DATABASE_URL}" -v ON_ERROR_STOP=1 -q
  done < <(list_migrations)
}

print_status() {
  ensure_migrations_table

  while IFS= read -r migration; do
    local version checksum applied_checksum state
    version="$(basename "${migration}")"
    validate_version "${version}"
    checksum="$(sha256sum "${migration}" | cut -d ' ' -f1)"
    applied_checksum="$(psql_query "SELECT checksum FROM schema_migrations WHERE version = '${version}'")"
    state="pending"
    if [[ "${applied_checksum}" == "${checksum}" ]]; then
      state="applied"
    elif [[ -n "${applied_checksum}" ]]; then
      state="checksum_mismatch"
    fi

    printf '%s %s\n' "${version}" "${state}"
  done < <(list_migrations)
}

case "${command_name}" in
  up)
    apply_migrations
    ;;
  status)
    print_status
    ;;
  *)
    printf 'usage: DATABASE_URL=... %s [up|status]\n' "$0" >&2
    exit 1
    ;;
esac
