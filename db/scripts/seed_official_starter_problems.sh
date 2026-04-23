#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "$0")/../.." && pwd)"
compose_file="$repo_root/infra/docker-compose.yml"
object_storage_endpoint="${OBJECT_STORAGE_ENDPOINT:-http://127.0.0.1:9000}"

postgres_container="$(docker compose -f "$compose_file" ps -q postgres)"

if [[ -z "$postgres_container" ]]; then
  docker compose -f "$compose_file" up -d postgres >/dev/null
fi

if [[ "$object_storage_endpoint" == "http://127.0.0.1:9000" || "$object_storage_endpoint" == "http://localhost:9000" ]]; then
  minio_container="$(docker compose -f "$compose_file" ps -q minio)"
  if [[ -z "$minio_container" ]]; then
    docker compose -f "$compose_file" up -d minio >/dev/null
  fi
fi

(cd "$repo_root/services/api" && go run ./cmd/seed-starter-problems)
