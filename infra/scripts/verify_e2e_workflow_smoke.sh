#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
POSTGRES_IMAGE="${POSTGRES_IMAGE:-postgres:17-alpine}"
MINIO_IMAGE="${MINIO_IMAGE:-minio/minio:RELEASE.2025-04-22T22-12-26Z}"
JUDGE_CPP_IMAGE="${JUDGE_CPP_IMAGE:-gcc:14.2.0}"
JUDGE_PYTHON_IMAGE="${JUDGE_PYTHON_IMAGE:-python:3.13.0-alpine3.20}"
PLAYWRIGHT_DOCKER_IMAGE="${PLAYWRIGHT_DOCKER_IMAGE:-mcr.microsoft.com/playwright:v1.59.1-noble}"
LOCAL_TEST_AUTH_ISSUER="https://cabugi.local.test"
LOCAL_TEST_AUTH_AUDIENCE="cabugi-local-test"
LOCAL_TEST_AUTH_DIR="${ROOT_DIR}/infra/testdata/local_test_auth"
postgres_container="cabugi-e2e-postgres-${RANDOM}-$$"
minio_container="cabugi-e2e-minio-${RANDOM}-$$"
api_log_file="$(mktemp)"
judge_log_file="$(mktemp)"
web_log_file="$(mktemp)"
jwks_log_file="$(mktemp)"
api_pid=""
judge_pid=""
web_pid=""
jwks_pid=""

cleanup() {
  if [[ -n "${web_pid}" ]]; then
    kill "${web_pid}" >/dev/null 2>&1 || true
    wait "${web_pid}" >/dev/null 2>&1 || true
  fi
  if [[ -n "${judge_pid}" ]]; then
    kill "${judge_pid}" >/dev/null 2>&1 || true
    wait "${judge_pid}" >/dev/null 2>&1 || true
  fi
  if [[ -n "${api_pid}" ]]; then
    kill "${api_pid}" >/dev/null 2>&1 || true
    wait "${api_pid}" >/dev/null 2>&1 || true
  fi
  if [[ -n "${jwks_pid}" ]]; then
    kill "${jwks_pid}" >/dev/null 2>&1 || true
    wait "${jwks_pid}" >/dev/null 2>&1 || true
  fi
  docker rm -f "${postgres_container}" "${minio_container}" >/dev/null 2>&1 || true
  rm -f "${api_log_file}" "${judge_log_file}" "${web_log_file}" "${jwks_log_file}"
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

print_log() {
  local label="$1"
  local path="$2"

  if [[ -s "${path}" ]]; then
    printf '%s log:\n' "${label}" >&2
    sed -n '1,200p' "${path}" >&2
  fi
}

wait_for_http() {
  local url="$1"
  local description="$2"
  local log_file="$3"

  for _ in $(seq 1 90); do
    if curl -fsS "${url}" >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done

  printf 'timed out waiting for %s at %s\n' "${description}" "${url}" >&2
  print_log "service" "${log_file}"
  exit 1
}

bootstrap_user() {
  local token="$1"
  local label="$2"
  local body_file
  local status

  body_file="$(mktemp)"
  status="$(curl -sS -o "${body_file}" -w "%{http_code}" -H "Authorization: Bearer ${token}" "${api_base_url}/v1/me")"
  if [[ "${status}" != "200" ]]; then
    printf 'failed to bootstrap %s user via /v1/me: status %s\n' "${label}" "${status}" >&2
    printf 'response body: %s\n' "$(<"${body_file}")" >&2
    print_log "api" "${api_log_file}"
    rm -f "${body_file}"
    exit 1
  fi

  rm -f "${body_file}"
}

author_token="$(<"${LOCAL_TEST_AUTH_DIR}/author-token.txt")"
moderator_token="$(<"${LOCAL_TEST_AUTH_DIR}/moderator-token.txt")"

docker pull "${JUDGE_CPP_IMAGE}" >/dev/null
docker pull "${JUDGE_PYTHON_IMAGE}" >/dev/null

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

wait_for_http "http://127.0.0.1:${minio_port}/minio/health/live" "MinIO" "${judge_log_file}"

database_url="postgres://cabugi:cabugi@127.0.0.1:${postgres_port}/cabugi?sslmode=disable"
object_storage_endpoint="http://127.0.0.1:${minio_port}"
api_port="$(random_port)"
judge_observability_port="$(random_port)"
web_port="$(random_port)"
jwks_port="$(random_port)"
api_address="127.0.0.1:${api_port}"
judge_observability_address="127.0.0.1:${judge_observability_port}"
web_base_url="http://127.0.0.1:${web_port}"
api_base_url="http://${api_address}"
jwks_url="http://127.0.0.1:${jwks_port}/jwks.json"

DATABASE_URL="${database_url}" POSTGRES_TOOLS_MODE=docker "${ROOT_DIR}/db/scripts/migrate.sh" up >/dev/null

(
  cd "${LOCAL_TEST_AUTH_DIR}"
  python3 -m http.server "${jwks_port}" --bind 127.0.0.1 >"${jwks_log_file}" 2>&1
) &
jwks_pid="$!"

wait_for_http "${jwks_url}" "JWKS fixture" "${jwks_log_file}"

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
    WEB_ALLOWED_ORIGINS="${web_base_url}" \
    CLERK_ISSUER="${LOCAL_TEST_AUTH_ISSUER}" \
    CLERK_JWKS_URL="${jwks_url}" \
    CLERK_ALLOWED_PARTIES="" \
    CLERK_ALLOWED_AUDIENCES="${LOCAL_TEST_AUTH_AUDIENCE}" \
    CLERK_PEM_PUBLIC_KEY="" \
    OBJECT_STORAGE_ENDPOINT="${object_storage_endpoint}" \
    OBJECT_STORAGE_REGION="us-east-1" \
    OBJECT_STORAGE_BUCKET="cabugi-hidden-tests" \
    OBJECT_STORAGE_ACCESS_KEY_ID="minioadmin" \
    OBJECT_STORAGE_SECRET_ACCESS_KEY="minioadmin" \
    OBJECT_STORAGE_USE_PATH_STYLE="true" \
    API_REQUIRE_DATABASE="true" \
    API_REQUIRE_HIDDEN_BUNDLE_VALIDATION="true" \
    API_REQUIRE_AUTH="true" \
    go run ./cmd/api >"${api_log_file}" 2>&1
) &
api_pid="$!"

