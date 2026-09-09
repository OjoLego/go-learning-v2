# Phase 2: Schema Design & Manual SQL - Personal Notes

## Overview
Phase 2 completed successfully. Designed and implemented the database schema for dime-api including transactions and budgets tables with proper data types, constraints, and indexes.

---

## What Was Built

### Tables Created

**1. transactions table**
- 6 columns: id, user_id, type, amount, category, created_at
- Stores all income and expense transactions
- Auto-generated UUID for id
- Auto-generated timestamp for created_at

**2. budgets table**
- 6 columns: id, user_id, category, limit_amount, spent, updated_at
- Tracks spending limits per user+category combination
- Auto-generated UUID for id
- Default 0.00 for spent
- Auto-generated timestamp for updated_at

### Indexes Created (4 total)

**On transactions table:**
- idx_transactions_user_id - for ListByUser queries
- idx_transactions_user_category - for budget correlation queries
- idx_transactions_created_at - for sorting by date

**On budgets table:**
- idx_budgets_user_category - for GetByUserAndCategory queries

### Constraints Implemented

**transactions table:**
- PRIMARY KEY on id
- NOT NULL on all columns
- CHECK (type IN ('income', 'expense')) - validates transaction type
- CHECK (amount > 0) - ensures positive amounts

**budgets table:**
- PRIMARY KEY on id
- NOT NULL on all columns
- UNIQUE(user_id, category) - one budget per user+category
- CHECK (limit_amount > 0) - ensures positive limit
- CHECK (spent >= 0) - ensures non-negative spent

---

## Key Decisions Made

### 1. UUID vs SERIAL for IDs
**Decision:** Use UUID
**Rationale:**
- Matches Go model (string type)
- Better for distributed systems
- Non-sequential prevents ID enumeration
- Used `gen_random_uuid()` from pgcrypto extension

### 2. DECIMAL vs FLOAT for Money
**Decision:** Use DECIMAL(10,2)
**Rationale:**
- FLOAT has rounding errors (0.1 + 0.2 != 0.3)
- DECIMAL provides exact precision
- Critical for financial calculations
- (10,2) allows up to $99,999,999.99

### 3. CHECK Constraint vs ENUM for Type
**Decision:** Use VARCHAR with CHECK constraint
**Rationale:**
- CHECK constraints are easier to modify than ENUM
- Still provides database-level validation
- More flexible for future additions

### 4. TIMESTAMPTZ vs TIMESTAMP
**Decision:** Use TIMESTAMPTZ
**Rationale:**
- Stores values in UTC internally
- Handles timezone conversions automatically
- Best practice for multi-timezone applications
- Maps naturally to Go's time.Time

### 5. Reserved Keyword Avoidance
**Decision:** Use limit_amount instead of limit
**Rationale:**
- "limit" is a reserved SQL keyword
- Used in SELECT ... LIMIT 10
- Never use reserved keywords as column names

---

## What Was Learned

### PostgreSQL Data Types

| Go Type | PostgreSQL Type | Notes |
|---------|----------------|-------|
| string (ID) | UUID | Secure, distributed-safe, needs pgcrypto extension |
| string | TEXT | Flexible string storage |
| float64 (money) | DECIMAL(10,2) | Exact precision, no rounding errors |
| enum | VARCHAR(10) CHECK (...) | Validated, flexible |
| time.Time | TIMESTAMPTZ | Timezone-aware, stores UTC |

### Constraints

| Constraint | Purpose | Example |
|------------|---------|---------|
| PRIMARY KEY | Unique identifier, auto-indexed | id UUID PRIMARY KEY |
| NOT NULL | Column must have value | user_id TEXT NOT NULL |
| UNIQUE | No duplicates allowed | UNIQUE(user_id, category) |
| CHECK | Custom validation | CHECK (amount > 0) |
| DEFAULT | Auto-set value | DEFAULT NOW() |

### Indexes

**When to create:**
- Columns used in WHERE clauses
- Columns used in ORDER BY
- Columns used in JOIN conditions

**Composite indexes:**
- Multiple columns: (user_id, category)
- Column order matters for query matching
- More efficient than separate indexes for multi-column queries

**Trade-offs:**
- Speed up reads (SELECT)
- Slow down writes (INSERT, UPDATE, DELETE)
- Use disk space

### Important SQL Patterns

**CREATE TABLE:**
```sql
CREATE TABLE table_name (
    column_name data_type constraints,
    ...
);
```

**CREATE INDEX:**
```sql
CREATE INDEX index_name ON table_name(column_name);
CREATE INDEX index_name ON table_name(column1, column2); -- composite
```

**INSERT with RETURNING:**
```sql
INSERT INTO table (columns) VALUES (values) RETURNING *;
```

