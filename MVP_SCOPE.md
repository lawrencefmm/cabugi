# MVP Scope

## Goals
- Let users browse and solve competitive programming problems in a modern, fast web interface.
- Let users submit `C++17` and `Python` solutions and receive verdicts from an asynchronous judge.
- Let users view their submission history and the result details for each submission.
- Let users create their own problems as drafts and send them through a moderation flow before publication.
- Keep the first release small enough to build the core solve-submit-judge loop before adding contest features.

## Non-Goals
- Live contests or virtual contests.
- Rating, ranking, or Elo-style progression.
- Codeforces-style hacks.
- Plagiarism detection.
- Public discussion forums, comments, or editorials.
- Broad multi-language support beyond `C++17` and `Python`.

## User Roles
- `user`: Browse published problems, submit solutions, view personal submission history, and create draft problems.
- `moderator`: Review user-created problems, request changes, reject drafts, and publish approved problems.
- `admin`: Manage platform configuration, staff permissions, official problems, and operational tools.

## Core Flows

### Problem Solving
1. A signed-in user opens the published problem list.
2. The user opens a problem detail page with statement, examples, limits, and supported languages.
3. The user writes code in the browser and submits a solution.
4. The API stores a submission record with queued status.
5. The judge fetches the submission, compiles or runs it in isolation, evaluates it against hidden tests, and writes the final verdict.
6. The user views the final verdict and per-submission result details.

### User-Created Problems
1. A signed-in user creates a problem draft with statement content, limits, tags, examples, and hidden tests.
2. The user submits the draft for review.
3. A moderator reviews the problem and either approves it for publication, rejects it, or requests changes.
4. Approved problems become visible in the public problem list as stable published versions.

## Technology Decisions
- Frontend: `Next.js` with `TypeScript`.
- Styling: checked-in application CSS and component class naming within the Next.js app.
- Data fetching: `TanStack Query`.
- Code editor: `Monaco Editor`.
- API service: `Go`.
- Judge worker: separate `Go` service.
- Database: `PostgreSQL`.
- Local development: `Docker Compose`.
- Hidden test and asset storage: `S3`-compatible object storage, with local development using `MinIO`.
- Authentication: Clerk-compatible managed auth verified by the API.
- Problem statement authoring: `Markdown` with `LaTeX` support.

## Success Criteria
- A user can sign in and open a published problem.
- A user can submit a `C++17` or `Python` solution.
- The system stores the submission, judges it asynchronously, and returns a final verdict.
- A user can view personal submission history.
- A user can create a problem draft and send it for review.
- A moderator can approve a draft and publish it.
- Published problems remain tied to stable versions so prior submissions stay reproducible.
