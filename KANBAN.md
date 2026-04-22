# KANBAN

## Rules
- Use this file as the source of truth for active and upcoming work.
- Before starting a new implementation batch, pause for planning with the user and ask clarifying questions if anything important is unclear.
- Complete at most two tasks after each planning checkpoint before stopping for another planning pass with the user.
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

## Ready

## Backlog

### SUB-API-02 - Add submission history endpoint
Description: Add an authenticated endpoint that returns the current user's recent submissions with summary metadata for history views.

Expected Result: The API supports personal submission history ordered from newest to oldest.

Acceptance Tests:
- The OpenAPI document defines `GET /v1/submissions`.
- The endpoint returns `401` without auth.
- The endpoint returns only submissions owned by the current user.
- The endpoint orders submissions by newest first.
- Each returned item includes problem slug, language, status, and queued time.
- `go test ./...` passes in `services/api`.

Notes:
- Not started.

### SUB-API-03 - Expand submission detail with result breakdown
Description: Enrich submission detail responses with aggregate counts and per-test case results so the UI can show meaningful feedback after judging.

Expected Result: The API exposes verdict breakdown data for an owned submission, including stored `submission_results` rows.

Acceptance Tests:
- The OpenAPI document defines the per-test result shape on `GET /v1/submissions/{id}`.
- The endpoint returns final status plus aggregate counts for total and passed tests.
- The endpoint returns per-test result items when they exist.
- The endpoint still returns `404` for submissions not owned by the current user.
- `go test ./...` passes in `services/api`.

Notes:
- Not started.

### WEB-03 - Build submission history page
Description: Add a page where signed-in users can review recent submissions and navigate back to the relevant problem or submission detail.

Expected Result: Users can see their own submission history inside the web app.

Acceptance Tests:
- The page fetches the submission history endpoint.
- The page shows problem slug, language, status, and queued time.
- The page links to problem detail and submission detail routes.
- Loading and error states are present.
- `pnpm --filter web typecheck` passes.
- `pnpm --filter web test --run` passes.

Notes:
- Not started.

### WEB-04 - Build submission detail result UI
Description: Add a submission detail screen that renders the final verdict, aggregate counts, and per-test results returned by the API.

Expected Result: Users can inspect an individual submission and understand what failed.

Acceptance Tests:
- The page fetches `GET /v1/submissions/{id}`.
- The page renders the final verdict, total tests, and passed tests.
- The page renders per-test result rows when present.
- The page handles compile errors or empty per-test results gracefully.
- `pnpm --filter web typecheck` passes.
- `pnpm --filter web test --run` passes.

Notes:
- Not started.

### JUDGE-03 - Load hidden test bundles from object storage
Description: Replace local file-path bundle loading with `S3`-compatible hidden test retrieval so the real worker path matches the intended architecture.

Expected Result: The judge loads hidden tests from private object storage instead of local disk paths.

Acceptance Tests:
- The worker can fetch hidden test bundles from configured object storage using the stored bundle key.
- Local development works with `MinIO`.
- A queued submission can still reach a final verdict using object-stored test data.
- `go test ./...` passes in `services/judge`.

Notes:
- Not started.

### JUDGE-04 - Add worker loop and failure handling
Description: Extend the judge from one-off processing to a repeatable worker loop with basic retry accounting and failure recording.

Expected Result: The judge can run continuously and recover cleanly from job errors.

Acceptance Tests:
- A documented long-running worker command exists.
- Failed jobs increment `attempts` and record `last_error`.
- Successful jobs are removed from `submission_jobs`.
- Empty queues do not cause the worker to exit with failure.
- `go test ./...` passes in `services/judge`.

Notes:
- Not started.

### DRAFT-API-01 - Add problem draft create and update endpoints
Description: Add authenticated API routes for users to create and edit draft problems and draft problem versions.

Expected Result: Users can save and update draft problem content in the backend.

Acceptance Tests:
- The OpenAPI document defines draft create and update endpoints.
- Authenticated users can create a draft problem and initial draft version.
- Draft owners can update their draft statement fields and limits.
- Non-owners cannot update another user's draft unless they are staff.
- Public problem endpoints continue to exclude drafts.
- `go test ./...` passes in `services/api`.

