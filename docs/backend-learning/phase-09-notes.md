# Phase 9: Pagination & Performance

## Overview

Phase 9 adds pagination to the transactions listing endpoint and optimizes database query performance with indexes. Without pagination, listing a user's transactions would return every transaction at once, which becomes slow and unusable as users accumulate thousands of records.

## Learning Objectives

- Understand offset-based pagination (LIMIT/OFFSET)
- Learn why offset pagination breaks at scale
- Compare offset vs cursor pagination
- Add database indexes for query performance
- Document paginated APIs with OpenAPI/Swagger
- Test paginated endpoints end-to-end

---

## Why Pagination Matters

### Before (No Pagination)

```bash
GET /transactions?user_id=john
```

**Result:** Returns all 10,000 transactions at once.

**Problems:**
- Slow response time (large JSON payload)
- High memory usage on server
- High memory usage on client
- Poor user experience (long loading, app crashes)

### After (Paginated)

```bash
GET /transactions?user_id=john&page=1&page_size=20
```

**Result:** Returns only 20 transactions plus metadata.

**Benefits:**
- ✅ Fast responses
- ✅ Lower memory usage
- ✅ Better user experience
- ✅ Scales with data growth

---

## Milestone 1: Offset Pagination

Offset pagination is the simplest form of pagination. It uses SQL `LIMIT` and `OFFSET` to return a specific slice of results.

### Concept

| Page | Page Size | OFFSET | SQL |
|------|-----------|--------|-----|
| 1 | 20 | 0 | `LIMIT 20 OFFSET 0` |
| 2 | 20 | 20 | `LIMIT 20 OFFSET 20` |
| 3 | 20 | 40 | `LIMIT 20 OFFSET 40` |

**Formula:** `OFFSET = (page - 1) * page_size`

### Files Changed

#### 1. `internal/model/pagination.go` (NEW)

Centralized pagination request/response models and validation.

```go
type PaginationRequest struct {
    Page     int
    PageSize int
}

type PaginatedResponse struct {
    Data       interface{} `json:"data"`
    Page       int         `json:"page"`
    PageSize   int         `json:"page_size"`
    TotalCount int         `json:"total_count"`
    TotalPages int         `json:"total_pages"`
    HasNext    bool        `json:"has_next"`
    HasPrev    bool        `json:"has_prev"`
}
```

**Key behaviors:**
- Page defaults to 1 if invalid
- Page size defaults to 20, capped at 100
- `Offset()` method calculates SQL OFFSET
- `NewPaginatedResponse()` computes total pages and next/prev flags

#### 2. `internal/repository/interface.go`

Added the paginated method signature to `TransactionRepository`:

```go
ListByUserPaginated(
    ctx context.Context,
    userID string,
    offset, limit int,
) ([]model.Transaction, int, error)
```

Returns transactions, total count, and error.

#### 3. `internal/repository/postgres_transaction_repo.go`

Implemented `ListByUserPaginated` with two queries:

```sql
-- Count total transactions for metadata
SELECT COUNT(*) FROM transactions WHERE user_id = $1;

-- Fetch paginated slice
SELECT * FROM transactions
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;
```

#### 4. `internal/service/transaction_service.go`

Added `GetUserTransactionsPaginated()`:

```go
func (s *TransactionService) GetUserTransactionsPaginated(
    ctx context.Context,
    userID string,
    page, pageSize int,
) ([]model.Transaction, int, error) {
    offset := (page - 1) * pageSize
    return s.repo.ListByUserPaginated(ctx, userID, offset, pageSize)
}
```

#### 5. `internal/handler/transaction_handler.go`

- Added `parsePagination()` helper to read `?page=` and `?page_size=` from the URL
- Updated `ListByUser()` to return `PaginatedResponse`
- Preserved `user_id` as required query parameter

#### 6. `internal/swagger/openapi.yaml`

Documented the pagination parameters and `PaginatedResponse` schema so frontend developers and API consumers know how to paginate.

---

## Milestone 2: Problems with Offset Pagination

Offset pagination is simple but has limitations that become serious at scale.

### Performance Degradation

As the offset grows, PostgreSQL must scan and discard more rows to reach the requested page.

| Page | Offset | Approximate Time |
|------|--------|------------------|
| 1 | 0 | 1 ms |
| 100 | 1,980 | 5 ms |
| 1,000 | 19,980 | 50 ms |
| 50,000 | 999,980 | 5,000 ms |

