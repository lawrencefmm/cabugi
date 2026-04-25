# Development

## Prerequisites
- `pnpm` for the web workspace.
- `Go` for the API and judge services.
- `Docker` with Compose support for PostgreSQL, MinIO, and the optional full local stack.

## Recommended Local Run Paths

### Interactive Browser Run
Use this path when you want the normal local development loop with separate service processes.

1. Install dependencies from the repository root:

```bash
pnpm install
```

2. Start PostgreSQL and MinIO:

```bash
docker compose -f infra/docker-compose.yml up -d postgres minio
```

3. Seed the official starter problems:

```bash
./db/scripts/seed_official_starter_problems.sh
```

4. Start the API from `services/api`:

```bash
(cd services/api && API_REQUIRE_DATABASE=true API_REQUIRE_HIDDEN_BUNDLE_VALIDATION=true go run ./cmd/api)
```

5. Prepare the pinned judge images, then start the judge worker from `services/judge`:

```bash
docker pull gcc:14.2.0
docker pull python:3.13.0-alpine3.20
(cd services/judge && go run ./cmd/judge)
```

6. Start the web app from the repository root:

```bash
NEXT_PUBLIC_API_BASE_URL=http://127.0.0.1:8080 pnpm --filter web dev
```

7. Open `http://127.0.0.1:3000`.

This default browser path gives you the public problem pages immediately. Authenticated browser flows still require real Clerk configuration, because `NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY` is unset by default.

### Full Local Compose Runtime
Run the checked-in full stack definition from the repository root:

```bash
docker compose -f infra/docker-compose.yml --env-file infra/full-stack.env.example up --build
```

Then seed the official starter problems in a second shell:

```bash
./db/scripts/seed_official_starter_problems.sh
```

The example env file is useful for packaging and public-stack development. It intentionally leaves browser auth off, so drafts, moderation, submission history, and browser solve submissions still need real Clerk configuration.

### Authenticated Workflow Verification
Use the checked-in local auth fixture when you want the full authenticated stack without external auth setup:

```bash
pnpm verify:integration
pnpm verify:smoke:e2e
```

`pnpm verify:integration` exercises the authenticated API and judge flow. `pnpm verify:smoke:e2e` adds the browser author, moderation, publication, and verdict loop on top.

## Local Shared Infrastructure
Current shared-service endpoints:
- PostgreSQL: `localhost:5432`
- MinIO API: `http://localhost:9000`
- MinIO Console: `http://localhost:9001`

Current default development credentials:
- PostgreSQL database: `cabugi`
- PostgreSQL user: `cabugi`
- PostgreSQL password: `cabugi`
- MinIO user: `minioadmin`
- MinIO password: `minioadmin`

## Repository Structure
- `apps/web`: Next.js frontend workspace.
- `services/api`: Go API service.
- `services/judge`: Go judge worker.
- `db`: schema and migration assets.
- `infra`: local runtime and verification scripts.
- `docs`: project documentation.

## Current Capabilities
- The web app serves published problems, solve workspace screens, submission history, submission detail diagnostics, draft authoring, and moderation pages.
- The API serves authenticated and public problem, draft, moderation, submission, and current-user routes backed by PostgreSQL.
- The judge worker claims queued submissions, executes `C++17` and `Python` in a hardened sandbox, and writes final verdicts plus safe diagnostics back to PostgreSQL.
- Official starter problems can be seeded into PostgreSQL and MinIO for immediate local browsing.
- CI and local verification include API smoke, platform integration, and browser end-to-end coverage.

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

Current broad verification commands:
- `pnpm --filter web typecheck`
- `pnpm --filter web test --run`
- `go test ./...` from `services/api`
- `go test ./...` from `services/judge`
- `./db/scripts/verify_initial_schema.sh`
- `./db/scripts/verify_migration_workflow.sh`
- `./infra/scripts/verify_runtime_packaging.sh`
- `./infra/scripts/verify_api_runtime_smoke.sh`
- `./infra/scripts/verify_platform_integration.sh`
- `./infra/scripts/verify_e2e_workflow_smoke.sh`
- `./infra/scripts/verify_go_vulnerabilities.sh`

Convenience commands from the repository root:

```bash
pnpm verify:runtime
pnpm verify:smoke:api
pnpm verify:integration
pnpm verify:smoke:e2e
pnpm verify:security
pnpm verify:db-ops
```