Notes:
- Not started.

### DRAFT-API-02 - Add hidden test bundle registration and validation
Description: Add draft-problem support for registering hidden test bundle metadata and validating bundle references before review.

Expected Result: Draft problems can reference real hidden test bundles safely.

Acceptance Tests:
- Draft problem routes can store hidden test bundle metadata.
- Invalid or missing hidden test bundle metadata is rejected.
- Draft versions retain the bundle key and checksum needed by the judge.
- `go test ./...` passes in `services/api`.

Notes:
- Not started.

### DRAFT-API-03 - Add submit-for-review transition
Description: Add the lifecycle transition that moves a user draft from `draft` to `in_review`.

Expected Result: Draft problems can enter the moderation queue through an explicit API action.

Acceptance Tests:
- The OpenAPI document defines a submit-for-review route.
- Draft owners can transition a draft to `in_review`.
- Invalid lifecycle transitions are rejected.
- Public problem endpoints continue to exclude `in_review` versions.
- `go test ./...` passes in `services/api`.

Notes:
- Not started.

### MOD-API-01 - Add moderation queue and decision endpoints
Description: Add moderator-only API routes to list drafts in review and approve, reject, or request changes.

Expected Result: Moderators can control the publish flow for user-created problems.

Acceptance Tests:
- The OpenAPI document defines moderation queue and decision endpoints.
- Non-moderators are denied access to moderation routes.
- Moderators can approve, reject, and request changes.
- Approval results in one published version for the problem.
- `go test ./...` passes in `services/api`.

Notes:
- Not started.

### WEB-05 - Build problem authoring UI
Description: Add authenticated web screens for creating and editing draft problems, including statement fields, limits, and hidden test bundle metadata.

Expected Result: Users can create and edit draft problems from the browser.

Acceptance Tests:
- Authenticated users can create a draft problem from the UI.
- Users can edit statement fields, limits, and hidden test bundle metadata.
- Users can submit a draft for review from the UI.
- The UI shows loading and validation error states.
- `pnpm --filter web typecheck` passes.
- `pnpm --filter web test --run` passes.

Notes:
- Not started.

### WEB-06 - Build moderation UI
Description: Add moderator web screens for reviewing queued drafts and applying moderation decisions.

Expected Result: Moderators can review and publish user-created problems from the browser.

Acceptance Tests:
- Moderators can view the review queue in the UI.
- Moderators can approve, reject, or request changes.
- Non-moderators cannot access the moderation screens.
- `pnpm --filter web typecheck` passes.
- `pnpm --filter web test --run` passes.

Notes:
- Not started.

### SEED-01 - Seed official starter problems
Description: Add a small official set of published starter problems with verified hidden tests so the platform is immediately usable.

Expected Result: The public problem list contains a starter set of official problems that can be solved and judged successfully.

Acceptance Tests:
- At least a small starter set of published problems exists in development or seed data.
- Public problem list endpoints return the seeded problems.
- A seeded problem can be submitted against successfully through the real submission flow.
- Relevant verification commands pass.

Notes:
- Not started.

## Done

### WEB-02 - Add authenticated solve and submission UX
Description: Add Clerk-based frontend auth integration, a solve page editor and language picker, submission creation from the browser, and polling until a final verdict is reached.

Expected Result: A signed-in user can submit `C++17` or `Python` code from the browser and see queued, running, and final verdict states.

Acceptance Tests:
- The web app can obtain a Clerk session token and send it to the API.
- The problem detail page includes a language picker for `C++17` and `Python`.
- The solve page includes a code editor area.
- Submitting from the page calls `POST /v1/submissions`.
- After submission creation, the UI polls `GET /v1/submissions/{id}` until a terminal status is reached.
- The UI displays at least `queued`, `running`, `accepted`, `wrong_answer`, `compile_error`, and `time_limit_exceeded`.
- Unauthenticated submission attempts prompt sign-in or block submission clearly.
- `pnpm --filter web typecheck` passes.
- `pnpm --filter web test --run` passes.

