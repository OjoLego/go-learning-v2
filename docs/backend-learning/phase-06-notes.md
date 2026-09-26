# Phase 6: Repository Implementation - Budgets

## Overview

Phase 6 implements the PostgreSQL-backed Budget Repository. While similar to the Transaction Repository (Phase 5), budgets have unique requirements: upsert behavior, category-based lookups, and spending tracking. This phase demonstrates handling more complex SQL patterns.

## Learning Objectives

- Implement repository with upsert logic (INSERT ... ON CONFLICT)
- Handle partial updates (incrementing spent amounts)
- Query by composite keys (user_id + category)
- Understand PostgreSQL's RETURNING clause for updates
- Implement transaction-aware methods for atomic budget updates

---

## Repository Interface Recap

From Phase 4, we have this interface:

```go
// internal/repository/interface.go
type BudgetRepository interface {
    Create(ctx context.Context, b model.Budget) (model.Budget, error)
    GetByUserAndCategory(ctx context.Context, userID, category string) (model.Budget, error)
    AddSpent(ctx context.Context, userID, category string, amount float64) (model.Budget, error)
}
```

**Key Differences from Transaction Repository:**
- No ID-based lookup (budgets are identified by user + category)
- `AddSpent` method for partial updates (increment spent field)
- Upsert behavior in Create (update limit if budget exists)

---

## Milestone 1: Basic Operations with Upsert

### File: `internal/repository/postgres_budget_repo.go`

#### Constructor

```go
type PostgresBudgetRepo struct {
    db *sql.DB
}

func NewPostgresBudgetRepo(db *sql.DB) BudgetRepository {
    return &PostgresBudgetRepo{db: db}
}
```

#### Create with Upsert (INSERT ... ON CONFLICT)

Budgets have a unique constraint on `(user_id, category)` - each user can have only one budget per category. When creating a budget, we want to:
- Insert if it doesn't exist
- Update the limit if it already exists (upsert behavior)

```go
func (r *PostgresBudgetRepo) Create(ctx context.Context, b model.Budget) (model.Budget, error) {
    query := `
        INSERT INTO budgets (id, user_id, category, limit_amount, spent)
        VALUES ($1, $2, $3, $4, $5)
        ON CONFLICT (user_id, category) DO UPDATE SET
            limit_amount = EXCLUDED.limit_amount
        RETURNING id, user_id, category, limit_amount, spent
    `

    err := r.db.QueryRowContext(ctx,
        query,
        b.ID,
        b.UserID,
        b.Category,
        b.Limit,
        b.Spent,
    ).Scan(&b.ID, &b.UserID, &b.Category, &b.Limit, &b.Spent)

    if err != nil {
        return model.Budget{}, fmt.Errorf("failed to create budget: %w", err)
    }

    return b, nil
}
```

**SQL Breakdown:**

```sql
INSERT INTO budgets (...)           -- Try to insert
VALUES ($1, $2, $3, $4, $5)         -- New values
ON CONFLICT (user_id, category)     -- If conflict on unique constraint
DO UPDATE SET                       -- Then update
    limit_amount = EXCLUDED.limit_amount  -- EXCLUDED = values we tried to insert
RETURNING ...                       -- Return the final row
```

**Key Concepts:**
- `ON CONFLICT` triggers when unique constraint is violated
- `EXCLUDED` refers to the values that caused the conflict
- Only updating `limit_amount`, not `spent` (preserves spending history)

**Use Case:**
```go
// First call - creates budget
repo.Create(ctx, Budget{UserID: "user1", Category: "food", Limit: 500, Spent: 0})
// Result: New budget created with Limit=500, Spent=0

// Second call - updates limit only
repo.Create(ctx, Budget{UserID: "user1", Category: "food", Limit: 600, Spent: 100})
// Result: Budget updated with Limit=600, Spent=0 (spent preserved from first insert!)
```

#### Read by Composite Key

```go
func (r *PostgresBudgetRepo) GetByUserAndCategory(ctx context.Context, userID, category string) (model.Budget, error) {
    query := `
        SELECT id, user_id, category, limit_amount, spent
        FROM budgets
        WHERE user_id = $1 AND category = $2
    `

    var b model.Budget
    err := r.db.QueryRowContext(ctx, query, userID, category).Scan(
        &b.ID,
        &b.UserID,
        &b.Category,
        &b.Limit,
        &b.Spent,
    )

    if err == sql.ErrNoRows {
        return model.Budget{}, ErrBudgetNotFound
    }
    if err != nil {
        return model.Budget{}, fmt.Errorf("failed to get budget: %w", err)
    }

    return b, nil
}
```

