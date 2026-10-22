# Dime API

A production-hardened Go REST API for tracking personal transactions and budgets. Built as a backend learning project with PostgreSQL, structured logging, JWT authentication, authorization, request validation, rate limiting, and OpenAPI documentation.

## Features

- **Transactions:** create and list paginated transactions (income/expense).
- **Budgets:** set category budgets and check spending status.
- **Authentication:** JWT bearer tokens via `POST /auth/token`.
- **Authorization:** users can only access their own data.
- **Input Validation:** field-level validation with clear error responses.
- **Rate Limiting:** per-user token-bucket rate limiting.
- **Structured Logging:** JSON logs with request IDs and configurable log levels.
- **Health & Metrics:** `/health` and `/metrics` endpoints.
- **Database Migrations:** managed with `golang-migrate`.
- **OpenAPI/Swagger:** interactive docs at `/swagger`.

## Tech Stack

- Go 1.26
- PostgreSQL
- `pgx` PostgreSQL driver
- `golang-migrate` for migrations
- `golang-jwt/jwt/v5` for JWT tokens
- `go-playground/validator/v10` for request validation
- `golang.org/x/time/rate` for rate limiting

## Quick Start

### 1. Prerequisites

- Go 1.26+
- PostgreSQL running locally, or Docker with the provided `docker-compose.yml`:

```powershell
docker-compose up -d
```

### 2. Configure the environment

```powershell
copy .env.example .env
```

Edit `.env` with your database credentials and a strong `JWT_SECRET`.

### 3. Run migrations

```powershell
migrate -path migrations -database "postgres://dime_user:dime_password@localhost:5432/dime?sslmode=disable" up
```

### 4. Start the server

```powershell
go run ./cmd/api/main.go
```

The API will be available at `http://localhost:8080`.

## API Overview

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| POST | `/auth/token` | Public | Generate a JWT token |
| POST | `/transactions` | Bearer | Create a transaction |
| GET | `/transactions` | Bearer | List user's transactions (paginated) |
| GET | `/transactions/{id}` | Bearer | Get a transaction by ID |
| POST | `/budgets` | Bearer | Create a budget |
| GET | `/budgets/status` | Bearer | Check budget status |
| GET | `/health` | Public | Health check |
| GET | `/metrics` | Public | Request metrics snapshot |
| GET | `/swagger` | Public | Interactive API docs |

See `internal/swagger/openapi.yaml` for the full spec.

## Example Usage

### Get a token

```powershell
$token = (curl.exe -s -H "Content-Type: application/json" -d '{\"user_id\":\"alice\"}' http://localhost:8080/auth/token | ConvertFrom-Json).token
```

### Create a transaction

```powershell
curl.exe -H "Authorization: Bearer $token" -H "Content-Type: application/json" -d '{\"user_id\":\"alice\",\"type\":\"expense\",\"amount\":10,\"category\":\"food\"}' http://localhost:8080/transactions
```

### List transactions

```powershell
curl.exe -H "Authorization: Bearer $token" http://localhost:8080/transactions?user_id=alice
```

## Running Tests

```powershell
# All tests (requires PostgreSQL for integration tests)
go test ./...

# Unit tests only
go test -short ./...

# Static analysis
go vet ./...
```

## Environment Variables

See `.env.example` for all available variables. Key ones:

| Variable | Description | Default |
|----------|-------------|---------|
| `APP_ENV` | Environment: development, test, production | `development` |
| `LOG_LEVEL` | Log verbosity: debug, info, warn, error | `info` |
| `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME` | PostgreSQL connection | see `.env.example` |
| `SERVER_PORT` | HTTP server port | `8080` |
| `JWT_SECRET` | Secret for signing JWT tokens | required |
| `JWT_EXPIRY_MINUTES` | JWT token lifetime | `60` |
| `RATE_LIMIT_RPS` | Requests per second per user/IP | `10` |
| `RATE_LIMIT_BURST` | Burst capacity per user/IP | `20` |
| `CORS_ALLOWED_ORIGINS` | Allowed CORS origins | `*` |

## Project Structure

```
.
├── cmd/api/main.go              # Application entry point
├── internal
│   ├── auth                     # JWT generation and validation
│   ├── config                   # Environment configuration
│   ├── database                 # Transaction manager
│   ├── handler                  # HTTP handlers
│   ├── idgen                    # ID generation
│   ├── metrics                  # Request metrics
│   ├── middleware               # Auth, logging, rate limiting, request ID
│   ├── model                    # Domain models
│   ├── repository               # Database repositories and interfaces
│   ├── service                  # Business logic
│   ├── swagger                  # OpenAPI spec and Swagger UI
│   └── validator                # Request validation
├── migrations                   # SQL migrations
├── docs/backend-learning        # Learning notes and progress
└── README.md
```

## Documentation

Phase-by-phase learning notes are in `docs/backend-learning/`:

- [Phase 10: Production Hardening](docs/backend-learning/phase-10-notes.md)
- [Progress](docs/backend-learning/PROGRESS.md)

## License

MIT
