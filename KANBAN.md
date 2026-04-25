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

### WEB-UX-01 - Simplify problemset browsing
Description: Rework the published problem list so it feels like a user-facing problemset instead of an internal catalog, including removing slug as a primary column and improving the scan path around titles and limits.

Expected Result: The home page emphasizes problem titles and solving context first, while still preserving essential metadata such as limits and supported languages.

Acceptance Tests:
- The published problem list no longer renders slug as a primary table column on `/`.
- Problem titles remain the primary clickable entry point into `/problems/[slug]`.
- Time and memory limits remain visible in the list view.
- Problem list tests cover the revised table structure or metadata rendering.

Notes:
- Planned from the frontend UX review after local interactive auth work; this is intended as UX polish, not a visual redesign.

### WEB-AUTHOR-01 - Add authoring preview and readiness cues
Description: Improve the draft authoring experience with live markdown preview, clearer validation cues, and visible readiness checks before submit-for-review.

Expected Result: Authors can preview statements before moderation, understand missing requirements earlier, and submit for review with more confidence.

Acceptance Tests:
- Draft authoring pages expose live or near-live preview for the statement sections.
- Hidden test bundle state remains visible while editing.
- The UI surfaces clear readiness cues before enabling or encouraging submit-for-review.
- Relevant authoring tests cover the new preview and validation affordances.

Notes:
- The current authoring screen already has the data needed for a stronger UX but offers little preview or structured readiness guidance.

### SUB-BE-01 - Add incremental judge progress
Description: Extend the submission pipeline so the backend can expose partial judging progress, such as current test progress or partial case results, before a submission fully completes.

Expected Result: The frontend can eventually show richer live submission progress than simple queued/running/final verdict polling.

Acceptance Tests:
- The judge persists partial progress during execution instead of only writing all case results at completion.
- The API exposes a safe progress model for running submissions.
- The frontend can render incremental progress on `/submissions/[id]` without waiting for final completion.
- Relevant service and frontend verification covers the new progress contract.

Notes:
- This is intentionally lower priority than redirect-plus-polling because it requires backend and judge changes, not just frontend UX work.

## Done

### WEB-POLISH-01 - Improve navigation, retries, and mobile behavior
Description: Tighten common frontend interaction quality by replacing client-side hard reload links where appropriate, adding retry and recovery affordances, and improving dense views on mobile.

Expected Result: The app feels faster, less brittle, and more usable across common page transitions and narrow screens.

Acceptance Tests:
- Internal navigation in client-rendered views uses `next/link` or equivalent client transitions where appropriate.
- Major error or empty states offer retry or next-step actions instead of dead ends.
- Problem list, submissions, and moderation views remain usable on small screens without relying only on wide desktop tables.
- Protected routes do not strand cold loads on generic `Loading ...` shells when a clearer sign-in gate or next step is available.
- Relevant web tests cover the updated navigation and at least one improved retry state.

Notes:
- Completed by replacing the remaining client-rendered hard-reload anchors with `next/link` transitions in the problem list, problem detail, submission history, and other shared web views.
- Major error and empty states now include retry or recovery actions across the problem list, drafts index, submission history, moderation queue, submission detail, and draft authoring flows.
- Problem list and submission history now render dedicated mobile card layouts so narrow screens are not forced through the desktop tables, while moderation kept its existing stacked single-column layout and gained clearer recovery actions.
- Protected auth-gated views now use page-specific “checking access” copy with clear next steps instead of dropping cold loads onto generic loading-only shells.
- Verified with `pnpm --filter web test --run src/components/problem-list-page.test.tsx src/components/submission-history-page.test.tsx src/components/problem-detail-page.test.tsx src/components/moderation-problem-drafts-page.test.tsx src/components/problem-drafts-page.test.tsx src/components/problem-authoring-page.test.tsx app/layout.test.tsx`, `pnpm --filter web test --run`, and `pnpm --filter web typecheck`.

### WEB-DRAFTS-01 - Add a drafts index and fix information architecture
Description: Add a drafts management page and stop using the `Drafts` navigation item to mean only “create new draft”.

Expected Result: Authors can browse, reopen, and manage their existing drafts from a dedicated drafts index.

Acceptance Tests:
- A `/drafts` route exists and lists the current user's drafts with relevant status metadata.
- The primary navigation links `Drafts` to the drafts index instead of directly to `/drafts/new`.
- The UI still offers an obvious `New draft` entry point.
- Required API support and frontend tests are added if the current backend contract does not already support draft listing.

Notes:
- Completed by adding an authenticated owner-only `GET /v1/problem-drafts` API route plus OpenAPI coverage for draft summaries with lifecycle, updated, and submitted-for-review metadata.
- `apps/web/src/components/problem-drafts-page.tsx` now powers a dedicated `/drafts` index with sign-in, empty, and populated states plus direct reopen links back into `/drafts/[slug]`.
- The primary app navigation now points `Drafts` to `/drafts`, while the drafts index and authoring flow still surface clear `New draft` entry points.
- Verified with `go test ./...` in `services/api`, `pnpm --filter web test --run app/layout.test.tsx src/components/problem-drafts-page.test.tsx src/components/problem-authoring-page.test.tsx`, `pnpm --filter web test --run`, and `pnpm --filter web typecheck`.

### WEB-AUTH-02 - Add real Clerk auth controls to the app shell
Description: Expose real Clerk sign-in, sign-up, and signed-in account controls in the global header so authentication is discoverable without relying only on route-level auth gates.

Expected Result: When Clerk mode is enabled, signed-out users can start authentication directly from the header and signed-in users can see account state plus an obvious sign-out or account-management entry.

