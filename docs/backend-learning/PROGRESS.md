# Backend Learning Progress

Persistent handoff document for PostgreSQL + Go backend learning project.

---

## Overall Roadmap

### Phase 1: Foundation - PostgreSQL Setup
- **Status**: COMPLETED
- **Goal**: Install and run PostgreSQL locally, understand basic concepts

### Phase 2: Schema Design & Manual SQL
- **Status**: NOT STARTED
- **Goal**: Design relational schema, learn SQL fundamentals

### Phase 3: Migrations
- **Status**: NOT STARTED
- **Goal**: Learn schema versioning with golang-migrate

### Phase 4: Go Database Integration
- **Status**: NOT STARTED
- **Goal**: Connect Go to PostgreSQL using database/sql and pgx

### Phase 5: Repository Implementation - Transactions
- **Status**: NOT STARTED
- **Goal**: Implement PostgreSQL-backed transaction repository

### Phase 6: Repository Implementation - Budgets
- **Status**: NOT STARTED
- **Goal**: Implement PostgreSQL-backed budget repository with transactions

### Phase 7: Integration Testing
- **Status**: NOT STARTED
- **Goal**: Test repositories against real PostgreSQL

### Phase 8: Configuration & Environment
- **Status**: NOT STARTED
- **Goal**: Production configuration patterns

### Phase 9: Pagination & Performance
- **Status**: NOT STARTED
- **Goal**: Handle large datasets, query optimization

### Phase 10: Production Hardening
- **Status**: NOT STARTED
- **Goal**: Logging, metrics, Docker, graceful shutdown

---

## Current Phase

**Phase 2: Schema Design & Manual SQL**

---

## Completed Phases

### Phase 1: Foundation - PostgreSQL Setup
- Docker Compose configured with PostgreSQL 16
- Local database running on localhost:5432
- psql client installed and configured
- Practice table created and CRUD operations tested
- All fundamental concepts understood

---

## Current Phase Details

### Phase 1: Foundation - PostgreSQL Setup

#### Objective
Get PostgreSQL running locally using Docker and understand basic database concepts before writing any Go code.

#### Tasks
- [x] Install Docker Desktop
- [x] Create docker-compose.yml for PostgreSQL
- [x] Start PostgreSQL container
- [x] Connect using psql or GUI tool
- [x] Create a test table manually
- [x] Practice basic SQL commands (INSERT, SELECT, UPDATE, DELETE)

#### Current Progress
100% - Phase 1 Complete

#### Problems Encountered
_None yet_

#### Decisions Made
_None yet_

#### Concepts Learned
- PostgreSQL architecture: Server → Database → Schema → Table → Row/Column
- Docker containerization for local development
- Connecting to PostgreSQL with psql
- Basic SQL CRUD operations:
  - CREATE TABLE with data types and constraints
  - INSERT to add data
  - SELECT to query data with WHERE and ORDER BY
  - UPDATE to modify data (always use WHERE!)
  - DELETE to remove data (always use WHERE!)
- Data types: SERIAL, INTEGER, TEXT, BOOLEAN, TIMESTAMP
- Constraints: PRIMARY KEY, NOT NULL, DEFAULT
- psql meta-commands: \l, \dt, \d, \dn, \du, \conninfo

#### Concepts Still to Understand
- Schema design for production tables
- Relationships between tables
- Indexes and performance
- Advanced SQL queries

---

## Next Phase

**Phase 2: Schema Design & Manual SQL**

Design the actual database schema for dime-api:
- transactions table
- budgets table
- Indexes and constraints
- Relationships between tables

---

## Important Architectural Decisions

### AD-001: PostgreSQL Driver Choice
**Decision**: Use `pgx` (github.com/jackc/pgx/v5) instead of `lib/pq`

**Rationale**:
- Better performance (native Go, no cgo)
- More features (PostgreSQL-specific types)
- Built-in connection pooling
- lib/pq is in maintenance mode

**Status**: Pending implementation

### AD-002: Migration Tool Choice
**Decision**: Use `golang-migrate` for schema migrations

**Rationale**:
- Industry standard in Go ecosystem
- Supports up/down migrations
- Version tracking in database
- Multiple source support (files, S3, etc.)

**Status**: Pending implementation

### AD-003: No ORM for Initial Learning
**Decision**: Use raw SQL with database/sql instead of an ORM

**Rationale**:
- Learn SQL deeply first
- Understand what ORMs abstract away
- Essential for performance tuning knowledge
- Raw SQL is idiomatic in Go

**Status**: Active - may revisit after Phase 9

### AD-004: Context Propagation Required
**Decision**: All repository and service methods will accept context.Context

**Rationale**:
- Enables request cancellation
- Supports timeouts
- Required for tracing
- Standard Go pattern

**Status**: Pending implementation (Milestone 4)

---

## Important Lessons

_None yet - to be populated as we progress_

---

## Session Handoff

### For Next OpenCode Session

**Current State**: Phase 1 Complete - PostgreSQL Setup

**Immediate Next Step**: 
Begin Phase 2: Design the actual database schema for dime-api (transactions and budgets tables).

**Phase 1 Summary**:
- Docker Compose configured with PostgreSQL 16
- Local database running on localhost:5432
- psql client installed and working
- Practice table created and CRUD operations tested
- Ready to design production schema

**Context for New Session**:
- This is a Go backend learning project (dime-api) for tracking transactions and budgets
- Current implementation uses in-memory storage
- Goal is to migrate to PostgreSQL while learning backend engineering deeply
- Architecture: Handler → Service → Repository → PostgreSQL
- Zero external dependencies currently (Go 1.22+ standard library only)

**Key Files**:
- `cmd/api/main.go` - Application entry point
- `internal/model/` - Domain models (Transaction, Budget)
- `internal/repository/` - Repository interfaces + in-memory implementations
- `internal/service/` - Business logic
- `internal/handler/` - HTTP handlers
- `docs/backend-learning/PROGRESS.md` - This file

**Master Plan Location**: Original plan is documented in conversation history.
Brief summary: 10-phase roadmap from PostgreSQL basics to production readiness.

**Current Blockers**: None

**Environment Requirements**:
- Docker Desktop installed and running
- Go 1.22+ installed
- PostgreSQL client (psql) available

---

## Quick Reference

### Project Structure
```
cmd/api/main.go                    # Entry point
internal/
  model/                           # Domain structs
    transaction.go
    budget.go
  repository/                      # Data access interfaces
    transaction_repo.go
    budget_repo.go
  service/                         # Business logic
    transaction_service.go
    budget_service.go
  handler/                         # HTTP layer
    transaction_handler.go
    budget_handler.go
docs/backend-learning/             # Learning documentation
  PROGRESS.md                      # This file
```

### Current Architecture Flow
```
HTTP Request
  → Handler (parsing, JSON)
    → Service (business logic)
      → Repository (interface)
        → InMemoryRepo (current)
        → PostgreSQL (future)
```

### Repositories (Current - In-Memory)
- `TransactionRepository`: Create, GetByID, ListByUser
- `BudgetRepository`: Create, GetByUserAndCategory, AddSpent

### Endpoints
- `POST /transactions` - Create transaction
- `GET /transactions/{id}` - Get transaction by ID
- `GET /transactions?user_id=X` - List user's transactions
- `POST /budgets` - Create budget
- `GET /budgets/status?user_id=X&category=Y` - Check budget status

---

*Last Updated*: Session start
*Next Session Should Begin*: Phase 1, Milestone 0
