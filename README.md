# cabugi

Cabugi is a competitive programming practice platform focused on published problems, asynchronous judging, submission history, and moderated user-created problems.

## Repository Layout
- `apps/web`: frontend application workspace.
- `services/api`: Go HTTP API service.
- `services/judge`: Go judge worker service.
- `db`: database assets such as schema docs and migrations.
- `infra`: local infrastructure configuration.
- `docs`: developer-facing project documentation.

## Current Bootstrap Status
- The repository layout and local shared infrastructure are bootstrapped.
- The application services are still scaffolds; feature implementation and full local startup will be added in later tasks.

## Local Shared Infrastructure
Start PostgreSQL and MinIO for local development:

```bash
docker compose -f infra/docker-compose.yml up -d postgres minio
```

Open MinIO Console at `http://localhost:9001`.

Seed the official starter problems after the shared services are up:

```bash
./db/scripts/seed_official_starter_problems.sh
```

For more details, see `docs/DEVELOPMENT.md`.
