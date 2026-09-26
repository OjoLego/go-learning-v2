# Phase 5: Repository Implementation - Transactions

## Overview

Phase 5 implements the PostgreSQL-backed Transaction Repository. This is where we create the concrete implementation of the `TransactionRepository` interface defined in Phase 4, using raw SQL queries with PostgreSQL.

## Learning Objectives

- Implement repository pattern with PostgreSQL
- Write SQL queries for CRUD operations
- Handle context cancellation in database operations
- Implement proper error handling (not found errors)
- Create transaction-aware methods for atomic operations

---

## Repository Interface Recap

From Phase 4, we have this interface:

```go
// internal/repository/interface.go
type TransactionRepository interface {
    Create(ctx context.Context, t model.Transaction) (model.Transaction, error)
    GetByID(ctx context.Context, id string) (model.Transaction, error)
    ListByUser(ctx context.Context, userID string) ([]model.Transaction, error)
}
```

**Key Design Decisions:**
- `context.Context` as first parameter (enables cancellation/timeouts)
- Return the created entity (useful for getting generated fields)
- Domain-specific errors (ErrNotFound)

---

## Milestone 1: Basic CRUD Implementation

### File: `internal/repository/postgres_transaction_repo.go`

#### Constructor

```go
type PostgresTransactionRepo struct {
    db *sql.DB
}

func NewPostgresTransactionRepo(db *sql.DB) TransactionRepository {
    return &PostgresTransactionRepo{db: db}
}
```

**Why this pattern?**
- Dependency injection: Pass `*sql.DB` rather than creating it
- Interface compliance: Return interface type, not concrete
- Testability: Can mock `*sql.DB` for unit tests

#### Create Operation

```go
func (r *PostgresTransactionRepo) Create(ctx context.Context, t model.Transaction) (model.Transaction, error) {
    query := `
        INSERT INTO transactions (id, user_id, type, amount, category, created_at)
        VALUES ($1, $2, $3, $4, $5, $6)
        RETURNING id, user_id, type, amount, category, created_at
    `

    err := r.db.QueryRowContext(ctx,
        query,
        t.ID,
        t.UserID,
        t.Type,
        t.Amount,
        t.Category,
        t.CreatedAt,
    ).Scan(&t.ID, &t.UserID, &t.Type, &t.Amount, &t.Category, &t.CreatedAt)

    if err != nil {
        return model.Transaction{}, fmt.Errorf("failed to create transaction: %w", err)
    }

    return t, nil
}
```

**Key Points:**
- Uses `$1, $2...` syntax for PostgreSQL parameter binding
- `QueryRowContext` for single-row INSERT with RETURNING clause
- Returns all fields (confirms database state)
- Wraps errors with context (fmt.Errorf with %w)

**SQL Explained:**
```sql
INSERT INTO transactions (...)  -- Insert the data
VALUES ($1, $2, $3, $4, $5, $6)  -- Parameter placeholders
RETURNING ...                   -- Return the inserted row
```

#### Read Operation (GetByID)

```go
func (r *PostgresTransactionRepo) GetByID(ctx context.Context, id string) (model.Transaction, error) {
    query := `
        SELECT id, user_id, type, amount, category, created_at
        FROM transactions
        WHERE id = $1
    `

    var t model.Transaction
    err := r.db.QueryRowContext(ctx, query, id).Scan(
        &t.ID,
        &t.UserID,
        &t.Type,
        &t.Amount,
        &t.Category,
        &t.CreatedAt,
    )

    if err == sql.ErrNoRows {
        return model.Transaction{}, ErrNotFound
    }
    if err != nil {
        return model.Transaction{}, fmt.Errorf("failed to get transaction: %w", err)
    }

    return t, nil
}
```

**Key Points:**
- `sql.ErrNoRows` indicates "not found" - convert to domain error
- Field order in Scan must match SELECT order
- Always check `sql.ErrNoRows` before other errors

**Error Pattern:**
```go
if err == sql.ErrNoRows {
    return model.Transaction{}, ErrNotFound  // Domain error
}
if err != nil {
    return model.Transaction{}, fmt.Errorf("...: %w", err)  // Wrap other errors
}
```

#### List Operation (ListByUser)

```go
func (r *PostgresTransactionRepo) ListByUser(ctx context.Context, userID string) ([]model.Transaction, error) {
    query := `
        SELECT id, user_id, type, amount, category, created_at
        FROM transactions
        WHERE user_id = $1
        ORDER BY created_at DESC
    `

    rows, err := r.db.QueryContext(ctx, query, userID)
    if err != nil {
        return nil, fmt.Errorf("failed to list transactions: %w", err)
    }
    defer rows.Close()  // Important! Prevents connection leak

    var transactions []model.Transaction
    for rows.Next() {
        var t model.Transaction
        err := rows.Scan(
            &t.ID,
            &t.UserID,
            &t.Type,
            &t.Amount,
            &t.Category,
            &t.CreatedAt,
        )
        if err != nil {
            return nil, fmt.Errorf("failed to scan transaction: %w", err)
        }
        transactions = append(transactions, t)
    }

    if err = rows.Err(); err != nil {
        return nil, fmt.Errorf("error iterating transactions: %w", err)
    }

    return transactions, nil
}
```

