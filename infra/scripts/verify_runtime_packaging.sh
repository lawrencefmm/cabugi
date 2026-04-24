#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
COMPOSE_FILE="${ROOT_DIR}/infra/docker-compose.yml"

required_files=(
  "apps/web/Dockerfile"
  "services/api/Dockerfile"
  "services/judge/Dockerfile"
  "infra/docker-compose.yml"
  "infra/full-stack.env.example"
)

for file in "${required_files[@]}"; do
  if [[ ! -f "${ROOT_DIR}/${file}" ]]; then
    printf 'missing required packaging file: %s\n' "${file}" >&2
    exit 1
  fi
done

required_services=(web api judge postgres minio)
for service in "${required_services[@]}"; do
  if ! grep -Eq "^[[:space:]]{2}${service}:$" "${COMPOSE_FILE}"; then
    printf 'compose file is missing service: %s\n' "${service}" >&2
    exit 1
  fi
done

required_runtime_settings=(
  "DATABASE_URL"
  "NEXT_PUBLIC_API_BASE_URL"
  "CLERK_PEM_PUBLIC_KEY"
  "OBJECT_STORAGE_ENDPOINT"
  "OBJECT_STORAGE_BUCKET"
  "OBJECT_STORAGE_SECRET_ACCESS_KEY"
  "JUDGE_OBSERVABILITY_ADDRESS"
  "/var/run/docker.sock:/var/run/docker.sock"
)

for setting in "${required_runtime_settings[@]}"; do
  if ! grep -Fq "${setting}" "${COMPOSE_FILE}"; then
    printf 'compose file is missing runtime setting: %s\n' "${setting}" >&2
    exit 1
  fi
done

if ! grep -Fq '"start": "next start -H 0.0.0.0"' "${ROOT_DIR}/apps/web/package.json"; then
  printf 'web package is missing production start script\n' >&2
  exit 1
fi

docker compose -f "${COMPOSE_FILE}" --env-file "${ROOT_DIR}/infra/full-stack.env.example" config >/dev/null

printf 'runtime packaging files verified\n'
