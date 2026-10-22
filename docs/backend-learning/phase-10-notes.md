# Phase 10: Production Hardening

## Overview

Phase 10 hardens the Dime API for production use. Up to this point the application works, but it lacks the controls needed to run safely in the real world: consistent error handling, input validation, authentication, authorization, rate limiting, observability, and accurate documentation. This phase adds those controls and documents them.

## Learning Objectives

- Centralize error handling so internal details never leak to clients.
- Validate request payloads with field-level error messages.
- Authenticate users with JWT tokens and protect private endpoints.
- Authorize requests so users cannot access each other's data.
- Add rate limiting to prevent abuse.
- Move health checks into handlers and expose lightweight metrics.
- Keep OpenAPI specs, environment examples, and README in sync with the code.
- Run a final security and correctness checklist.

---

## What Was Added in Each Milestone

### Milestone 1: Structured Logging + Config

**Files:** `cmd/api/main.go`, `internal/config/config.go`

- Added `LOG_LEVEL` configuration.
- Configured `slog` with JSON output and a configurable level.
- Replaced the old `loggingMiddleware` with `middleware.StructuredLogger` and `middleware.RequestIDMiddleware`.

**Why it matters:** Structured JSON logs are easy to index and search. Request IDs make it possible to trace a single request through every log line. A configurable log level keeps development verbose and production quiet.

### Milestone 2: Centralized Error Handling

**Files:** `internal/handler/errors.go`, `internal/handler/transaction_handler.go`, `internal/handler/budget_handler.go`, `internal/repository/interface.go`, `internal/service/transaction_service.go`, `internal/service/budget_service.go`

- Added `APIError`, `ErrCodeNotFound`, `ErrCodeInvalidRequest`, `ErrCodeInternal`, etc.
- Created `RespondError` to map domain errors to HTTP status codes and safe messages.
- Added sentinel errors such as `repository.ErrNotFound`, `repository.ErrBudgetNotFound`, `service.ErrInvalidAmount`, and `service.ErrInvalidLimit`.
- Replaced direct `respondError(w, status, message)` calls with `RespondError(w, r, err)`.

**Why it matters:** A single place for error mapping guarantees that 500 responses never expose stack traces or database internals. Real errors are logged with request IDs for debugging, while clients receive stable, machine-readable codes.

### Milestone 3: Request Validation

**Files:** `internal/handler/transaction_handler.go`, `internal/handler/budget_handler.go`, `internal/validator/validator.go`

- Added struct tags (`validate:"required"`, `validate:"gt=0"`, `validate:"oneof=income expense"`) to request structs.
- Added `validator.ValidateStruct(req)` in handlers after JSON decode.
- Returned `VALIDATION_ERROR` with a `fields` array pointing to every invalid field.

**Why it matters:** Validating at the boundary prevents bad data from reaching the service or database. Field-level errors let clients highlight the exact input problem instead of showing a generic message.

### Milestone 4: JWT Authentication

**Files:** `internal/auth/jwt.go`, `internal/handler/auth_handler.go`, `internal/middleware/auth.go`, `cmd/api/main.go`

- Added `GenerateToken` and `ValidateToken` using `golang-jwt/jwt/v5`.
- Added `POST /auth/token` to issue tokens.
- Added `AuthMiddleware` to validate the `Authorization: Bearer <token>` header.
- Split routes into `publicMux` and `protectedMux`.

**Why it matters:** Without authentication anyone can call private endpoints. JWTs are stateless and scale well because the server only needs the signing secret to validate them.

### Milestone 5: Authorization

**Files:** `internal/handler/transaction_handler.go`, `internal/handler/budget_handler.go`, `internal/handler/errors.go`

- Extracted the authenticated `user_id` from the request context.
- Compared `req.UserID` / query `user_id` with the authenticated user.
- Added `ErrUnauthorized` (401) and `ErrForbidden` (403) responses.

**Why it matters:** Authentication only proves who you are. Authorization proves what you are allowed to do. This prevents Alice from reading or writing Bob's transactions and budgets.

### Milestone 6: Rate Limiting

**Files:** `internal/ratelimit/limiter.go`, `internal/ratelimit/memory.go`, `internal/middleware/ratelimit.go`, `internal/config/config.go`, `cmd/api/main.go`

- Defined a `Limiter` interface and an in-memory token-bucket implementation using `golang.org/x/time/rate`.
- Added `RateLimitMiddleware` and `RateLimitKey`.
- Per-user rate limiting for authenticated routes, IP-based fallback for public routes.
- Added `RATE_LIMIT_RPS` and `RATE_LIMIT_BURST` configuration.

