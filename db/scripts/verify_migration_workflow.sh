#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
POSTGRES_IMAGE="${POSTGRES_IMAGE:-postgres:17-alpine}"
container_name="cabugi-migration-verify-${RANDOM}-$$"
temp_db="cabugi_migration_verify"
backup_dir=""

cleanup() {
  docker rm -f "${container_name}" >/dev/null 2>&1 || true
  if [[ -n "${backup_dir}" ]]; then
    rm -rf "${backup_dir}"
  fi
}
trap cleanup EXIT

docker run -d \
  --name "${container_name}" \
  -e POSTGRES_DB=cabugi \
  -e POSTGRES_USER=cabugi \
  -e POSTGRES_PASSWORD=cabugi \
  -p 127.0.0.1::5432 \
  "${POSTGRES_IMAGE}" >/dev/null

until docker exec "${container_name}" pg_isready -U cabugi -d cabugi >/dev/null 2>&1; do
  sleep 1
done

until docker exec -i "${container_name}" psql -U cabugi -d cabugi -v ON_ERROR_STOP=1 -Atqc "SELECT 1" >/dev/null 2>&1; do
  sleep 1
done

port_line="$(docker port "${container_name}" 5432/tcp)"
host_port="${port_line##*:}"

docker exec -i "${container_name}" createdb -U cabugi "${temp_db}" >/dev/null

database_url="postgres://cabugi:cabugi@127.0.0.1:${host_port}/${temp_db}?sslmode=disable"
expected_migrations="$(find "${ROOT_DIR}/db/migrations" -maxdepth 1 -name '*.sql' | wc -l | tr -d ' ')"

DATABASE_URL="${database_url}" POSTGRES_TOOLS_MODE=docker "${ROOT_DIR}/db/scripts/migrate.sh" up >/dev/null
applied_migrations="$(docker exec -i "${container_name}" psql -U cabugi -d "${temp_db}" -v ON_ERROR_STOP=1 -Atqc "SELECT COUNT(*) FROM schema_migrations")"
if [[ "${applied_migrations}" != "${expected_migrations}" ]]; then
  printf 'expected %s applied migrations, got %s\n' "${expected_migrations}" "${applied_migrations}" >&2
  exit 1
fi

applied_status_count="$(DATABASE_URL="${database_url}" POSTGRES_TOOLS_MODE=docker "${ROOT_DIR}/db/scripts/migrate.sh" status | grep -c ' applied$')"
if [[ "${applied_status_count}" != "${expected_migrations}" ]]; then
  printf 'migration status did not report all migrations as applied\n' >&2
  exit 1
fi

DATABASE_URL="${database_url}" POSTGRES_TOOLS_MODE=docker "${ROOT_DIR}/db/scripts/migrate.sh" up >/dev/null
applied_after_second_run="$(docker exec -i "${container_name}" psql -U cabugi -d "${temp_db}" -v ON_ERROR_STOP=1 -Atqc "SELECT COUNT(*) FROM schema_migrations")"
if [[ "${applied_after_second_run}" != "${expected_migrations}" ]]; then
  printf 'migration runner is not idempotent\n' >&2
  exit 1
fi

docker exec -i "${container_name}" psql -U cabugi -d "${temp_db}" -v ON_ERROR_STOP=1 -q <<'SQL'
INSERT INTO users (auth_subject, handle, display_name)
VALUES ('verify_subject', 'verify_handle', 'Verify User');
SQL

backup_dir="$(mktemp -d)"
backup_path="$(DATABASE_URL="${database_url}" BACKUP_DIR="${backup_dir}" POSTGRES_TOOLS_MODE=docker "${ROOT_DIR}/db/scripts/backup.sh")"

docker exec -i "${container_name}" dropdb -U cabugi "${temp_db}" >/dev/null
docker exec -i "${container_name}" createdb -U cabugi "${temp_db}" >/dev/null
CONFIRM_RESTORE=yes DATABASE_URL="${database_url}" POSTGRES_TOOLS_MODE=docker "${ROOT_DIR}/db/scripts/restore.sh" "${backup_path}" >/dev/null

restored_users="$(docker exec -i "${container_name}" psql -U cabugi -d "${temp_db}" -v ON_ERROR_STOP=1 -Atqc "SELECT COUNT(*) FROM users WHERE auth_subject = 'verify_subject'")"
if [[ "${restored_users}" != "1" ]]; then
  printf 'backup restore did not preserve verification user\n' >&2
  exit 1
fi

restored_migrations="$(docker exec -i "${container_name}" psql -U cabugi -d "${temp_db}" -v ON_ERROR_STOP=1 -Atqc "SELECT COUNT(*) FROM schema_migrations")"
if [[ "${restored_migrations}" != "${expected_migrations}" ]]; then
  printf 'backup restore did not preserve migration history\n' >&2
  exit 1
fi

printf 'migration workflow verification passed\n'
