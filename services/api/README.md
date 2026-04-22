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
- `GET /v1/submissions`
- `POST /v1/submissions`
- `GET /v1/submissions/{id}`

## Clerk Auth Environment
- `CLERK_PEM_PUBLIC_KEY`: Clerk JWT verification public key in PEM format.
- `CLERK_ALLOWED_PARTIES`: optional comma-separated allowed `azp` values such as `http://localhost:3000`.

## Database Environment
- `DATABASE_URL`: PostgreSQL connection string for problem and submission data. Defaults to the local Docker Compose database.
- `WEB_ALLOWED_ORIGINS`: optional comma-separated browser origins allowed by API CORS. Defaults to `http://127.0.0.1:3000,http://localhost:3000`.

## Verification
Run the API test suite from `services/api`:

```bash
go test ./...
```
