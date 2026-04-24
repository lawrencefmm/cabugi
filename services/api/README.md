# API Service

This service will own authentication checks, business logic, persistence, and submission lifecycle state.

## Run Locally
Start the API from `services/api`:

```bash
go run ./cmd/api
```

The API listens on `127.0.0.1:8080` by default. Override with `API_ADDRESS`.

Build the deployable API image from the repository root:

```bash
pnpm build:image:api
```

Available bootstrap routes:
- `GET /healthz`
- `GET /readyz`
- `GET /metricsz`
- `GET /openapi/v1.yaml`
- `GET /v1/me` with Clerk session authentication and DB-backed app user bootstrap
- `GET /v1/problems`
- `GET /v1/problems/{slug}`
- `POST /v1/problem-drafts/hidden-test-bundles`
- `POST /v1/problem-drafts`
- `GET /v1/problem-drafts/{slug}`
- `PATCH /v1/problem-drafts/{slug}`
- `POST /v1/problem-drafts/{slug}/submit-for-review`
- `GET /v1/moderation/problem-drafts`
- `POST /v1/moderation/problem-drafts/{slug}/decision`
- `GET /v1/submissions`
- `POST /v1/submissions`
- `GET /v1/submissions/{id}`

## Clerk Auth Environment
- `CLERK_PEM_PUBLIC_KEY`: Clerk JWT verification public key in PEM format.
- `CLERK_ALLOWED_PARTIES`: optional comma-separated allowed `azp` values such as `http://localhost:3000`.

## Startup Requirement Environment
- `API_REQUIRE_AUTH`: when `true`, the API fails startup unless Clerk auth verification is configured.
- `API_REQUIRE_DATABASE`: when `true`, the API fails startup unless PostgreSQL-backed stores can connect successfully.
- `API_REQUIRE_HIDDEN_BUNDLE_VALIDATION`: when `true`, the API fails startup unless hidden test bundle validation can reach the configured object storage bucket.

## Database Environment
- `DATABASE_URL`: PostgreSQL connection string for problem and submission data. Defaults to the local Docker Compose database.
- `WEB_ALLOWED_ORIGINS`: optional comma-separated browser origins allowed by API CORS. Defaults to `http://127.0.0.1:3000,http://localhost:3000`.

## Hidden Test Bundle Validation Environment
- `OBJECT_STORAGE_ENDPOINT`: object storage endpoint used to validate hidden test bundle references. Defaults to local MinIO at `http://127.0.0.1:9000`.
- `OBJECT_STORAGE_REGION`: object storage region. Defaults to `us-east-1`.
- `OBJECT_STORAGE_BUCKET`: hidden test bundle bucket. Defaults to `cabugi-hidden-tests`.
- `OBJECT_STORAGE_ACCESS_KEY_ID`: object storage access key. Defaults to `minioadmin`.
- `OBJECT_STORAGE_SECRET_ACCESS_KEY`: object storage secret key. Defaults to `minioadmin`.
- `OBJECT_STORAGE_USE_PATH_STYLE`: optional path-style toggle for S3-compatible APIs. Defaults to `true` for local MinIO.

## Readiness Behavior
- `GET /healthz` is a liveness endpoint and returns `200` when the process is running.
- `GET /readyz` returns `200` only when auth, database stores, and hidden bundle validation are configured and ready; otherwise it returns `503` with dependency readiness details.

## Observability
- API logs are written to stdout as JSON records. Request logs use the `http_request` message and include `request_id`, `method`, `path`, `route`, `status`, and `latency_ms`.
- `GET /metricsz` returns a JSON snapshot with API request counters grouped by method, route, and status, plus the current submission queue depth.
- Pass `X-Request-ID` on requests to reuse an external request identifier; otherwise the API generates one and returns it in the response header.

## Verification
Run the API test suite from `services/api`:

```bash
go test ./...
```
