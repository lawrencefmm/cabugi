# Deployment Packaging

Cabugi has checked-in container packaging for each deployable application unit:

- `apps/web/Dockerfile`: Next.js web app image.
- `services/api/Dockerfile`: Go API image.
- `services/judge/Dockerfile`: Go judge worker image with Docker CLI for sandbox orchestration.

## Image Builds
Run image builds from the repository root:

```bash
pnpm build:image:web
pnpm build:image:api
pnpm build:image:judge
```

CI runs the same Dockerfile build paths on pushes and pull requests.

## Local Full Stack Runtime
Run the local application stack from the repository root:

```bash
docker compose -f infra/docker-compose.yml --env-file infra/full-stack.env.example up --build
```

The runtime definition includes:

- `web` on `http://127.0.0.1:3000`.
- `api` on `http://127.0.0.1:8080`.
- `judge` observability on `http://127.0.0.1:8082`.
- `postgres` on `localhost:5432`.
- `minio` API on `http://127.0.0.1:9000` and console on `http://127.0.0.1:9001`.

The judge container mounts `/var/run/docker.sock` and uses `/tmp/cabugi-judge-workspaces` as a shared host path so sandbox containers can bind worker scratch directories. Treat Docker socket access as privileged and only use this Compose mode for local development or isolated judge hosts.

## Runtime Inputs
Required production secrets and environment inputs:

- `POSTGRES_PASSWORD`: PostgreSQL password used by the local runtime and by service connection strings.
- `DATABASE_URL`: API and judge PostgreSQL connection string in deployed environments.
- `NEXT_PUBLIC_API_BASE_URL`: browser-visible API base URL compiled into the web image.
- `NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY`: Clerk frontend publishable key.
- `CLERK_PEM_PUBLIC_KEY`: Clerk JWT verification public key for API auth.
- `CLERK_ALLOWED_PARTIES`: comma-separated allowed Clerk `azp` values.
- `WEB_ALLOWED_ORIGINS`: comma-separated origins accepted by API CORS.
- `OBJECT_STORAGE_ENDPOINT`: private object storage endpoint for hidden test bundles.
- `OBJECT_STORAGE_REGION`: object storage region.
- `OBJECT_STORAGE_BUCKET`: hidden test bundle bucket name.
- `OBJECT_STORAGE_ACCESS_KEY_ID`: object storage access key.
- `OBJECT_STORAGE_SECRET_ACCESS_KEY`: object storage secret key.
- `OBJECT_STORAGE_USE_PATH_STYLE`: set to `true` for MinIO-style local object storage.
- `JUDGE_MAX_JOB_ATTEMPTS`: retry limit before `judge_failed`.
- `JUDGE_JOB_LEASE_DURATION`: maximum unrenewed job lease duration.
- `JUDGE_JOB_LEASE_RENEW_INTERVAL`: active job lease renewal cadence.
- `JUDGE_POLL_INTERVAL`: idle judge poll interval.
- `JUDGE_RETRY_DELAY`: delay before retrying failed jobs.
- `JUDGE_OBSERVABILITY_ADDRESS`: judge health and metrics bind address.

Local defaults live in `infra/full-stack.env.example`; do not commit real production secrets.

## Verification
Verify packaging metadata without building images:

```bash
pnpm verify:runtime
```

Baseline CI also builds all three service images to catch Dockerfile regressions.

## Database Migrations And Recovery
Production migrations use the non-destructive runner in `db/scripts/migrate.sh`. This is separate from `db/scripts/verify_initial_schema.sh`, which is destructive verification logic and intentionally runs only against an isolated temporary database container.

Check pending migrations against a target database:

```bash
DATABASE_URL="postgres://..." ./db/scripts/migrate.sh status
```

Before applying migrations, create a backup:

```bash
DATABASE_URL="postgres://..." BACKUP_DIR="/secure/backups" ./db/scripts/backup.sh
```

Apply pending migrations:

```bash
DATABASE_URL="postgres://..." ./db/scripts/migrate.sh up
```

The migration runner records applied migration filenames and SHA-256 checksums in `schema_migrations`. If an applied migration file changes later, the runner fails instead of applying more changes on top of an unknown schema history.

Restore from a backup only after selecting the correct target database and backup file:

```bash
CONFIRM_RESTORE=yes DATABASE_URL="postgres://..." ./db/scripts/restore.sh /secure/backups/cabugi-YYYYMMDDTHHMMSSZ.dump
```

Restore is destructive because it runs `pg_restore --clean --if-exists` against the target database. Prefer restoring into a separate database first when validating a recovery path.

Verify the migration and recovery workflow locally:

```bash
pnpm verify:db-ops
```