Acceptance Tests:
- With `NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY` configured and local-test auth disabled, the app shell shows `Sign in` and `Create account` entry points while signed out.
- Starting auth from the header opens the configured Clerk flow and exposes the Clerk-managed providers already enabled for the instance.
- After sign-in, the header renders a signed-in account state with a clear account or sign-out action.
- Local-test auth mode still renders the existing local test auth controls instead of the Clerk header controls.
- Relevant auth UI tests cover both Clerk-mode and local-test-mode header behavior.

Notes:
- Completed by replacing the header’s mode-specific gap with a shared auth control surface that switches cleanly between disabled, local-test, and Clerk auth modes.
- Clerk mode now exposes `Sign in` and `Create account` directly in the app shell, while signed-in Clerk sessions render compact account state plus an explicit `Sign out` action.
- Verified in-browser that the header controls open the expected Clerk sign-in and sign-up modal flows and surface the configured GitHub, Google, email-or-username, and sign-up options.
- Verified with `pnpm --filter web test --run src/components/auth.test.tsx`, `pnpm --filter web test --run`, `pnpm --filter web typecheck`, and live browser checks against the Clerk-configured local stack.

### WEB-SUB-01 - Redirect to a live submission page after submit
Description: Change the solve flow so successful submissions navigate to `/submissions/[id]` and that page live-refreshes until the submission reaches a terminal verdict.

Expected Result: Users submit once and land on a dedicated live status page instead of staying on the problem page with only a small verdict badge.

Acceptance Tests:
- Submitting from `/problems/[slug]` redirects the browser to `/submissions/[id]` after a `201` response.
- `/submissions/[id]` polls the existing submission detail endpoint while status is `queued` or `running`.
- The submission page updates automatically to the final verdict and renders final per-test results without manual reload.
- Relevant web tests cover the redirect and live-refresh behavior.

Notes:
- Completed by redirecting the solve workspace to `/submissions/[id]` immediately after a successful submission create response instead of keeping live status on the problem page.
- `apps/web/src/components/submission-summary-page.tsx` now polls the existing submission detail endpoint until the verdict becomes terminal and updates the final verdict plus per-test rows without a manual reload.
- The submission page now surfaces a live-status callout while judging is still in flight and keeps direct navigation back to the problem and submission history.
- Verified with `pnpm --filter web test --run src/components/solve-workspace.test.tsx src/components/submission-summary-page.test.tsx`, `pnpm --filter web typecheck`, and `pnpm verify:smoke:e2e`.

### WEB-SOLVE-01 - Improve solve workspace feedback and safety
Description: Improve the solve workspace so it is harder to lose work and easier to move between solving, active submissions, and history.

Expected Result: The solve workspace provides clearer submission feedback, preserves work more safely, and offers stronger continuity between editing and result inspection.

Acceptance Tests:
- Switching languages does not silently wipe edited source code.
- The solve workspace exposes a clear path to the latest submission detail or submission history.
- Submission failures render more actionable messages than the current generic error copy.
- Relevant solve workspace tests cover the new behavior.

Notes:
- Completed by changing `apps/web/src/components/solve-workspace.tsx` to keep separate source buffers per language instead of resetting editor contents to the starter template on every language switch.
- The workspace now persists the active buffers plus the latest submission id in browser storage per problem, so a user can reopen the latest run and continue editing without silently losing the prior buffer.
- Submission errors now render specific guidance for invalid language, expired auth, missing problems, and temporary pipeline unavailability instead of a single generic failure string.
- Verified with `pnpm --filter web test --run src/components/solve-workspace.test.tsx src/components/submission-summary-page.test.tsx`, `pnpm --filter web typecheck`, and `pnpm verify:smoke:e2e`.

### WEB-LOCAL-01 - Make local moderation usable
Description: Fix the local moderation developer path so the checked-in local test moderator identity, staff-role bootstrap guidance, and frontend moderation experience line up.

Expected Result: Local moderation no longer feels broken; developers can either moderate successfully after the documented setup or see precise guidance about what local role setup is missing.

Acceptance Tests:
- Local development docs reference the actual local test auth subjects used by the browser auth controls.
- A locally granted moderator can load `/moderation/problem-drafts` and apply moderation decisions.
- A non-moderator still receives a clear forbidden state with actionable local setup guidance.
- Relevant moderation UI tests and API verification pass.

Notes:
- The current local mismatch is that browser local test auth uses `user_e2e_moderator`, while the documented grant examples still reference different subjects.
- Local development docs and API docs now reference `user_e2e_author` and `user_e2e_moderator` for the checked-in browser auth controls.
- The moderation `403` state now renders local-test diagnostics plus exact bootstrap and grant commands instead of only generic forbidden copy.
- Verified with `pnpm --filter web typecheck`, `pnpm --filter web test --run src/components/moderation-problem-drafts-page.test.tsx`, `go test ./...` in `services/api`, and `pnpm verify:smoke:e2e`.

### OPS-04 - Add local test auth support for interactive local runs
Description: Extend the local development runtime so browser submissions and other authenticated flows can work without a real Clerk setup.

Expected Result: Developers can opt into a checked-in local test auth mode, sign in through the browser, and use authenticated workflows such as solution submission against the local stack.

Acceptance Tests:
- The local runtime can opt into a checked-in local test auth mode without external Clerk credentials.
- The web app offers a working browser sign-in path when local test auth mode is enabled.
- The API accepts the corresponding local test tokens in that mode.
- Developer-facing docs explain how to start and use the local test auth path.
- Relevant verification commands pass.

