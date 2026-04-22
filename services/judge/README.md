# Judge Service

This service will claim submission jobs and execute untrusted code in isolation on dedicated judge hosts.

## Worker Command
Run the long-running judge worker from `services/judge`:

```bash
go run ./cmd/judge
```

Process at most one queued submission from `services/judge`:

```bash
go run ./cmd/judge --once
```

Current local worker behavior:
- claims one queued submission job from PostgreSQL
- downloads hidden test cases from private object storage using the stored `hidden_test_bundle_key`
- verifies the downloaded bundle against `hidden_test_bundle_sha256`
- executes the submission with the existing Docker-based spike runner
- requeues failed jobs with `last_error` until the configured max-attempt limit is reached
- marks submissions as `judge_failed` when the worker exhausts retry attempts
- writes final submission status plus `submission_results`

## Judge Spike
Run the local Docker-based judging spike from `services/judge`:

```bash
go run ./cmd/judge-spike --all
```

This command exercises the built-in `C++17` and `Python` scenarios for:
- `Accepted`
- `Wrong Answer`
- `Compile Error`
- `Time Limit Exceeded`

Automated verification:

```bash
go test ./...
```
