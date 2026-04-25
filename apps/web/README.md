# Web Workspace

This workspace contains the published problem browsing UI, solve workspace, draft authoring flow, moderation screens, submission history, and frontend test tooling for the Cabugi web app.

## Current Routes
- `/`: published problem list
- `/problems/[slug]`: published problem detail page
- `/drafts/new`: create a problem draft
- `/drafts/[slug]`: edit or submit a problem draft for review
- `/moderation/problem-drafts`: moderator review queue
- `/submissions`: authenticated submission history page
- `/submissions/[id]`: submission summary page

## Environment
- `NEXT_PUBLIC_API_BASE_URL`: browser-visible base URL for the Go API. Defaults to `http://127.0.0.1:8080`.
- `NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY`: Clerk publishable key used for frontend authentication and token retrieval.
- `NEXT_PUBLIC_LOCAL_TEST_AUTH_ENABLED`: test-only local auth adapter used by the browser smoke suite.

Without auth-related environment values, the public problem pages still work but solve, draft, moderation, and submission-history pages stay unavailable in the browser.

## Local Run
From the repository root:

```bash
NEXT_PUBLIC_API_BASE_URL=http://127.0.0.1:8080 pnpm --filter web dev
```

## Packaging
Build the deployable web image from the repository root:

```bash
pnpm build:image:web
```

## Verification
- `pnpm --filter web typecheck`
- `pnpm --filter web test --run`
