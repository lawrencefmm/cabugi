# Judge Service

This service will claim submission jobs and execute untrusted code in isolation on dedicated judge hosts.

## Worker Command
Process one queued submission from `services/judge`:

```bash
go run ./cmd/judge --once
```

Current local worker behavior:
- claims one queued submission job from PostgreSQL
- loads hidden test cases from the file path stored in `hidden_test_bundle_key`
- executes the submission with the existing Docker-based spike runner
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
