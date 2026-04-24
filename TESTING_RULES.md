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
| Submission evaluation is asynchronous | Integration and worker tests cover `queued -> running -> final verdict` state transitions, lease-based recovery of abandoned jobs, plus terminal `judge_failed` handling after repeated worker failures |
| Published problems are judged against stable versions | Database or service tests prove submissions reference immutable published problem versions |
| Hidden tests remain private | API and storage integration tests prove hidden test bundles are not exposed through public endpoints or frontend assets, API draft validation rejects missing or mismatched bundle metadata, and judge-side storage tests verify object-fetched bundle checksums before execution |
| Moderated user-created problems follow lifecycle rules | Tests cover `draft`, `in_review`, `published`, and `archived` transitions and permission checks |
| Supported languages are `C++17` and `Python` | Judge tests cover successful execution and common failure verdicts for both languages |
| Verdict reporting is trustworthy | Tests cover final verdict aggregation from per-test results, owner-scoped submission detail reads, and verdict rendering for `Accepted`, `Wrong Answer`, `Compile Error`, `Runtime Error`, and `Time Limit Exceeded` |
| Role-based access is enforced | API or service tests cover `user`, `moderator`, and `admin` permissions for protected actions |
| Service observability is available for production operations | API tests cover structured request logs and metrics snapshots; judge tests cover claim, completion, retry, terminal failure counters, and heartbeat freshness |

## Current Gaps
- Initial automated test commands now exist for the frontend workspace and both Go services.
- The database schema now has a repeatable verification script, but higher-level integration tests across the API and judge pipeline do not exist yet.
- Integration tests for the API, database, object storage, and end-to-end submission flow do not exist yet.
- The judge worker now uses a stricter sandbox runner than the original local spike path, but broader cross-host hardening and production operations still need additional integration coverage.
- CI still needs to be added so the documented verification commands run automatically on every push or pull request.
- This file should be updated whenever a new project rule, security constraint, or core product behavior is introduced.