(
  cd "${ROOT_DIR}/services/judge"
  DATABASE_URL="${database_url}" \
    OBJECT_STORAGE_ENDPOINT="${object_storage_endpoint}" \
    OBJECT_STORAGE_REGION="us-east-1" \
    OBJECT_STORAGE_BUCKET="cabugi-hidden-tests" \
    OBJECT_STORAGE_ACCESS_KEY_ID="minioadmin" \
    OBJECT_STORAGE_SECRET_ACCESS_KEY="minioadmin" \
    OBJECT_STORAGE_USE_PATH_STYLE="true" \
    JUDGE_OBSERVABILITY_ADDRESS="${judge_observability_address}" \
    JUDGE_POLL_INTERVAL="250ms" \
    JUDGE_RETRY_DELAY="500ms" \
    JUDGE_JOB_LEASE_DURATION="10s" \
    JUDGE_JOB_LEASE_RENEW_INTERVAL="2s" \
    go run ./cmd/judge >"${judge_log_file}" 2>&1
) &
judge_pid="$!"

wait_for_http "${api_base_url}/healthz" "API health" "${api_log_file}"
wait_for_http "http://${judge_observability_address}/healthz" "judge health" "${judge_log_file}"

bootstrap_user "${author_token}" "author"
bootstrap_user "${moderator_token}" "moderator"

docker exec -e PGPASSWORD=cabugi "${postgres_container}" psql -U cabugi -d cabugi -c "INSERT INTO user_roles (user_id, role) SELECT id, 'moderator' FROM users WHERE auth_subject = 'user_e2e_moderator' ON CONFLICT (user_id, role) DO NOTHING" >/dev/null

(
  cd "${ROOT_DIR}"
  NEXT_PUBLIC_API_BASE_URL="${api_base_url}" \
    NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY="" \
    NEXT_PUBLIC_LOCAL_TEST_AUTH_ENABLED="true" \
    pnpm --filter web build >/dev/null
)

(
  cd "${ROOT_DIR}"
  NEXT_PUBLIC_API_BASE_URL="${api_base_url}" \
    NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY="" \
    NEXT_PUBLIC_LOCAL_TEST_AUTH_ENABLED="true" \
    pnpm --filter web exec next start -H 127.0.0.1 -p "${web_port}" >"${web_log_file}" 2>&1
) &
web_pid="$!"

wait_for_http "${web_base_url}" "web app" "${web_log_file}"

if ! docker run --rm \
  --network host \
  --ipc host \
  -u "$(id -u):$(id -g)" \
  -v "${ROOT_DIR}:/work" \
  -w /work \
  -e E2E_API_BASE_URL="${api_base_url}" \
  -e E2E_WEB_BASE_URL="${web_base_url}" \
  -e E2E_AUTHOR_TOKEN="${author_token}" \
  -e E2E_MODERATOR_TOKEN="${moderator_token}" \
  "${PLAYWRIGHT_DOCKER_IMAGE}" \
  node /work/infra/scripts/run_e2e_smoke.mjs; then
  print_log "api" "${api_log_file}"
  print_log "judge" "${judge_log_file}"
  print_log "web" "${web_log_file}"
  exit 1
fi

printf 'end-to-end workflow smoke verification passed\n'