Notes:
- Completed by adding API-side `LOCAL_TEST_AUTH_ENABLED` defaults for the checked-in local test issuer, audience, and RSA public key, so the Go API can validate the existing local test JWTs without any external JWKS or Clerk project.
- Added a local web session route at `app/api/local-test-auth/session` plus local auth session controls in the header, so the browser can sign in as the checked-in `author` or `moderator` profiles and set the existing local test auth cookies interactively.
- Wired the local test auth flags through the web Docker build, web runtime env, API runtime env, and `infra/full-stack.env.example`, while keeping the default local Compose path public-only unless the developer explicitly enables the new mode.
- Updated the root, development, deployment, web, and API docs so the local authenticated path is now documented for interactive browser submissions without real Clerk credentials.
- Verified `go test ./...` in `services/api`, `pnpm --filter web typecheck`, `pnpm --filter web test --run`, `pnpm --filter web build`, and `pnpm verify:runtime`.

### OPS-02 - Make local container startup one command
Description: Improve the checked-in local container runtime so `docker compose up --build` prepares dependencies, migrates the database, seeds official starter problems, and starts the runnable stack without extra manual startup steps.

Expected Result: Local container startup becomes a one-command path for public problem browsing and judge-backed runtime verification.

Acceptance Tests:
- `docker compose -f infra/docker-compose.yml --env-file infra/full-stack.env.example up --build` starts the local stack without separate manual migration or seed commands.
- The Compose runtime prepares the pinned judge runtime images before the long-running judge worker starts.
- The local stack exposes seeded published problems after startup.
- Developer-facing docs describe the new one-command container workflow and any remaining auth limitations.
- Relevant verification commands pass.

Notes:
- Completed by extending `infra/docker-compose.yml` with one-shot `migrate`, `judge-image-prep`, and `seed-starter-problems` services so `docker compose ... up --build` can prepare the local runtime automatically before `api`, `judge`, and `web` start.
- Updated the API image to include the `seed-starter-problems` binary so the Compose seed service can run without host Go tooling.
- Added health checks plus `depends_on` conditions so the public web stack waits for database migrations, hidden bundle bucket creation, starter problem seeding, and pinned judge image preparation.
- Made the exported host ports configurable through `infra/full-stack.env.example` so the one-command startup can still work on machines already using the default local ports.
- Updated the root and developer-facing docs to make the one-command Compose path the default containerized local startup flow.

### DOCS-01 - Refresh project documentation and local run guidance
Description: Refresh the top-level project documentation so the current architecture, feature surface, and local run paths match the repository as it exists today.

Expected Result: A new contributor can read the checked-in docs and understand what the project does, how the services fit together, and how to run the stack locally without guessing at stale setup steps.

Acceptance Tests:
- `README.md` reflects the current product surface instead of the old bootstrap state.
- `ARCHITECTURE.md` matches the current API, judge, storage, and diagnostic responsibilities.
- Developer-facing docs clearly describe the recommended local run paths for interactive development, full Compose runtime, and authenticated smoke verification.
- Stale project documentation such as outdated styling or auth assumptions is removed or updated.
- Relevant readmes and documentation index files point to the current guides.

Notes:
- Completed by rewriting `README.md`, `ARCHITECTURE.md`, `docs/DEVELOPMENT.md`, and `docs/DEPLOYMENT.md` around the current feature set and the actual local run paths used by the repository today.
- Updated `docs/README.md`, `apps/web/README.md`, `db/README.md`, `services/judge/README.md`, and `MVP_SCOPE.md` so the smaller documentation surfaces match the current runtime, starter seeds, diagnostics, and auth model.
- Added clearer instructions for the three common local modes: interactive public-stack development, full Compose runtime, and authenticated smoke verification without external auth setup.

### SEED-02 - Expand official starter problem set
Description: Add more published official starter problems so new users can explore a broader range of supported problem types immediately.

Expected Result: The public problem list contains a larger verified starter set with real hidden tests and varied verdict coverage.

Acceptance Tests:
- Additional official starter problems exist in repeatable seed data.
- The seeded set covers more than the current starter problems and includes varied examples or difficulty.
- Each added seeded problem has a verified hidden test bundle.
- Public problem list endpoints return the expanded set.
- Relevant verification commands pass.

Notes:
- Completed by expanding `services/api/internal/starterproblems` from two official starter problems to five: `a-plus-b`, `reverse-string`, `count-positives`, `palindrome-check`, and `running-sum`.
- Added varied starter coverage across arithmetic, counting, string symmetry, and prefix-sum style output, with hidden test bundles generated and checksum-verified for every seeded problem.
- Tightened `services/api/internal/starterproblems/starterproblems_test.go` so the seed set must include the expanded slug list and every hidden bundle must decode and match its stored SHA-256.
- Updated `./infra/scripts/verify_api_runtime_smoke.sh` so the API smoke now proves the public problem list returns the expanded seed set and that a new seeded problem detail can be read successfully.
- Verified `go test ./...` in `services/api` and `./infra/scripts/verify_api_runtime_smoke.sh`.

### E2E-01 - Add end-to-end workflow smoke tests
Description: Add end-to-end smoke coverage for the core user and moderation workflows across the real local stack.

Expected Result: The repository has repeatable smoke tests that prove the main product loops work across `web`, `api`, `postgres`, `object-storage`, and `judge`.

Acceptance Tests:
- A documented command exists to run the end-to-end smoke suite locally.
- The suite proves a signed-in user can open a published problem, submit code, and reach a final verdict.
- The suite proves a signed-in user can create a draft and submit it for review.
- The suite proves a moderator can approve a reviewed draft and make it visible in the public problem list.
- The suite runs in CI or has a follow-up task recorded if CI execution is not yet practical.

