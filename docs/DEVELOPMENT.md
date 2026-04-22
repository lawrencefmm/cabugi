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
- The frontend workspace now has an initial `Next.js` scaffold and automated unit test setup.
- The API and judge services now have initial Go unit-test targets.
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
