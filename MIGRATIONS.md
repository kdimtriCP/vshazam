# Database Migrations Guide

## Overview

VShazam uses an automatic migration system that tracks and applies database schema changes for PostgreSQL.

## How It Works

1. **Automatic Execution**: Migrations run automatically when the server starts
2. **Version Tracking**: Applied migrations are tracked in `schema_migrations` table
3. **Idempotent**: Running migrations multiple times is safe - already applied migrations are skipped
4. **Transaction Safety**: Each migration runs in a transaction - either fully succeeds or rolls back

## Migration Files

Migrations are stored in the `migrations/` directory with naming convention:
```
XXX_description.sql
```

Where `XXX` is a 3-digit version number (e.g., `001`, `002`, etc.)

## Current Migrations

- `001_init.sql` - Initial database schema (videos table)
- `002_frame_analyses.sql` - AI analysis results table

## Running Migrations

### Development (Local)
```bash
# Check migration status
make migrate-status

# Run pending migrations
make migrate
```

### Docker
```bash
# Migrations run automatically on startup
docker-compose up

# Check status manually
docker-compose exec app go run cmd/migrate/main.go -status

# Force run migrations
docker-compose exec app go run cmd/migrate/main.go
```

## Creating New Migrations

1. Create a new file in `migrations/` directory:
```bash
touch migrations/003_your_description.sql
```

2. Write your SQL:
```sql
-- migrations/003_your_description.sql
CREATE TABLE new_feature (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_new_feature_name ON new_feature(name);
```

3. Restart the server or run migrations manually

## Best Practices

1. **Always use transactions** - Migrations automatically run in transactions
2. **Make migrations reversible** - Consider how to undo changes if needed
3. **Test locally first** - Run migrations on local PostgreSQL before production
4. **One change per migration** - Keep migrations focused and simple
5. **Use IF NOT EXISTS** - Make migrations idempotent when possible

## Troubleshooting

### Migration Failed
Check the error message and fix the SQL. The migration will be retried on next run.

### Reset Migrations (Development Only)
```sql
-- Connect to PostgreSQL
DROP TABLE schema_migrations;
-- Then restart the server to re-run all migrations
```

### Check Applied Migrations
```sql
SELECT * FROM schema_migrations ORDER BY version;
```

## Environment Variables

- `MIGRATIONS_PATH` - Path to migrations directory (default: `./migrations`)

## Notes

- SQLite databases don't use this migration system (tables are created directly)
- Migrations only run for PostgreSQL databases
- The migration system is integrated into the main server startup