Notes:
- Completed by adding `./infra/scripts/verify_e2e_workflow_smoke.sh`, which starts temporary PostgreSQL and MinIO containers, serves a local JWKS fixture, boots the API and judge worker, builds and starts the web app, and runs the browser smoke through the official Playwright Docker image.
- Added a test-only local web auth adapter gated by `NEXT_PUBLIC_LOCAL_TEST_AUTH_ENABLED` so the real Next.js app can exercise signed-in author and moderator flows without depending on an external Clerk environment.
- Added checked-in local auth fixtures under `infra/testdata/local_test_auth/` plus `infra/scripts/run_e2e_smoke.mjs`, which proves published problem browsing, draft creation, submit-for-review, moderator approval, public visibility, and final-verdict submission behavior.
- While implementing the smoke, fixed the judge sandbox bind-mount flag so worker submissions no longer fail with an invalid Docker `--mount` argument, and fixed problem draft creation to return rows reliably by joining the `inserted_problem` CTE instead of the base `problems` table.
- Wired the browser smoke into `.github/workflows/ci.yml` and documented the local command in `docs/DEVELOPMENT.md` and `docs/DEPLOYMENT.md`.

### QUAL-02 - Add integration coverage and refresh testing rules
Description: Add higher-level integration verification for the core platform flows and refresh `TESTING_RULES.md` so it matches the current project state.

Expected Result: The testing policy reflects current capabilities, and the repository gains integration coverage for important cross-service behavior.

Acceptance Tests:
- `TESTING_RULES.md` no longer lists stale gaps that have already been addressed.
- The requirement coverage map includes the current moderation, hidden bundle, and judge failure behaviors.
- New integration checks cover at least one cross-service submission flow and one hidden-test or moderation rule.
- The new integration verification command is documented in developer-facing docs.
- Relevant verification commands pass.

Notes:
- Completed by adding `./infra/scripts/verify_platform_integration.sh` plus the root command `pnpm verify:integration`, which starts temporary PostgreSQL and MinIO containers, serves the checked-in JWKS fixture, boots the API and judge worker, and verifies a moderated problem plus accepted submission through the real services.
- The new integration flow proves an in-review draft stays out of the public problem list until moderator approval, then becomes visible and can be solved through the authenticated submission pipeline with persisted per-test results.
- Updated `.github/workflows/ci.yml` so the existing `integration-smoke` job now runs both the API runtime smoke and the higher-level platform integration verification.
- Refreshed `TESTING_RULES.md` so the requirement map now reflects current moderation visibility checks, hidden bundle handling, browser smoke coverage, and the remaining gap around full multi-service lease-recovery or `judge_failed` integration coverage.

### JUDGE-05 - Store and expose compile and runtime artifacts
Description: Persist useful compile and runtime artifacts from the judge so users and staff can inspect failures without accessing judge hosts directly.

Expected Result: Submission detail responses can include safe references or excerpts for compile and runtime diagnostics.

Acceptance Tests:
- The judge stores compile logs or runtime artifacts for failed submissions when useful output exists.
- The API exposes safe artifact metadata or excerpts on owned submission detail reads.
- The web submission detail UI renders compile-error or runtime diagnostics when present.
- Hidden tests and other private data are not exposed through the new artifact flow.
- `go test ./...` passes in `services/judge`.
- `go test ./...` passes in `services/api`.
- `pnpm --filter web typecheck` passes.
- `pnpm --filter web test --run` passes.

Notes:
- Completed by adding `db/migrations/0003_submission_artifact_excerpts.sql`, which extends `submissions` with `compile_output_excerpt` so compile failures can persist a safe top-level diagnostic without exposing hidden test inputs or other private judge state.
- Updated the judge worker to persist clipped compile output excerpts for `compile_error` submissions while continuing to store per-test stdout and stderr excerpts for runtime failure diagnostics.
- Extended API submission detail reads and the OpenAPI contract to return `compileOutputExcerpt` on owned submission detail responses.
- Updated the web submission detail page to render a dedicated compile-output panel when the API returns compiler diagnostics, while preserving the existing per-test runtime diagnostic rendering.
- Verified `go test ./...` in `services/judge`, `go test ./...` in `services/api`, `pnpm --filter web typecheck`, `pnpm --filter web test --run`, `./db/scripts/verify_initial_schema.sh`, and `./db/scripts/verify_migration_workflow.sh`.

### ADMIN-01 - Add staff role bootstrap tooling
Description: Add an operational path to assign and manage `moderator` and `admin` roles for local development and early production operations.

Expected Result: Staff roles can be granted safely without manual database edits.

Acceptance Tests:
- A documented admin-only command, script, or endpoint exists to grant staff roles.
- Local development setup documents how to bootstrap the first moderator or admin.
- Non-admin users cannot grant privileged roles.
- Automated tests cover the authorization behavior for staff role assignment if an API route is introduced.
- Relevant verification commands pass.

Notes:
- Completed by adding `services/api/cmd/grant-staff-role`, a Go command that bootstraps the first admin only while no admin exists and then requires an existing admin requester subject for later `moderator` or `admin` grants.
- Added `services/api/internal/staffroles` to centralize the grant rules, so non-admin requesters are rejected and bootstrap-specific validation is covered by unit tests without expanding the HTTP API surface.
- Extended `services/api/internal/users` with role-count and role-grant helpers used by the new command and added store tests for the new database operations.
- Documented the bootstrap and follow-up grant commands in `docs/DEVELOPMENT.md` and `services/api/README.md` for local development and early operations.
- Verified `go test ./...` in `services/api` and a temporary-PostgreSQL command flow that bootstraps the first admin, grants a moderator from that admin subject, and rejects a non-admin attempt to grant `admin`.

### AUTH-OPS-01 - Harden production auth validation and transport rules
Description: Tighten API authentication for deployed environments by clarifying token transport, validating production JWT claims, and documenting key-rotation behavior.

