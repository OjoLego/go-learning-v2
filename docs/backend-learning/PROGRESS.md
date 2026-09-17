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
- **Status**: COMPLETED
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

**Phase 4: Go Database Integration**

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

### Phase 3: Database Migrations
- Installed golang-migrate CLI tool (v4.17.1)
- Created proper migration file structure with up/down pairs
- Converted Phase 2 SQL into version-controlled migrations:
  * 001_create_transactions_table.up.sql / .down.sql
  * 002_create_budgets_table.up.sql / .down.sql
- Learned migration naming conventions (sequential numbering)
- Implemented programmatic migration execution in Go
- Created PostgreSQL repository implementations
- Removed in-memory repositories (production-standard)
- Added automatic migration on application startup
- Tested rollback (down) and re-apply (up) workflows
- Created comprehensive documentation (phase-03-notes.md)
- Current migration version: 2 (both migrations applied)

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
- Go database integration with pgx
- Transaction management in Go
- Connection pooling

---

### Phase 3: Database Migrations

#### Objective
Learn schema versioning with golang-migrate and convert manual SQL into version-controlled migrations that run programmatically from Go.

#### Tasks
- [x] Understand what database migrations are and why they're essential
- [x] Install golang-migrate CLI tool (v4.17.1)
- [x] Learn migration file naming conventions (timestamp-based, up/down pairs)
- [x] Create 001_create_transactions_table migration (up/down)
- [x] Create 002_create_budgets_table migration (up/down)
- [x] Learn about idempotent migrations (IF NOT EXISTS)
- [x] Understand up vs down migrations
- [x] Learn version tracking via schema_migrations table
- [x] Run migrations using CLI
- [x] Implement programmatic migration execution in Go
- [x] Add migration status checking in application startup
- [x] Test rollback (down) and re-apply (up) workflows
- [x] Create PostgreSQL repository implementations
- [x] Remove in-memory repositories (production-standard)
- [x] Clean up old SQL files
- [x] Document migration workflow in phase-03-notes.md

#### Current Progress
100% - Phase 3 Complete

#### Problems Encountered
- Initial go install of golang-migrate lacked PostgreSQL drivers
- Solution: Downloaded pre-built binary with all drivers included
- Port conflicts when testing server multiple times
- Solution: Kill existing processes before restarting

#### Decisions Made
- **Migration Tool**: golang-migrate (industry standard, supports up/down)
- **File Naming**: Sequential numbers (001, 002) instead of timestamps for clarity
- **Idempotency**: Use IF NOT EXISTS for safe re-runs
- **Architecture**: Always use PostgreSQL in production (removed in-memory switching)
- **Migration Timing**: Run automatically on application startup

#### Concepts Learned
- Database migrations = version control for database schema
- Up migrations apply changes (CREATE, ALTER)
- Down migrations undo changes (DROP, ALTER back)
- Version tracking via schema_migrations table
- Idempotent migrations (safe to run multiple times)
- Transaction safety in migrations (all-or-nothing)
- golang-migrate CLI commands (up, down, version, force)
- Programmatic migration execution from Go code
- Migration file structure and naming conventions
- Rollback strategies and when to use them
- Never modify applied migrations (create new ones instead)

#### Concepts Still to Understand
- None - Phase 3 complete! Moving to Phase 4 for advanced database topics.

---

## Next Phase

**Phase 4: Go Database Integration**

Learn advanced Go database patterns:
- Connection pooling configuration (MaxOpenConns, MaxIdleConns, timeouts)
- Context-aware queries with cancellation and timeouts
- Transaction management (Begin, Commit, Rollback, isolation levels)
- Go database integration with pgx driver
- Error handling patterns and PostgreSQL error codes
- Query debugging, logging, and observability
- Testing database code (integration tests, test database setup)
- Install and configure database/sql with pgx driver
- Connection pooling configuration
- Query execution with context
- Error handling patterns
- Testing database connections

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

**Status**: ✅ Implemented - Migrations running automatically on startup

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

**Current State**: Phase 3 Complete - Database Migrations

**Immediate Next Step**: 
Begin Phase 4: Go Database Integration with database/sql and pgx.

**Phase 3 Summary**:
- Installed golang-migrate CLI tool (migrate.exe v4.17.1)
- Created proper migration files with up/down pairs:
  * 001_create_transactions_table.up.sql / .down.sql
  * 002_create_budgets_table.up.sql / .down.sql
- Implemented programmatic migration execution in cmd/api/main.go
- Added automatic migration on application startup
- Created PostgreSQL repository implementations (postgres_budget_repo.go, postgres_transaction_repo.go)
- Removed in-memory repositories (production-standard code)
- Tested rollback (down 1) and re-apply (up) workflows
- Current migration version: 2 (both migrations applied)
- Documentation created in phase-03-notes.md