Notes:
- Completed by adding a Clerk-aware solve workspace on the problem detail page, including language selection, Monaco editor integration, authenticated submission creation, and verdict polling through React Query.
- Added frontend submission helpers plus starter code templates for `C++17` and `Python`, and documented the required `NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY` environment variable.
- The solve workspace now blocks clearly when auth is unavailable, prompts sign-in when the user is signed out, and shows live `queued`, `running`, and terminal verdict states after submission.
- Verified `pnpm --filter web typecheck` and `pnpm --filter web test --run`.

### WEB-01 - Build published problem browsing UI
Description: Replace the placeholder web page with a real app shell, a published problem list screen, and a problem detail page powered by the existing API.

Expected Result: A user can open the web app, browse published problems, and read a full problem statement with limits and metadata.

Acceptance Tests:
- The home route renders a published problem list screen instead of the placeholder landing page.
- The web app fetches `GET /v1/problems` and renders returned problems.
- A problem detail route exists and fetches `GET /v1/problems/{slug}`.
- The problem detail page renders statement, input, output, constraints, notes, time limit, and memory limit.
- Loading and error states exist for both list and detail fetches.
- `pnpm --filter web typecheck` passes.
- `pnpm --filter web test --run` passes.

Notes:
- Completed by replacing the placeholder landing page with a published problem list and adding `/problems/[slug]` for full published problem detail rendering.
- Added a small browser-facing API client, markdown plus KaTeX rendering for statement sections, and a responsive app shell for the first real web experience.
- Added direct-browser API support by documenting `NEXT_PUBLIC_API_BASE_URL` and enabling API CORS for local web origins.
- Verified `pnpm --filter web typecheck`, `pnpm --filter web test --run`, and `go test ./...` in `services/api` for the new CORS middleware support.

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
- Completed by adding a PostgreSQL-backed judge worker store, a file-based local hidden-test bundle loader, and a processor that claims one queued job, runs it through the existing spike runner, and persists the final verdict plus per-test results.
- Implemented `go run ./cmd/judge --once` as the local worker command and documented it in `services/judge/README.md` and `docs/DEVELOPMENT.md`.
- Added automated worker tests covering `accepted`, `wrong_answer`, `compile_error`, and `time_limit_exceeded` verdict persistence paths.
- Verified `go test ./...` in `services/judge`, and confirmed locally that a queued submission was processed to `accepted`, wrote one `submission_results` row, and was removed from `submission_jobs`.

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
- Completed by adding a PostgreSQL-backed submission store, authenticated handlers for `POST /v1/submissions` and `GET /v1/submissions/{id}`, and OpenAPI definitions for both endpoints.
- Submission creation now resolves the published problem version by slug, inserts one `submissions` row, and inserts one `submission_jobs` row through a single database-backed workflow.
- Added handler tests for `401`, `201`, and `404` cases plus store tests that assert published-only submission creation and owner-scoped reads.
- Verified `go test ./...` in `services/api`.

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
- Completed by adding a PostgreSQL-backed user store and wiring `GET /v1/me` to create or load an app user from the authenticated Clerk subject.
- Temporary profile values are derived deterministically from the Clerk subject, giving each new user a unique placeholder `handle` and `display_name` until profile editing exists.
- Added automated coverage for generated-handle uniqueness, repeated subject reuse in the store, and a route test that accepts a real signed JWT through the Clerk verifier path.
- Verified `go test ./...` in `services/api`.

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
- Completed by moving `pnpm/action-setup` ahead of `actions/setup-node` in the `web` job so `cache: pnpm` can resolve the `pnpm` executable on GitHub runners.
- Recorded the new workflow rules in both `AGENTS.md` and `KANBAN.md`: use Conventional Commits and inspect GitHub Actions after each completed batch.
- Verified `pnpm --filter web typecheck` and `pnpm --filter web test --run` locally.
- Verified the GitHub Actions run on `task/ci-02-fix-pnpm-setup-in-github-actions` completed successfully.

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
