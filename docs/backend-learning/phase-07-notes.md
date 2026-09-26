# Phase 7: Integration Testing

## Overview

Phase 7 implements integration tests for our PostgreSQL repositories. Unlike unit tests that mock dependencies, integration tests run against a real PostgreSQL database in Docker, verifying that our SQL queries work correctly with the actual database.

## Learning Objectives

- Write integration tests with real PostgreSQL
- Set up Docker test environment
- Implement test utilities for database setup/cleanup
- Test context cancellation and timeout behavior
- Test transaction rollback behavior
- Understand test isolation strategies

---

## Why Integration Testing?

### Unit Tests vs Integration Tests

| Aspect | Unit Tests | Integration Tests |
|--------|-----------|-------------------|
| **Dependencies** | Mocked | Real (PostgreSQL) |
| **Speed** | Fast (< 100ms) | Slower (100ms - 2s) |
| **Coverage** | Code logic | Database interactions |
| **Reliability** | High | Higher (catches real issues) |
| **CI/CD** | Always run | Conditional or scheduled |

### What Integration Tests Catch

1. **SQL Syntax Errors**: Invalid queries that compile but fail at runtime
2. **Type Mismatches**: Go types that don't match PostgreSQL types
3. **Constraint Violations**: Missed unique constraints or check constraints
4. **Race Conditions**: Concurrent access issues
5. **Context Cancellation**: Timeout and cancellation handling
6. **Transaction Behavior**: Rollback and isolation issues

---

## Milestone 1: Docker Test Environment

### File: `docker-compose.test.yml`

```yaml
version: '3.8'

services:
  postgres-test:
    image: postgres:15-alpine
    container_name: dime-postgres-test
    environment:
      POSTGRES_DB: dime_test
      POSTGRES_USER: test_user
      POSTGRES_PASSWORD: test_password
    ports:
      - "5433:5432"  # Different port to avoid conflicts with main database
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U test_user -d dime_test"]
      interval: 5s
      timeout: 5s
      retries: 5
    volumes:
      - postgres_test_data:/var/lib/postgresql/data

volumes:
  postgres_test_data:
    driver: local
```

**Key Design Decisions:**
- **Separate database**: `dime_test` not `dime` (isolation from dev data)
- **Different port**: `5433` not `5432` (run dev and tests simultaneously)
- **Health check**: Ensures database is ready before tests run
- **Named volume**: Persists data between test runs (faster than recreating)

### Running Tests

```bash
# Start test database
docker-compose -f docker-compose.test.yml up -d

# Run integration tests
go test -v ./internal/repository/...

# Run with short flag (skip integration tests)
go test -short ./...

# Stop test database
docker-compose -f docker-compose.test.yml down
```

---

## Milestone 2: Test Utilities

### File: `internal/testutil/db.go`

Test utilities provide reusable functions for database setup and cleanup.

#### Configuration

```go
// TestDBConfig holds configuration for test database
var TestDBConfig = struct {
    Host     string
    Port     string
    User     string
    Password string
    DBName   string
}{
    Host:     getEnvOrDefault("TEST_DB_HOST", "localhost"),
    Port:     getEnvOrDefault("TEST_DB_PORT", "5433"),
    User:     getEnvOrDefault("TEST_DB_USER", "test_user"),
    Password: getEnvOrDefault("TEST_DB_PASSWORD", "test_password"),
    DBName:   getEnvOrDefault("TEST_DB_NAME", "dime_test"),
}
```

**Why configurable?**
- CI/CD can override with environment variables
- Developers can use local PostgreSQL if preferred
- Port flexibility (5433 default, but configurable)

#### Database Setup

```go
// SetupTestDB creates a test database connection with migrations applied
func SetupTestDB(t *testing.T) *sql.DB {
    t.Helper()  // Marks this as a test helper (better error reporting)

    // Build connection string
    connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
        TestDBConfig.Host,
        TestDBConfig.Port,
        TestDBConfig.User,
        TestDBConfig.Password,
        TestDBConfig.DBName,
    )

    // Open connection
    db, err := sql.Open("pgx", connStr)
    if err != nil {
        t.Fatalf("Failed to open test database: %v", err)
    }

    // Verify connection with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    if err := db.PingContext(ctx); err != nil {
        t.Fatalf("Failed to ping test database: %v", err)
    }

    // Configure connection pool for tests (smaller than production)
    db.SetMaxOpenConns(5)
    db.SetMaxIdleConns(2)
    db.SetConnMaxLifetime(5 * time.Minute)

    // Run migrations
    if err := runMigrations(db); err != nil {
        t.Fatalf("Failed to run migrations: %v", err)
    }

    log.Println("Test database setup complete")
    return db
}
```

