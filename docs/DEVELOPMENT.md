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

## API Service
Run the bootstrap API locally from `services/api`:

```bash
go run ./cmd/api
```

Current bootstrap API routes:
- `GET /healthz`
- `GET /openapi/v1.yaml`
- `GET /v1/me` with Clerk session authentication and DB-backed app user bootstrap
- `GET /v1/problems`
- `GET /v1/problems/{slug}`
- `GET /v1/submissions`
- `POST /v1/submissions`
- `GET /v1/submissions/{id}`

Clerk environment variables for protected API routes:
- `CLERK_PEM_PUBLIC_KEY`: Clerk JWT verification public key in PEM format.
- `CLERK_ALLOWED_PARTIES`: optional comma-separated allowed frontend origins used to validate the `azp` claim.

Database environment for API routes backed by PostgreSQL:
- `DATABASE_URL`: PostgreSQL connection string. Defaults to `postgres://cabugi:cabugi@127.0.0.1:5432/cabugi?sslmode=disable`.

Web environment:
- `NEXT_PUBLIC_API_BASE_URL`: web runtime base URL for the Go API. Defaults to `http://127.0.0.1:8080`.
- `NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY`: Clerk publishable key used by the web app for authentication.

API browser access environment:
- `WEB_ALLOWED_ORIGINS`: optional comma-separated origins the API should allow for browser requests. Defaults to `http://127.0.0.1:3000,http://localhost:3000`.

## CI
GitHub Actions runs the same baseline verification in `.github/workflows/ci.yml` on pushes to `dev`, `main`, and `task/**`, plus pull requests.

Current CI checks:
- `pnpm --filter web typecheck`
- `pnpm --filter web test --run`
- `go test ./...` from `services/api`
- `go test ./...` from `services/judge`
- `./db/scripts/verify_initial_schema.sh`

Judge spike command from `services/judge`:

```bash
go run ./cmd/judge-spike --all
```

Judge worker command from `services/judge`:

```bash
go run ./cmd/judge --once
```