## API Service
Run the API locally from `services/api`:

```bash
go run ./cmd/api
```

Current routes:
- `GET /healthz`
- `GET /readyz`
- `GET /metricsz`
- `GET /openapi/v1.yaml`
- `GET /v1/me`
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

Grant a moderator or another admin after bootstrap:

```bash
DATABASE_URL="postgres://cabugi:cabugi@127.0.0.1:5432/cabugi?sslmode=disable" \
  go run ./cmd/grant-staff-role --requester-subject user_local_admin --target-subject user_local_moderator --role moderator
```

Important environment values:
- `CLERK_ISSUER`: expected `iss` claim for Clerk session tokens when auth verification is enabled.
- `CLERK_JWKS_URL`: explicit JWKS endpoint, or leave unset to derive `<CLERK_ISSUER>/.well-known/jwks.json`.
- `CLERK_PEM_PUBLIC_KEY`: static Clerk JWT verification public key for local or fallback use.
- `CLERK_ALLOWED_PARTIES`: allowed `azp` values.
- `CLERK_ALLOWED_AUDIENCES`: allowed `aud` values.
- `API_REQUIRE_AUTH`: fail startup when auth is required but not configured.
- `API_REQUIRE_DATABASE`: fail startup when the PostgreSQL-backed stores are unavailable.
- `API_REQUIRE_HIDDEN_BUNDLE_VALIDATION`: fail startup when hidden bundle validation cannot reach object storage.
- `DATABASE_URL`: PostgreSQL connection string. Defaults to `postgres://cabugi:cabugi@127.0.0.1:5432/cabugi?sslmode=disable`.
- `WEB_ALLOWED_ORIGINS`: comma-separated browser origins allowed by API CORS.

Browser auth transport:
- The API accepts bearer tokens from the `Authorization` header and from the `__session` cookie.
- If both are present, the `Authorization` header takes precedence.

## Web Environment
- `NEXT_PUBLIC_API_BASE_URL`: web runtime base URL for the Go API. Defaults to `http://127.0.0.1:8080`.
- `NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY`: Clerk publishable key used by the web app for auth.
- `NEXT_PUBLIC_LOCAL_TEST_AUTH_ENABLED`: test-only local auth adapter used by the browser smoke suite.

## Judge Environment
- `OBJECT_STORAGE_ENDPOINT`: object storage endpoint. Defaults to `http://127.0.0.1:9000`.
- `OBJECT_STORAGE_REGION`: object storage region. Defaults to `us-east-1`.
- `OBJECT_STORAGE_BUCKET`: hidden test bundle bucket. Defaults to `cabugi-hidden-tests`.
- `OBJECT_STORAGE_ACCESS_KEY_ID`: object storage access key. Defaults to `minioadmin`.
- `OBJECT_STORAGE_SECRET_ACCESS_KEY`: object storage secret key. Defaults to `minioadmin`.
- `OBJECT_STORAGE_USE_PATH_STYLE`: path-style toggle for S3-compatible APIs. Defaults to `true`.
- `JUDGE_MAX_JOB_ATTEMPTS`: retry limit before `judge_failed`. Defaults to `3`.
- `JUDGE_JOB_LEASE_DURATION`: unrenewed claim window. Defaults to `30s`.
- `JUDGE_JOB_LEASE_RENEW_INTERVAL`: lease renewal cadence. Defaults to `10s`.
- `JUDGE_POLL_INTERVAL`: idle poll interval. Defaults to `3s`.
- `JUDGE_RETRY_DELAY`: retry delay after worker failures. Defaults to `5s`.
- `JUDGE_OBSERVABILITY_ADDRESS`: bind address for `GET /healthz` and `GET /metricsz`. Defaults to `127.0.0.1:8082`.

## Database Operations
- `DATABASE_URL="postgres://..." ./db/scripts/migrate.sh status`
- `DATABASE_URL="postgres://..." ./db/scripts/migrate.sh up`
- `DATABASE_URL="postgres://..." BACKUP_DIR="/secure/backups" ./db/scripts/backup.sh`
- `CONFIRM_RESTORE=yes DATABASE_URL="postgres://..." ./db/scripts/restore.sh <backup.dump>`

`./db/scripts/verify_initial_schema.sh` is destructive verification logic and intentionally targets only an isolated temporary PostgreSQL container.

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