Expected Result: Production authentication behavior is explicit, testable, and suitable for long-lived deployments.

Acceptance Tests:
- Production token validation checks the required issuer and expected authorized-party or audience claims.
- The repository documents or implements a key-rotation strategy such as JWKS-backed verification.
- Browser auth transport rules are explicit and consistent between header and cookie-based flows.
- Automated tests cover invalid issuer or audience behavior if those checks are introduced.
- `go test ./...` passes in `services/api`.

Notes:
- Completed by extending `ClerkConfig` with issuer, JWKS URL, and allowed audience support, plus automatic JWKS URL derivation from `CLERK_ISSUER`.
- Hardened the verifier to reject invalid issuer, invalid `azp`, and invalid `aud` claims, and to support JWKS-backed key refresh when a token presents a new `kid`.
- Kept both bearer-header and `__session` cookie transport paths, with explicit header precedence when both are present.
- Added auth unit coverage for cookie transport, header precedence, invalid issuer, invalid audience, JWKS key rotation, and JWKS URL derivation, plus route-level cookie transport coverage in API server tests.
- Updated API and deployment docs to make the production auth inputs and transport rules explicit.
- Verified `go test ./...` in `services/api`.

### CI-03 - Expand CI for builds, integration, and security checks
Description: Extend CI beyond unit tests so it verifies deployable builds, integration behavior, and basic security hygiene for the production path.

Expected Result: CI covers the most important production regressions before changes reach shared branches.

Acceptance Tests:
- CI builds the production web app.
- CI builds the API and judge deployable artifacts.
- CI runs at least one integration or end-to-end smoke job against backing services.
- CI runs dependency or image security checks, or records an explicit follow-up if that is not yet practical.
- Workflow docs stay aligned with the new CI behavior.

Notes:
- Completed by extending `.github/workflows/ci.yml` to build the production web app, run the existing image packaging job, add an API backing-services smoke job, and add a Go dependency security job.
- Added `infra/scripts/verify_api_runtime_smoke.sh`, which starts temporary PostgreSQL and MinIO containers, applies migrations, seeds starter problems, boots the API with required backing services enabled, and verifies published problem reads.
- Added `infra/scripts/verify_go_vulnerabilities.sh`, which installs and runs `govulncheck` against the API and judge Go modules.
- Bumped the pinned Go toolchain and Go build images to `1.25.9` so the new security gate runs against the patched standard library version instead of the older vulnerable `1.25.0` baseline.
- Updated package scripts and developer-facing docs so the smoke and security checks are reproducible outside CI.
- Verified `./infra/scripts/verify_api_runtime_smoke.sh`, `./infra/scripts/verify_go_vulnerabilities.sh`, `pnpm --filter web typecheck`, `pnpm --filter web test --run`, and `pnpm --filter web build`.

### DB-OPS-01 - Add production-safe migration and recovery workflow
Description: Add a documented and repeatable database migration process for deploys, including rollback guidance and backup or restore procedures.

Expected Result: Database changes can be applied and recovered safely in production without relying on destructive development verification scripts.

Acceptance Tests:
- A versioned migration workflow or runner is documented and checked into the repository.
- Deployment docs define the production migration procedure separately from destructive local schema verification.
- Backup and restore commands, scripts, or documented procedures exist and are verified locally.
- Relevant verification commands pass.

Notes:
- Completed by adding `db/scripts/migrate.sh`, which applies versioned SQL migrations in filename order and records applied filenames plus SHA-256 checksums in `schema_migrations`.
- Added `db/scripts/backup.sh` and `db/scripts/restore.sh` for custom-format PostgreSQL backups and explicit-confirmation restores.
- Added `db/scripts/verify_migration_workflow.sh`, which verifies migration idempotency plus backup and restore behavior against an isolated temporary PostgreSQL container with a dynamic port.
- Updated `db/scripts/verify_initial_schema.sh` to use an isolated temporary PostgreSQL container so destructive schema verification no longer depends on the local shared runtime or port `5432`.
- Documented production migration, backup, and restore procedures in `db/README.md`, `docs/DEPLOYMENT.md`, and `docs/DEVELOPMENT.md`, and added CI coverage for the migration workflow.
- Verified `./db/scripts/verify_initial_schema.sh`, `./db/scripts/verify_migration_workflow.sh`, and `./infra/scripts/verify_runtime_packaging.sh`.

### OPS-01 - Add deployment packaging and full-stack runtime definitions
Description: Add production-oriented packaging for the web, API, and judge services plus a full local runtime definition that matches the intended service boundaries.

Expected Result: The repository can build deployable artifacts consistently and run the full application stack through checked-in runtime definitions.

Acceptance Tests:
- Dockerfiles or equivalent build definitions exist for `apps/web`, `services/api`, and `services/judge`.
- A checked-in local runtime definition exists for `web`, `api`, `judge`, `postgres`, and object storage.
- Developer-facing docs list the required environment variables and secret inputs for the full stack.
- The new build or runtime commands are exercised in CI or have a recorded follow-up if CI execution is not yet practical.

Notes:
- Completed by adding Dockerfiles for `apps/web`, `services/api`, and `services/judge`, plus root package scripts for image builds.
- Expanded `infra/docker-compose.yml` into a full local runtime for `web`, `api`, `judge`, `postgres`, and MinIO object storage, with `infra/full-stack.env.example` documenting local defaults.
- Added `docs/DEPLOYMENT.md` and updated development and service docs with runtime inputs, secret requirements, image build commands, and local judge Docker socket caveats.
- Added `./infra/scripts/verify_runtime_packaging.sh` and a CI `packaging` job that validates runtime packaging metadata and builds all three service images.
- Verified `./infra/scripts/verify_runtime_packaging.sh`, `docker compose -f infra/docker-compose.yml --env-file infra/full-stack.env.example config`, `pnpm --filter web typecheck`, `pnpm --filter web test --run`, `pnpm --filter web build`, `go test ./...` in `services/api`, `go test ./...` in `services/judge`, and local Docker builds for the web, API, and judge images.

