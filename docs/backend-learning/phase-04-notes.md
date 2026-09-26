# Phase 4: Go Database Integration

## Overview

Phase 4 focuses on production-ready database connectivity patterns in Go. We migrate from `lib/pq` to `pgx`, implement connection pooling, add context support for timeouts and cancellation, create a transaction management system, and add observability through structured logging and health checks.

This phase establishes the foundation that Phases 5, 6, and 7 build upon.

## Learning Objectives

- Understand why `pgx` is preferred over `lib/pq`
- Configure production-ready connection pooling
- Implement context-aware database operations
- Build a transaction management system for atomic operations
- Add structured logging and health checks
- Handle database errors properly

---

## Milestone 1: pgx Migration & Connection Pooling

### Why Migrate from lib/pq to pgx?

**lib/pq Issues:**
- In maintenance mode (no new features being added)
- Uses cgo (C bindings) which is slower and has C dependencies
- Limited support for PostgreSQL-specific features

**pgx Advantages:**
- Native Go implementation (30-50% better performance)
- Actively maintained with frequent updates
- Better PostgreSQL type support (arrays, JSONB, UUIDs)
- Superior context support
- Built-in connection pooling

### Migration Changes

The change is minimal - mostly just the import:

**Before (lib/pq):**
```go
import _ "github.com/lib/pq"

db, err := sql.Open("postgres", connStr)
```

**After (pgx):**
```go
import _ "github.com/jackc/pgx/v5/stdlib"

db, err := sql.Open("pgx", connStr)
```

**Key difference:** The driver name changes from `"postgres"` to `"pgx"`.

### Connection Pool Configuration

```go
// Based on PostgreSQL max_connections = 100
// Assuming 4 app instances: 100/4 = 25 max connections per instance
db.SetMaxOpenConns(25)                  // Maximum concurrent connections
db.SetMaxIdleConns(10)                  // Connections kept warm
db.SetConnMaxLifetime(5 * time.Minute)  // Recycle connections periodically
db.SetConnMaxIdleTime(1 * time.Minute)  // Close idle connections
```

**Pool Settings Explained:**

| Setting | Value | Rationale |
|---------|-------|-----------|
| `MaxOpenConns` | 25 | 25% of total, leaves headroom for admin connections |
| `MaxIdleConns` | 10 | Quick response for burst traffic |
| `ConnMaxLifetime` | 5 min | Recycle before server-side timeout |
| `ConnMaxIdleTime` | 1 min | Free up unused resources |

**Why these numbers?**
- Default PostgreSQL `max_connections` is 100
- With 4 app instances: 100 / 4 = 25 per instance
- 10 idle keeps connections warm without consuming too many
- 5-minute lifetime prevents connection leaks and handles network issues

### Pool Monitoring

```go
// Log pool statistics
stats := db.Stats()
slog.Info("Database pool statistics",
    slog.Int("open_connections", stats.OpenConnections),
    slog.Int("in_use", stats.InUse),
    slog.Int("idle", stats.Idle),
    slog.Int64("wait_count", stats.WaitCount),
    slog.Duration("wait_duration", stats.WaitDuration),
)
```

**Key Metrics:**
- `OpenConnections`: Currently open (in_use + idle)
- `InUse`: Actively executing queries
- `Idle`: Available in pool
- `WaitCount`: Total waits for connection (should be low)
- `WaitDuration`: Total time waiting (should be low)

---

## Milestone 2: Context-Aware Operations

### Why Context Matters

Without context:
```go
// Can hang forever if database is slow or network is down!
rows, err := db.Query("SELECT * FROM large_table")
```

With context:
```go
// Fails after 5 seconds with context deadline exceeded
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
rows, err := db.QueryContext(ctx, "SELECT * FROM large_table")
```

### Context Flow Architecture

