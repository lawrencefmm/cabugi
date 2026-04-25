# Development

## Prerequisites
- `pnpm` for frontend workspace management.
- `Go` for the API and judge services.
- `Docker` with Compose support for local shared infrastructure.

## Local Shared Infrastructure
Start the shared services used by local development:

```bash
docker compose -f infra/docker-compose.yml up -d postgres minio
```

Current endpoints:
- PostgreSQL: `localhost:5432`
- MinIO API: `http://localhost:9000`
- MinIO Console: `http://localhost:9001`

Current development credentials:
- PostgreSQL database: `cabugi`
- PostgreSQL user: `cabugi`
- PostgreSQL password: `cabugi`
- MinIO user: `minioadmin`
- MinIO password: `minioadmin`

Run the full local application runtime instead of only shared infrastructure:

```bash
docker compose -f infra/docker-compose.yml --env-file infra/full-stack.env.example up --build
```

See `docs/DEPLOYMENT.md` for image build commands, runtime environment variables, and Docker socket notes for the local judge worker.

## Repository Structure
- `apps/web`: frontend workspace.
- `services/api`: Go API service.
- `services/judge`: Go judge worker.
- `db`: schema and migration assets.
- `infra`: local Docker Compose configuration.
- `docs`: project documentation.

## Current State
- This task bootstraps the repository structure and shared local infrastructure only.
- The frontend workspace now has published problem list and detail UI plus automated unit tests.
- The frontend workspace now has submission history and summary UI for authenticated users.
- The API service now has a bootstrap HTTP server with a health route and a versioned OpenAPI document.
- The API now includes PostgreSQL-backed published problem read endpoints.
- The judge service now has a Docker-based spike plus Go tests for verdict handling.
- Integration tests and full service wiring will be expanded in subsequent tasks.

## Test Commands
Install frontend dependencies from the repository root:

```bash
pnpm install
```

Run focused verification:

```bash
pnpm --filter web typecheck
pnpm --filter web test --run
```

```bash
go test ./...
```

Run the Go command from `services/api` for API tests or from `services/judge` for judge tests.

Current broad verification for the bootstrapped repo:
- `pnpm --filter web typecheck`
- `pnpm --filter web test --run`
- `go test ./...` from `services/api`
- `go test ./...` from `services/judge`
- `./db/scripts/verify_initial_schema.sh`
- `./db/scripts/verify_migration_workflow.sh`
- `./infra/scripts/verify_platform_integration.sh`
- `./infra/scripts/verify_runtime_packaging.sh`
- `./infra/scripts/verify_api_runtime_smoke.sh`
- `./infra/scripts/verify_e2e_workflow_smoke.sh`

Run the API and judge integration verification from the repository root:

```bash
pnpm verify:integration
```

This command starts temporary PostgreSQL and MinIO containers, serves the local JWKS fixture, boots the API and judge worker, creates and moderates a user-authored problem through the real API, then verifies an accepted submission plus per-test results against the published draft.

Run the browser end-to-end smoke from the repository root:

```bash
pnpm verify:smoke:e2e
```

The browser smoke starts temporary PostgreSQL and MinIO containers, boots the API and judge worker against them, builds and starts the web app, and runs Playwright from the checked-in Docker image. It uses the local test-auth fixture under `infra/testdata/local_test_auth/` so the real web app can exercise signed-in author and moderator flows without depending on an external Clerk environment.

Seed official starter problems into PostgreSQL and object storage:

```bash
./db/scripts/seed_official_starter_problems.sh
```

## API Service
Run the bootstrap API locally from `services/api`:

```bash
go run ./cmd/api
```

Current bootstrap API routes:
- `GET /healthz`
- `GET /readyz`
- `GET /metricsz`
- `GET /openapi/v1.yaml`
- `GET /v1/me` with Clerk session authentication and DB-backed app user bootstrap
- `GET /v1/problems`
- `GET /v1/problems/{slug}`
- `POST /v1/problem-drafts/hidden-test-bundles`
- `POST /v1/problem-drafts`
- `GET /v1/problem-drafts/{slug}`
- `PATCH /v1/problem-drafts/{slug}`
- `POST /v1/problem-drafts/{slug}/submit-for-review`
- `GET /v1/moderation/problem-drafts`
- `POST /v1/moderation/problem-drafts/{slug}/decision`
- `GET /v1/submissions`
- `POST /v1/submissions`
- `GET /v1/submissions/{id}`