**Context for New Session**:
- This is a Go backend learning project (dime-api) for tracking transactions and budgets
- Current implementation uses PostgreSQL with automatic migrations
- Goal is to deepen Go database integration knowledge
- Architecture: Handler → Service → Repository → PostgreSQL
- Migrations run automatically on startup
- Ready to implement advanced database patterns (Phase 4+)

**Key Files**:
- `cmd/api/main.go` - Entry point with migration logic
- `internal/model/` - Domain models (Transaction, Budget)
- `internal/repository/interface.go` - Repository interfaces
- `internal/repository/postgres_budget_repo.go` - PostgreSQL implementation
- `internal/repository/postgres_transaction_repo.go` - PostgreSQL implementation
- `internal/service/` - Business logic
- `internal/handler/` - HTTP handlers
- `migrations/001_*.sql` and `002_*.sql` - Version-controlled migrations
- `docs/backend-learning/PROGRESS.md` - This file
- `docs/backend-learning/phase-03-notes.md` - Phase 3 documentation
- `.env` - Database configuration
- `migrate.exe` - CLI tool for manual migrations

**Database Schema**:
- **transactions**: id (UUID), user_id (TEXT), type (VARCHAR), amount (DECIMAL), category (TEXT), created_at (TIMESTAMPTZ)
- **budgets**: id (UUID), user_id (TEXT), category (TEXT), limit_amount (DECIMAL), spent (DECIMAL), updated_at (TIMESTAMPTZ)
- **schema_migrations**: version (INTEGER), dirty (BOOLEAN) - tracks migration state
- Constraints: CHECK (amount > 0), CHECK (type IN ('income', 'expense')), UNIQUE(user_id, category)
- Indexes: idx_transactions_user_id, idx_transactions_user_category, idx_transactions_created_at, idx_budgets_user_category

**Migration Commands**:
```powershell
# Check version
.\migrate.exe -path migrations -database "postgres://dime_user:dime_password@localhost:5432/dime?sslmode=disable" version

# Rollback 1
.\migrate.exe -path migrations -database "postgres://dime_user:dime_password@localhost:5432/dime?sslmode=disable" down 1

# Apply migrations
.\migrate.exe -path migrations -database "postgres://dime_user:dime_password@localhost:5432/dime?sslmode=disable" up
```

**Master Plan Location**: Original 10-phase roadmap in conversation history.
Phases 1-3 complete. Phase 4: Go Database Integration starting.

**Current Blockers**: None

**Environment Requirements**:
- Docker Desktop installed and running
- PostgreSQL 16 container running on localhost:5432
- Database: dime, User: dime_user, Password: dime_password
- psql client available
- Go 1.22+ installed
- migrate.exe CLI tool available

---

## Quick Reference

### Project Structure
```
cmd/api/main.go                    # Entry point with migrations
internal/
  model/                           # Domain structs
    transaction.go
    budget.go
  repository/                      # Data access
    interface.go                   # Repository interfaces
    postgres_budget_repo.go        # PostgreSQL implementation
    postgres_transaction_repo.go   # PostgreSQL implementation
  service/                         # Business logic
    transaction_service.go
    budget_service.go
  handler/                         # HTTP layer
    transaction_handler.go
    budget_handler.go
migrations/                        # Database migrations
  001_create_transactions_table.up.sql
  001_create_transactions_table.down.sql
  002_create_budgets_table.up.sql
  002_create_budgets_table.down.sql
docs/backend-learning/             # Learning documentation
  PROGRESS.md                      # This file
  phase-01-notes.md               # Phase 1: PostgreSQL setup
  phase-02-notes.md               # Phase 2: Schema design
  phase-03-notes.md               # Phase 3: Migrations
.env                               # Database configuration
migrate.exe                        # CLI tool for migrations
dime-api.exe                       # Compiled binary
```

### Current Architecture Flow
```
HTTP Request
  → Handler (parsing, JSON)
    → Service (business logic)
      → Repository (interface)
        → PostgreSQL (current)
```

### Repositories (Current - PostgreSQL)
- `TransactionRepository`: Create, GetByID, ListByUser
- `BudgetRepository`: Create, GetByUserAndCategory, AddSpent

### Endpoints
- `POST /transactions` - Create transaction
- `GET /transactions/{id}` - Get transaction by ID
- `GET /transactions?user_id=X` - List user's transactions
- `POST /budgets` - Create budget
- `GET /budgets/status?user_id=X&category=Y` - Check budget status
- `GET /swagger` - Swagger UI documentation
- `GET /openapi.yaml` - OpenAPI specification

### Running the Application
```powershell
# Start PostgreSQL (if not running)
docker-compose up -d

# Run the application (automatically runs migrations)
.\dime-api.exe
# or
go run ./cmd/api

# Application will output:
# - Current migration version
# - PostgreSQL connection status
# - Migration application status
# - Server startup confirmation
```

---

*Last Updated*: Phase 3 Complete
*Next Session Should Begin*: Phase 4, Go Database Integration