### OBS-01 - Add structured observability for API and judge
Description: Add structured logs, service health signals, and metrics so operators can monitor request flow, queue health, worker progress, and failure modes in production.

Expected Result: Operators can understand what the API and judge are doing without debugging from inside hosts or databases directly.

Acceptance Tests:
- The API emits structured request logs that include method, path, status, latency, and a request identifier.
- The judge emits structured logs for claim, retry, completion, and terminal failure paths.
- Metrics or status endpoints expose at least API request counts, submission queue depth, worker outcomes, and worker heartbeat freshness.
- Local development docs describe how to inspect the new logs and metrics.
- Relevant verification commands pass.

Notes:
- Completed by adding API JSON request logging with generated or propagated `X-Request-ID`, request counters, and `GET /metricsz` exposing request totals plus submission queue depth.
- Added judge JSON observability for claim, completion, retry, and terminal failure paths, and exposed `GET /healthz` plus `GET /metricsz` on `JUDGE_OBSERVABILITY_ADDRESS` with worker outcome counters and heartbeat freshness.
- Updated local API, judge, and development docs with log and metrics inspection guidance, and mapped the new observability behavior in `TESTING_RULES.md`.
- Added automated coverage for API request logs, API metrics, queue depth, judge metrics, judge structured log events, and worker observer hooks for completion, retry, and terminal failure paths.
- Verified `go test ./...` in `services/api` and `go test ./...` in `services/judge`.

### JUDGE-07 - Harden judge sandbox for production
Description: Replace the current Docker-spike execution path with a production-ready sandbox model for untrusted code on dedicated judge hosts.

Expected Result: The default judge worker path uses a hardened isolation boundary that is appropriate for hostile submissions in production rather than a local proof-of-concept runner.

Acceptance Tests:
- The default long-running worker path no longer depends directly on the local spike runner for production execution.
- Submissions execute as a non-root user with network disabled and only a tightly scoped writable scratch area.
- Runtime images, toolchains, or sandbox assets are pinned and prepared ahead of execution instead of being pulled opportunistically during job processing.
- Sandbox assumptions and dedicated-host requirements are documented clearly.
- Automated tests or reproducible local verification still prove `accepted`, `wrong_answer`, `compile_error`, and `time_limit_exceeded` handling.
- `go test ./...` passes in `services/judge`.

Notes:
- Completed by adding a new worker-only sandbox runner in `services/judge/internal/sandbox` and wiring `cmd/judge` to use it instead of the local spike runner path.
- The hardened runner now requires preloaded pinned runtime images, enforces `--pull never`, runs as a non-root user, uses a read-only root filesystem, disables network access, and limits writable paths to the scoped workspace plus tmpfs scratch mounts.
- Kept `cmd/judge-spike` as the local proof-of-concept command while documenting the stricter long-running worker path for dedicated judge hosts.
- Added sandbox unit tests that assert pinned images and the key Docker hardening flags used by the worker runner.
- Verified `go test ./...` in `services/judge`, `go test ./...` in `services/api`, and `./db/scripts/verify_initial_schema.sh`.

### API-OPS-01 - Harden API startup, readiness, and HTTP limits
Description: Make the API safer for deployed environments by failing fast on required dependency problems and adding readiness checks, explicit HTTP server timeouts, body limits, and correct browser preflight behavior.

Expected Result: Broken deployments no longer look healthy, and public API traffic is handled with safer defaults and clearer operational signals.

Acceptance Tests:
- The API can be configured to fail startup when required auth, database, or hidden-bundle validation dependencies are unavailable.
- `GET /readyz` reports dependency readiness separately from `GET /healthz` liveness.
- The HTTP server sets explicit read, read-header, write, and idle timeouts.
- Write routes enforce request body size limits suitable for source-code and markdown payloads.
- Browser preflight responses include the methods used by the web app, including `PATCH`.
- Required environment and readiness behavior are documented in developer-facing docs.
- `go test ./...` passes in `services/api`.

Notes:
- Completed by adding startup requirement flags for auth, database, and hidden bundle validation, plus actual PostgreSQL connectivity checks during store initialization and object-storage bucket readiness checks before enabling bundle validation.
- Added `GET /readyz` with explicit dependency status reporting separate from `GET /healthz`, and configured the HTTP server with explicit read, read-header, write, and idle timeouts plus a bounded max header size.
- Added JSON request body limits for draft create or update, moderation decisions, and submission create routes, and updated CORS preflight responses to advertise `PATCH` for the browser draft editor flow.
- Documented the new startup requirement environment variables and readiness behavior in the API and development docs, and updated the OpenAPI document to include `/readyz`.
- Verified `go test ./...` in `services/api` and `pnpm --filter web typecheck`.

### AUTHOR-01 - Add hidden test bundle upload flow
Description: Add an author-facing upload flow so draft creators can upload hidden test bundles directly instead of manually entering object storage metadata.

Expected Result: Draft authors can upload a hidden test bundle from the web app, and the backend stores validated bundle metadata automatically.

Acceptance Tests:
- The web authoring flow includes a hidden test bundle upload action.
- Uploading a valid bundle stores or returns the object key and SHA-256 needed by draft routes.
- Invalid bundles are rejected with a clear validation error.
- Draft submit-for-review continues to reject drafts that do not have a valid hidden test bundle.
- `pnpm --filter web typecheck` passes.
- `pnpm --filter web test --run` passes.
- `go test ./...` passes in `services/api`.