```
HTTP Request
    ↓ (inherits HTTP context with cancellation)
Handler: ctx, cancel := context.WithTimeout(r.Context(), 5s)
    ↓ (new context with 5s timeout)
Service: RecordTransaction(ctx, ...)
    ↓
Repository: repo.Create(ctx, transaction)
    ↓
Database: db.QueryRowContext(ctx, query, ...)
    ↓
PostgreSQL: Receives context, cancels query if context expires
```

**Benefits:**
- Request cancellation propagates to database
- Timeout protection against slow queries
- Resource cleanup when clients disconnect
- Distributed tracing support

### Repository Interface with Context

```go
// internal/repository/interface.go
type TransactionRepository interface {
    Create(ctx context.Context, t model.Transaction) (model.Transaction, error)
    GetByID(ctx context.Context, id string) (model.Transaction, error)
    ListByUser(ctx context.Context, userID string) ([]model.Transaction, error)
}

type BudgetRepository interface {
    Create(ctx context.Context, b model.Budget) (model.Budget, error)
    GetByUserAndCategory(ctx context.Context, userID, category string) (model.Budget, error)
    AddSpent(ctx context.Context, userID, category string, amount float64) (model.Budget, error)
}
```

**Key Design Decision:** `context.Context` as the first parameter - this is the Go standard pattern.

### Handler Implementation

```go
const dbTimeout = 5 * time.Second

func (h *TransactionHandler) Create(w http.ResponseWriter, r *http.Request) {
    // Create context with timeout from HTTP request context
    ctx, cancel := context.WithTimeout(r.Context(), dbTimeout)
    defer cancel()
    
    // Parse request...
    
    // Pass context through the stack
    t, err := h.service.RecordTransaction(ctx, userID, txType, amount, category)
    if err != nil {
        // Handle error...
    }
    
    // Return response...
}
```

### Context Cancellation Example

```go
// Client disconnects after 100ms
// Handler creates 5s timeout context
// Database query takes 10s

// What happens:
// 1. Client disconnects → HTTP context cancelled
// 2. Handler's context inherits cancellation
// 3. Database query receives cancellation
// 4. Query aborted, connection returned to pool
// 5. Resources freed
```

---

## Milestone 3: Transaction Management

### Why Database Transactions?

**Without transactions (risk of inconsistency):**
```
1. Create transaction record ✓
2. Update budget spent ✗ (fails!)
Result: Transaction exists but budget not updated (INCONSISTENT)
```

**With transactions (atomic):**
```
1. Create transaction record ✓
2. Update budget spent ✗ (fails!)
Result: Both rolled back, database remains CONSISTENT
```

### Transaction Manager Implementation

File: `internal/database/transaction.go`

```go
package database

import (
    "context"
    "database/sql"
)

// TransactionManager handles database transactions
type TransactionManager struct {
    db *sql.DB
}

// NewTransactionManager creates a new transaction manager
func NewTransactionManager(db *sql.DB) *TransactionManager {
    return &TransactionManager{db: db}
}

// RunInTransaction executes the given function within a database transaction
// Automatically handles commit/rollback based on function return value
func (tm *TransactionManager) RunInTransaction(
    ctx context.Context,
    fn func(*sql.Tx) error,
) error {
    // Begin transaction with context support
    tx, err := tm.db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    
    // Handle panics by rolling back
    defer func() {
        if p := recover(); p != nil {
            tx.Rollback()
            panic(p)  // Re-panic after rollback
        }
    }()
    
    // Execute the function
    if err := fn(tx); err != nil {
        tx.Rollback()  // Rollback on error
        return err
    }
    
    return tx.Commit()  // Commit on success
}
```

**Key Features:**
- `BeginTx`: Starts transaction with context support
- Panic recovery: Ensures rollback even on panic
- Automatic rollback: On any error from the function
- Automatic commit: When function returns nil

### Transaction-Aware Repository Pattern

For atomic operations, repositories need transaction-aware methods:

```go
// Standard method (uses connection pool)
func (r *PostgresTransactionRepo) Create(ctx context.Context, t model.Transaction) (model.Transaction, error)

// Transaction method (uses provided transaction)
func (r *PostgresTransactionRepo) CreateTx(ctx context.Context, tx *sql.Tx, t model.Transaction) (model.Transaction, error)
```

**The only difference:**
- Standard: `r.db.QueryRowContext(...)`
- Transaction: `tx.QueryRowContext(...)`

### Using the Transaction Manager

```go
func (s *TransactionService) RecordTransactionAtomic(
    ctx context.Context,
    userID string,
    txType model.TransactionType,
    amount float64,
    category string,
) (model.Transaction, error) {
    var saved model.Transaction
    
    err := s.txManager.RunInTransaction(ctx, func(tx *sql.Tx) error {
        // Step 1: Create transaction within the database transaction
        t := model.Transaction{
            ID:        generateID(),
            UserID:    userID,
            Type:      txType,
            Amount:    amount,
            Category:  category,
            CreatedAt: time.Now(),
        }
        
        created, err := s.txRepo.CreateTx(ctx, tx, t)
        if err != nil {
            return err  // Triggers rollback
        }
        saved = created
        
        // Step 2: Update budget (atomic with transaction creation)
        if txType == model.TypeExpense {
            _, err := s.budgetRepo.AddSpentTx(ctx, tx, userID, category, amount)
            if err != nil && err != repository.ErrBudgetNotFound {
                return err  // Triggers rollback
            }
        }
        
        return nil  // Triggers commit
    })
    
    return saved, err
}
```

**Flow:**
1. `RunInTransaction` starts database transaction
2. `CreateTx` runs within transaction
3. `AddSpentTx` runs within same transaction
4. If any error → automatic rollback
5. If all succeed → automatic commit

### Transaction Isolation Levels

```go
// Default (Read Committed)
txManager.RunInTransaction(ctx, fn)

// Serializable (highest isolation)
txManager.RunInTransactionWithIsolation(ctx, sql.LevelSerializable, fn)
```

**Isolation Levels:**

| Level | Dirty Read | Non-repeatable | Phantom | Use Case |
|-------|-----------|----------------|---------|----------|
| Read Uncommitted | ✓ | ✓ | ✓ | Rarely used |
| Read Committed | ✗ | ✓ | ✓ | Default, good balance |
| Repeatable Read | ✗ | ✗ | ✓ | Long transactions |
| Serializable | ✗ | ✗ | ✗ | Critical consistency |

---

## Milestone 4: Error Handling & Observability

### Structured Logging with log/slog

**Why slog over standard log?**
- Structured output (JSON for production)
- Key-value pairs for filtering/searching
- Standard library (Go 1.21+)
- Levels (Debug, Info, Warn, Error)

**Setup:**
```go
// Initialize
logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
slog.SetDefault(logger)

// Usage
slog.Info("Database connection pool configured",
    slog.Int("max_open_conns", 25),
    slog.Duration("conn_max_lifetime", 5*time.Minute),
)

// Output: {"time":"2026-09-18T...","level":"INFO","msg":"...","max_open_conns":25}
```

### Health Check Endpoint

```go
// GET /health
func healthCheck(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Log pool stats on each health check
        logPoolStats(db)
        
        // Check database connectivity with timeout
        ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
        defer cancel()
        
        if err := db.PingContext(ctx); err != nil {
            slog.Error("Health check failed", slog.String("error", err.Error()))
            w.WriteHeader(http.StatusServiceUnavailable)
            w.Header().Set("Content-Type", "application/json")
            fmt.Fprintf(w, `{"status":"unhealthy","error":"database unreachable"}`)
            return
        }
        
        // Get pool stats
        stats := db.Stats()
        
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        fmt.Fprintf(w, `{"status":"healthy","open_connections":%d,"in_use":%d,"idle":%d}`,
            stats.OpenConnections, stats.InUse, stats.Idle)
    }
}
```