**Key Points:**
- `defer rows.Close()` - Always close rows to free connection
- Check `rows.Err()` after iteration (catches errors during scan)
- Empty result is `nil` error with empty slice (not an error)

**Common Pitfall:**
```go
// BAD - connection leak!
rows, _ := db.QueryContext(ctx, query)
for rows.Next() { ... }
// rows never closed

// GOOD - proper cleanup
rows, err := db.QueryContext(ctx, query)
if err != nil { return err }
defer rows.Close()
for rows.Next() { ... }
if err = rows.Err(); err != nil { return err }
```

---

## Milestone 2: Transaction-Aware Methods

### Why Transaction-Aware Methods?

When we need to perform multiple database operations atomically (all succeed or all fail), we use SQL transactions. The repository needs methods that can work within an existing transaction.

### Pattern: Dual Methods

For each operation, we create two versions:
1. **Standard version**: Uses connection pool (`r.db`)
2. **Transaction version**: Uses provided transaction (`*sql.Tx`)

#### CreateTx Implementation

```go
func (r *PostgresTransactionRepo) CreateTx(ctx context.Context, tx *sql.Tx, t model.Transaction) (model.Transaction, error) {
    query := `
        INSERT INTO transactions (id, user_id, type, amount, category, created_at)
        VALUES ($1, $2, $3, $4, $5, $6)
        RETURNING id, user_id, type, amount, category, created_at
    `

    err := tx.QueryRowContext(ctx,  // Note: uses tx, not r.db
        query,
        t.ID,
        t.UserID,
        t.Type,
        t.Amount,
        t.Category,
        t.CreatedAt,
    ).Scan(&t.ID, &t.UserID, &t.Type, &t.Amount, &t.Category, &t.CreatedAt)

    if err != nil {
        return model.Transaction{}, fmt.Errorf("failed to create transaction: %w", err)
    }

    return t, nil
}
```

**Key Differences:**
- `tx.QueryRowContext` instead of `r.db.QueryRowContext`
- No need to check `sql.ErrNoRows` for INSERT
- Same query, different executor

#### GetByIDTx Implementation

```go
func (r *PostgresTransactionRepo) GetByIDTx(ctx context.Context, tx *sql.Tx, id string) (model.Transaction, error) {
    query := `
        SELECT id, user_id, type, amount, category, created_at
        FROM transactions
        WHERE id = $1
    `

    var t model.Transaction
    err := tx.QueryRowContext(ctx, query, id).Scan(
        &t.ID,
        &t.UserID,
        &t.Type,
        &t.Amount,
        &t.Category,
        &t.CreatedAt,
    )

    if err == sql.ErrNoRows {
        return model.Transaction{}, ErrNotFound
    }
    if err != nil {
        return model.Transaction{}, fmt.Errorf("failed to get transaction: %w", err)
    }

    return t, nil
}
```

### When to Use Which?

| Scenario | Method to Use |
|----------|--------------|
| Single operation | `Create()`, `GetByID()` |
| Multiple operations atomic | `CreateTx()`, `GetByIDTx()` within `RunInTransaction()` |
| Read-only, consistency needed | `GetByIDTx()` within transaction |

---

## Complete Implementation

### File Structure

```
internal/repository/
├── interface.go                    # Repository interfaces
├── postgres_transaction_repo.go    # This phase's implementation
└── postgres_budget_repo.go         # Phase 6 implementation
```

### Full Code

