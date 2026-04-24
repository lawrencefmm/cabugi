#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
POSTGRES_IMAGE="${POSTGRES_IMAGE:-postgres:17-alpine}"
MINIO_IMAGE="${MINIO_IMAGE:-minio/minio:RELEASE.2025-04-22T22-12-26Z}"
postgres_container="cabugi-api-smoke-postgres-${RANDOM}-$$"
minio_container="cabugi-api-smoke-minio-${RANDOM}-$$"
api_log_file="$(mktemp)"
api_pid=""

cleanup() {
  if [[ -n "${api_pid}" ]]; then
    kill "${api_pid}" >/dev/null 2>&1 || true
    wait "${api_pid}" >/dev/null 2>&1 || true
  fi
  docker rm -f "${postgres_container}" "${minio_container}" >/dev/null 2>&1 || true
  rm -f "${api_log_file}"
}
trap cleanup EXIT

random_port() {
  python3 - <<'PY'
import socket

sock = socket.socket()
sock.bind(("127.0.0.1", 0))
print(sock.getsockname()[1])
sock.close()
PY
}

assert_contains() {
  local haystack="$1"
  local needle="$2"
  local message="$3"

  if [[ "${haystack}" != *"${needle}"* ]]; then
    printf 'verification failed: %s\n' "${message}" >&2
    printf 'response body: %s\n' "${haystack}" >&2
    exit 1
  fi
}

wait_for_http() {
  local url="$1"
  local description="$2"

  for _ in $(seq 1 60); do
    if curl -fsS "${url}" >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done

  printf 'timed out waiting for %s at %s\n' "${description}" "${url}" >&2
  if [[ -s "${api_log_file}" ]]; then
    printf 'api log:\n' >&2
    sed -n '1,200p' "${api_log_file}" >&2
  fi
  exit 1
}

docker run -d \
  --name "${postgres_container}" \
  -e POSTGRES_DB=cabugi \
  -e POSTGRES_USER=cabugi \
  -e POSTGRES_PASSWORD=cabugi \
  -p 127.0.0.1::5432 \
  "${POSTGRES_IMAGE}" >/dev/null

docker run -d \
  --name "${minio_container}" \
  -e MINIO_ROOT_USER=minioadmin \
  -e MINIO_ROOT_PASSWORD=minioadmin \
  -p 127.0.0.1::9000 \
  "${MINIO_IMAGE}" server /data >/dev/null

until docker exec "${postgres_container}" pg_isready -U cabugi -d cabugi >/dev/null 2>&1; do
  sleep 1
done

postgres_port_line="$(docker port "${postgres_container}" 5432/tcp)"
postgres_port="${postgres_port_line##*:}"
minio_port_line="$(docker port "${minio_container}" 9000/tcp)"
minio_port="${minio_port_line##*:}"

wait_for_http "http://127.0.0.1:${minio_port}/minio/health/live" "MinIO"

database_url="postgres://cabugi:cabugi@127.0.0.1:${postgres_port}/cabugi?sslmode=disable"
object_storage_endpoint="http://127.0.0.1:${minio_port}"
api_port="$(random_port)"
api_address="127.0.0.1:${api_port}"
api_base_url="http://${api_address}"

DATABASE_URL="${database_url}" POSTGRES_TOOLS_MODE=docker "${ROOT_DIR}/db/scripts/migrate.sh" up >/dev/null

(
  cd "${ROOT_DIR}/services/api"
  DATABASE_URL="${database_url}" \
    OBJECT_STORAGE_ENDPOINT="${object_storage_endpoint}" \
    OBJECT_STORAGE_REGION="us-east-1" \
    OBJECT_STORAGE_BUCKET="cabugi-hidden-tests" \
    OBJECT_STORAGE_ACCESS_KEY_ID="minioadmin" \
    OBJECT_STORAGE_SECRET_ACCESS_KEY="minioadmin" \
    OBJECT_STORAGE_USE_PATH_STYLE="true" \
    go run ./cmd/seed-starter-problems >/dev/null
)

(
  cd "${ROOT_DIR}/services/api"
  API_ADDRESS="${api_address}" \
    DATABASE_URL="${database_url}" \
    OBJECT_STORAGE_ENDPOINT="${object_storage_endpoint}" \
    OBJECT_STORAGE_REGION="us-east-1" \
    OBJECT_STORAGE_BUCKET="cabugi-hidden-tests" \
    OBJECT_STORAGE_ACCESS_KEY_ID="minioadmin" \
    OBJECT_STORAGE_SECRET_ACCESS_KEY="minioadmin" \
    OBJECT_STORAGE_USE_PATH_STYLE="true" \
    API_REQUIRE_DATABASE="true" \
    API_REQUIRE_HIDDEN_BUNDLE_VALIDATION="true" \
    API_REQUIRE_AUTH="false" \
    go run ./cmd/api >"${api_log_file}" 2>&1
) &
api_pid="$!"

wait_for_http "${api_base_url}/healthz" "API health"

problems_response="$(curl -fsS "${api_base_url}/v1/problems")"
problem_detail_response="$(curl -fsS "${api_base_url}/v1/problems/a-plus-b")"

assert_contains "${problems_response}" '"slug":"a-plus-b"' 'published problems smoke must include seeded a-plus-b problem'
assert_contains "${problems_response}" '"slug":"reverse-string"' 'published problems smoke must include seeded reverse-string problem'
assert_contains "${problem_detail_response}" '"slug":"a-plus-b"' 'problem detail smoke must return seeded problem detail'
assert_contains "${problem_detail_response}" '"title":"A + B"' 'problem detail smoke must return seeded problem title'

printf 'api runtime smoke verification passed\n'