**Response when healthy:**
```json
{
  "status": "healthy",
  "open_connections": 12,
  "in_use": 3,
  "idle": 9
}
```

**Response when unhealthy:**
```json
{
  "status": "unhealthy",
  "error": "database unreachable"
}
```

### Error Classification

```go
// IsRetryableError checks if error is transient (network, deadlock, etc.)
func IsRetryableError(err error) bool {
    var pgErr *pgconn.PgError
    if errors.As(err, &pgErr) {
        switch pgErr.Code {
        case "40001": // serialization_failure
        case "40P01": // deadlock_detected
        case "08006": // connection_failure
            return true
        }
    }
    return false
}

// IsNotFoundError checks for "not found" errors
func IsNotFoundError(err error) bool {
    return errors.Is(err, sql.ErrNoRows)
}
```

---

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                         HTTP Layer                          │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐ │
│  │   Handler   │  │   Handler   │  │   Health Check      │ │
│  │ Transaction │  │   Budget    │  │   /health           │ │
│  └──────┬──────┘  └──────┬──────┘  └─────────────────────┘ │
└─────────┼────────────────┼──────────────────────────────────┘
          │                │
          ▼                ▼
┌─────────────────────────────────────────────────────────────┐
│                     Service Layer                           │
│  ┌──────────────────────┐  ┌─────────────────────────────┐ │
│  │ TransactionService   │  │     BudgetService           │ │
│  │ - RecordTransaction  │  │  - CreateBudget             │ │
│  │ - RecordTransaction  │  │  - CheckBudgetStatus        │ │
│  │   Atomic             │  │  - RecordSpend              │ │
│  └──────────┬───────────┘  └──────────────┬──────────────┘ │
└─────────────┼─────────────────────────────┼──────────────────┘
              │                             │
              ▼                             ▼
┌─────────────────────────────────────────────────────────────┐
│                    Repository Layer                         │
│  ┌────────────────────────┐  ┌──────────────────────────┐  │
│  │ PostgresTransactionRepo│  │  PostgresBudgetRepo      │  │
│  │ - Create(ctx, ...)     │  │  - Create(ctx, ...)      │  │
│  │ - GetByID(ctx, ...)    │  │  - GetByUserAndCategory()│  │
│  │ - ListByUser(ctx, ...) │  │  - AddSpent(ctx, ...)    │  │
│  └──────────┬─────────────┘  └────────────┬───────────────┘  │
└─────────────┼─────────────────────────────┼──────────────────┘
              │                             │
              └─────────────┬───────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                   Database Layer                            │
│  ┌───────────────────────────────────────────────────────┐  │
│  │              PostgreSQL (pgx driver)                  │  │
│  │  - Connection Pool (25 max open, 10 max idle)        │  │
│  │  - Connection Lifetime: 5 min                        │  │
│  │  - Context Support                                   │  │
│  │  - Transaction Support                               │  │
│  └───────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

**Context Flow:**
```
HTTP Request Context → Handler Timeout Context → Service → Repository → Database
        ↓                      ↓                    ↓           ↓           ↓
   Connection Close        5s Timeout          Propagate   Propagate   Query Cancel
```

---

## File Changes Summary

### New Files

1. **`internal/database/transaction.go`** - Transaction manager
2. **`internal/repository/interface.go`** - Repository interfaces with context

### Modified Files

1. **`cmd/api/main.go`**
   - Migrated to pgx driver
   - Added connection pool configuration
   - Added structured logging
   - Added health check endpoint
   - Added TransactionManager wiring

2. **`go.mod`** - Added pgx dependencies

---

## Performance Considerations

### Connection Pool Tuning

**Formula for MaxOpenConns:**
```
MaxOpenConns = PostgreSQL_max_connections / Number_of_app_instances

Example:
- PostgreSQL max_connections: 100
- App instances: 4
- MaxOpenConns per instance: 25
```

