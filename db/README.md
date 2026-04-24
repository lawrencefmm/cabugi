# Database Assets

This directory holds schema documentation, migrations, and verification helpers for the platform.

## Current Files
- `migrations/0001_initial_schema.sql`: initial PostgreSQL schema for users, problems, versions, submissions, and moderation.
- `migrations/0002_submission_job_leases.sql`: adds lease-based submission job recovery so abandoned claims can be reclaimed safely.
- `scripts/verify_initial_schema.sh`: applies the checked-in migrations to the local PostgreSQL container and checks the key MVP schema rules.

## Verification
Run the initial schema verification from the repository root:

```bash
./db/scripts/verify_initial_schema.sh
```
