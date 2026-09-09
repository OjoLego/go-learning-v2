# Backend Learning Progress

Persistent handoff document for PostgreSQL + Go backend learning project.

---

## Overall Roadmap

### Phase 1: Foundation - PostgreSQL Setup
- **Status**: COMPLETED
- **Goal**: Install and run PostgreSQL locally, understand basic concepts

### Phase 2: Schema Design & Manual SQL
- **Status**: COMPLETED
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

**Phase 3: Database Migrations**

---

## Completed Phases

### Phase 1: Foundation - PostgreSQL Setup
- Docker Compose configured with PostgreSQL 16
- Local database running on localhost:5432
- psql client installed and configured
- Practice table created and CRUD operations tested
- All fundamental concepts understood

### Phase 2: Schema Design & Manual SQL
- Designed database schema for transactions and budgets tables
- Selected appropriate data types (UUID, DECIMAL, TIMESTAMPTZ)
- Created CHECK constraints for data integrity (type validation, positive amounts)
- Implemented UNIQUE constraint on (user_id, category) for budgets
- Created 4 indexes for query performance
- Executed SQL files in PostgreSQL using psql
- Inserted test data and validated constraints work correctly
- Wrote and tested repository queries (Create, GetByID, ListByUser, AddSpent)
- Created comprehensive notes documenting all learnings
- Schema ready for Go integration

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
- All concepts from Phase 2 now understood!

---

### Phase 2: Schema Design & Manual SQL

#### Objective
Design the actual database schema for dime-api and practice manual SQL execution before implementing migrations.

#### Tasks
- [x] Review existing domain models (Transaction, Budget)
- [x] Review repository interfaces to understand query patterns
- [x] Learn PostgreSQL data types (UUID, DECIMAL, TIMESTAMPTZ, VARCHAR with CHECK)
- [x] Learn about constraints (PRIMARY KEY, NOT NULL, UNIQUE, CHECK, DEFAULT)
- [x] Create transactions table with appropriate columns and constraints
- [x] Create budgets table with UNIQUE constraint on (user_id, category)
- [x] Learn about indexes and when to create them
- [x] Create 4 indexes for query performance
- [x] Execute SQL files in PostgreSQL using psql
- [x] Insert test data and validate constraints
- [x] Test constraint violations (ensure invalid data is rejected)
- [x] Write SQL queries matching repository interfaces
- [x] Test all repository queries (Create, GetByID, ListByUser, AddSpent)
- [x] Document schema design in phase-02-notes.md

#### Current Progress
100% - Phase 2 Complete

#### Problems Encountered
_None_

#### Decisions Made
- **Data Types**: UUID for IDs, DECIMAL(10,2) for money, TIMESTAMPTZ for timestamps
- **Constraints**: CHECK for type validation (income/expense), CHECK for positive amounts, UNIQUE for budget uniqueness
- **Reserved Keywords**: Avoided 'limit' by using 'limit_amount'
- **Index Strategy**: Index all columns used in WHERE and ORDER BY clauses

#### Concepts Learned
- PostgreSQL data types and Go type mappings
- DECIMAL vs FLOAT for money (rounding errors with FLOAT)
- UUID generation with gen_random_uuid()
- CHECK constraints for data validation
- UNIQUE constraints for business rules
- DEFAULT values for auto-generated data
- Index creation for performance optimization
- Composite indexes (user_id, category)
- RETURNING clause for INSERT/UPDATE
- Parameterized queries ($1, $2, etc.)

#### Concepts Still to Understand
- Database migrations and version control
- Go database integration with pgx
- Transaction management in Go
- Connection pooling

---

## Next Phase

**Phase 3: Database Migrations**

Learn schema versioning with golang-migrate:
- Install golang-migrate CLI
- Create migration files (up/down)
- Apply migrations programmatically
- Version tracking in database

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

**Current State**: Phase 2 Complete - Schema Design & Manual SQL

**Immediate Next Step**: 
Begin Phase 3: Database Migrations with golang-migrate.

**Phase 2 Summary**:
- Database schema designed for transactions and budgets tables
- Tables created with appropriate data types (UUID, DECIMAL, TIMESTAMPTZ)
- CHECK constraints implemented for data integrity (type validation, positive amounts)
- UNIQUE constraint on (user_id, category) for budgets
- 4 indexes created for query performance
- Test data inserted and validated
- Repository queries written and tested
- Schema design documented in phase-02-notes.md
- SQL files created in migrations/ folder

**Context for New Session**:
- This is a Go backend learning project (dime-api) for tracking transactions and budgets
- Current implementation uses in-memory storage
- Goal is to migrate to PostgreSQL while learning backend engineering deeply
- Architecture: Handler → Service → Repository → PostgreSQL
- Database schema is complete and tested
- Ready to implement migrations (Phase 3) then Go integration (Phase 4)

**Key Files**:
- `cmd/api/main.go` - Application entry point
- `internal/model/` - Domain models (Transaction, Budget)
- `internal/repository/` - Repository interfaces + in-memory implementations
- `internal/service/` - Business logic
- `internal/handler/` - HTTP handlers
- `docs/backend-learning/PROGRESS.md` - This file
- `docs/backend-learning/phase-02-notes.md` - Phase 2 learnings and documentation
- `migrations/01_create_transactions.sql` - Transactions table schema
- `migrations/02_create_budgets.sql` - Budgets table schema

**Database Schema**:
- **transactions**: id (UUID), user_id (TEXT), type (VARCHAR), amount (DECIMAL), category (TEXT), created_at (TIMESTAMPTZ)
- **budgets**: id (UUID), user_id (TEXT), category (TEXT), limit_amount (DECIMAL), spent (DECIMAL), updated_at (TIMESTAMPTZ)
- Constraints: CHECK (amount > 0), CHECK (type IN ('income', 'expense')), UNIQUE(user_id, category)
- Indexes: idx_transactions_user_id, idx_transactions_user_category, idx_transactions_created_at, idx_budgets_user_category

**Master Plan Location**: Original plan is documented in conversation history.
Brief summary: 10-phase roadmap from PostgreSQL basics to production readiness.

**Current Blockers**: None

**Environment Requirements**:
- Docker Desktop installed and running
- PostgreSQL 16 container running on localhost:5432
- Database: dime, User: dime_user, Password: dime_password
- psql client available
- Go 1.22+ installed

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
