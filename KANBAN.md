# KANBAN

## Rules
- Use this file as the source of truth for active and upcoming work.
- Before starting a new implementation batch, pause for planning with the user and ask clarifying questions if anything important is unclear.
- Complete at most four tasks after each planning checkpoint before stopping for another planning pass with the user.
- After each completed batch, inspect the current GitHub Actions runs and confirm the pipelines are working as intended.
- Keep exactly one task in `In Progress` unless the user explicitly approves parallel work.
- Use `dev` as the integration branch for ongoing work.
- Start each task branch from `dev`.
- Use one git branch per task, push each completed task branch to `origin`, then merge it into `dev` and push `dev` after its acceptance checks pass.
- Use Conventional Commits for all new commit messages.
- Every task must include `Description`, `Expected Result`, `Acceptance Tests`, and `Notes`.
- Acceptance tests must be objective and directly verifiable from files, commands, endpoints, or UI behavior.
- After finishing a task, add implementation details and important discoveries to that task's `Notes` section.
- Add or update automated tests for meaningful behavior whenever practical, and run the smallest relevant verification continuously while implementing.
- Use `TESTING_RULES.md` to keep important project rules and requirements mapped to verification.
- If a task cannot yet be covered by automated tests, record the gap in `Notes` and add follow-up work.

## In Progress

### CI-02 - Fix pnpm setup in GitHub Actions
Description: Fix the failing GitHub Actions web job by ensuring `pnpm` is installed before `actions/setup-node` uses `cache: pnpm`, and record the new workflow rules introduced after the last batch.

Expected Result: The `web` CI job succeeds, and the repository instructions explicitly require Conventional Commits plus post-batch CI inspection.

Acceptance Tests:
- `.github/workflows/ci.yml` installs `pnpm` before `actions/setup-node` in the `web` job.
- `AGENTS.md` records the Conventional Commits rule.
- `AGENTS.md` records the post-batch GitHub Actions inspection rule.
- `KANBAN.md` rules record the same two workflow expectations.
- A GitHub Actions run on the task branch completes successfully.

Notes:
- In progress.

## Ready

### USER-01 - Bootstrap app users from Clerk subjects
Description: Create or load a database-backed app user from an authenticated Clerk subject, using generated temporary values for `handle` and `display_name` on first bootstrap.

Expected Result: Authenticated API requests resolve a stable `users` row that later features can reference by `user_id`.

Acceptance Tests:
- `GET /v1/me` returns `401` without a token.
- `GET /v1/me` returns `200` with a verified token.
- The first successful authenticated request creates a `users` row keyed by `auth_subject`.
- Repeated authenticated requests reuse the same user row.
- Generated temporary `handle` values are unique.
- `go test ./...` passes in `services/api`.

Notes:
- Not started.

### SUB-API-01 - Add submission create and read endpoints
Description: Add API routes to create a submission for a published problem and fetch a stored submission by id.

Expected Result: The API can accept a submission, queue it in PostgreSQL, and expose its current state to the owner.

Acceptance Tests:
- The OpenAPI document defines `POST /v1/submissions`.
- The OpenAPI document defines `GET /v1/submissions/{id}`.
- `POST /v1/submissions` returns `401` without auth.
- `POST /v1/submissions` returns `201` for an authenticated user on a published problem.
- `POST /v1/submissions` creates one `submissions` row and one `submission_jobs` row.
- `POST /v1/submissions` rejects draft-only or missing problems.
- `GET /v1/submissions/{id}` returns `404` for unknown ids.
- `GET /v1/submissions/{id}` returns the stored submission for its owner.
- `go test ./...` passes in `services/api`.

Notes:
- Not started.

### JUDGE-02 - Process queued submissions and persist verdicts
Description: Extend the judge from a local spike into a worker flow that claims queued submission jobs, evaluates them, and writes final verdicts plus per-test results back to PostgreSQL.

Expected Result: A queued submission can move through `queued`, `running`, and a final verdict using the real database-backed workflow.

Acceptance Tests:
- A documented worker command exists for local execution.
- The worker can claim one queued submission job from PostgreSQL.
- The worker updates submission state to `running` and then a final verdict.
- The worker inserts `submission_results` rows.
- The worker can persist at least `accepted`, `wrong_answer`, `compile_error`, and `time_limit_exceeded`.
- `go test ./...` passes in `services/judge`.

