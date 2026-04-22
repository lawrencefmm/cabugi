# API Service

This service will own authentication checks, business logic, persistence, and submission lifecycle state.

## Run Locally
Start the API from `services/api`:

```bash
go run ./cmd/api
```

The API listens on `127.0.0.1:8080` by default. Override with `API_ADDRESS`.

Available bootstrap routes:
- `GET /healthz`
- `GET /openapi/v1.yaml`
- `GET /v1/me` with Clerk session authentication and DB-backed app user bootstrap
- `GET /v1/problems`
- `GET /v1/problems/{slug}`
- `POST /v1/problem-drafts`
- `GET /v1/problem-drafts/{slug}`
- `PATCH /v1/problem-drafts/{slug}`
- `GET /v1/submissions`
- `POST /v1/submissions`
- `GET /v1/submissions/{id}`

## Clerk Auth Environment
- `CLERK_PEM_PUBLIC_KEY`: Clerk JWT verification public key in PEM format.
- `CLERK_ALLOWED_PARTIES`: optional comma-separated allowed `azp` values such as `http://localhost:3000`.

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

## Verification
Run the API test suite from `services/api`:

```bash
go test ./...
```