**Why:** The database sorts all matching rows, skips the offset, then returns a small slice.

### Inconsistent Results

If a new transaction is inserted while a user is paging through results, items can shift between pages, causing duplicates or missing records.

**Example:**
- User fetches page 1: `[A, B, C, D]`
- New transaction `Z` is inserted
- User fetches page 2: now contains `[D, E, F, G]`
- Result: `D` appears twice, `Z` is never seen

### When Offset Is Fine

- Small to medium datasets (< 100,000 rows per user)
- Admin dashboards with page numbers
- Data that does not change frequently

### Better Alternative

**Cursor pagination** uses a pointer to the last seen item instead of an offset. It is faster on deep pages and consistent under concurrent inserts. It is the right choice for infinite scroll and very large datasets, but adds implementation complexity. This project intentionally kept offset pagination for simplicity.

---

## Milestone 3: Database Indexes for Performance

### Index Created

**Migration:** `migrations/003_add_transaction_indexes.up.sql`

```sql
CREATE INDEX IF NOT EXISTS idx_transactions_user_created
ON transactions(user_id, created_at DESC);
```

**Rollback:** `migrations/003_add_transaction_indexes.down.sql`

```sql
DROP INDEX IF EXISTS idx_transactions_user_created;
```

### Why This Index?

The paginated query does two things:
1. **Filter** by `user_id`
2. **Sort** by `created_at DESC`

A composite index on `(user_id, created_at DESC)` lets PostgreSQL do both in a single index scan.

### Verification

Run `EXPLAIN ANALYZE` on the paginated query:

```sql
EXPLAIN (ANALYZE, BUFFERS)
SELECT * FROM transactions
WHERE user_id = 'test-user-id'
ORDER BY created_at DESC
LIMIT 20 OFFSET 0;
```

**Expected output (uses index):**

```
Index Scan using idx_transactions_user_created on transactions
  Index Cond: (user_id = 'test-user-id'::text)
  Execution Time: 0.065 ms
```

**Without the index, expect:**

```
Seq Scan on transactions
  Filter: (user_id = 'test-user-id'::text)
  Execution Time: 5-50 ms (or worse)
```

### Performance Comparison

| Scenario | Without Index | With Index | Improvement |
|----------|---------------|------------|-------------|
| 100 rows, page 1 | ~5 ms | ~0.06 ms | ~80× |
| 100,000 rows, page 100 | ~200 ms | ~1 ms | ~200× |
| 1,000,000 rows, deep page | 5,000+ ms | ~2 ms | ~2,500× |

---

## File Structure

```
dime-api/
├── cmd/api/main.go
├── internal/
│   ├── handler/
│   │   └── transaction_handler.go      # Added parsePagination + paginated response
│   ├── model/
│   │   └── pagination.go               # NEW: Pagination models
│   ├── repository/
│   │   ├── interface.go                # Added ListByUserPaginated
│   │   ├── postgres_transaction_repo.go # Implemented pagination
│   │   ├── memory_budget_repo.go       # NEW: in-memory budget repo for tests
│   │   └── memory_transaction_repo.go  # NEW: in-memory transaction repo for tests
│   ├── service/
│   │   └── transaction_service.go      # Added GetUserTransactionsPaginated
│   └── swagger/
│       └── openapi.yaml                # Documented pagination params
├── migrations/
│   ├── 001_create_transactions_table.up.sql
│   ├── 002_create_budgets_table.up.sql
│   ├── 003_add_transaction_indexes.up.sql     # NEW
│   └── 003_add_transaction_indexes.down.sql   # NEW
└── docs/backend-learning/phase-09-notes.md    # This file
```

---

## API Reference

### List User Transactions (Paginated)

```bash
GET /transactions?user_id={user_id}&page={page}&page_size={page_size}
```

### Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `user_id` | string | Yes | - | User identifier |
| `page` | integer | No | 1 | Page number (1-based) |
| `page_size` | integer | No | 20 | Items per page (max 100) |

### Example Request

```bash
curl -UseBasicParsing "http://localhost:8080/transactions?user_id=john&page=2&page_size=10"
```

### Example Response

```json
{
  "data": [
    {
      "id": "abc123",
      "user_id": "john",
      "type": "expense",
      "amount": 45.50,
      "category": "food",
      "created_at": "2024-01-15T10:30:00Z"
    }
  ],
  "page": 2,
  "page_size": 10,
  "total_count": 3500,
  "total_pages": 350,
  "has_next": true,
  "has_prev": true
}
```

