# Integration Testing Guide

## Prerequisites

1. **Docker and Docker Compose** must be installed
2. **Go 1.21+** must be installed

## Running Integration Tests

### Step 1: Start Test Database

```bash
# Start the test PostgreSQL container
docker-compose -f docker-compose.test.yml up -d

# Wait for database to be ready (optional - tests will retry)
docker-compose -f docker-compose.test.yml ps
```

### Step 2: Run Integration Tests

```bash
# Run all integration tests (without -short flag)
go test -v ./internal/repository/...

# Run only integration tests (skip unit tests)
go test -v -run Integration ./internal/repository/...

# Run with race detection
go test -race -v ./internal/repository/...
```

### Step 3: Stop Test Database

```bash
# Stop and remove the test container
docker-compose -f docker-compose.test.yml down

# Stop and remove volumes (clean slate)
docker-compose -f docker-compose.test.yml down -v
```

## Test Configuration

Tests use the following default configuration (can be overridden with environment variables):

| Variable | Default | Description |
|----------|---------|-------------|
| `TEST_DB_HOST` | `localhost` | Database host |
| `TEST_DB_PORT` | `5433` | Database port (different from main app) |
| `TEST_DB_USER` | `test_user` | Database user |
| `TEST_DB_PASSWORD` | `test_password` | Database password |
| `TEST_DB_NAME` | `dime_test` | Database name |

## Test Isolation Strategy

Tests use **table truncation** for isolation:

1. Before each test: Tables are truncated
2. After each test: Tables are truncated again
3. No transaction rollback (simpler but slightly slower)

Alternative approaches considered:
- **Transaction rollback**: Faster, but doesn't test commit behavior
- **Database per test**: Too slow for most cases

## What's Tested

### Transaction Repository Tests
- ✅ Create transaction
- ✅ Get transaction by ID
- ✅ Get transaction (not found)
- ✅ List transactions by user
- ✅ Create within transaction (Tx methods)
- ✅ Transaction rollback
- ✅ Context timeout handling

### Budget Repository Tests
- ✅ Create budget
- ✅ Create budget with upsert (ON CONFLICT)
- ✅ Get budget by user and category
- ✅ Get budget (not found)
- ✅ Add spent amount
- ✅ Add spent (not found)
- ✅ Transaction methods (CreateTx, AddSpentTx)
- ✅ Transaction rollback

## Quick Test Run

```bash
# One-liner to run all integration tests
docker-compose -f docker-compose.test.yml up -d && \
  sleep 5 && \
  go test -v ./internal/repository/... && \
  docker-compose -f docker-compose.test.yml down
```

## Troubleshooting

### "connection refused" errors
- Ensure Docker container is running: `docker-compose -f docker-compose.test.yml ps`
- Wait a few seconds for PostgreSQL to start
- Check logs: `docker-compose -f docker-compose.test.yml logs`

### "migrations" not found
- Ensure you're running from project root
- Check that migrations folder exists: `ls migrations/`

### Port already in use
- Change `TEST_DB_PORT` or stop conflicting service
- Default test port is 5433 to avoid conflict with main PostgreSQL (5432)
