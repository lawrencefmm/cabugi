# Database Assets

This directory holds schema documentation, migrations, and verification helpers for the platform.

## Current Files
- `migrations/0001_initial_schema.sql`: initial PostgreSQL schema for users, problems, versions, submissions, and moderation.
- `scripts/verify_initial_schema.sh`: applies the initial migration to the local PostgreSQL container and checks the key MVP schema rules.

## Verification
Run the initial schema verification from the repository root:

```bash
./db/scripts/verify_initial_schema.sh
```