**Why it matters:** Rate limiting protects the API from accidental loops, brute-force attacks, and runaway clients. Different limits can later be applied per plan or per user tier.

### Milestone 9: Health Checks + Metrics

**Files:** `internal/handler/health_handler.go`, `internal/metrics/metrics.go`, `internal/middleware/logging.go`, `cmd/api/main.go`

- Moved health check logic from `main.go` into `HealthHandler`.
- Added a `/metrics` endpoint that returns request counters.
- Updated `StructuredLogger` to increment `requests_total` and `requests_<status>`.

**Why it matters:** Load balancers and orchestrators need a `/health` endpoint to know when to restart or stop sending traffic. Metrics give visibility into traffic patterns without requiring a full observability stack.

### Milestone 10: Documentation + Security Review

**Files:** `internal/swagger/openapi.yaml`, `.env.example`, `README.md`, `docs/backend-learning/phase-10-notes.md`

- Added `bearerAuth` security scheme to OpenAPI.
- Documented 401, 403, 429 responses on protected routes.
- Updated `.env.example` with comments for every variable.
- Rewrote `README.md` to describe the Dime API.
- Created this phase note and updated `PROGRESS.md`.

**Why it matters:** Code and docs drift apart over time. Keeping them aligned makes onboarding, testing, and security reviews easier.

---

## How to Run Tests

### Unit tests

```powershell
go test ./...
```

If you are behind the corporate proxy, set the proxy first:

```powershell
$env:HTTP_PROXY='http://trendgate.interswitchng.com:8080'
$env:HTTPS_PROXY='http://trendgate.interswitchng.com:8080'
go test ./...
```

### Skip integration tests

Integration tests need a running PostgreSQL instance. To run only unit tests:

```powershell
go test -short ./...
```

### Static analysis

```powershell
go vet ./...
go build ./cmd/api/main.go
```

### Manual verification

Start the server:

```powershell
go run ./cmd/api/main.go
```

Get a token:

```powershell
$token = (curl.exe -s -H "Content-Type: application/json" -d '{\"user_id\":\"alice\"}' http://localhost:8080/auth/token | ConvertFrom-Json).token
```

Create a transaction:

```powershell
curl.exe -H "Authorization: Bearer $token" -H "Content-Type: application/json" -d '{\"user_id\":\"alice\",\"type\":\"expense\",\"amount\":10,\"category\":\"food\"}' http://localhost:8080/transactions
```

Try cross-user access (should return 403):

```powershell
curl.exe -H "Authorization: Bearer $token" -H "Content-Type: application/json" -d '{\"user_id\":\"bob\",\"type\":\"expense\",\"amount\":10,\"category\":\"food\"}' http://localhost:8080/transactions
```

---

## Security Checklist

Before considering this phase complete, verify every item below.

| # | Check | How to verify |
|---|-------|---------------|
| 1 | Protected endpoints reject requests without a valid token. | `curl.exe http://localhost:8080/transactions?user_id=alice` should return 401. |
| 2 | Users cannot access other users' data. | Request with Alice's token but Bob's `user_id` should return 403. |
| 3 | Invalid input returns 400 with field-level errors. | `POST /transactions` with `amount: -10` returns `VALIDATION_ERROR` and a `fields` array. |
| 4 | Rate limiting returns 429. | Send many rapid requests until `429 Too Many Requests` is returned. |
| 5 | Logs do not contain `user_id`, `amount`, or raw bodies. | Review structured logs; only `method`, `path`, `status`, `duration`, and `request_id` are logged. |
| 6 | Security headers present on all responses. | Inspect `curl.exe -I` output for `X-Request-ID`, `Content-Type`, and `Retry-After` where applicable. |
| 7 | Graceful shutdown works. | Send `SIGINT` (`Ctrl+C`) and confirm the server exits cleanly. |
| 8 | Server timeouts are configured. | Check `SERVER_READ_TIMEOUT` and `SERVER_WRITE_TIMEOUT` values in `.env.example` and `internal/config/config.go`. |
| 9 | OpenAPI spec reflects authentication and new response codes. | Open `http://localhost:8080/swagger` and confirm bearer auth + 401/403/429 are shown. |
| 10 | `.env.example` documents every environment variable. | Compare `internal/config/config.go` fields with `.env.example` entries. |

---

## Key Takeaways

- **Defense in depth:** validation, authentication, authorization, and rate limiting each protect against different threats.
- **Never trust the client:** `user_id` in the body must always be checked against the authenticated identity.
- **Errors are UX:** Good error responses have a stable code, a safe message, and field-level detail where appropriate.
- **Docs are code:** OpenAPI, README, and `.env.example` should be updated as the API changes so future developers (and security reviewers) can understand the system.
