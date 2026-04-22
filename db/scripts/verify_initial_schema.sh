#!/usr/bin/env bash

set -euo pipefail

compose_file="infra/docker-compose.yml"
container_name="$(docker compose -f "$compose_file" ps -q postgres)"

if [[ -z "$container_name" ]]; then
  docker compose -f "$compose_file" up -d postgres >/dev/null
  container_name="$(docker compose -f "$compose_file" ps -q postgres)"
fi

until docker exec "$container_name" pg_isready -U cabugi -d cabugi >/dev/null 2>&1; do
  sleep 1
done

until docker exec -i "$container_name" psql -U cabugi -d cabugi -v ON_ERROR_STOP=1 -Atqc "SELECT 1" >/dev/null 2>&1; do
  sleep 1
done

psql_exec() {
  docker exec -i "$container_name" psql -U cabugi -d cabugi -v ON_ERROR_STOP=1 -Atqc "$1"
}

docker exec -i "$container_name" psql -U cabugi -d cabugi -v ON_ERROR_STOP=1 -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;" >/dev/null
docker exec -i "$container_name" psql -U cabugi -d cabugi -v ON_ERROR_STOP=1 < db/migrations/0001_initial_schema.sql >/dev/null

assert_equals() {
  local actual="$1"
  local expected="$2"
  local message="$3"

  if [[ "$actual" != "$expected" ]]; then
    printf 'verification failed: %s (got %s, want %s)\n' "$message" "$actual" "$expected" >&2
    exit 1
  fi
}

assert_equals "$(psql_exec "SELECT to_regclass('public.users') IS NOT NULL")" "t" "users table must exist"
assert_equals "$(psql_exec "SELECT to_regclass('public.user_roles') IS NOT NULL")" "t" "user_roles table must exist"
assert_equals "$(psql_exec "SELECT string_agg(enumlabel, ',' ORDER BY enumsortorder) FROM pg_enum JOIN pg_type ON pg_enum.enumtypid = pg_type.oid WHERE typname = 'problem_version_status'")" "draft,in_review,published,archived" "problem version lifecycle enum must match MVP states"
assert_equals "$(psql_exec "SELECT string_agg(enumlabel, ',' ORDER BY enumsortorder) FROM pg_enum JOIN pg_type ON pg_enum.enumtypid = pg_type.oid WHERE typname = 'submission_status'")" "queued,running,accepted,wrong_answer,compile_error,runtime_error,time_limit_exceeded,judge_failed" "submission status enum must include judge failure handling"
assert_equals "$(psql_exec "SELECT COUNT(*) FROM information_schema.table_constraints tc JOIN information_schema.key_column_usage kcu ON tc.constraint_name = kcu.constraint_name AND tc.table_schema = kcu.table_schema JOIN information_schema.constraint_column_usage ccu ON tc.constraint_name = ccu.constraint_name AND tc.table_schema = ccu.table_schema WHERE tc.constraint_type = 'FOREIGN KEY' AND tc.table_name = 'submissions' AND kcu.column_name = 'problem_version_id' AND ccu.table_name = 'problem_versions' AND ccu.column_name = 'id'")" "1" "submissions must reference problem_versions"
assert_equals "$(psql_exec "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name IN ('contests', 'contest_registrations', 'contest_submissions', 'standings')")" "0" "contest tables must not exist in the MVP schema"

printf 'initial schema verification passed\n'
