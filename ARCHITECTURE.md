# Architecture

## Deployable Units
- `web`: `Next.js` frontend that renders the public site, authenticated app screens, and problem-solving UI.
- `api`: `Go` HTTP API that owns business logic, persistence, authorization, and submission lifecycle state.
- `judge`: separate `Go` worker service that compiles and executes untrusted submissions in isolation.
- `postgres`: primary relational database for users, problems, versions, submissions, moderation, and judge results.
- `object-storage`: private `S3`-compatible storage for hidden test bundles and large judge artifacts. Local development will use `MinIO`.
- `auth`: managed authentication provider used by `web` and verified by `api`.

## Service Responsibilities

### web
- Renders public pages, problem lists, problem detail pages, submission history, and authoring or moderation screens.
- Collects source code submissions and sends them to `api`.
- Never compiles or executes user code.

### api
- Verifies authentication and role-based access.
- Stores problems, problem versions, moderation states, submissions, and verdict summaries in `postgres`.
- Creates submission jobs in a database-backed queue table inside `postgres` for the `judge` to consume.
- Issues short-lived private object storage access for hidden test bundles when needed by trusted services.
- Never compiles or executes user code.

### judge
- Polls or claims queued submissions from the database-backed queue.
- Fetches submission source, language, limits, and problem-version metadata from `api`-owned data in `postgres`.
- Downloads hidden test bundles from private object storage.
- Compiles and runs untrusted code inside a sandbox on dedicated judge hosts only.
- Writes per-test results and final verdicts back to `postgres`.
- Uploads large compile logs and execution artifacts to private object storage when they are too large or noisy for relational storage.

### postgres
- Stores canonical application data.
- Stores submission lifecycle state such as `queued`, `running`, `accepted`, `wrong_answer`, `compile_error`, `runtime_error`, `time_limit_exceeded`, and terminal worker failures like `judge_failed`.
- Stores structured verdict summaries and per-test metadata needed by the product UI.

### object-storage
- Stores hidden test bundles keyed by stable problem version.
- Stores optional large judge artifacts such as compile logs, stderr captures, and sandbox metadata.
- Remains private; end users do not receive direct public access to hidden tests.

## Submission Flow
1. A signed-in user opens a published problem in `web` and submits code in `C++17` or `Python`.
2. `web` sends the submission payload to `api`.
3. `api` validates the request, checks that the problem version is published, creates a `submissions` row in `postgres`, and records the initial status as `queued`.
4. `api` inserts a job for that submission into a database-backed queue in `postgres`.
5. `judge` claims the queued job and marks the submission as `running`.
6. `judge` reads submission metadata from `postgres` and downloads the hidden test bundle for the referenced problem version from private object storage.
7. `judge` compiles or runs the submission inside an isolated sandbox on the judge host.
8. `judge` evaluates program output against the expected results, computes per-test outcomes, and derives the final verdict.
9. `judge` writes per-test results and the final verdict back to `postgres`.
10. `judge` uploads large compile or runtime artifacts to private object storage when needed.
11. `web` reads the updated submission state through `api` and shows the final verdict to the user.

## Isolation Rules
- Untrusted user code runs only inside the `judge` service sandbox, never inside `web` or `api`.
- Judge hosts are the only machines that need compilers, interpreters, and sandbox tooling.
- Hidden tests are stored outside the repo and outside the public frontend bundle.
- Published problems must reference stable problem versions so old submissions remain reproducible.

## Data Ownership
- `postgres` is the source of truth for product state and structured judge results.
- Private object storage is the source of truth for hidden test bundles and oversized judge artifacts.
- `web` is a presentation layer and does not own business-critical state.
- `judge` is a worker and does not own long-term canonical data beyond transient local execution files.

## Local Development And Future Cloud Deployment
- Local development should run `web`, `api`, `judge`, `postgres`, and `MinIO` through `Docker Compose`.
- The same boundaries should carry into cloud deployment: `web` and `api` may move to managed platforms, while `judge` remains isolated on dedicated Linux hosts.
- Keeping storage and execution concerns separate in v1 avoids a major architecture change when moving from local Docker to cloud infrastructure.