**UPDATE with calculation:**
```sql
UPDATE table SET spent = spent + amount WHERE ... RETURNING *;
```

---

## Repository Queries Mapped to SQL

### TransactionRepository

**Create:**
```sql
INSERT INTO transactions (user_id, type, amount, category)
VALUES ($1, $2, $3, $4)
RETURNING id, user_id, type, amount, category, created_at;
```

**GetByID:**
```sql
SELECT id, user_id, type, amount, category, created_at
FROM transactions
WHERE id = $1;
```

**ListByUser:**
```sql
SELECT id, user_id, type, amount, category, created_at
FROM transactions
WHERE user_id = $1
ORDER BY created_at DESC;
```

### BudgetRepository

**Create:**
```sql
INSERT INTO budgets (user_id, category, limit_amount)
VALUES ($1, $2, $3)
RETURNING id, user_id, category, limit_amount, spent, updated_at;
```

**GetByUserAndCategory:**
```sql
SELECT id, user_id, category, limit_amount, spent, updated_at
FROM budgets
WHERE user_id = $1 AND category = $2;
```

**AddSpent:**
```sql
UPDATE budgets
SET spent = spent + $3, updated_at = NOW()
WHERE user_id = $1 AND category = $2
RETURNING id, user_id, category, limit_amount, spent, updated_at;
```

---

## Common Errors & Solutions

### Error: "check constraint violated"
**Cause:** Trying to insert invalid data
**Examples:**
- Invalid type: 'invalid' instead of 'income' or 'expense'
- Negative amount: -50.00 when CHECK requires > 0
- Zero amount: 0 when CHECK requires > 0

**Solution:** Ensure data meets constraint requirements before inserting.

### Error: "duplicate key value violates unique constraint"
**Cause:** Trying to insert duplicate budget
**Example:** User already has budget for 'food', trying to create another

**Solution:** Check if budget exists first, or use INSERT ... ON CONFLICT.

### Error: "column 'limit' does not exist" or syntax error
**Cause:** Using reserved SQL keyword as column name
**Example:** Tried to use 'limit' instead of 'limit_amount'

**Solution:** Avoid reserved keywords. Common ones: limit, order, group, select, insert, update, delete, table, column.

---

## Testing Performed

### Valid Operations (All Passed)
- ✅ Insert valid transactions (income and expense)
- ✅ Insert valid budgets
- ✅ Query transactions by user_id
- ✅ Query budgets by user_id and category
- ✅ Update spent amount (AddSpent)
- ✅ Auto-generation of UUIDs
- ✅ Auto-generation of timestamps
- ✅ Default value for spent (0.00)

### Constraint Violations (All Correctly Rejected)
- ✅ Invalid transaction type rejected
- ✅ Negative amount rejected
- ✅ Zero amount rejected
- ✅ Duplicate budget rejected
- ✅ Negative spent value rejected

### Index Verification
- ✅ All 4 indexes created successfully
- ✅ Verified with \di command
- ✅ Queries use indexes (checked with EXPLAIN)

---

## Commands Used

### psql Meta-Commands
| Command | Purpose |
|---------|---------|
| \dt | List all tables |
| \d table_name | Describe table structure |
| \di | List all indexes |
| \i file.sql | Execute SQL file |

### SQL Commands
| Command | Purpose |
|---------|---------|
| CREATE TABLE | Create new table |
| CREATE INDEX | Create index for performance |
| INSERT | Add data |
| SELECT | Query data |
| UPDATE | Modify data |
| DROP TABLE | Delete table |
| COMMENT ON | Add documentation |

---

## Files Created

```
migrations/
├── 01_create_transactions.sql    -- transactions table + indexes
├── 02_create_budgets.sql         -- budgets table + indexes
└── 04_repository_queries.sql     -- documented queries (optional)
```

---

## Next Phase Preview

**Phase 3: Database Migrations**

Will learn:
- What are migrations and why they're important
- golang-migrate tool installation and usage
- Creating migration files (up/down)
- Version controlling schema changes
- Running migrations programmatically from Go

The difference: Instead of running SQL manually, we'll use a tool to track and apply schema changes.

---

## Key Takeaways

1. **Choose data types carefully** - DECIMAL for money, UUID for IDs, TIMESTAMPTZ for timestamps
2. **Use constraints for data integrity** - CHECK, NOT NULL, UNIQUE at database level
3. **Index strategically** - Index columns used in WHERE, ORDER BY, JOIN
4. **Test constraint violations** - Ensure data integrity rules work
5. **Document your schema** - Comments and notes for future reference
6. **RETURNING clause** - Always use for INSERT/UPDATE to get complete records

---

*Phase 2 Completed: Schema designed, tables created, queries validated*