Notes:
- Completed by adding the protected API upload route `POST /v1/problem-drafts/hidden-test-bundles`, object-storage upload support in `services/api/internal/problems/bundles.go`, and OpenAPI plus docs updates for the new flow.
- Hidden bundle uploads are now validated as JSON bundles with at least one test case before being written to object storage, and successful uploads return the generated object key plus SHA-256 checksum used by draft create or update routes.
- The web authoring page now lets users choose a bundle file, upload it directly from the browser, and uses the returned metadata as read-only draft bundle fields instead of requiring manual object-storage values.
- Added API handler and bundle tests plus web authoring tests for successful upload and upload validation failures.
- Verified `go test ./...` in `services/api`, `pnpm --filter web typecheck`, and `pnpm --filter web test --run`.

### WEB-07 - Shift the web UI to a developer-tool visual system
Description: Refactor the frontend visual system so the product feels like a serious competitive-programming tool instead of a generic dark SaaS shell.

Expected Result: The web app uses a clean, dense, dark-first interface that prioritizes tables, code, statements, verdicts, filters, and operational readability.

Acceptance Tests:
- Global visual tokens use flat dark surfaces, subtle borders, minimal shadows, and a single strong accent color.
- The home problem list and submission history views use dense table or compact row-list presentations instead of large glossy cards.
- Verdicts render with a shared compact badge style.
- Problem detail and solve workspace layouts prioritize readable statement content, code, and metadata without marketing-style hero sections.
- `pnpm --filter web typecheck` passes.
- `pnpm --filter web test --run` passes.

Notes:
- Completed by replacing the glossy gradient-heavy shell with flatter dark tokens, tighter spacing, monospace metadata, compact surfaces, and a single violet accent in `apps/web/app/globals.css`.
- Reworked the home problem list and submission history into dense tables, added a real shared verdict badge component, and refreshed the submission detail view to feel more like an operational judge dashboard.
- Updated the problem detail header and solve workspace copy and layout so the core solve flow reads like a practical developer tool instead of a marketing-style landing screen.
- Verified `pnpm --filter web typecheck` and `pnpm --filter web test --run`.

### JUDGE-06 - Add job leases and stuck submission recovery
Description: Replace permanent submission job claims with a lease-based worker flow that can recover abandoned work after judge crashes, restarts, or host loss.

Expected Result: A claimed submission job cannot remain stuck forever; expired claims become recoverable and user-visible submission state progresses to completion or a terminal failure.

Acceptance Tests:
- `submission_jobs` or equivalent worker state includes the data needed to detect and recover stale claims.
- Active workers can renew claims while processing long-running submissions.
- Claimed jobs become reclaimable after lease expiry when a worker stops heartbeating.
- Automated tests cover a stale-claim recovery path and prove the submission does not remain `running` indefinitely.
- Relevant API or worker status handling continues to return correct terminal states for recovered or failed submissions.
- `go test ./...` passes in `services/judge`.
- `go test ./...` passes in `services/api` if shared submission-state behavior changes.

Notes:
- Completed by adding `db/migrations/0002_submission_job_leases.sql` plus schema verification updates so `submission_jobs` now tracks a renewable `lease_token` and `lease_expires_at` for abandoned-claim recovery.
- The judge worker now claims jobs with a lease, renews the lease while processing long-running submissions, and gates completion or failure writes on the active lease token so an expired claimant cannot overwrite a newer worker's result.
- Recovery handling now treats lost leases as reclaimable work instead of permanent failure, while poisoned jobs are parked with `available_at = 'infinity'` so terminal `judge_failed` jobs are not claimed again.
- Added worker tests for lease renewal, lost-lease cancellation, heartbeat failure recording, and lease-aware store behavior.
- Verified `go test ./...` in `services/judge` and `./db/scripts/verify_initial_schema.sh`.

### SEED-01 - Seed official starter problems
Description: Add a small official set of published starter problems with verified hidden tests so the platform is immediately usable.

Expected Result: The public problem list contains a starter set of published problems that can be solved and judged successfully.

Acceptance Tests:
- At least a small starter set of published problems exists in development or seed data.
- Public problem list endpoints return the seeded problems.
- A seeded problem can be submitted against successfully through the real submission flow.
- Relevant verification commands pass.

Notes:
- Completed by adding a repeatable Go seed command in `services/api/cmd/seed-starter-problems` plus `./db/scripts/seed_official_starter_problems.sh` to upload hidden bundles and populate published starter problems.
- Seeded the first official published starter problems `a-plus-b` and `reverse-string` with object-stored hidden tests and deterministic checksums.
- Verified locally that `GET /v1/problems` returned the seeded problems and that a real authenticated submission against `a-plus-b` reached final `accepted` through the judge worker path.
- Verified `go test ./...` in `services/api`.

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
- Completed by adding `/moderation/problem-drafts` with a moderator queue, selected-draft detail view, moderation notes input, and decision actions for approve, reject, and request changes.
- The page now handles moderator access through API-driven `403` responses instead of guessing roles client-side, so non-moderators get a clear forbidden state.
- Added a `Moderation` link to the main header so the review queue is reachable from the app shell.
- Verified `pnpm --filter web typecheck` and `pnpm --filter web test --run`.

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
- Completed by adding `/drafts/new` and `/drafts/[slug]` plus a shared authenticated authoring page that handles draft creation, draft editing, and submit-for-review from the browser.
- Added draft API client helpers and authoring form state for statement fields, limits, and hidden bundle metadata, including inline error handling for server-side validation failures.
- Added a `Drafts` link to the main header so the new authoring flow is discoverable from the app shell.
- Verified `pnpm --filter web typecheck` and `pnpm --filter web test --run`.

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
- Completed by adding moderator-only queue and decision routes that read from `in_review` problem versions and apply `approve`, `reject`, or `request_changes` transitions.
- The moderation decision flow now maps to existing lifecycle states: `approve -> published`, `request_changes -> draft`, and `reject -> archived`.
- Reused the staff role lookup introduced for draft editing so non-moderators receive `403` on moderation routes while staff can still read in-review draft detail through the existing draft read path.
- Verified `go test ./...` in `services/api`.

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
- Completed by adding `POST /v1/problem-drafts/{slug}/submit-for-review` and a store transition that moves an owned draft from `draft` to `in_review` while preserving the validated hidden bundle metadata.
- The transition now rejects submissions that are already in review or still missing hidden bundle metadata so incomplete drafts cannot enter moderation.
- Kept public problem reads unchanged so only `published` versions continue to appear through the public API.
- Verified `go test ./...` in `services/api`.

