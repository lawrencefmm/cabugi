# AGENTS.md

## Current State
- Planning files now exist: `KANBAN.md`, `MVP_SCOPE.md`, `ARCHITECTURE.md`, and `TESTING_RULES.md`.
- The repository now has a pnpm workspace, a Go workspace, service placeholder modules, and local shared infrastructure config.
- Verified test commands now exist for `pnpm --filter web typecheck`, `pnpm --filter web test --run`, and `go test ./...` in both `services/api` and `services/judge`.
- The database directory now includes an initial PostgreSQL migration and `./db/scripts/verify_initial_schema.sh` for schema verification.
- The git remote is `origin` at `git@github.com:lawrencefmm/cabugi.git`.

## Working Rules
- Use `KANBAN.md` as the source of truth for active and upcoming work.
- Before starting a new batch of implementation work, pause for planning with the user and ask clarifying questions if anything important is unclear.
- After that planning step, complete at most four tasks before stopping for another planning checkpoint with the user.
- Work on exactly one task at a time unless the user explicitly approves parallel work.
- Use one git branch per task.
- After each completed task, run the relevant verification, create a commit, and push that task branch to `origin`.
- Every task in `KANBAN.md` must include `Description`, `Expected Result`, `Acceptance Tests`, and `Notes`.
- Keep acceptance tests objective and directly verifiable from files, commands, endpoints, or UI behavior.
- After completing a task, add a short `Notes` section that records what was implemented and any key discoveries.
- Add automated tests for meaningful behavior whenever practical, and run the smallest relevant verification repeatedly while implementing.
- Use `TESTING_RULES.md` to map project rules and requirements to verification; update it when new important behavior or constraints are introduced.
- If a task cannot yet be covered by automated tests, record the gap in the task notes and create follow-up work instead of silently skipping coverage.

## Project Decisions
- V1 is a competitive programming practice platform, not a full Codeforces clone.
- Initial in-scope features are published problems, submissions, verdicts, submission history, and moderated user-created problems.
- Initial non-goals are contests, ratings, hacks, plagiarism detection, and discussion forums.
- The planned v1 stack is `Next.js` + `TypeScript` frontend, `Go` API, separate `Go` judge worker, `PostgreSQL`, `Docker Compose` for local development, and `Markdown` plus `LaTeX` for problem statements.
- Initial submission language support is `C++17` and `Python`.
- Hidden tests should live in `S3`-compatible object storage rather than the public app bundle or git-tracked fixtures.
- Initial deployable units are `web`, `api`, `judge`, `postgres`, private `S3`-compatible object storage, and managed `auth`.
- Use a database-backed queue in `PostgreSQL` for the first submission job flow.
- Only the `judge` service may run untrusted code, and it must do so on isolated judge hosts.
