#!/usr/bin/env bash
set -euo pipefail

POSTGRES_TOOLS_IMAGE="${POSTGRES_TOOLS_IMAGE:-postgres:17-alpine}"
backup_path="${1:-}"

if [[ -z "${DATABASE_URL:-}" ]]; then
  printf 'DATABASE_URL is required\n' >&2
  exit 1
fi

if [[ -z "${backup_path}" || ! -f "${backup_path}" ]]; then
  printf 'usage: CONFIRM_RESTORE=yes DATABASE_URL=... %s <backup.dump>\n' "$0" >&2
  exit 1
fi

if [[ "${CONFIRM_RESTORE:-}" != "yes" ]]; then
  printf 'restore is destructive; set CONFIRM_RESTORE=yes to continue\n' >&2
  exit 1
fi

backup_dir="$(cd "$(dirname "${backup_path}")" && pwd)"
backup_name="$(basename "${backup_path}")"

if command -v pg_restore >/dev/null 2>&1; then
  pg_restore --dbname "${DATABASE_URL}" --clean --if-exists --no-owner --no-acl --single-transaction "${backup_dir}/${backup_name}"
else
  docker run --rm --network host -v "${backup_dir}:/backup:ro" "${POSTGRES_TOOLS_IMAGE}" \
    pg_restore --dbname "${DATABASE_URL}" --clean --if-exists --no-owner --no-acl --single-transaction "/backup/${backup_name}"
fi

printf 'restore completed from %s\n' "${backup_dir}/${backup_name}"