Notes:
- Not started.

## Done

### PROB-API-01 - Implement published problem read endpoints
Description: Implement the first product endpoints for listing published problems and fetching a published problem by slug from PostgreSQL.

Expected Result: The API can return published problem data using the stable problem version model defined in the schema.

Acceptance Tests:
- The OpenAPI document defines `GET /v1/problems`.
- The OpenAPI document defines `GET /v1/problems/{slug}`.
- `GET /v1/problems` returns only `published` problem versions in tests.
- `GET /v1/problems/{slug}` returns `200` for a published problem.
- `GET /v1/problems/{slug}` returns `404` for a draft-only or missing slug.
- `go test ./...` passes in `services/api`.

Notes:
- Completed by adding a PostgreSQL-backed problem store, public API handlers for `GET /v1/problems` and `GET /v1/problems/{slug}`, and OpenAPI definitions for both endpoints.
- Added store-level query tests to prove the SQL only reads `published` problem versions, plus handler tests for `200` and `404` behavior.
- Documented `DATABASE_URL` usage in `services/api/README.md` and `docs/DEVELOPMENT.md`.
- Verified `go test ./...` in `services/api`, and confirmed locally that a seeded published problem returns `200` while a draft-only slug returns `404`.

### AUTH-01 - Add Clerk auth verification to the API
Description: Add API-side authentication plumbing for Clerk, including token verification, current-user resolution, and protected-route middleware.

Expected Result: The API can distinguish public and authenticated routes and resolve the authenticated user subject safely.

Acceptance Tests:
- The API has a protected test route that returns `401` with no token.
- The same protected route returns `401` for an invalid token.
- Middleware tests prove authenticated requests can pass when the verifier accepts the token.
- Required Clerk-related environment variables are documented in `docs/DEVELOPMENT.md`.
- `go test ./...` passes in `services/api`.

Notes:
- Completed by adding Clerk-style session token middleware, request token extraction, current-user context handling, and a protected `GET /v1/me` bootstrap route.
- Used Clerk's documented manual JWT verification approach with `CLERK_PEM_PUBLIC_KEY` and optional allowed `azp` validation via `CLERK_ALLOWED_PARTIES`.
- Documented the required auth environment variables in `services/api/README.md` and `docs/DEVELOPMENT.md`, and extended the OpenAPI document with the protected route and security scheme.
- Verified `go test ./...` in `services/api` and confirmed `GET /v1/me` returns `401` without a token when running `go run ./cmd/api` locally.

### API-01 - Bootstrap API service and publish OpenAPI
Description: Turn `services/api` from a placeholder into a minimal HTTP service with configuration loading, a health route, and a versioned OpenAPI document.

Expected Result: The API starts locally, exposes a health endpoint, and serves a versioned contract that future handlers will implement.

Acceptance Tests:
- `services/api/openapi/v1.yaml` exists.
- Running the API locally exposes `GET /healthz` and returns `200`.
- Running the API locally exposes the OpenAPI document and returns `200`.
- `go test ./...` passes in `services/api`.
- Handler tests cover the health route and OpenAPI document route.

Notes:
- Completed by adding a bootstrap HTTP server in `services/api` with config loading, route registration, and graceful shutdown wiring.
- Added `services/api/openapi/v1.yaml` plus embedded serving at `GET /openapi/v1.yaml` and a `GET /healthz` JSON health route.
- Documented the local run command in `services/api/README.md` and `docs/DEVELOPMENT.md`.
- Verified `go test ./...` in `services/api` and confirmed `GET /healthz` and `GET /openapi/v1.yaml` return `200` when running `go run ./cmd/api` locally.

### CI-01 - Add baseline CI
Description: Add GitHub Actions that run the repository's verified checks on pushes and pull requests.

Expected Result: The current local verification commands run automatically in CI for `dev`, `main`, and task branches.

