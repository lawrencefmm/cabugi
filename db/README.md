# Database Assets

This directory holds schema documentation, migrations, and verification helpers for the platform.

## Current Files
- `migrations/0001_initial_schema.sql`: initial PostgreSQL schema for users, problems, versions, submissions, and moderation.
- `migrations/0002_submission_job_leases.sql`: adds lease-based submission job recovery so abandoned claims can be reclaimed safely.
- `scripts/migrate.sh`: non-destructive migration runner that records applied migration filenames and SHA-256 checksums in `schema_migrations`.
- `scripts/backup.sh`: creates a PostgreSQL custom-format backup with `pg_dump`.
- `scripts/restore.sh`: restores a PostgreSQL custom-format backup after explicit `CONFIRM_RESTORE=yes` confirmation.
- `scripts/verify_initial_schema.sh`: applies the checked-in migrations to a temporary PostgreSQL container and checks the key MVP schema rules.
- `scripts/verify_migration_workflow.sh`: verifies the production-style migration, backup, and restore workflow against a temporary local database.

## Production Migration Workflow
Set `DATABASE_URL` to the target database and run the migration runner from the repository root:

```bash
DATABASE_URL="postgres://..." ./db/scripts/migrate.sh status
DATABASE_URL="postgres://..." ./db/scripts/migrate.sh up
```

The runner creates `schema_migrations` if needed, applies pending files from `db/migrations` in filename order, and refuses to continue if an already-applied migration file has a different checksum.

Do not use `verify_initial_schema.sh` against production. That script is destructive verification logic and intentionally runs only against an isolated temporary database container.

## Backup And Restore
Create a backup before production migrations:

```bash
DATABASE_URL="postgres://..." BACKUP_DIR="/secure/backups" ./db/scripts/backup.sh
```

Restore requires explicit confirmation because it replaces objects in the target database:

```bash
CONFIRM_RESTORE=yes DATABASE_URL="postgres://..." ./db/scripts/restore.sh /secure/backups/cabugi-YYYYMMDDTHHMMSSZ.dump
```

The scripts use local PostgreSQL client tools when available and fall back to the `postgres:17-alpine` Docker image for local and CI verification.

## Verification
Run the initial schema verification from the repository root:

```bash
./db/scripts/verify_initial_schema.sh
```

Run production workflow verification from the repository root:

```bash
./db/scripts/verify_migration_workflow.sh
```