**Key Points:**
- `t.Helper()`: Marks function as helper (errors report caller's line)
- Ping with timeout: Ensures database is actually accessible
- Smaller pool: Tests don't need production pool sizes
- Runs migrations: Ensures schema is up to date

#### Migration Runner

```go
// runMigrations executes all pending migrations
func runMigrations(db *sql.DB) error {
    driver, err := postgres.WithInstance(db, &postgres.Config{})
    if err != nil {
        return fmt.Errorf("failed to create migration driver: %w", err)
    }

    m, err := migrate.NewWithDatabaseInstance(
        "file://../../migrations", // Relative path from test file
        "postgres",
        driver,
    )
    if err != nil {
        // Try alternative path (for different test locations)
        m, err = migrate.NewWithDatabaseInstance(
            "file://./migrations",
            "postgres",
            driver,
        )
        if err != nil {
            return fmt.Errorf("failed to create migrate instance: %w", err)
        }
    }

    if err := m.Up(); err != nil && err != migrate.ErrNoChange {
        return fmt.Errorf("failed to run migrations: %w", err)
    }

    return nil
}
```

**Why migrations in tests?**
- Ensures schema matches code expectations
- Tests run against actual database schema
- Catches migration/schema mismatches

#### Cleanup Strategy

```go
// CleanupTestDB truncates all tables to ensure test isolation
func CleanupTestDB(t *testing.T, db *sql.DB) {
    t.Helper()

    tables := []string{"transactions", "budgets"}

    for _, table := range tables {
        _, err := db.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
        if err != nil {
            t.Logf("Warning: failed to truncate table %s: %v", table, err)
        }
    }
}

// TeardownTestDB closes the database connection
func TeardownTestDB(t *testing.T, db *sql.DB) {
    t.Helper()

    if err := db.Close(); err != nil {
        t.Logf("Warning: failed to close test database: %v", err)
    }
}
```

**TRUNCATE vs DELETE:**
- `TRUNCATE`: Fast, resets sequences, cannot rollback
- `DELETE`: Slower, preserves sequences, can rollback
- We use `TRUNCATE` for speed and clean state

#### Transaction Isolation Helper

```go
// WithTransaction runs a test function within a transaction that gets rolled back
// This provides automatic test isolation without truncating tables
func WithTransaction(t *testing.T, db *sql.DB, fn func(*sql.Tx)) {
    t.Helper()

    ctx := context.Background()
    tx, err := db.BeginTx(ctx, nil)
    if err != nil {
        t.Fatalf("Failed to begin transaction: %v", err)
    }

    // Ensure rollback happens even if test panics
    defer func() {
        if r := recover(); r != nil {
            tx.Rollback()
            panic(r)  // Re-panic after rollback
        }
    }()

    // Run the test function
    fn(tx)

    // Always rollback to maintain test isolation
    if err := tx.Rollback(); err != nil {
        t.Logf("Warning: failed to rollback transaction: %v", err)
    }
}
```

**Alternative to Truncation:**
- Wrap test in transaction
- Always rollback after test
- Fast and fully isolated
- Best for unit-test-like speed with real database

---

## Milestone 3: Repository Integration Tests

### File: `internal/repository/postgres_transaction_repo_test.go`

#### Test Structure

```go
func TestPostgresTransactionRepo_Integration(t *testing.T) {
    // Skip if not running integration tests
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    // Setup
    db := testutil.SetupTestDB(t)
    defer testutil.TeardownTestDB(t, db)
    
    // Cleanup before and after tests
    testutil.CleanupTestDB(t, db)
    defer testutil.CleanupTestDB(t, db)

    repo := NewPostgresTransactionRepo(db)
    ctx := context.Background()

    // Subtests...
}
```

**Pattern: Setup → Test → Cleanup**
- Setup database connection
- Clean state before tests
- Run subtests
- Clean state after tests (defer)
- Close connection (defer)

#### Test: Create

```go
t.Run("Create", func(t *testing.T) {
    tx := model.Transaction{
        ID:        "test-tx-1",
        UserID:    "user-1",
        Type:      model.TypeExpense,
        Amount:    50.00,
        Category:  "food",
        CreatedAt: time.Now(),
    }

    created, err := repo.Create(ctx, tx)
    if err != nil {
        t.Fatalf("Create failed: %v", err)
    }

    // Verify all fields
    if created.ID != tx.ID {
        t.Errorf("Expected ID %s, got %s", tx.ID, created.ID)
    }
    if created.UserID != tx.UserID {
        t.Errorf("Expected UserID %s, got %s", tx.UserID, created.UserID)
    }
    if created.Amount != tx.Amount {
        t.Errorf("Expected Amount %f, got %f", tx.Amount, created.Amount)
    }
})
```

**What it verifies:**
- Insert works correctly
- All fields are persisted
- No data loss or type conversion issues

#### Test: GetByID (Found)

```go
t.Run("GetByID", func(t *testing.T) {
    // Create first
    tx := model.Transaction{
        ID:        "test-tx-2",
        UserID:    "user-2",
        Type:      model.TypeIncome,
        Amount:    100.00,
        Category:  "salary",
        CreatedAt: time.Now(),
    }

    _, err := repo.Create(ctx, tx)
    if err != nil {
        t.Fatalf("Create failed: %v", err)
    }

    // Retrieve
    retrieved, err := repo.GetByID(ctx, tx.ID)
    if err != nil {
        t.Fatalf("GetByID failed: %v", err)
    }

    // Verify
    if retrieved.ID != tx.ID {
        t.Errorf("Expected ID %s, got %s", tx.ID, retrieved.ID)
    }
    if retrieved.Amount != tx.Amount {
        t.Errorf("Expected Amount %f, got %f", tx.Amount, retrieved.Amount)
    }
})
```

#### Test: GetByID (Not Found)

```go
t.Run("GetByID_NotFound", func(t *testing.T) {
    _, err := repo.GetByID(ctx, "non-existent-id")
    if err != ErrNotFound {
        t.Errorf("Expected ErrNotFound, got %v", err)
    }
})
```

**Critical test:** Verifies correct error type for "not found" scenarios.

#### Test: ListByUser with Ordering

```go
t.Run("ListByUser", func(t *testing.T) {
    userID := "user-3"

    // Create multiple transactions
    transactions := []model.Transaction{
        {
            ID:        "test-tx-3a",
            UserID:    userID,
            Type:      model.TypeExpense,
            Amount:    25.00,
            Category:  "food",
            CreatedAt: time.Now().Add(-1 * time.Hour),  // Older
        },
        {
            ID:        "test-tx-3b",
            UserID:    userID,
            Type:      model.TypeExpense,
            Amount:    30.00,
            Category:  "transport",
            CreatedAt: time.Now(),  // Newer
        },
    }

    for _, tx := range transactions {
        _, err := repo.Create(ctx, tx)
        if err != nil {
            t.Fatalf("Create failed: %v", err)
        }
    }

    // List
    list, err := repo.ListByUser(ctx, userID)
    if err != nil {
        t.Fatalf("ListByUser failed: %v", err)
    }

    if len(list) != 2 {
        t.Errorf("Expected 2 transactions, got %d", len(list))
    }

    // Verify ORDER BY created_at DESC
    if list[0].ID != "test-tx-3b" {
        t.Errorf("Expected first transaction to be test-tx-3b (newest), got %s", list[0].ID)
    }
})
```

**What it verifies:**
- Multiple records returned
- WHERE clause works correctly
- ORDER BY works (newest first)

#### Test: Transaction Methods (CreateTx)

```go
t.Run("CreateTx", func(t *testing.T) {
    tx := model.Transaction{
        ID:        "test-tx-4",
        UserID:    "user-4",
        Type:      model.TypeExpense,
        Amount:    75.00,
        Category:  "entertainment",
        CreatedAt: time.Now(),
    }

    // Start a transaction
    dbTx, err := db.Begin()
    if err != nil {
        t.Fatalf("Failed to begin transaction: %v", err)
    }

    // Use concrete type to access Tx methods
    concreteRepo := repo.(*PostgresTransactionRepo)
    created, err := concreteRepo.CreateTx(ctx, dbTx, tx)
    if err != nil {
        dbTx.Rollback()
        t.Fatalf("CreateTx failed: %v", err)
    }

    // Commit
    if err := dbTx.Commit(); err != nil {
        t.Fatalf("Failed to commit transaction: %v", err)
    }

    // Verify creation
    if created.ID != tx.ID {
        t.Errorf("Expected ID %s, got %s", tx.ID, created.ID)
    }

    // Verify it was actually persisted
    retrieved, err := repo.GetByID(ctx, tx.ID)
    if err != nil {
        t.Fatalf("GetByID failed: %v", err)
    }
    if retrieved.ID != tx.ID {
        t.Errorf("Expected to retrieve created transaction, got %s", retrieved.ID)
    }
})
```

**Why use concrete type?**
- Interface doesn't expose `CreateTx` (only for internal use)
- Type assertion `repo.(*PostgresTransactionRepo)` accesses additional methods

#### Test: Transaction Rollback

```go
t.Run("CreateTx_Rollback", func(t *testing.T) {
    tx := model.Transaction{
        ID:        "test-tx-5",
        UserID:    "user-5",
        Type:      model.TypeExpense,
        Amount:    99.00,
        Category:  "test",
        CreatedAt: time.Now(),
    }

    // Start transaction
    dbTx, err := db.Begin()
    if err != nil {
        t.Fatalf("Failed to begin transaction: %v", err)
    }

    // Create within transaction
    concreteRepo := repo.(*PostgresTransactionRepo)
    _, err = concreteRepo.CreateTx(ctx, dbTx, tx)
    if err != nil {
        dbTx.Rollback()
        t.Fatalf("CreateTx failed: %v", err)
    }

    // Rollback instead of commit
    if err := dbTx.Rollback(); err != nil {
        t.Fatalf("Failed to rollback transaction: %v", err)
    }

    // Verify the transaction was NOT created
    _, err = repo.GetByID(ctx, tx.ID)
    if err != ErrNotFound {
        t.Errorf("Expected transaction to not exist after rollback, got error: %v", err)
    }
})
```

**Critical test:** Verifies rollback actually works (data not persisted).

#### Test: Context Timeout

```go
func TestPostgresTransactionRepo_ContextTimeout(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    db := testutil.SetupTestDB(t)
    defer testutil.TeardownTestDB(t, db)
    testutil.CleanupTestDB(t, db)

    repo := NewPostgresTransactionRepo(db)

    t.Run("ContextTimeout", func(t *testing.T) {
        // Create a context with very short timeout
        ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
        defer cancel()

        // Wait for context to expire
        time.Sleep(10 * time.Millisecond)

        tx := model.Transaction{
            ID:        "test-timeout",
            UserID:    "user-timeout",
            Type:      model.TypeExpense,
            Amount:    10.00,
            Category:  "test",
            CreatedAt: time.Now(),
        }

        _, err := repo.Create(ctx, tx)
        if err == nil {
            t.Error("Expected error due to context timeout, got nil")
        }
    })
}
```

**What it verifies:**
- Context cancellation propagates to database
- Operations fail fast when context expires
- No hanging operations

### File: `internal/repository/postgres_budget_repo_test.go`

Similar structure with budget-specific tests:

- **Create**: Basic insert
- **Create_Upsert**: Update limit on conflict
- **GetByUserAndCategory**: Lookup by composite key
- **GetByUserAndCategory_NotFound**: Error handling
- **AddSpent**: Atomic increment
- **AddSpent_NotFound**: Error when budget missing
- **CreateTx**: Works within transaction
- **AddSpentTx**: Works within transaction
- **Transaction_Rollback**: Changes not persisted

---

## Test Isolation Strategies

### Strategy 1: Truncate Tables (Used)

```go
testutil.CleanupTestDB(t, db)  // TRUNCATE TABLE transactions, budgets
```

**Pros:**
- Fast (TRUNCATE is O(1))
- Clean slate for each test
- Works with auto-increment/UUID

**Cons:**
- Destroys all data (can't inspect after failure)
- Must run sequentially (no parallel tests)

### Strategy 2: Transaction Rollback

```go
testutil.WithTransaction(t, db, func(tx *sql.Tx) {
    repo := NewPostgresTransactionRepo(db)
    repo.CreateTx(ctx, tx, transaction)
    // Automatically rolled back
})
```

**Pros:**
- Fast (no disk writes committed)
- Perfect isolation
- Can run tests in parallel

**Cons:**
- Requires Tx versions of all methods
- Can't test commit behavior

### Strategy 3: Unique Test Data

```go
userID := fmt.Sprintf("test-user-%d", time.Now().UnixNano())
```

**Pros:**
- Tests can run in parallel
- No cleanup needed

**Cons:**
- Database grows over time
- Harder to debug

---

## Running Tests

### Local Development

```bash
# Start test database
docker-compose -f docker-compose.test.yml up -d

# Run all tests (including integration)
go test -v ./internal/repository/...

# Run only unit tests (skip integration)
go test -short ./...

# Run specific test
go test -v -run TestPostgresTransactionRepo_Integration ./internal/repository/

# Stop test database
docker-compose -f docker-compose.test.yml down
```

### Makefile (Recommended)

```makefile
.PHONY: test test-integration test-unit test-db-up test-db-down

test-db-up:
	docker-compose -f docker-compose.test.yml up -d

test-db-down:
	docker-compose -f docker-compose.test.yml down

test-unit:
	go test -short ./...

test-integration: test-db-up
	go test -v ./internal/repository/...

test: test-unit test-integration
```

### CI/CD (GitHub Actions Example)

```yaml
name: Tests
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    
    services:
      postgres:
        image: postgres:15-alpine
        env:
          POSTGRES_DB: dime_test
          POSTGRES_USER: test_user
          POSTGRES_PASSWORD: test_password
        ports:
          - 5433:5432
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5

    steps:
    - uses: actions/checkout@v3
    
    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.21'
    
    - name: Run tests
      env:
        TEST_DB_HOST: localhost
        TEST_DB_PORT: 5433
      run: go test -v ./internal/repository/...
```

---

## Key Concepts Learned

### 1. Test Isolation

Each test must start with clean state:
```go
testutil.CleanupTestDB(t, db)
defer testutil.CleanupTestDB(t, db)
```

Prevents tests from affecting each other.

### 2. Subtests

```go
t.Run("Create", func(t *testing.T) { ... })
t.Run("GetByID", func(t *testing.T) { ... })
```

- Organizes related tests
- Can run individual subtest: `-run TestName/SubtestName`
- Shared setup/teardown

### 3. Short Flag

```go
if testing.Short() {
    t.Skip("Skipping integration test")
}
```

- `go test -short`: Fast unit tests only
- `go test`: Full test suite
- Useful for CI/CD speed vs thoroughness

### 4. Helper Methods

```go
t.Helper()  // Marks function as test helper
```

Makes error messages show caller's line number, not helper's line.

### 5. Parallel Tests

```go
t.Parallel()  // Marks test as parallelizable
```

Run with `go test -parallel 4` for concurrent execution (requires careful isolation).

---

## Test Coverage Summary

| Component | Tests | Coverage |
|-----------|-------|----------|
| **Create** | Create, CreateTx | Insert operations, RETURNING clause |
| **Read** | GetByID, GetByID_NotFound | Single row lookup, error handling |
| **List** | ListByUser | Multi-row queries, ORDER BY |
| **Update** | AddSpent | Atomic increment, partial updates |
| **Transactions** | CreateTx, AddSpentTx, Rollback | Atomic operations, rollback |
| **Context** | ContextTimeout | Cancellation propagation |
| **Error Handling** | NotFound variants | Error types, error messages |

---

## Next Phase

Phase 8 focuses on configuration management and environment setup for production readiness.

---

*Part of: Backend Learning Project - Phase 7*
*Prerequisite: Phases 5-6 (Repository Implementations)*
*Next: Phase 8 (Configuration & Environment)*