Acceptance Tests:
- `.github/workflows/ci.yml` exists at the repository root.
- `.github/workflows/ci.yml` runs on pushes to `dev`, `main`, and `task/**`, plus pull requests.
- `.github/workflows/ci.yml` runs `pnpm --filter web typecheck`.
- `.github/workflows/ci.yml` runs `pnpm --filter web test --run`.
- `.github/workflows/ci.yml` runs `go test ./...` in `services/api`.
- `.github/workflows/ci.yml` runs `go test ./...` in `services/judge`.
- `.github/workflows/ci.yml` runs `./db/scripts/verify_initial_schema.sh`.

Notes:
- Completed by adding `.github/workflows/ci.yml` with separate web, Go service, and database verification jobs for pushes and pull requests.
- Recorded the new `dev` integration-branch workflow in both `AGENTS.md` and `KANBAN.md`, and added the next planned API tasks to `KANBAN.md`.
- Kept the CI commands aligned with the already-verified local commands documented in `docs/DEVELOPMENT.md`.
- Verified `pnpm --filter web typecheck`, `pnpm --filter web test --run`, `go test ./...` in `services/api`, `go test ./...` in `services/judge`, and `./db/scripts/verify_initial_schema.sh` locally.

### JUDGE-01 - Prove sandbox execution
Description: Build a small technical spike that compiles and runs C++17 and Python in an isolated environment with enforced resource limits.

Expected Result: A working prototype that proves the judging model is feasible before broader product work continues.

Acceptance Tests:
- A documented local command exists to run the spike.
- The spike can return `Accepted` for a correct C++17 submission.
- The spike can return `Wrong Answer` for an incorrect submission.
- The spike can return `Compile Error` for invalid C++17 code.
- The spike can return `Time Limit Exceeded` for an intentionally slow program.
- The spike can run a Python submission under the same isolation flow.

Notes:
- Completed by adding `services/judge/internal/spike` plus `cmd/judge-spike`, a Docker-based local runner that compiles and executes submissions with network disabled and resource limits.
- Added built-in fixture scenarios for `C++17` and `Python`, covering `Accepted`, `Wrong Answer`, `Compile Error`, and `Time Limit Exceeded`.
- Documented the local spike command in `services/judge/README.md` and `docs/DEVELOPMENT.md` as `go run ./cmd/judge-spike --all` from `services/judge`.
- Verified `go test ./...` and `go run ./cmd/judge-spike --all` in `services/judge`.

### DB-01 - Define initial schema
Description: Design the first database schema for users, roles, problems, problem versions, submissions, and moderation state.

Expected Result: A schema document or first migration set that supports the MVP flows without contest features.

Acceptance Tests:
- The schema defines users and role-aware access control.
- The schema defines problem lifecycle states including `draft`, `in_review`, `published`, and `archived`.
- The schema defines submissions linked to a stable problem version.
- The schema excludes contest-only tables from v1.

Notes:
- Completed by adding `db/migrations/0001_initial_schema.sql` with users, role grants, problems, stable problem versions, tags, submissions, submission jobs, and per-test submission results.
- Stored the moderation lifecycle as the `problem_version_status` enum and linked each submission directly to `problem_versions.id` so old submissions remain reproducible.
- Added `db/scripts/verify_initial_schema.sh` to apply the migration to the local PostgreSQL container and verify the MVP schema rules automatically.
- Verified the schema with `./db/scripts/verify_initial_schema.sh`, including the absence of contest tables.

### TEST-01 - Bootstrap automated test tooling
Description: Establish the first test runners, test directory conventions, and developer commands for the frontend, API, and judge services.

Expected Result: The repository has an initial automated testing foundation and documented commands that future tasks can use for continuous verification.

Acceptance Tests:
- The repository includes test runner configuration for the frontend, API, and judge code once those packages exist.
- A developer-facing document lists the exact commands for running focused tests and broader verification.
- The setup supports adding unit and integration tests without restructuring the repo.

Notes:
- Completed by adding a minimal `Next.js` frontend scaffold with `Vitest`, `jsdom`, and `Testing Library`, plus a passing component test.
- Added initial Go unit-test targets for API submission statuses and judge language support so both services have real test coverage instead of empty runners.
- Documented the exact focused and broad verification commands in `docs/DEVELOPMENT.md` and recorded the remaining gap in `TESTING_RULES.md`: integration and end-to-end coverage still need to be added.
- Verified `pnpm --filter web typecheck`, `pnpm --filter web test --run`, `go test ./...` in `services/api`, and `go test ./...` in `services/judge`.

