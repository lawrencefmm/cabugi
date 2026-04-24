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
- assigns each claimed job a renewable lease so abandoned claims can be recovered after worker loss
- downloads hidden test cases from private object storage using the stored `hidden_test_bundle_key`
- verifies the downloaded bundle against `hidden_test_bundle_sha256`
- executes the submission with a dedicated worker sandbox runner instead of the local spike command path
- uses pinned runtime images prepared ahead of time rather than pulling images during job execution
- runs containers as a non-root user with network disabled, read-only root filesystems, and only a scoped writable workspace plus tmpfs scratch areas
- renews the submission job lease while long-running work is still active
- requeues failed jobs with `last_error` until the configured max-attempt limit is reached
- marks submissions as `judge_failed` when the worker exhausts retry attempts
- writes final submission status plus `submission_results`
- emits structured JSON logs for claimed, completed, retried, and terminally failed jobs
- serves local observability routes on `JUDGE_OBSERVABILITY_ADDRESS`, which defaults to `127.0.0.1:8082`

Prepare the worker sandbox images before starting the judge service on a host:

```bash
docker pull gcc:14.2.0
docker pull python:3.13.0-alpine3.20
```

Judge worker lease environment:
- `JUDGE_JOB_LEASE_DURATION`: maximum time a claimed submission job can go without renewal before another worker may reclaim it. Defaults to `30s`.
- `JUDGE_JOB_LEASE_RENEW_INTERVAL`: how often an active worker renews its current job lease. Defaults to `10s`.

Judge observability environment:
- `JUDGE_OBSERVABILITY_ADDRESS`: bind address for `GET /healthz` and `GET /metricsz`. Defaults to `127.0.0.1:8082`.

Inspect the worker locally:
- Read stdout for JSON log events named `judge_job_claimed`, `judge_job_completed`, `judge_job_retried`, and `judge_job_terminal_failure`.
- Request `http://127.0.0.1:8082/metricsz` to inspect worker outcome counters and heartbeat freshness.

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

The spike remains a local technical proof-of-concept command. The long-running worker now uses a separate hardened sandbox runner path intended for dedicated judge hosts.

Automated verification:

```bash
go test ./...
```