**Key Points:**
- Lookup by two fields: `user_id` AND `category`
- Database has index on `(user_id, category)` for performance
- Returns `ErrBudgetNotFound` (different from transaction's `ErrNotFound`)

#### Partial Update (AddSpent)

Instead of reading-modifying-writing, we use atomic increment:

```go
func (r *PostgresBudgetRepo) AddSpent(ctx context.Context, userID, category string, amount float64) (model.Budget, error) {
    query := `
        UPDATE budgets
        SET spent = spent + $3
        WHERE user_id = $1 AND category = $2
        RETURNING id, user_id, category, limit_amount, spent
    `

    var b model.Budget
    err := r.db.QueryRowContext(ctx, query, userID, category, amount).Scan(
        &b.ID,
        &b.UserID,
        &b.Category,
        &b.Limit,
        &b.Spent,
    )

    if err == sql.ErrNoRows {
        return model.Budget{}, ErrBudgetNotFound
    }
    if err != nil {
        return model.Budget{}, fmt.Errorf("failed to add spent: %w", err)
    }

    return b, nil
}
```

**SQL Breakdown:**
```sql
UPDATE budgets
SET spent = spent + $3      -- Atomic increment (no read-modify-write race condition)
WHERE user_id = $1 AND category = $2
RETURNING ...               -- Return the updated row
```

**Why Atomic Increment?**

```
Without atomic increment (race condition):
┌─────────────┐           ┌─────────────┐
│  Request A  │           │  Request B  │
├─────────────┤           ├─────────────┤
│ Read: 100   │           │ Read: 100   │
│ Add: 50     │           │ Add: 30     │
│ Write: 150  │           │ Write: 130  │
└─────────────┘           └─────────────┘
Result: 130 (B overwrites A, lost 50!)

With atomic increment:
┌─────────────┐           ┌─────────────┐
│  Request A  │           │  Request B  │
├─────────────┤           ├─────────────┤
│ spent + 50  │           │ spent + 30  │
└─────────────┘           └─────────────┘
Result: 180 (both adds applied)
```

**Use Case:**
```go
// User spends money
updated, err := repo.AddSpent(ctx, "user1", "food", 25.50)
if err != nil {
    if err == ErrBudgetNotFound {
        // No budget for this category
    }
}
fmt.Printf("Spent: %.2f / %.2f\n", updated.Spent, updated.Limit)
```

---

## Milestone 2: Transaction-Aware Methods

Just like Phase 5, we need transaction-aware versions for atomic operations:

### CreateTx

```go
func (r *PostgresBudgetRepo) CreateTx(ctx context.Context, tx *sql.Tx, b model.Budget) (model.Budget, error) {
    query := `
        INSERT INTO budgets (id, user_id, category, limit_amount, spent)
        VALUES ($1, $2, $3, $4, $5)
        ON CONFLICT (user_id, category) DO UPDATE SET
            limit_amount = EXCLUDED.limit_amount
        RETURNING id, user_id, category, limit_amount, spent
    `

    err := tx.QueryRowContext(ctx,
        query,
        b.ID,
        b.UserID,
        b.Category,
        b.Limit,
        b.Spent,
    ).Scan(&b.ID, &b.UserID, &b.Category, &b.Limit, &b.Spent)

    if err != nil {
        return model.Budget{}, fmt.Errorf("failed to create budget: %w", err)
    }

    return b, nil
}
```

### GetByUserAndCategoryTx

```go
func (r *PostgresBudgetRepo) GetByUserAndCategoryTx(ctx context.Context, tx *sql.Tx, userID, category string) (model.Budget, error) {
    query := `
        SELECT id, user_id, category, limit_amount, spent
        FROM budgets
        WHERE user_id = $1 AND category = $2
    `

    var b model.Budget
    err := tx.QueryRowContext(ctx, query, userID, category).Scan(
        &b.ID,
        &b.UserID,
        &b.Category,
        &b.Limit,
        &b.Spent,
    )

    if err == sql.ErrNoRows {
        return model.Budget{}, ErrBudgetNotFound
    }
    if err != nil {
        return model.Budget{}, fmt.Errorf("failed to get budget: %w", err)
    }

    return b, nil
}
```

### AddSpentTx

```go
func (r *PostgresBudgetRepo) AddSpentTx(ctx context.Context, tx *sql.Tx, userID, category string, amount float64) (model.Budget, error) {
    query := `
        UPDATE budgets
        SET spent = spent + $3
        WHERE user_id = $1 AND category = $2
        RETURNING id, user_id, category, limit_amount, spent
    `

    var b model.Budget
    err := tx.QueryRowContext(ctx, query, userID, category, amount).Scan(
        &b.ID,
        &b.UserID,
        &b.Category,
        &b.Limit,
        &b.Spent,
    )

    if err == sql.ErrNoRows {
        return model.Budget{}, ErrBudgetNotFound
    }
    if err != nil {
        return model.Budget{}, fmt.Errorf("failed to add spent: %w", err)
    }

    return b, nil
}
```

### Why AddSpentTx is Critical

When recording a transaction, we must update the budget **atomically**:

```go
// In TransactionService
func (s *TransactionService) RecordTransactionAtomic(...) (model.Transaction, error) {
    return s.txManager.RunInTransaction(ctx, func(tx *sql.Tx) error {
        // Step 1: Create transaction record
        created, err := txRepo.CreateTx(ctx, tx, transaction)
        if err != nil {
            return err
        }
        
        // Step 2: Update budget (ATOMIC with transaction creation)
        _, err := budgetRepo.AddSpentTx(ctx, tx, userID, category, amount)
        if err != nil {
            return err  // Both operations rolled back
        }
        
        return nil  // Both committed
    })
}
```

Without `AddSpentTx`:
- Transaction created ✓
- Budget update fails ✗
- Result: Data inconsistency (transaction exists but budget not updated)

With `AddSpentTx`:
- Both succeed → both committed ✓
- Either fails → both rolled back ✓
- Result: Always consistent

---

## Complete Implementation

```go
package repository

import (
    "context"
    "database/sql"
    "dime-api/internal/model"
    "fmt"
)

// PostgresBudgetRepo implements BudgetRepository using PostgreSQL
type PostgresBudgetRepo struct {
    db *sql.DB
}

// NewPostgresBudgetRepo creates a new PostgreSQL budget repository
func NewPostgresBudgetRepo(db *sql.DB) BudgetRepository {
    return &PostgresBudgetRepo{db: db}
}

// Create inserts a new budget into the database (upserts on conflict)
func (r *PostgresBudgetRepo) Create(ctx context.Context, b model.Budget) (model.Budget, error) {
    query := `
        INSERT INTO budgets (id, user_id, category, limit_amount, spent)
        VALUES ($1, $2, $3, $4, $5)
        ON CONFLICT (user_id, category) DO UPDATE SET
            limit_amount = EXCLUDED.limit_amount
        RETURNING id, user_id, category, limit_amount, spent
    `

    err := r.db.QueryRowContext(ctx,
        query,
        b.ID,
        b.UserID,
        b.Category,
        b.Limit,
        b.Spent,
    ).Scan(&b.ID, &b.UserID, &b.Category, &b.Limit, &b.Spent)

    if err != nil {
        return model.Budget{}, fmt.Errorf("failed to create budget: %w", err)
    }

    return b, nil
}

// GetByUserAndCategory retrieves a budget by user ID and category
func (r *PostgresBudgetRepo) GetByUserAndCategory(ctx context.Context, userID, category string) (model.Budget, error) {
    query := `
        SELECT id, user_id, category, limit_amount, spent
        FROM budgets
        WHERE user_id = $1 AND category = $2
    `

    var b model.Budget
    err := r.db.QueryRowContext(ctx, query, userID, category).Scan(
        &b.ID,
        &b.UserID,
        &b.Category,
        &b.Limit,
        &b.Spent,
    )

    if err == sql.ErrNoRows {
        return model.Budget{}, ErrBudgetNotFound
    }
    if err != nil {
        return model.Budget{}, fmt.Errorf("failed to get budget: %w", err)
    }

    return b, nil
}

// AddSpent increments the spent amount on a budget (atomic)
func (r *PostgresBudgetRepo) AddSpent(ctx context.Context, userID, category string, amount float64) (model.Budget, error) {
    query := `
        UPDATE budgets
        SET spent = spent + $3
        WHERE user_id = $1 AND category = $2
        RETURNING id, user_id, category, limit_amount, spent
    `

    var b model.Budget
    err := r.db.QueryRowContext(ctx, query, userID, category, amount).Scan(
        &b.ID,
        &b.UserID,
        &b.Category,
        &b.Limit,
        &b.Spent,
    )

    if err == sql.ErrNoRows {
        return model.Budget{}, ErrBudgetNotFound
    }
    if err != nil {
        return model.Budget{}, fmt.Errorf("failed to add spent: %w", err)
    }

    return b, nil
}

// CreateTx inserts a new budget within an existing transaction
func (r *PostgresBudgetRepo) CreateTx(ctx context.Context, tx *sql.Tx, b model.Budget) (model.Budget, error) {
    query := `
        INSERT INTO budgets (id, user_id, category, limit_amount, spent)
        VALUES ($1, $2, $3, $4, $5)
        ON CONFLICT (user_id, category) DO UPDATE SET
            limit_amount = EXCLUDED.limit_amount
        RETURNING id, user_id, category, limit_amount, spent
    `

    err := tx.QueryRowContext(ctx,
        query,
        b.ID,
        b.UserID,
        b.Category,
        b.Limit,
        b.Spent,
    ).Scan(&b.ID, &b.UserID, &b.Category, &b.Limit, &b.Spent)

    if err != nil {
        return model.Budget{}, fmt.Errorf("failed to create budget: %w", err)
    }

    return b, nil
}

// GetByUserAndCategoryTx retrieves a budget within an existing transaction
func (r *PostgresBudgetRepo) GetByUserAndCategoryTx(ctx context.Context, tx *sql.Tx, userID, category string) (model.Budget, error) {
    query := `
        SELECT id, user_id, category, limit_amount, spent
        FROM budgets
        WHERE user_id = $1 AND category = $2
    `

    var b model.Budget
    err := tx.QueryRowContext(ctx, query, userID, category).Scan(
        &b.ID,
        &b.UserID,
        &b.Category,
        &b.Limit,
        &b.Spent,
    )

    if err == sql.ErrNoRows {
        return model.Budget{}, ErrBudgetNotFound
    }
    if err != nil {
        return model.Budget{}, fmt.Errorf("failed to get budget: %w", err)
    }

    return b, nil
}

// AddSpentTx increments spent amount within an existing transaction
func (r *PostgresBudgetRepo) AddSpentTx(ctx context.Context, tx *sql.Tx, userID, category string, amount float64) (model.Budget, error) {
    query := `
        UPDATE budgets
        SET spent = spent + $3
        WHERE user_id = $1 AND category = $2
        RETURNING id, user_id, category, limit_amount, spent
    `

    var b model.Budget
    err := tx.QueryRowContext(ctx, query, userID, category, amount).Scan(
        &b.ID,
        &b.UserID,
        &b.Category,
        &b.Limit,
        &b.Spent,
    )

    if err == sql.ErrNoRows {
        return model.Budget{}, ErrBudgetNotFound
    }
    if err != nil {
        return model.Budget{}, fmt.Errorf("failed to add spent: %w", err)
    }

    return b, nil
}
```

---

## Key Concepts Learned

### 1. UPSERT Pattern (INSERT ... ON CONFLICT)

**When to use:** When you want "insert or update" behavior.

**Syntax:**
```sql
INSERT INTO table (...)
VALUES (...)
ON CONFLICT (conflict_column) DO UPDATE SET
    column = EXCLUDED.column
```

**EXCLUDED** refers to the row values that attempted to insert.

### 2. Atomic Operations

**Read-Modify-Write (Problematic):**
```go
b, _ := repo.GetByUserAndCategory(ctx, userID, category)
b.Spent += amount  // Race condition here!
repo.Update(ctx, b)
```

**Atomic (Correct):**
```sql
UPDATE budgets SET spent = spent + $3 WHERE ...
```

### 3. Composite Key Lookups

Budgets are identified by combination of fields:
- Single field lookup: `WHERE id = $1`
- Composite lookup: `WHERE user_id = $1 AND category = $2`

Database index on `(user_id, category)` makes this fast.

### 4. Partial Updates with RETURNING

```sql
UPDATE budgets
SET spent = spent + $3           -- Only update spent
WHERE user_id = $1 AND category = $2
RETURNING ...                     -- Get all fields back
```

This pattern:
- Updates only what changed
- Returns complete updated entity
- Single round-trip to database

---

## Testing

See Phase 7 for integration tests. Key test scenarios:

1. **Create**: New budget created
2. **Create Upsert**: Existing budget limit updated, spent preserved
3. **GetByUserAndCategory (found)**: Retrieve existing budget
4. **GetByUserAndCategory (not found)**: Returns ErrBudgetNotFound
5. **AddSpent**: Spent amount incremented atomically
6. **AddSpent_NotFound**: Error when budget doesn't exist
7. **CreateTx**: Works within transaction
8. **AddSpentTx**: Works within transaction
9. **TransactionRollback**: Changes rolled back on error

---

## Comparison: Transaction vs Budget Repositories

| Aspect | Transaction Repository | Budget Repository |
|--------|----------------------|-------------------|
| **Primary Key** | `id` (UUID) | `(user_id, category)` composite |
| **Lookup** | By ID | By user + category |
| **Upsert** | No | Yes (ON CONFLICT) |
| **Partial Update** | No | Yes (AddSpent) |
| **Update Pattern** | N/A | Atomic increment |
| **Unique Constraint** | Primary key | `(user_id, category)` |

---

## Next Phase

Phase 7 implements integration testing for both repositories against real PostgreSQL.

---

*Part of: Backend Learning Project - Phase 6*
*Prerequisite: Phase 5 (Transaction Repository)*
*Next: Phase 7 (Integration Testing)*