Staff role bootstrap command from `services/api`:

```bash
DATABASE_URL="postgres://cabugi:cabugi@127.0.0.1:5432/cabugi?sslmode=disable" \
  go run ./cmd/grant-staff-role --bootstrap-first-admin --target-subject user_local_admin --role admin
```

Grant a moderator or another admin after the first admin exists:

```bash
DATABASE_URL="postgres://cabugi:cabugi@127.0.0.1:5432/cabugi?sslmode=disable" \
  go run ./cmd/grant-staff-role --requester-subject user_local_admin --target-subject user_local_moderator --role moderator
```

This command path is operator-only. The bootstrap flag works only while no admin exists yet, and later grants require the requester subject to already hold the `admin` role.

Clerk environment variables for protected API routes:
- `CLERK_ISSUER`: required expected `iss` claim for Clerk session tokens when auth verification is enabled.
- `CLERK_JWKS_URL`: optional explicit JWKS endpoint. Defaults to `<CLERK_ISSUER>/.well-known/jwks.json`.
- `CLERK_PEM_PUBLIC_KEY`: optional static Clerk JWT verification public key in PEM format.
- `CLERK_ALLOWED_PARTIES`: optional comma-separated allowed frontend origins used to validate the `azp` claim.
- `CLERK_ALLOWED_AUDIENCES`: optional comma-separated allowed token audiences used to validate the `aud` claim.
- `API_REQUIRE_AUTH`: when `true`, the API refuses to start unless Clerk auth verification is configured.
- `API_REQUIRE_DATABASE`: when `true`, the API refuses to start unless the PostgreSQL-backed stores can connect successfully.
- `API_REQUIRE_HIDDEN_BUNDLE_VALIDATION`: when `true`, the API refuses to start unless hidden test bundle validation can reach the configured object storage bucket.

Browser auth transport:
- The API accepts bearer tokens from the `Authorization` header and from the `__session` cookie.
- If both are present, the `Authorization` header takes precedence.
- Production auth should set `CLERK_ISSUER` plus at least one of `CLERK_ALLOWED_PARTIES` or `CLERK_ALLOWED_AUDIENCES`.

Database environment for API routes backed by PostgreSQL:
- `DATABASE_URL`: PostgreSQL connection string. Defaults to `postgres://cabugi:cabugi@127.0.0.1:5432/cabugi?sslmode=disable`.

Web environment:
- `NEXT_PUBLIC_API_BASE_URL`: web runtime base URL for the Go API. Defaults to `http://127.0.0.1:8080`.
- `NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY`: Clerk publishable key used by the web app for authentication.
- `NEXT_PUBLIC_LOCAL_TEST_AUTH_ENABLED`: test-only local auth adapter used by the browser smoke suite. Do not enable this in deployed environments.

API browser access environment:
- `WEB_ALLOWED_ORIGINS`: optional comma-separated origins the API should allow for browser requests. Defaults to `http://127.0.0.1:3000,http://localhost:3000`.

Readiness endpoints:
- `GET /healthz` reports process liveness only.
- `GET /readyz` reports dependency readiness for auth, database-backed stores, and hidden test bundle validation, and returns `503` when any of them is unavailable.

API observability:
- API logs are JSON records on stdout. Request logs use the `http_request` message and include `request_id`, `method`, `path`, `route`, `status`, and `latency_ms`.
- `GET /metricsz` returns request counters and current submission queue depth.

