# Phase 1: PostgreSQL Foundation - Personal Notes

## Overview
Phase 1 completed successfully. PostgreSQL is now running locally in Docker, and all fundamental concepts have been learned.

---

## What I Learned

### Docker & PostgreSQL Setup
- Docker Desktop must be running for docker commands to work
- `docker compose up -d` starts containers in detached (background) mode
- `docker compose ps` shows container status and health
- `docker compose logs` shows what's happening inside the container
- `docker compose down` stops and removes containers
- Named volumes (`postgres_data`) persist data even if container is removed

### Connection Details
```text
Host: localhost
Port: 5432
Database: dime
Username: dime_user
Password: dime_password
```

### Connecting with psql
```bash
# Connect to database
psql -h localhost -p 5432 -U dime_user -d dime

# Enter password when prompted: dime_password
```

### psql Meta-Commands
| Command | Description |
|---------|-------------|
| `\l` | List all databases |
| `\c database_name` | Connect to a different database |
| `\dt` | List all tables |
| `\d table_name` | Describe a table (show columns and types) |
| `\dn` | List schemas |
| `\du` | List users |
| `\conninfo` | Show current connection info |
| `\q` | Quit psql |

---

## PostgreSQL Architecture Hierarchy

```
PostgreSQL Server
  └── Database (dime)
        └── Schema (public)
              ├── Table (practice_items)
              │     ├── Columns (id, name, quantity, is_active, created_at)
              │     └── Rows (actual data records)
              └── Table (future tables...)
```

### Key Concepts

**Database**: A named collection of schemas and tables. Like a dedicated filing cabinet.

**Schema**: A namespace within a database. Default is `public`. Like a drawer in the filing cabinet.

**Table**: Structure with defined columns that holds rows. Like a spreadsheet.

**Column**: A field with a specific data type. Defines what data can be stored.

**Row**: One record in a table. Represents a single entity.

**Primary Key**: Unique identifier for each row. Usually `SERIAL` (auto-incrementing integer).

**NULL**: Represents "no value" or "unknown". Different from empty string ("") or zero (0).

---

## Data Types Used

| Type | Purpose | Example |
|------|---------|---------|
| `SERIAL` | Auto-incrementing integer for IDs | 1, 2, 3, ... |
| `INTEGER` | Whole numbers | 42, -7, 0 |
| `TEXT` | Variable-length strings | 'Hello world' |
| `BOOLEAN` | True/false values | true, false |
| `TIMESTAMP` | Date and time | '2024-01-15 10:30:00' |

---

## Constraints Used

| Constraint | Purpose |
|------------|---------|
| `PRIMARY KEY` | Unique identifier, automatically indexed |
| `NOT NULL` | Column must have a value (cannot be NULL) |
| `DEFAULT` | Automatic value if none provided |

---

## SQL Quick Reference

### CREATE TABLE
```sql
CREATE TABLE practice_items (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    quantity INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### INSERT
```sql
-- Single row
INSERT INTO practice_items (name, quantity) 
VALUES ('Learn PostgreSQL', 1);

-- Multiple rows
INSERT INTO practice_items (name, quantity, is_active) 
VALUES 
    ('Practice SQL queries', 5, true),
    ('Review concepts', 3, true);
```

### SELECT
```sql
-- All columns, all rows
SELECT * FROM practice_items;

-- Specific columns
SELECT name, quantity FROM practice_items;

-- With WHERE filter
SELECT * FROM practice_items WHERE is_active = true;
SELECT * FROM practice_items WHERE quantity > 2;

-- Sorted results
SELECT * FROM practice_items ORDER BY quantity DESC;
```

### UPDATE
```sql
-- ⚠️ ALWAYS use WHERE with UPDATE!
UPDATE practice_items 
SET quantity = 10 
WHERE name = 'Practice SQL queries';

-- Update multiple columns
UPDATE practice_items 
SET quantity = 2, is_active = false 
WHERE name = 'Take a break';
```

### DELETE
```sql
-- ⚠️ ALWAYS use WHERE with DELETE!
DELETE FROM practice_items 
WHERE is_active = false;
```

---

## Common Mistakes to Avoid

1. **Forgetting semicolons** at end of SQL statements
   - psql waits for `;` to execute the command

2. **Forgetting WHERE clause** in UPDATE/DELETE
   - Without WHERE, it affects ALL rows!
   - Always preview with SELECT first:
   ```sql
   SELECT * FROM table WHERE condition;  -- Preview
   UPDATE table SET col = val WHERE condition;  -- Then update
   ```

3. **Not using quotes** around string values
   - Strings need single quotes: `'text'`
   - Numbers don't: `42`

4. **Confusing NULL with empty string or zero**
   - NULL = unknown/no value
   - Empty string = known to be empty
   - Zero = known to be zero

5. **Wrong directory when running docker compose**
   - Must be in the directory with `docker-compose.yml`

---

## Files Created in Phase 1

| File | Purpose |
|------|---------|
| `docker-compose.yml` | PostgreSQL container configuration |
| `docs/backend-learning/phase-01-notes.md` | This reference file |

---

## Next Phase Preview

**Phase 2: Schema Design & Manual SQL**

Will design the actual database schema for dime-api:
- `transactions` table
- `budgets` table
- Relationships between tables
- Indexes for performance
- Constraints for data integrity

---

## Important Safety Rules

1. **Always use WHERE with UPDATE and DELETE**
2. **Preview changes with SELECT first**
3. **Use transactions when modifying multiple related rows** (learned in later phases)
4. **Keep docker-compose.yml credentials for local dev only**
5. **Never commit production credentials to git**

---

*Phase 1 Completed: [Today's Date]*