**When to Adjust:**
- **Increase** if: High concurrency, low latency requirements, low instance count
- **Decrease** if: Memory constraints, many app instances, database CPU issues

### Context Timeout Guidelines

| Operation Type | Recommended Timeout | Reason |
|----------------|---------------------|---------|
| Simple read | 1-3 seconds | Fast index lookup |
| Complex query | 5-10 seconds | Joins, aggregations |
| Transaction | 10-30 seconds | Multiple operations |
| Batch operation | 30-60 seconds | Large data operations |

### Query Best Practices

- Use `QueryRowContext` for single-row results
- Use `QueryContext` with `defer rows.Close()`
- Use `ExecContext` for INSERT/UPDATE/DELETE
- Always pass context for cancellation support
- Use transactions for atomic operations
- Add appropriate database indexes

---

## Common Pitfalls & Solutions

### 1. Connection Leaks

**Problem:**
```go
rows, _ := db.QueryContext(ctx, "SELECT ...")
// Forgot to close!
```

**Solution:**
```go
rows, err := db.QueryContext(ctx, "SELECT ...")
if err != nil {
    return err
}
defer rows.Close() // Always defer close
```

### 2. Context Cancellation Ignored

**Problem:**
```go
func (r *Repo) Get(ctx context.Context, id string) (Model, error) {
    row := r.db.QueryRow("SELECT ...", id) // Ignoring ctx!
}
```

**Solution:**
```go
func (r *Repo) Get(ctx context.Context, id string) (Model, error) {
    row := r.db.QueryRowContext(ctx, "SELECT ...", id)
}
```

### 3. Pool Exhaustion

**Problem:**
```
"pq: sorry, too many clients already"
```

**Solution:**
- Monitor `WaitCount` and `WaitDuration` metrics
- Adjust `MaxOpenConns` based on load
- Check for connection leaks (unclosed rows)
- Scale horizontally (more app instances with lower MaxOpenConns each)

---

## Monitoring & Alerts

### Key Metrics to Monitor

```go
stats := db.Stats()

// Critical metrics:
stats.OpenConnections  // Current open connections
stats.InUse            // Connections currently in use
stats.WaitCount        // Total number of connection waits
stats.WaitDuration     // Total time waited for connections

// Alert if:
// - OpenConnections > MaxOpenConns * 0.8 (80% capacity)
// - WaitCount is increasing rapidly (pool exhaustion)
// - WaitDuration > 1 second (slow connection acquisition)
```

### Health Check Monitoring

- HTTP 200 with `{"status": "healthy"}` = OK
- HTTP 503 with `{"status": "unhealthy"}` = Alert
- No response/timeout = Critical alert

---

## Next Steps (Phases 5-7)

Phase 4 establishes the infrastructure. The next phases build upon this:

- **Phase 5**: Implement Transaction Repository (uses context, transaction methods)
- **Phase 6**: Implement Budget Repository (uses context, transaction methods)
- **Phase 7**: Integration testing (tests all Phase 4 features with real PostgreSQL)

---

## References

- [pgx Documentation](https://github.com/jackc/pgx)
- [Go database/sql Tutorial](http://go-database-sql.org/)
- [PostgreSQL Connection Pooling](https://www.postgresql.org/docs/current/runtime-config-connection.html)
- [Go Context Package](https://pkg.go.dev/context)
- [log/slog Package](https://pkg.go.dev/log/slog)

---

## Conclusion

Phase 4 establishes the database infrastructure foundation:

- **Performance:** pgx driver with optimized connection pooling
- **Reliability:** Context timeouts and cancellation support
- **Consistency:** Transaction management for atomic operations
- **Observability:** Structured logging and health checks

This infrastructure enables Phases 5-7 to focus on business logic while inheriting production-ready patterns.

---

*Part of: Backend Learning Project - Phase 4*
*Prerequisite: Phase 3 (Database Migrations)*
*Next: Phase 5 (Transaction Repository Implementation)*
