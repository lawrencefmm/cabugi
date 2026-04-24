# Testing Rules

## Purpose
- Treat testing as a first-class project requirement, not cleanup after implementation.
- Map important product rules and architecture constraints to explicit verification so regressions are caught early.
- Keep the project shippable by running focused tests during development and broader verification before closing tasks.

## Principles
- Every meaningful feature, rule, bug fix, and security-sensitive behavior should be covered by automated tests when practical.
- Every important project rule or requirement should be traceable to at least one verification step.
- Prefer the smallest test that proves the behavior: unit tests first, then integration tests, then end-to-end tests where the boundary requires it.
- Add regression tests for every bug that is fixed, unless the environment makes that impossible.
- If a requirement is not yet testable, document the gap and create follow-up work instead of silently skipping coverage.

## Verification Workflow
- While implementing a task, run the smallest relevant automated tests repeatedly for the code being changed.
- Before marking a task complete, run the relevant verification for every touched area and record the result in the task notes if it is non-obvious.
- When a task introduces a new rule, permission check, state transition, or API contract, add or update tests that prove it.
- When a task cannot include automated tests yet, record the reason in the task notes and add a follow-up task to close the gap.

## Requirement Coverage Map

| Rule or Requirement | Minimum Verification Expectation |
| --- | --- |
| Only the `judge` service may run untrusted code | Service and integration tests prove `web` and `api` never execute submissions directly, and judge execution is delegated through the async flow |
| Submission evaluation is asynchronous | Integration verification covers authenticated submission create plus final verdict persistence through the real API and judge services, and worker tests cover `queued -> running -> final verdict`, lease-based recovery of abandoned jobs, and terminal `judge_failed` handling after repeated worker failures |
| Published problems are judged against stable versions | Database or service tests prove submissions reference immutable published problem versions |
| Hidden tests remain private | API, integration, and storage tests prove hidden test bundles are uploaded through protected routes, kept out of public problem reads, rejected when metadata is missing or mismatched, and checksum-verified by the judge before execution |
| Moderated user-created problems follow lifecycle rules | API tests cover lifecycle transitions and permission checks, platform integration keeps in-review drafts out of the public problem list until moderator approval, and browser smoke verifies the author-to-moderator publication loop |
| Supported languages are `C++17` and `Python` | Judge tests cover successful execution and common failure verdicts for both languages |
| Verdict reporting is trustworthy | Integration and browser smoke tests cover accepted final verdicts with stored per-test results, and API plus web tests cover owner-scoped submission detail reads and verdict rendering for `Accepted`, `Wrong Answer`, `Compile Error`, `Runtime Error`, and `Time Limit Exceeded` |
| Role-based access is enforced | API or service tests cover `user`, `moderator`, and `admin` permissions for protected actions |
| Service observability is available for production operations | API tests cover structured request logs and metrics snapshots; judge tests cover claim, completion, retry, terminal failure counters, and heartbeat freshness |
| Deployable service packaging remains valid | CI verifies runtime packaging metadata and builds the web, API, and judge container images |
| Production database changes are recoverable | Migration workflow verification applies migrations idempotently to a temporary database and proves backup plus restore preserves schema history and data |
| CI protects the production path | CI builds the production web app, runs API backing-service smoke verification, runs platform integration and browser workflow smoke verification, and executes dependency security checks for Go services |
| Production auth validation is explicit | API auth tests cover cookie transport, header precedence, invalid issuer or audience rejection, and JWKS-backed signing-key refresh |

## Current Gaps
- Automated verification now exists for the frontend workspace, both Go services, database schema verification, database migration or recovery verification, runtime packaging, API runtime smoke, API plus judge integration, and browser end-to-end smoke.
- Full multi-service integration still does not cover lease-expiry recovery or terminal `judge_failed` behavior yet; those remain covered at the worker and service-test level.
- The browser smoke uses a checked-in local test-auth fixture rather than a live external Clerk tenant; production auth behavior remains covered separately through API auth tests.
- The judge worker now uses a stricter sandbox runner than the original local spike path, but broader cross-host hardening and production operations still need additional integration coverage.
- CI still needs image scanning and deployable artifact publishing beyond the current build, smoke, and dependency-security checks.
- This file should be updated whenever a new project rule, security constraint, or core product behavior is introduced.
