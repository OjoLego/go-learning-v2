# Phase 4: Advanced Database Integration

## Overview

Phase 4 builds upon Phase 3's basic database connectivity to implement production-ready database patterns. We migrated from `lib/pq` to `pgx`, added connection pooling, context support, transaction management, comprehensive error handling, and integration testing.

## Key Achievements

✅ Migrated from `lib/pq` to `pgx` (better performance, active maintenance)
✅ Configured production-ready connection pooling
✅ Added context support with 5-second timeouts
✅ Implemented transaction management for atomic operations
✅ Added structured logging with `log/slog`
✅ Created health check endpoint with pool metrics
✅ Built integration test suite with Docker

---

## Milestone 1: pgx Migration & Connection Pooling

### Why pgx?

**lib/pq issues:**
- In maintenance mode (no new features)
- Uses cgo (slower, C dependency)
- Limited PostgreSQL-specific features

**pgx advantages:**
- Native Go implementation (30-50% faster)
- Actively maintained
- Better PostgreSQL type support (arrays, JSONB, UUIDs)
- Superior context support

### Migration Changes

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

### Connection Pool Configuration

```go
// Based on PostgreSQL max_connections = 100
// Assuming 4 app instances: 100/4 = 25 per instance
db.SetMaxOpenConns(25)                  // Max concurrent connections
db.SetMaxIdleConns(10)                  // Keep connections warm
db.SetConnMaxLifetime(5 * time.Minute)  // Recycle connections
db.SetConnMaxIdleTime(1 * time.Minute)  // Close idle connections
```

**Pool Settings Explained:**

| Setting | Value | Rationale |
|---------|-------|-----------|
| `MaxOpenConns` | 25 | 25% of total, leaves headroom |
| `MaxIdleConns` | 10 | Quick response for burst traffic |
| `ConnMaxLifetime` | 5 min | Recycle before server timeout |
| `ConnMaxIdleTime` | 1 min | Free up unused resources |

### Pool Monitoring

```go
// Log pool statistics
stats := db.Stats()
slog.Info("Database pool statistics",
    slog.Int("open_connections", stats.OpenConnections),
    slog.Int("in_use", stats.InUse),
    slog.Int("idle", stats.Idle),
    slog.Int64("wait_count", stats.WaitCount),
)
```

---

## Milestone 2: Context-Aware Operations

### Why Context Matters

Without context:
```go
// Can hang forever if database is slow!
rows, err := db.Query("SELECT * FROM large_table")
```

With context:
```go
// Fails after 5 seconds
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
rows, err := db.QueryContext(ctx, "SELECT * FROM large_table")
```

### Context Flow Architecture

```
HTTP Request
    ↓ (inherits cancellation)
Handler: ctx, cancel := context.WithTimeout(r.Context(), 5s)
    ↓
Service: RecordTransaction(ctx, ...)
    ↓
Repository: repo.Create(ctx, transaction)
    ↓
Database: db.QueryRowContext(ctx, query, ...)
```

### Implementation Pattern

**Repository Interface:**
```go
type TransactionRepository interface {
    Create(ctx context.Context, t model.Transaction) (model.Transaction, error)
    GetByID(ctx context.Context, id string) (model.Transaction, error)
    ListByUser(ctx context.Context, userID string) ([]model.Transaction, error)
}
```

**Handler Usage:**
```go
const dbTimeout = 5 * time.Second

func (h *TransactionHandler) Create(w http.ResponseWriter, r *http.Request) {
    ctx, cancel := context.WithTimeout(r.Context(), dbTimeout)
    defer cancel()
    
    t, err := h.service.RecordTransaction(ctx, ...)
    // ...
}
```

**Database Query:**
```go
func (r *PostgresTransactionRepo) Create(ctx context.Context, t model.Transaction) (model.Transaction, error) {
    err := r.db.QueryRowContext(ctx, query, ...).Scan(...)
    // ...
}
```

---

## Milestone 3: Transaction Management

### Why Transactions?

**Without transactions (risk of inconsistency):**
```
1. Create transaction record ✓
2. Update budget spent ✗ (fails!)
Result: Transaction exists but budget not updated
```

**With transactions (atomic):**
```
1. Create transaction record ✓
2. Update budget spent ✗ (fails!)
Result: Both rolled back, database remains consistent
```

### Transaction Manager

Created `internal/database/transaction.go`:

```go
type TransactionManager struct {
    db *sql.DB
}

func (tm *TransactionManager) RunInTransaction(
    ctx context.Context,
    fn func(*sql.Tx) error,
) error {
    tx, err := tm.db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    
    defer func() {
        if p := recover(); p != nil {
            tx.Rollback()
            panic(p)
        }
    }()
    
    if err := fn(tx); err != nil {
        tx.Rollback()
        return err
    }
    
    return tx.Commit()
}
```

