# Web Workspace

This workspace now contains the published problem browsing UI and the frontend test tooling for the Cabugi web app.

## Current Routes
- `/`: published problem list
- `/problems/[slug]`: published problem detail page

## Environment
- `NEXT_PUBLIC_API_BASE_URL`: browser-visible base URL for the Go API. Defaults to `http://127.0.0.1:8080`.
- `NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY`: Clerk publishable key used for frontend authentication and token retrieval.

## Verification
- `pnpm --filter web typecheck`
- `pnpm --filter web test --run`
