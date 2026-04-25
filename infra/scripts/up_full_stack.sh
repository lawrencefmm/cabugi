#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
COMPOSE_FILE="${ROOT_DIR}/infra/docker-compose.yml"
DEFAULT_ENV_FILE="${ROOT_DIR}/infra/full-stack.env.example"
LOCAL_ENV_FILE="${ROOT_DIR}/infra/full-stack.env"

CHECK_ONLY=false
ENV_FILE=""

usage() {
  cat <<'EOF'
Usage: ./infra/scripts/up_full_stack.sh [--check] [--env-file PATH]

Starts the local full-stack Compose runtime after validating published host ports.

Options:
  --check          Validate the selected env and exit without starting containers.
  --env-file PATH  Use a specific env file instead of the default local path.

Environment variable overrides still take precedence over the env file.
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --check)
      CHECK_ONLY=true
      ;;
    --env-file)
      shift
      if [[ $# -eq 0 ]]; then
        printf '%s\n' 'missing value for --env-file' >&2
        exit 1
      fi
      ENV_FILE="$1"
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      printf 'unknown argument: %s\n' "$1" >&2
      usage >&2
      exit 1
      ;;
  esac
  shift
done

if [[ -z "${ENV_FILE}" ]]; then
  if [[ -f "${LOCAL_ENV_FILE}" ]]; then
    ENV_FILE="${LOCAL_ENV_FILE}"
  else
    ENV_FILE="${DEFAULT_ENV_FILE}"
  fi
fi

if [[ "${ENV_FILE}" != /* ]]; then
  ENV_FILE="$(pwd)/${ENV_FILE}"
fi

if [[ ! -f "${ENV_FILE}" ]]; then
  printf 'env file not found: %s\n' "${ENV_FILE}" >&2
  exit 1
fi

load_env_defaults() {
  local line=""
  local key=""
  local value=""

  while IFS= read -r line || [[ -n "${line}" ]]; do
    if [[ -z "${line//[[:space:]]/}" ]] || [[ "${line}" =~ ^[[:space:]]*# ]]; then
      continue
    fi

    key="${line%%=*}"
    value="${line#*=}"
    key="${key#${key%%[![:space:]]*}}"
    key="${key%${key##*[![:space:]]}}"
    value="${value#${value%%[![:space:]]*}}"
    value="${value%${value##*[![:space:]]}}"

    if [[ -z "${key}" ]]; then
      continue
    fi

    if [[ -z "${!key+x}" ]]; then
      export "${key}=${value}"
    fi
  done <"${ENV_FILE}"
}

relative_path() {
  local path="$1"
  printf '%s\n' "${path#${ROOT_DIR}/}"
}

port_is_in_use() {
  local port="$1"

  if command -v ss >/dev/null 2>&1; then
    ss -H -ltn "sport = :${port}" 2>/dev/null | grep -q .
    return
  fi

  if command -v lsof >/dev/null 2>&1; then
    lsof -nP -iTCP:"${port}" -sTCP:LISTEN >/dev/null 2>&1
    return
  fi

  printf '%s\n' 'warning: skipping host port preflight because neither ss nor lsof is available' >&2
  return 1
}

suggest_port() {
  local port="$1"

  if (( port < 10000 )); then
    printf '1%s\n' "${port}"
    return
  fi

  printf '%s\n' "$((port + 1000))"
}

load_env_defaults

WEB_PORT="${WEB_PORT:-3000}"
API_PORT="${API_PORT:-8080}"
JUDGE_OBSERVABILITY_PORT="${JUDGE_OBSERVABILITY_PORT:-8082}"
POSTGRES_PORT="${POSTGRES_PORT:-5432}"
MINIO_PORT="${MINIO_PORT:-9000}"
MINIO_CONSOLE_PORT="${MINIO_CONSOLE_PORT:-9001}"
NEXT_PUBLIC_API_BASE_URL="${NEXT_PUBLIC_API_BASE_URL:-http://127.0.0.1:${API_PORT}}"

env_file_display="$(relative_path "${ENV_FILE}")"
local_env_display="$(relative_path "${LOCAL_ENV_FILE}")"

conflicts=()

check_port_conflict() {
  local variable_name="$1"
  local port="$2"
  local service_name="$3"
  local suggestion

  if ! port_is_in_use "${port}"; then
    return
  fi

  suggestion="$(suggest_port "${port}")"
  case "${variable_name}" in
    API_PORT)
      conflicts+=("- ${variable_name} publishes ${service_name} on host port ${port}, but that port is already in use. Try: API_PORT=${suggestion} NEXT_PUBLIC_API_BASE_URL=http://127.0.0.1:${suggestion} ./infra/scripts/up_full_stack.sh")
      ;;
    WEB_PORT)
      conflicts+=("- ${variable_name} publishes ${service_name} on host port ${port}, but that port is already in use. Try: WEB_PORT=${suggestion} WEB_ALLOWED_ORIGINS=http://127.0.0.1:${suggestion},http://localhost:${suggestion} ./infra/scripts/up_full_stack.sh")
      ;;
    *)
      conflicts+=("- ${variable_name} publishes ${service_name} on host port ${port}, but that port is already in use. Try: ${variable_name}=${suggestion} ./infra/scripts/up_full_stack.sh")
      ;;
  esac
}

check_port_conflict "WEB_PORT" "${WEB_PORT}" "web"
check_port_conflict "API_PORT" "${API_PORT}" "api"
check_port_conflict "JUDGE_OBSERVABILITY_PORT" "${JUDGE_OBSERVABILITY_PORT}" "judge observability"
check_port_conflict "POSTGRES_PORT" "${POSTGRES_PORT}" "postgres"
check_port_conflict "MINIO_PORT" "${MINIO_PORT}" "minio api"
check_port_conflict "MINIO_CONSOLE_PORT" "${MINIO_CONSOLE_PORT}" "minio console"

if [[ "${NEXT_PUBLIC_API_BASE_URL}" != "http://127.0.0.1:${API_PORT}" && "${NEXT_PUBLIC_API_BASE_URL}" != "http://localhost:${API_PORT}" ]]; then
  conflicts+=("- NEXT_PUBLIC_API_BASE_URL=${NEXT_PUBLIC_API_BASE_URL} does not match API_PORT=${API_PORT}. Keep them aligned, for example: API_PORT=${API_PORT} NEXT_PUBLIC_API_BASE_URL=http://127.0.0.1:${API_PORT} ./infra/scripts/up_full_stack.sh")
fi

if (( ${#conflicts[@]} > 0 )); then
  printf 'local full-stack startup is blocked by env or host-port issues in %s\n' "${env_file_display}" >&2
  printf '%s\n' >&2
  printf '%s\n' "${conflicts[@]}" >&2
  printf '%s\n' >&2
  printf 'Set overrides in your shell for one run, or create %s for local-only values and rerun.\n' "${local_env_display}" >&2
  exit 1
fi

if [[ "${CHECK_ONLY}" == "true" ]]; then
  printf 'local full-stack env verified: %s\n' "${env_file_display}"
  exit 0
fi

exec docker compose -f "${COMPOSE_FILE}" --env-file "${ENV_FILE}" up --build -d