### DRAFT-API-02 - Add hidden test bundle registration and validation
Description: Add draft-problem support for registering hidden test bundle metadata and validating bundle references before review.

Expected Result: Draft problems can reference real hidden test bundles safely.

Acceptance Tests:
- Draft problem routes can store hidden test bundle metadata.
- Invalid or missing hidden test bundle metadata is rejected.
- Draft versions retain the bundle key and checksum needed by the judge.
- `go test ./...` passes in `services/api`.

Notes:
- Completed by extending draft create and update routes with optional `hiddenTestBundleKey` and `hiddenTestBundleSha256` fields plus API-side object storage validation against local MinIO or other S3-compatible storage.
- Added checksum and object-existence validation before persistence, and preserved existing bundle metadata on draft updates when the request omits those fields.
- Added validator tests for checksum matching and route tests that reject partial or missing bundle metadata while storing validated references for the judge.
- Verified `go test ./...` in `services/api`.

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
- Completed by extending the problem store with owner-scoped draft create, read, and update flows backed by `problems` plus `problem_versions` rows.
- Added staff-aware authorization using `user_roles` lookups so moderators and admins can update another user's draft while non-owners still receive `404`.
- Added `GET /v1/problem-drafts/{slug}` alongside create and update so the next web authoring task can reopen saved drafts without immediate follow-up API work.
- Verified `go test ./...` in `services/api`.

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
- Completed by turning `go run ./cmd/judge` into a real polling worker loop while keeping `--once` for one-shot processing.
- Added retry handling that requeues claimed jobs with `last_error`, then marks submissions as terminal `judge_failed` after the configured max-attempt limit is reached.
- Extended the schema, API status helpers, OpenAPI contract, and web submission status rendering so poison jobs appear as a final user-visible state instead of polling forever.
- Added worker store tests for requeue, poison-job handling, and successful queue deletion plus command-loop tests that prove idle queues do not fail the worker.
- Verified `go test ./...` in `services/judge`, `go test ./...` in `services/api`, `pnpm --filter web typecheck`, `pnpm --filter web test --run`, and `./db/scripts/verify_initial_schema.sh`.

### JUDGE-03 - Load hidden test bundles from object storage
Description: Replace local file-path bundle loading with `S3`-compatible hidden test retrieval so the real worker path matches the intended architecture.

Expected Result: The judge loads hidden tests from private object storage instead of local disk paths.

Acceptance Tests:
- The worker can fetch hidden test bundles from configured object storage using the stored bundle key.
- Local development works with `MinIO`.
- A queued submission can still reach a final verdict using object-stored test data.
- `go test ./...` passes in `services/judge`.

Notes:
- Completed by replacing the local file loader with an S3-compatible bundle loader configured through judge environment variables and defaulting to local MinIO.
- Added SHA-256 verification against `hidden_test_bundle_sha256` before hidden tests are decoded or executed.
- Added loader tests for object fetch and checksum mismatch plus a processor test that completes a submission using object-stored bundle data.
- Verified `go test ./...` in `services/judge`.

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
- Completed by replacing the placeholder submission summary with a richer detail view that shows final verdict, aggregate test counts, and per-test result cards with output excerpts when present.
- The page now handles compile errors and other empty-result cases with explicit messaging instead of rendering a blank section.
- Added detail-page component tests that prove the authenticated fetch path, verdict breakdown rendering, and compile-error empty-state handling.
- Verified `pnpm --filter web typecheck` and `pnpm --filter web test --run`.

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
- Completed by extending `GET /v1/submissions/{id}` from a summary response to a detail response that includes `totalTests`, `passedTests`, and ordered `submission_results` rows.
- Added explicit API/store coverage for owner-scoped reads, aggregate counts, ordered per-test results, and the existing `404` behavior.
- Updated `services/api/openapi/v1.yaml` with `SubmissionDetail` and `SubmissionResult` schemas so the contract matches the stored judge data.
- Verified `go test ./...` in `services/api`.

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
- Completed by adding `/submissions` for authenticated history browsing and `/submissions/[id]` for a minimal submission summary page linked from the history list.
- The history UI handles signed-out, loading, empty, and error states while reusing Clerk auth and React Query for direct API calls.
- Added tests for signed-out behavior, populated history rendering with real links, and empty-history handling.
- Verified `pnpm --filter web typecheck` and `pnpm --filter web test --run`.

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
- Completed by extending the submission store with owner-scoped history queries and adding an authenticated `GET /v1/submissions` route plus OpenAPI contract.
- The history endpoint reuses the existing Clerk-backed user bootstrap flow and returns summary rows ordered newest first.
- Added route tests for `401` and successful history reads plus store tests that verify owner filtering and newest-first ordering.
- Verified `go test ./...` in `services/api`.

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
