#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
BACKUP_DIR="${BACKUP_DIR:-${ROOT_DIR}/db/backups}"
POSTGRES_TOOLS_IMAGE="${POSTGRES_TOOLS_IMAGE:-postgres:17-alpine}"

if [[ -z "${DATABASE_URL:-}" ]]; then
  printf 'DATABASE_URL is required\n' >&2
  exit 1
fi

mkdir -p "${BACKUP_DIR}"
backup_dir_abs="$(cd "${BACKUP_DIR}" && pwd)"
backup_name="cabugi-$(date -u +%Y%m%dT%H%M%SZ).dump"
backup_path="${backup_dir_abs}/${backup_name}"

if command -v pg_dump >/dev/null 2>&1; then
  pg_dump --dbname "${DATABASE_URL}" --format=custom --no-owner --no-acl --file "${backup_path}"
else
  docker run --rm --network host -v "${backup_dir_abs}:/backup" "${POSTGRES_TOOLS_IMAGE}" \
    pg_dump --dbname "${DATABASE_URL}" --format=custom --no-owner --no-acl --file "/backup/${backup_name}"
fi

printf '%s\n' "${backup_path}"
