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

## Verification
Run the API test suite from `services/api`:

```bash
go test ./...
```
