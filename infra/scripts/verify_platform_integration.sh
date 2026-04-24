#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
POSTGRES_IMAGE="${POSTGRES_IMAGE:-postgres:17-alpine}"
MINIO_IMAGE="${MINIO_IMAGE:-minio/minio:RELEASE.2025-04-22T22-12-26Z}"
JUDGE_CPP_IMAGE="${JUDGE_CPP_IMAGE:-gcc:14.2.0}"
JUDGE_PYTHON_IMAGE="${JUDGE_PYTHON_IMAGE:-python:3.13.0-alpine3.20}"
LOCAL_TEST_AUTH_ISSUER="https://cabugi.local.test"
LOCAL_TEST_AUTH_AUDIENCE="cabugi-local-test"
LOCAL_TEST_AUTH_DIR="${ROOT_DIR}/infra/testdata/local_test_auth"
postgres_container="cabugi-integration-postgres-${RANDOM}-$$"
minio_container="cabugi-integration-minio-${RANDOM}-$$"
api_log_file="$(mktemp)"
judge_log_file="$(mktemp)"
jwks_log_file="$(mktemp)"
bundle_file="$(mktemp)"
api_pid=""
judge_pid=""
jwks_pid=""

cleanup() {
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
  rm -f "${api_log_file}" "${judge_log_file}" "${jwks_log_file}" "${bundle_file}"
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

json_field() {
  local file="$1"
  local field_path="$2"

  python3 - "$file" "$field_path" <<'PY'
import json
import sys

with open(sys.argv[1], encoding="utf-8") as handle:
    value = json.load(handle)

for part in sys.argv[2].split("."):
    if isinstance(value, list):
        value = value[int(part)]
    else:
        value = value[part]

if isinstance(value, (dict, list)):
    print(json.dumps(value))
else:
    print(value)
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

assert_not_contains() {
  local haystack="$1"
  local needle="$2"
  local message="$3"

  if [[ "${haystack}" == *"${needle}"* ]]; then
    printf 'verification failed: %s\n' "${message}" >&2
    printf 'response body: %s\n' "${haystack}" >&2
    exit 1
  fi
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

request_json() {
  local method="$1"
  local path="$2"
  local token="$3"
  local body="$4"
  local content_type="$5"
  local response_file="$6"
  local status

  local args=(curl -sS -o "${response_file}" -w "%{http_code}" -X "${method}" -H "Accept: application/json")
  if [[ -n "${token}" ]]; then
    args+=( -H "Authorization: Bearer ${token}" )
  fi
  if [[ -n "${content_type}" ]]; then
    args+=( -H "Content-Type: ${content_type}" )
  fi
  if [[ -n "${body}" ]]; then
    args+=( --data "${body}" )
  fi
  args+=( "${api_base_url}${path}" )

  status="$("${args[@]}")"
  printf '%s' "${status}"
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
jwks_port="$(random_port)"
api_address="127.0.0.1:${api_port}"
judge_observability_address="127.0.0.1:${judge_observability_port}"
api_base_url="http://${api_address}"
jwks_url="http://127.0.0.1:${jwks_port}/jwks.json"
draft_slug="integration-mirror-number-${RANDOM}-$$"
draft_title="Integration Mirror Number"
accepted_source=$'def main() -> None:\n    import sys\n\n    value = sys.stdin.read().strip()\n    print(value)\n\n\nif __name__ == "__main__":\n    main()\n'

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
    WEB_ALLOWED_ORIGINS="http://127.0.0.1:3000" \
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

python3 - >"${bundle_file}" <<'PY'
import json
import sys

json.dump({
    "cases": [
        {"input": "7\n", "expectedOutput": "7\n"},
        {"input": "42\n", "expectedOutput": "42\n"},
    ]
}, sys.stdout)
PY

public_before="$(curl -fsS "${api_base_url}/v1/problems")"
assert_not_contains "${public_before}" "\"slug\":\"${draft_slug}\"" "draft slug must not be visible in the public list before moderation"

upload_response="$(mktemp)"
upload_status="$(curl -sS -o "${upload_response}" -w "%{http_code}" -H "Accept: application/json" -H "Authorization: Bearer ${author_token}" -F "bundle=@${bundle_file};type=application/json" "${api_base_url}/v1/problem-drafts/hidden-test-bundles")"
if [[ "${upload_status}" != "201" ]]; then
  printf 'hidden bundle upload failed with status %s\n' "${upload_status}" >&2
  printf 'response body: %s\n' "$(<"${upload_response}")" >&2
  exit 1
fi

bundle_key="$(json_field "${upload_response}" "hiddenTestBundleKey")"
bundle_sha="$(json_field "${upload_response}" "hiddenTestBundleSha256")"
rm -f "${upload_response}"

create_body="$(python3 - "$draft_slug" "$draft_title" "$bundle_key" "$bundle_sha" <<'PY'
import json
import sys

payload = {
    "slug": sys.argv[1],
    "title": sys.argv[2],
    "statementMarkdown": "Read one integer and print the same integer.",
    "inputMarkdown": "One integer n.",
    "outputMarkdown": "Print n.",
    "constraintsMarkdown": "|n| <= 10^9",
    "notesMarkdown": "Integration smoke draft.",
    "timeLimitMs": 5000,
    "memoryLimitMb": 256,
    "hiddenTestBundleKey": sys.argv[3],
    "hiddenTestBundleSha256": sys.argv[4],
}
print(json.dumps(payload))
PY
)"

create_response="$(mktemp)"
create_status="$(request_json "POST" "/v1/problem-drafts" "${author_token}" "${create_body}" "application/json" "${create_response}")"
if [[ "${create_status}" != "201" ]]; then
  printf 'draft creation failed with status %s\n' "${create_status}" >&2
  printf 'response body: %s\n' "$(<"${create_response}")" >&2
  print_log "api" "${api_log_file}"
  exit 1
fi
rm -f "${create_response}"

submit_response="$(mktemp)"
submit_status="$(request_json "POST" "/v1/problem-drafts/${draft_slug}/submit-for-review" "${author_token}" "" "" "${submit_response}")"
if [[ "${submit_status}" != "200" ]]; then
  printf 'submit-for-review failed with status %s\n' "${submit_status}" >&2
  printf 'response body: %s\n' "$(<"${submit_response}")" >&2
  exit 1
fi
rm -f "${submit_response}"

public_in_review="$(curl -fsS "${api_base_url}/v1/problems")"
assert_not_contains "${public_in_review}" "\"slug\":\"${draft_slug}\"" "draft slug must stay hidden from the public list while it is still in review"

approve_body='{"decision":"approve","moderationNotes":"integration smoke approval"}'
approve_response="$(mktemp)"
approve_status="$(request_json "POST" "/v1/moderation/problem-drafts/${draft_slug}/decision" "${moderator_token}" "${approve_body}" "application/json" "${approve_response}")"
if [[ "${approve_status}" != "200" ]]; then
  printf 'moderation approval failed with status %s\n' "${approve_status}" >&2
  printf 'response body: %s\n' "$(<"${approve_response}")" >&2
  exit 1
fi
rm -f "${approve_response}"

public_after="$(curl -fsS "${api_base_url}/v1/problems")"
assert_contains "${public_after}" "\"slug\":\"${draft_slug}\"" "approved draft must become visible in the public problem list"
assert_contains "${public_after}" "\"title\":\"${draft_title}\"" "approved draft must expose the published title"

submission_body="$(python3 - "$draft_slug" "$accepted_source" <<'PY'
import json
import sys

print(json.dumps({
    "problemSlug": sys.argv[1],
    "language": "python",
    "sourceCode": sys.argv[2],
}))
PY
)"

submission_response="$(mktemp)"
submission_status="$(request_json "POST" "/v1/submissions" "${author_token}" "${submission_body}" "application/json" "${submission_response}")"
if [[ "${submission_status}" != "201" ]]; then
  printf 'submission create failed with status %s\n' "${submission_status}" >&2
  printf 'response body: %s\n' "$(<"${submission_response}")" >&2
  exit 1
fi

submission_id="$(json_field "${submission_response}" "id")"
rm -f "${submission_response}"

terminal_status=""
submission_detail="$(mktemp)"
for _ in $(seq 1 90); do
  detail_status="$(request_json "GET" "/v1/submissions/${submission_id}" "${author_token}" "" "" "${submission_detail}")"
  if [[ "${detail_status}" != "200" ]]; then
    printf 'submission detail read failed with status %s\n' "${detail_status}" >&2
    printf 'response body: %s\n' "$(<"${submission_detail}")" >&2
    exit 1
  fi

  terminal_status="$(json_field "${submission_detail}" "status")"
  if [[ "${terminal_status}" == "accepted" || "${terminal_status}" == "wrong_answer" || "${terminal_status}" == "compile_error" || "${terminal_status}" == "runtime_error" || "${terminal_status}" == "time_limit_exceeded" || "${terminal_status}" == "judge_failed" ]]; then
    break
  fi
  sleep 1
done

if [[ "${terminal_status}" != "accepted" ]]; then
  printf 'expected accepted submission, got %s\n' "${terminal_status}" >&2
  printf 'response body: %s\n' "$(<"${submission_detail}")" >&2
  print_log "judge" "${judge_log_file}"
  exit 1
fi

passed_tests="$(json_field "${submission_detail}" "passedTests")"
total_tests="$(json_field "${submission_detail}" "totalTests")"
first_result_verdict="$(json_field "${submission_detail}" "results.0.verdict")"
if [[ "${passed_tests}" != "${total_tests}" || "${first_result_verdict}" != "accepted" ]]; then
  printf 'submission detail did not record accepted per-test results\n' >&2
  printf 'response body: %s\n' "$(<"${submission_detail}")" >&2
  exit 1
fi

rm -f "${submission_detail}"

printf 'platform integration verification passed\n'