### Transaction-Aware Repository Methods

Added Tx methods to repositories:

```go
// Regular method (uses pool)
func (r *PostgresTransactionRepo) Create(ctx context.Context, t model.Transaction) (model.Transaction, error)

// Transaction method (uses provided tx)
func (r *PostgresTransactionRepo) CreateTx(ctx context.Context, tx *sql.Tx, t model.Transaction) (model.Transaction, error)
```

### Atomic Operation Example

```go
func (s *TransactionService) RecordTransactionAtomic(...) (model.Transaction, error) {
    var saved model.Transaction
    
    err := s.txManager.RunInTransaction(ctx, func(tx *sql.Tx) error {
        // Step 1: Create transaction
        created, err := txRepo.CreateTx(ctx, tx, t)
        if err != nil {
            return err
        }
        saved = created
        
        // Step 2: Update budget (atomic with transaction creation)
        if txType == model.TypeExpense {
            _, err := budgetRepo.AddSpentTx(ctx, tx, userID, category, amount)
            if err != nil && err != repository.ErrBudgetNotFound {
                return err // Triggers rollback
            }
        }
        
        return nil // Triggers commit
    })
    
    return saved, err
}
```

### Isolation Levels

```go
// Default (Read Committed)
txManager.RunInTransaction(ctx, fn)

// Serializable (highest isolation)
txManager.RunInTransactionWithIsolation(ctx, sql.LevelSerializable, fn)
```

**Isolation Levels Reference:**

| Level | Dirty Read | Non-repeatable Read | Phantom Read |
|-------|-----------|---------------------|--------------|
| Read Uncommitted | ✓ | ✓ | ✓ |
| Read Committed | ✗ | ✓ | ✓ |
| Repeatable Read | ✗ | ✗ | ✓ |
| Serializable | ✗ | ✗ | ✗ |

---

## Milestone 4: Error Handling & Observability

### Structured Logging

Migrated from `log` to `log/slog`:

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
        ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
        defer cancel()
        
        if err := db.PingContext(ctx); err != nil {
            w.WriteHeader(http.StatusServiceUnavailable)
            json.NewEncoder(w).Encode(map[string]string{
                "status": "unhealthy",
                "error": "database unreachable",
            })
            return
        }
        
        stats := db.Stats()
        json.NewEncoder(w).Encode(map[string]interface{}{
            "status":           "healthy",
            "open_connections": stats.OpenConnections,
            "in_use":          stats.InUse,
            "idle":            stats.Idle,
        })
    }
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

## Milestone 5: Integration Testing

### Docker Test Setup

**docker-compose.test.yml:**
```yaml
services:
  postgres-test:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: dime_test
      POSTGRES_USER: test_user
      POSTGRES_PASSWORD: test_password
    ports:
      - "5433:5432"  # Different from production
```

### Running Tests

```bash
# Start test database
docker-compose -f docker-compose.test.yml up -d

# Run integration tests
go test -v ./internal/repository/...

# Stop test database
docker-compose -f docker-compose.test.yml down
```

### Test Utilities

```go
// Setup test database with migrations
func SetupTestDB(t *testing.T) *sql.DB {
    db, err := sql.Open("pgx", testConnStr)
    // ...
    runMigrations(db)
    return db
}

// Cleanup between tests
func CleanupTestDB(t *testing.T, db *sql.DB) {
    db.Exec("TRUNCATE TABLE transactions, budgets CASCADE")
}

// Transaction isolation for tests
func WithTransaction(t *testing.T, db *sql.DB, fn func(*sql.Tx)) {
    tx, _ := db.Begin()
    defer tx.Rollback() // Always rollback
    fn(tx)
}
```

### Test Coverage

**Transaction Repository Tests:**
- Create transaction
- Get by ID (found/not found)
- List by user
- Context timeout handling
- Transaction methods (CreateTx)
- Transaction rollback

**Budget Repository Tests:**
- Create budget
- Upsert behavior (ON CONFLICT)
- Get by user/category
- Add spent amount
- Transaction methods
- Transaction rollback

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
│  │ - CreateTx(ctx, tx, ...)│  │  - CreateTx(ctx, tx, ...)│  │
│  │ - GetByID(ctx, ...)    │  │  - AddSpentTx(ctx, ...)  │  │
│  │ - ListByUser(ctx, ...) │  │  - GetByUserAndCategory()│  │
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
│  │  - Transaction Support                               │  │
│  └───────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

---

## File Changes Summary

### New Files