Judge object storage environment:
- `OBJECT_STORAGE_ENDPOINT`: judge object storage endpoint. Defaults to `http://127.0.0.1:9000` for local MinIO.
- `OBJECT_STORAGE_REGION`: object storage region. Defaults to `us-east-1`.
- `OBJECT_STORAGE_BUCKET`: hidden test bundle bucket. Defaults to `cabugi-hidden-tests`.
- `OBJECT_STORAGE_ACCESS_KEY_ID`: object storage access key. Defaults to `minioadmin`.
- `OBJECT_STORAGE_SECRET_ACCESS_KEY`: object storage secret key. Defaults to `minioadmin`.
- `OBJECT_STORAGE_USE_PATH_STYLE`: optional path-style toggle for S3-compatible APIs. Defaults to `true` for local MinIO.
- `JUDGE_MAX_JOB_ATTEMPTS`: retry limit before the worker marks a submission as `judge_failed`. Defaults to `3`.
- `JUDGE_JOB_LEASE_DURATION`: how long a claimed submission job remains owned without renewal before another worker may reclaim it. Defaults to `30s`.
- `JUDGE_JOB_LEASE_RENEW_INTERVAL`: how often the active worker renews its current job lease. Defaults to `10s`.
- `JUDGE_POLL_INTERVAL`: idle poll interval for the long-running worker loop. Defaults to `3s`.
- `JUDGE_RETRY_DELAY`: delay before retrying a failed claimed job. Defaults to `5s`.
- `JUDGE_OBSERVABILITY_ADDRESS`: bind address for judge `GET /healthz` and `GET /metricsz`. Defaults to `127.0.0.1:8082`.

Judge observability:
- Judge logs are JSON records on stdout for claim, completion, retry, and terminal failure paths.
- `GET http://127.0.0.1:8082/metricsz` returns worker outcome counters and heartbeat freshness when the worker runs with the default observability address.

Local judge bundle storage:
- Create the `cabugi-hidden-tests` bucket in MinIO.
- Upload JSON bundle objects to that bucket, and store the object key in `hidden_test_bundle_key`.
- Store the uploaded bundle SHA-256 hex digest in `hidden_test_bundle_sha256` so the API can validate draft bundle references and the judge can verify integrity before execution.

Database operations:
- Use `DATABASE_URL="postgres://..." ./db/scripts/migrate.sh status` to inspect production-style migration state.
- Use `DATABASE_URL="postgres://..." ./db/scripts/migrate.sh up` to apply pending migrations without dropping schemas.
- Use `DATABASE_URL="postgres://..." BACKUP_DIR="/secure/backups" ./db/scripts/backup.sh` before production migrations.
- Use `CONFIRM_RESTORE=yes DATABASE_URL="postgres://..." ./db/scripts/restore.sh <backup.dump>` for explicit restore operations.
- `./db/scripts/verify_initial_schema.sh` uses an isolated temporary PostgreSQL container for destructive schema verification; do not adapt it for production databases.

## CI
GitHub Actions runs the same baseline verification in `.github/workflows/ci.yml` on pushes to `dev`, `main`, and `task/**`, plus pull requests.

Current CI checks:
- `pnpm --filter web typecheck`
- `pnpm --filter web test --run`
- `pnpm --filter web build`
- `go test ./...` from `services/api`
- `go test ./...` from `services/judge`
- `./db/scripts/verify_initial_schema.sh`
- `./db/scripts/verify_migration_workflow.sh`
- `./infra/scripts/verify_runtime_packaging.sh`
- `./infra/scripts/verify_api_runtime_smoke.sh`
- `./infra/scripts/verify_platform_integration.sh`
- `./infra/scripts/verify_e2e_workflow_smoke.sh`
- `./infra/scripts/verify_go_vulnerabilities.sh`
- Docker image builds for `apps/web`, `services/api`, and `services/judge`

Judge spike command from `services/judge`:

```bash
go run ./cmd/judge-spike --all
```

Prepare the pinned worker sandbox images from `services/judge` before starting the long-running judge worker on a host:

```bash
docker pull gcc:14.2.0
docker pull python:3.13.0-alpine3.20
```

Long-running judge worker command from `services/judge`:

```bash
go run ./cmd/judge
```

One-shot judge worker command from `services/judge`:

```bash
go run ./cmd/judge --once
```

## Local MVP Smoke Test
1. Start shared infrastructure:

```bash
docker compose -f infra/docker-compose.yml up -d postgres minio
```

2. Seed starter problems and hidden bundles:

```bash
./db/scripts/seed_official_starter_problems.sh
```

3. Run `pnpm verify:smoke:e2e` from the repository root.
