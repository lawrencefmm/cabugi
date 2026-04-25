# cabugi

Cabugi is a competitive programming practice platform with published problems, asynchronous judging, submission history, compile and runtime diagnostics, and moderated user-created problems.

## Current Product Surface
- Published problem list and detail pages backed by the Go API.
- Authenticated solve workspace with asynchronous verdict polling.
- Submission history and submission detail pages with per-test results and compile-output excerpts.
- User-authored problem drafts with hidden bundle upload and submit-for-review flow.
- Moderator review queue with approve, reject, and request-changes actions.
- Official starter problem seeding plus local runtime, integration, and browser smoke verification.

## Repository Layout
- `apps/web`: Next.js frontend.
- `services/api`: Go HTTP API service.
- `services/judge`: Go judge worker service.
- `db`: schema, migrations, and database scripts.
- `infra`: local runtime and verification scripts.
- `docs`: developer-facing documentation.

## Run Locally

### Recommended Interactive Local Run
1. Install workspace dependencies from the repository root:

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

### One-Command Full Stack Runtime
Run the checked-in full-stack helper from the repository root:

```bash
./infra/scripts/up_full_stack.sh
# or: pnpm runtime:up
```

That one command now:
- starts PostgreSQL and MinIO
- applies database migrations
- prepares the pinned judge runtime images
- seeds the official starter problems
- starts `api`, `judge`, and `web`

The helper uses `infra/full-stack.env` when present, otherwise it falls back to `infra/full-stack.env.example`. It fails early with actionable guidance if your machine already uses `3000`, `5432`, `8080`, `8082`, `9000`, or `9001`. If you change the published API or web ports, keep `NEXT_PUBLIC_API_BASE_URL` and `WEB_ALLOWED_ORIGINS` aligned with those overrides.

### Authenticated Workflow Verification
Use the checked-in smoke paths when you want the full authenticated workflow without external auth setup:

```bash
pnpm verify:integration
pnpm verify:smoke:e2e
```

## Key Documentation
- `docs/DEVELOPMENT.md`: detailed local development and verification commands.
- `docs/DEPLOYMENT.md`: container packaging, runtime inputs, and local full-stack runtime.
- `ARCHITECTURE.md`: service boundaries, submission flow, and storage responsibilities.
- `TESTING_RULES.md`: verification policy and current coverage map.