### REPO-01 - Bootstrap repository layout
Description: Create the initial monorepo structure for the web app, Go API, Go judge worker, shared docs, and infrastructure config.

Expected Result: A minimal but usable repo layout that matches the architecture and can hold the first implementation slices without reorganization.

Acceptance Tests:
- Root directories exist for the web app, API service, judge service, and infrastructure or local environment config.
- The root includes a developer-facing file that explains how to start the local stack.
- The layout matches the service boundaries defined in the architecture document.

Notes:
- Completed by creating the initial monorepo skeleton under `apps/web`, `services/api`, `services/judge`, `db`, `infra`, and `docs`.
- Added executable local shared infrastructure with `infra/docker-compose.yml` for PostgreSQL and MinIO, plus developer-facing setup docs in `README.md` and `docs/DEVELOPMENT.md`.
- Added a root pnpm workspace and Go workspace so the next tasks can attach real app code and test tooling without restructuring.
- Verified the Compose file with `docker compose -f infra/docker-compose.yml config` and smoke-checked both Go service placeholders with `go test ./...`.

### QUAL-01 - Define testing and verification rules
Description: Document how the project will use automated tests, requirement traceability, and continuous verification during development.

Expected Result: A markdown document and workflow updates that make testing a required part of future implementation work.

Acceptance Tests:
- `TESTING_RULES.md` exists at the repository root.
- `TESTING_RULES.md` contains sections named `Purpose`, `Principles`, `Verification Workflow`, `Requirement Coverage Map`, and `Current Gaps`.
- `AGENTS.md` states that meaningful behavior should get automated tests when practical and that `TESTING_RULES.md` is the traceability source.
- `KANBAN.md` rules state that tasks should add or update automated tests when practical and document uncovered gaps.

Notes:
- Completed by creating `TESTING_RULES.md` with the initial project-wide testing policy and requirement coverage map.
- Recorded the current testing gap explicitly: the repo still has no application code or runnable test commands, so the first code bootstrap must establish the actual tooling.

### ARCH-01 - Define service boundaries
Description: Document the initial deployment units and the request flow between web, API, database, object storage, and judge worker.

Expected Result: A concise architecture document that defines the first production-ready system boundary decisions for v1.

Acceptance Tests:
- `ARCHITECTURE.md` exists at the repository root.
- `ARCHITECTURE.md` names the initial deployable units for v1.
- `ARCHITECTURE.md` defines the submission flow from user request to stored verdict.
- `ARCHITECTURE.md` states that untrusted code does not run inside the main web or API process.
- `ARCHITECTURE.md` names the storage location for hidden tests and verdict artifacts.

Notes:
- Completed by creating `ARCHITECTURE.md` with the initial deployable units: `web`, `api`, `judge`, `postgres`, private `S3`-compatible object storage, and managed `auth`.
- Chose a database-backed submission queue in `postgres` for the first implementation so local development stays simple and cloud migration does not require changing service boundaries.
- Recorded that hidden tests live in private object storage and that large compile or runtime artifacts can also live there instead of in the public app or git.

### PLAN-01 - Lock MVP scope
Description: Define the exact v1 scope so the project stays focused on the core practice and judging loop instead of drifting into full Codeforces feature parity.

Expected Result: A short scope document that defines goals, non-goals, roles, core flows, and initial technology decisions for v1.

Acceptance Tests:
- `MVP_SCOPE.md` exists at the repository root.
- `MVP_SCOPE.md` contains sections named `Goals`, `Non-Goals`, `User Roles`, `Core Flows`, `Technology Decisions`, and `Success Criteria`.
- `MVP_SCOPE.md` explicitly includes problem solving, submissions, verdicts, and moderated user-created problems in scope.
- `MVP_SCOPE.md` explicitly excludes contests, rating, hacks, plagiarism detection, and discussion forums from v1.
- `MVP_SCOPE.md` defines the initial supported languages as `C++17` and `Python`.

Notes:
- Completed by creating `MVP_SCOPE.md` with the agreed v1 scope, non-goals, roles, flows, and stack direction.
- Recorded the current high-level technology choices: Next.js frontend, Go API, Go judge worker, PostgreSQL, Docker-based local development, and Markdown plus LaTeX problem statements.
