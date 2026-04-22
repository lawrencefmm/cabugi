# Judge Service

This service will claim submission jobs and execute untrusted code in isolation on dedicated judge hosts.

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