```go
package repository

import (
    "context"
    "database/sql"
    "dime-api/internal/model"
    "fmt"
)

// PostgresTransactionRepo implements TransactionRepository using PostgreSQL
type PostgresTransactionRepo struct {
    db *sql.DB
}

// NewPostgresTransactionRepo creates a new PostgreSQL transaction repository
func NewPostgresTransactionRepo(db *sql.DB) TransactionRepository {
    return &PostgresTransactionRepo{db: db}
}

// Create inserts a new transaction into the database
func (r *PostgresTransactionRepo) Create(ctx context.Context, t model.Transaction) (model.Transaction, error) {
    query := `
        INSERT INTO transactions (id, user_id, type, amount, category, created_at)
        VALUES ($1, $2, $3, $4, $5, $6)
        RETURNING id, user_id, type, amount, category, created_at
    `

    err := r.db.QueryRowContext(ctx,
        query,
        t.ID,
        t.UserID,
        t.Type,
        t.Amount,
        t.Category,
        t.CreatedAt,
    ).Scan(&t.ID, &t.UserID, &t.Type, &t.Amount, &t.Category, &t.CreatedAt)

    if err != nil {
        return model.Transaction{}, fmt.Errorf("failed to create transaction: %w", err)
    }

    return t, nil
}

// GetByID retrieves a transaction by its ID
func (r *PostgresTransactionRepo) GetByID(ctx context.Context, id string) (model.Transaction, error) {
    query := `
        SELECT id, user_id, type, amount, category, created_at
        FROM transactions
        WHERE id = $1
    `

    var t model.Transaction
    err := r.db.QueryRowContext(ctx, query, id).Scan(
        &t.ID,
        &t.UserID,
        &t.Type,
        &t.Amount,
        &t.Category,
        &t.CreatedAt,
    )

    if err == sql.ErrNoRows {
        return model.Transaction{}, ErrNotFound
    }
    if err != nil {
        return model.Transaction{}, fmt.Errorf("failed to get transaction: %w", err)
    }

    return t, nil
}

// ListByUser retrieves all transactions for a specific user
func (r *PostgresTransactionRepo) ListByUser(ctx context.Context, userID string) ([]model.Transaction, error) {
    query := `
        SELECT id, user_id, type, amount, category, created_at
        FROM transactions
        WHERE user_id = $1
        ORDER BY created_at DESC
    `

    rows, err := r.db.QueryContext(ctx, query, userID)
    if err != nil {
        return nil, fmt.Errorf("failed to list transactions: %w", err)
    }
    defer rows.Close()

    var transactions []model.Transaction
    for rows.Next() {
        var t model.Transaction
        err := rows.Scan(
            &t.ID,
            &t.UserID,
            &t.Type,
            &t.Amount,
            &t.Category,
            &t.CreatedAt,
        )
        if err != nil {
            return nil, fmt.Errorf("failed to scan transaction: %w", err)
        }
        transactions = append(transactions, t)
    }

    if err = rows.Err(); err != nil {
        return nil, fmt.Errorf("error iterating transactions: %w", err)
    }

    return transactions, nil
}

// CreateTx inserts a new transaction within an existing transaction
func (r *PostgresTransactionRepo) CreateTx(ctx context.Context, tx *sql.Tx, t model.Transaction) (model.Transaction, error) {
    query := `
        INSERT INTO transactions (id, user_id, type, amount, category, created_at)
        VALUES ($1, $2, $3, $4, $5, $6)
        RETURNING id, user_id, type, amount, category, created_at
    `

    err := tx.QueryRowContext(ctx,
        query,
        t.ID,
        t.UserID,
        t.Type,
        t.Amount,
        t.Category,
        t.CreatedAt,
    ).Scan(&t.ID, &t.UserID, &t.Type, &t.Amount, &t.Category, &t.CreatedAt)

    if err != nil {
        return model.Transaction{}, fmt.Errorf("failed to create transaction: %w", err)
    }

    return t, nil
}

// GetByIDTx retrieves a transaction by its ID within an existing transaction
func (r *PostgresTransactionRepo) GetByIDTx(ctx context.Context, tx *sql.Tx, id string) (model.Transaction, error) {
    query := `
        SELECT id, user_id, type, amount, category, created_at
        FROM transactions
        WHERE id = $1
    `

    var t model.Transaction
    err := tx.QueryRowContext(ctx, query, id).Scan(
        &t.ID,
        &t.UserID,
        &t.Type,
        &t.Amount,
        &t.Category,
        &t.CreatedAt,
    )

    if err == sql.ErrNoRows {
        return model.Transaction{}, ErrNotFound
    }
    if err != nil {
        return model.Transaction{}, fmt.Errorf("failed to get transaction: %w", err)
    }

    return t, nil
}
```

---

## Key Concepts Learned

### 1. Parameter Binding

PostgreSQL uses `$1, $2, $3...` for parameters:

```go
query := "SELECT * FROM transactions WHERE user_id = $1 AND category = $2"
db.QueryContext(ctx, query, userID, category)
```

**Benefits:**
- Prevents SQL injection
- Type-safe
- Query plan caching

### 2. Error Handling Hierarchy

```go
err := row.Scan(...)

if err == sql.ErrNoRows {
    // Specific: Not found
    return ErrNotFound
}
if err != nil {
    // General: Database error
    return fmt.Errorf("query failed: %w", err)
}
```

### 3. Connection Management

| Method | Use For | Must Close? |
|--------|---------|-------------|
| `QueryRowContext` | Single row result | No |
| `QueryContext` | Multiple rows | Yes (defer rows.Close()) |
| `ExecContext` | INSERT/UPDATE/DELETE | No |

### 4. Context Propagation

Context flows through all layers:
```
HTTP Handler → Service → Repository → Database
     ↓              ↓           ↓            ↓
  5s timeout    5s timeout  5s timeout   Query timeout
```

If any layer cancels, all downstream operations stop.

---

## Testing

See Phase 7 for integration tests. Key test scenarios:

1. **Create**: Verify transaction is created with correct fields
2. **GetByID (found)**: Retrieve existing transaction
3. **GetByID (not found)**: Returns ErrNotFound
4. **ListByUser**: Returns transactions ordered by date
5. **CreateTx**: Works within transaction, commits successfully
6. **CreateTx Rollback**: Rolled back when transaction fails

---

## Next Phase

Phase 6 implements the Budget Repository with similar patterns but different SQL (upserts, updates).

---

*Part of: Backend Learning Project - Phase 5*
*Prerequisite: Phase 4 (Advanced Database Integration)*
*Next: Phase 6 (Budget Repository Implementation)*
