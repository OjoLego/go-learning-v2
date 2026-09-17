# Phase 3: Database Migrations - Learning Notes

## Overview

This phase covered database schema versioning using **golang-migrate**. We converted manual SQL files into proper version-controlled migrations that can be run programmatically from Go code.

## What We Learned

### 1. What Are Database Migrations?

Database migrations are **version-controlled scripts** that manage changes to your database schema over time. Think of them like **Git for your database structure**.

**Key Benefits:**
- Track schema changes over time
- Teams can sync database state
- Rollback capability (up/down migrations)
- Essential for production deployments

### 2. Up vs Down Migrations

| Type | Purpose | Example |
|------|---------|---------|
| **Up** | Apply changes | `CREATE TABLE`, `ALTER TABLE` |
| **Down** | Undo changes | `DROP TABLE`, remove columns |

### 3. Migration File Naming Convention

```
migrations/
├── 001_create_transactions_table.up.sql
├── 001_create_transactions_table.down.sql
├── 002_create_budgets_table.up.sql
└── 002_create_budgets_table.down.sql
```

**Rules:**
- Use **three-digit sequential numbers** (001, 002, 003...)
- **`.up.sql`** for forward migrations
- **`.down.sql`** for rollback migrations
- Never reuse version numbers

### 4. golang-migrate Tool

We use `golang-migrate` which provides:
- CLI tool for manual migration management
- Go library for programmatic migration execution
- Support for multiple databases (PostgreSQL, MySQL, etc.)
- Version tracking via `schema_migrations` table

## Files Created

### Migration Files
```
migrations/
├── 001_create_transactions_table.up.sql     # Create transactions table + indexes
├── 001_create_transactions_table.down.sql   # Drop transactions table
├── 002_create_budgets_table.up.sql          # Create budgets table + indexes
└── 002_create_budgets_table.down.sql        # Drop budgets table
```

### Go Implementation Files
```
internal/repository/
├── interface.go                        # Repository interfaces
├── postgres_budget_repo.go            # PostgreSQL budget implementation
└── postgres_transaction_repo.go       # PostgreSQL transaction implementation

cmd/api/main.go                         # Updated with migration logic
```

### Configuration
```
.env                                    # Database configuration
```

## Key Commands

### CLI Commands (using migrate.exe)

```powershell
# Check current migration version
.\migrate.exe -path migrations -database "postgres://dime_user:dime_password@localhost:5432/dime?sslmode=disable" version

# Run all pending migrations (up)
.\migrate.exe -path migrations -database "postgres://dime_user:dime_password@localhost:5432/dime?sslmode=disable" up

# Rollback last migration (down 1)
.\migrate.exe -path migrations -database "postgres://dime_user:dime_password@localhost:5432/dime?sslmode=disable" down 1

# Rollback all migrations
.\migrate.exe -path migrations -database "postgres://dime_user:dime_password@localhost:5432/dime?sslmode=disable" down

# Force version (fix dirty state)
.\migrate.exe -path migrations -database "postgres://dime_user:dime_password@localhost:5432/dime?sslmode=disable" force 1
```

### Running the Application

```powershell
# Build the application
go build -o dime-api.exe ./cmd/api

# Run (automatically connects to PostgreSQL and runs migrations)
.\dime-api.exe

# Or run directly without building
go run ./cmd/api
```

### Expected Output

```
2026/09/16 15:37:15 Current migration version: 2, Dirty: false
2026/09/16 15:37:15 Connected to PostgreSQL
2026/09/16 15:37:15 No new migrations to apply
2026/09/16 15:37:15 Migrations applied successfully
2026/09/16 15:37:15 Final migration version: 2, Dirty: false
Server running on :8080
```

## Migration Best Practices

### 1. Idempotent Migrations
Always use `IF NOT EXISTS` to make migrations safe to run multiple times:

```sql
-- Good
CREATE TABLE IF NOT EXISTS users (...);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

-- Bad
CREATE TABLE users (...);  -- Fails if table exists
```

### 2. Transaction Safety
Keep migrations small and focused. Each migration file is wrapped in a transaction by default - if any part fails, the entire migration rolls back.

### 3. Never Modify Applied Migrations
Once a migration is applied to any environment (dev, staging, production), **never change it**. Create a new migration instead.

```
❌ BAD: Modifying 001_create_users.sql after it's been applied
✅ GOOD: Creating 003_add_user_profile.sql to add new columns
```

### 4. Test Both Up and Down
Always test your down migrations work correctly:

```powershell
# Test full cycle
.\migrate.exe ... up      # Apply migrations
.\migrate.exe ... down 1  # Rollback one
.\migrate.exe ... up      # Re-apply
```

### 5. Schema vs Data Migrations
Separate schema changes from data migrations:

```sql
-- Schema migration (fast)
ALTER TABLE users ADD COLUMN email TEXT;

-- Data migration (slow, may need separate process)
UPDATE users SET email = 'temp@example.com' WHERE email IS NULL;
```

## Troubleshooting

### Error: "Dirty database version"

**Cause:** A migration failed halfway through.

**Fix:**
```powershell
# Check current version
.\migrate.exe ... version

# Force to clean version (use with caution!)
.\migrate.exe ... force <version_number>
```

### Error: "unknown driver postgres"

**Cause:** Using wrong migrate binary (go-installed version without drivers).

**Fix:** Use the downloaded binary with PostgreSQL support:
```powershell
# ❌ Wrong
migrate ...

# ✅ Correct
.\migrate.exe ...
```

### Error: "relation does not exist"

**Cause:** Trying to reference a table that doesn't exist yet.

**Fix:** Ensure migrations are ordered correctly (dependencies first).

## Architecture Overview

```
┌─────────────────────────────────────┐
│         Application Startup         │
└──────────────────┬──────────────────┘
                   │
                   ▼
┌─────────────────────────────────────┐
│     Connect to PostgreSQL           │
└──────────────────┬──────────────────┘
                   │
                   ▼
┌─────────────────────────────────────┐
│     Check Migration Status          │
│     (schema_migrations table)       │
└──────────────────┬──────────────────┘
                   │
                   ▼
┌─────────────────────────────────────┐
│     Run Pending Migrations          │
│     (001_*.up.sql, 002_*.up.sql)    │
└──────────────────┬──────────────────┘
                   │
                   ▼
┌─────────────────────────────────────┐
│     Update Version                  │
│     (schema_migrations table)       │
└──────────────────┬──────────────────┘
                   │
                   ▼
┌─────────────────────────────────────┐
│     Start Application Server        │
└─────────────────────────────────────┘
```

## What Makes This Production-Ready

✅ **Automatic migrations** on startup  
✅ **Version tracking** via database table  
✅ **Rollback capability** via down migrations  
✅ **Idempotent migrations** (safe to retry)  
✅ **Environment-based configuration**  
✅ **Clean separation** of concerns  
✅ **Interface-based repositories** (testable)  

## Connection to Previous Phases

- **Phase 1:** Set up project structure
- **Phase 2:** Designed database schema and wrote SQL manually
- **Phase 3:** ✅ **Converted SQL to version-controlled migrations**

## Next Steps

Future phases can build on this foundation:
- Add more tables via new migrations
- Implement data migrations for backfills
- Add migration testing in CI/CD
- Explore migration squashing for long-running projects

## Summary

We successfully:
1. ✅ Learned what migrations are and why they're essential
2. ✅ Installed and configured golang-migrate
3. ✅ Created proper up/down migration pairs
4. ✅ Implemented programmatic migration execution in Go
5. ✅ Tested rollback and re-apply workflows
6. ✅ Built a production-ready migration system

**Phase 3 Complete!** 🎉