1. **`internal/database/transaction.go`** - Transaction manager
2. **`internal/testutil/db.go`** - Test utilities
3. **`docker-compose.test.yml`** - Test database configuration
4. **`internal/repository/postgres_transaction_repo_test.go`** - Transaction tests
5. **`internal/repository/postgres_budget_repo_test.go`** - Budget tests
6. **`docs/backend-learning/phase-04-notes.md`** - This documentation

### Modified Files

1. **`cmd/api/main.go`**
   - Migrated to pgx driver
   - Added connection pool configuration
   - Added structured logging
   - Added health check endpoint
   - Added TransactionManager wiring

2. **`go.mod`** - Added pgx dependencies

3. **`internal/repository/interface.go`**
   - Added `context.Context` to all methods

4. **`internal/repository/postgres_transaction_repo.go`**
   - Added context support
   - Added transaction methods (CreateTx, GetByIDTx)

5. **`internal/repository/postgres_budget_repo.go`**
   - Added context support
   - Added transaction methods (CreateTx, AddSpentTx, GetByUserAndCategoryTx)

6. **`internal/service/transaction_service.go`**
   - Added context support
   - Added RecordTransactionAtomic method
   - Added TransactionManager dependency

7. **`internal/service/budget_service.go`**
   - Added context support
   - Exported Repo field for transaction support

8. **`internal/handler/transaction_handler.go`**
   - Added context with 5-second timeout

9. **`internal/handler/budget_handler.go`**
   - Added context with 5-second timeout

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
- **Increase** if: High concurrency, low latency requirements
- **Decrease** if: Memory constraints, many app instances

### Context Timeout Guidelines

| Operation Type | Recommended Timeout |
|----------------|-------------------|
| Simple read | 1-3 seconds |
| Complex query | 5-10 seconds |
| Transaction | 10-30 seconds |
| Batch operation | 30-60 seconds |

### Query Optimization Checklist

- [ ] Use `QueryRowContext` for single-row results
- [ ] Use `QueryContext` with proper `rows.Close()`
- [ ] Use `ExecContext` for INSERT/UPDATE/DELETE
- [ ] Always pass context for cancellation support
- [ ] Use transactions for atomic operations
- [ ] Add appropriate database indexes

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
    // Ignoring ctx!
    row := r.db.QueryRow("SELECT ...", id)
}
```

**Solution:**
```go
func (r *Repo) Get(ctx context.Context, id string) (Model, error) {
    row := r.db.QueryRowContext(ctx, "SELECT ...", id)
}
```

### 3. Transaction Rollback Forgotten

**Problem:**
```go
tx, _ := db.Begin()
// Do work...
if err != nil {
    return err // Forgot to rollback!
}
tx.Commit()
```

**Solution:**
```go
err := txManager.RunInTransaction(ctx, func(tx *sql.Tx) error {
    // Do work...
    if err != nil {
        return err // Automatic rollback
    }
    return nil // Automatic commit
})
```

### 4. Pool Exhaustion

**Problem:**
```
"pq: sorry, too many clients already"
```

**Solution:**
- Monitor `WaitCount` and `WaitDuration` metrics
- Adjust `MaxOpenConns` based on load
- Check for connection leaks (unclosed rows)

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
// - OpenConnections > MaxOpenConns * 0.8
// - WaitCount is increasing rapidly
// - WaitDuration > 1 second
```

### Health Check Response

```json
{
  "status": "healthy",
  "open_connections": 12,
  "in_use": 3,
  "idle": 9
}
```

---

## Next Steps / Future Enhancements

1. **Query Logging:** Log slow queries (>1s) for optimization
2. **Metrics Export:** Expose Prometheus metrics
3. **Retry Logic:** Automatic retry for transient failures
4. **Circuit Breaker:** Fail fast when database is down
5. **Read Replicas:** Route reads to replica databases
6. **Connection Pool Tuning:** Dynamic adjustment based on load

---

## References

- [pgx Documentation](https://github.com/jackc/pgx)
- [Go database/sql Tutorial](http://go-database-sql.org/)
- [PostgreSQL Connection Pooling](https://www.postgresql.org/docs/current/runtime-config-connection.html)
- [Go Context Package](https://pkg.go.dev/context)

---

## Conclusion

Phase 4 transforms the basic database connectivity from Phase 3 into a production-ready system with:

- **Performance:** pgx driver with optimized connection pooling
- **Reliability:** Context timeouts and cancellation support
- **Consistency:** Transaction management for atomic operations
- **Observability:** Structured logging and health checks
- **Quality:** Comprehensive integration test suite

The application is now ready for production workloads with proper resource management, error handling, and monitoring capabilities.
