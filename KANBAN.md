# KANBAN

## Rules
- Use this file as the source of truth for active and upcoming work.
- Before starting a new implementation batch, pause for planning with the user and ask clarifying questions if anything important is unclear.
- Complete at most four tasks after each planning checkpoint before stopping for another planning pass with the user.
- Keep exactly one task in `In Progress` unless the user explicitly approves parallel work.
- Use one git branch per task and push each completed task branch to `origin` after its acceptance checks pass.
- Every task must include `Description`, `Expected Result`, `Acceptance Tests`, and `Notes`.
- Acceptance tests must be objective and directly verifiable from files, commands, endpoints, or UI behavior.
- After finishing a task, add implementation details and important discoveries to that task's `Notes` section.
- Add or update automated tests for meaningful behavior whenever practical, and run the smallest relevant verification continuously while implementing.
- Use `TESTING_RULES.md` to keep important project rules and requirements mapped to verification.
- If a task cannot yet be covered by automated tests, record the gap in `Notes` and add follow-up work.

## In Progress

## Ready

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
- Not started.

## Done

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
