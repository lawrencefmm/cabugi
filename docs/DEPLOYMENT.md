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
docker compose -f infra/docker-compose.yml --env-file infra/full-stack.env.example up --build -d
```

The runtime definition includes:
- `web` on `http://127.0.0.1:3000`
- `api` on `http://127.0.0.1:8080`
- `judge` observability on `http://127.0.0.1:8082`
- `postgres` on `localhost:5432`
- `minio` API on `http://127.0.0.1:9000` and console on `http://127.0.0.1:9001`

The Compose startup also runs three one-shot init services automatically:
- `migrate`: applies checked-in database migrations before the API and judge start
- `judge-image-prep`: pulls the pinned judge runtime images through the host Docker socket
- `seed-starter-problems`: seeds the official published starter problems and hidden bundles before the public web flow comes up

The checked-in `infra/full-stack.env.example` values are intended for local runtime packaging and public problem browsing. They do not enable browser auth by default, so authenticated browser flows still need real Clerk values or the checked-in authenticated smoke commands.

If your host already uses the default ports, override these values in the env file before running Compose:
- `WEB_PORT`
- `API_PORT`
- `JUDGE_OBSERVABILITY_PORT`
- `POSTGRES_PORT`
- `MINIO_PORT`
- `MINIO_CONSOLE_PORT`

When you override the published API or web ports, keep `NEXT_PUBLIC_API_BASE_URL` and `WEB_ALLOWED_ORIGINS` aligned with those values. If browser auth is enabled, `CLERK_ALLOWED_PARTIES` should match the published web origin too.

The judge container mounts `/var/run/docker.sock` and uses `/tmp/cabugi-judge-workspaces` as a shared host path so sandbox containers can bind worker scratch directories. Treat Docker socket access as privileged and only use this Compose mode for local development or isolated judge hosts.

## Runtime Inputs
Important production secrets and environment inputs:
- `POSTGRES_PASSWORD`: PostgreSQL password used by the local runtime and service connection strings.
- `DATABASE_URL`: API and judge PostgreSQL connection string in deployed environments.
- `NEXT_PUBLIC_API_BASE_URL`: browser-visible API base URL compiled into the web image.
- `NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY`: Clerk frontend publishable key.
- `CLERK_ISSUER`: expected issuer for Clerk session tokens.
- `CLERK_JWKS_URL`: optional explicit JWKS endpoint; defaults from `CLERK_ISSUER`.
- `CLERK_PEM_PUBLIC_KEY`: optional static Clerk JWT verification public key.
- `CLERK_ALLOWED_PARTIES`: comma-separated allowed Clerk `azp` values.
- `CLERK_ALLOWED_AUDIENCES`: comma-separated allowed Clerk `aud` values.
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

For production auth, set `CLERK_ISSUER` and at least one of `CLERK_ALLOWED_PARTIES` or `CLERK_ALLOWED_AUDIENCES`. Prefer the JWKS path for deployed environments so key rotation does not require a static PEM rollout. The API accepts either bearer headers or the `__session` cookie; if both are sent, the bearer header is authoritative.

## Verification
Verify packaging metadata without building images:

```bash
pnpm verify:runtime
```

Baseline CI also builds all three service images to catch Dockerfile regressions.

CI also runs:
- `./infra/scripts/verify_api_runtime_smoke.sh` for public API plus backing-service smoke.
- `./infra/scripts/verify_platform_integration.sh` for authenticated API and judge publication plus submission coverage.
- `./infra/scripts/verify_e2e_workflow_smoke.sh` for browser author, moderation, publication, and submission flows end to end.
- `./infra/scripts/verify_go_vulnerabilities.sh` for `govulncheck` against the Go services.

The browser smoke uses the checked-in local JWKS fixture plus the web app's test-only local auth adapter. The Playwright browser step runs inside the official Playwright Docker image so the smoke stays reproducible across CI and local Linux environments without extra host browser packages.

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

The database scripts use local PostgreSQL client tools by default. Set `POSTGRES_TOOLS_MODE=docker` to force the checked workflow to use the `postgres:17-alpine` client tools, which avoids local client and server version mismatches.

Restore from a backup only after selecting the correct target database and backup file:

```bash
CONFIRM_RESTORE=yes DATABASE_URL="postgres://..." ./db/scripts/restore.sh /secure/backups/cabugi-YYYYMMDDTHHMMSSZ.dump
```

Restore is destructive because it runs `pg_restore --clean --if-exists` against the target database. Prefer restoring into a separate database first when validating a recovery path.

Verify the migration and recovery workflow locally:

```bash
pnpm verify:db-ops
```