### Response Fields

| Field | Type | Description |
|-------|------|-------------|
| `data` | array | Transactions for the current page |
| `page` | integer | Current page number |
| `page_size` | integer | Number of items per page |
| `total_count` | integer | Total transactions for the user |
| `total_pages` | integer | Total pages available |
| `has_next` | boolean | True if a next page exists |
| `has_prev` | boolean | True if a previous page exists |

---

## Testing Pagination

### Unit Tests

The existing mock repositories implement `ListByUserPaginated`, so service-layer tests can verify pagination logic without a database.

### Integration Tests

Run with a real PostgreSQL database:

```bash
# Start test database (if using Docker)
docker-compose -f docker-compose.test.yml up -d

# Run tests
go test ./internal/repository/... -v
```

### Manual Testing

Start the server:

```bash
go run ./cmd/api
```

Test page 1:

```bash
curl -UseBasicParsing "http://localhost:8080/transactions?user_id=john&page=1&page_size=10"
```

Test via Swagger:

1. Open http://localhost:8080/swagger
2. Expand `GET /transactions`
3. Click **Try it out**
4. Enter `user_id`, `page`, and `page_size`
5. Click **Execute**

---

## Common Issues & Solutions

### Issue: "user_id query param is required"

**Cause:** You did not include `user_id` in the URL.

**Solution:** Add `?user_id=john` to the request.

```bash
curl -UseBasicParsing "http://localhost:8080/transactions?user_id=john&page=1"
```

### Issue: `curl` Security Warning in PowerShell

**Cause:** PowerShell's `curl` alias invokes `Invoke-WebRequest`, which warns about script execution.

**Solution:** Use the `-UseBasicParsing` flag.

```bash
curl -UseBasicParsing "http://localhost:8080/transactions?user_id=john"
```

### Issue: `EXPLAIN ANALYZE` Shows `Seq Scan`

**Cause 1:** The table is very small. PostgreSQL may choose a sequential scan because it is faster for tiny tables.

**Solution:** Insert more test rows (1,000+) and re-run `EXPLAIN ANALYZE`.

**Cause 2:** The index has not been created.

**Solution:** Run migrations by starting the server or using the migrate CLI.

### Issue: Slow Pagination on Deep Pages

**Cause:** Offset pagination is inherently slow for large offsets.

**Solution:**
- Keep users on early pages when possible
- Use larger `page_size` to reduce total pages
- For very large datasets, plan to implement cursor pagination

---

## Offset vs Cursor Pagination

| Feature | Offset (Implemented) | Cursor (Future) |
|---------|----------------------|-----------------|
| **Implementation** | Simple | More complex |
| **Performance** | Slow on deep pages | Fast everywhere |
| **Consistency** | Can duplicate/miss items | Always consistent |
| **Jump to page** | Easy (page numbers) | Hard (need cursor) |
| **Best for** | Admin UIs, small datasets | Infinite scroll, large datasets |

**Decision for this project:** Keep offset pagination for simplicity, add performance warnings, and leave cursor pagination as a future enhancement.

---

## Performance Checklist

- [ ] Created migration `003_add_transaction_indexes.up.sql`
- [ ] Ran migrations successfully
- [ ] Verified index exists with `pg_indexes`
- [ ] Verified `EXPLAIN ANALYZE` uses `Index Scan`
- [ ] Tested paginated endpoint with curl or Swagger
- [ ] Confirmed metadata is correct (total_count, total_pages, has_next, has_prev)

---

## Next Phase Preview

**Phase 10: Production Hardening**

Will cover:
- Authentication and authorization
- Input validation and sanitization
- Rate limiting
- Structured logging and observability
- Error handling improvements
- Deployment preparation

---

## Summary

Phase 9 transforms the transaction listing endpoint from a scalability liability into a paginated, performant API:

✅ **Offset pagination** implemented for `GET /transactions`  
✅ **Validation** for page and page_size parameters  
✅ **Paginated response metadata** for frontend navigation  
✅ **Database index** added for fast filter + sort  
✅ **OpenAPI/Swagger** updated with pagination docs  
✅ **In-memory test repositories** created to fix test compilation  

The API now scales with user data and provides the foundation for future cursor-based pagination if needed.

**Phase 9 Complete!** 🎉

---

*Part of: Backend Learning Project - Phase 9*  
*Prerequisite: Phase 8 (Configuration & Environment)*  
*Next: Phase 10 (Production Hardening)*
