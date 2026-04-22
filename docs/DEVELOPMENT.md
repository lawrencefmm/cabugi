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
- The frontend app, API service, judge worker, and automated test tooling will be expanded in subsequent tasks.
