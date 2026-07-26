# dime-api — Go API Capstone

A layered Go REST API for tracking transactions and budgets (Dime Diary domain),
structured close to a FastAPI router/service/repository setup.

No external dependencies — pure Go standard library, requires **Go 1.22+**
(uses `net/http`'s built-in method+path routing and `r.PathValue`).

## Project structure

```
cmd/api/main.go              — entrypoint, wires everything together (composition root)
internal/model/               — domain structs (Transaction, Budget)
internal/repository/          — data access interfaces + in-memory implementations
internal/service/              — business logic (validation, budget status, async checks)
internal/handler/              — HTTP layer (request parsing, JSON responses)
internal/idgen/                 — tiny dependency-free random ID generator
```

## Run it

```bash
go run ./cmd/api
```

Server starts on `:8080`.

## Swagger UI

After starting the server, open:

```text
http://localhost:8080/swagger
```

Use **Try it out** in Swagger UI to call the endpoints from your browser. The
OpenAPI spec is also available at:

```text
http://localhost:8080/openapi.yaml
```

## Endpoints

### Transactions

**Create a transaction**
```bash
curl -X POST localhost:8080/transactions \
  -H "Content-Type: application/json" \
  -d '{"user_id":"u1","type":"expense","amount":75.25,"category":"food"}'
```

**Get a transaction by ID**
```bash
curl localhost:8080/transactions/<id>
```

**List a user's transactions**
```bash
curl "localhost:8080/transactions?user_id=u1"
```

### Budgets

**Create a budget**
```bash
curl -X POST localhost:8080/budgets \
  -H "Content-Type: application/json" \
  -d '{"user_id":"u1","category":"food","limit":100}'
```

**Check budget status**
```bash
curl "localhost:8080/budgets/status?user_id=u1&category=food"
```
Returns one of: `"within budget"`, `"near limit"` (>80% spent), `"over budget"` (>=100% spent).

## How the pieces fit together

- `TransactionRepository` / `BudgetRepository` are **interfaces**. The service layer
  depends on the interface, not the concrete `InMemory*` struct — swap in a Postgres
  or Firestore implementation later without touching `service/` or `handler/` at all.
- `TransactionService` holds a reference to `BudgetService` (constructor-injected in
  `main.go`, same idea as injecting a repository into a FastAPI service class).
- When `RecordTransaction` is called with `type: "expense"`, it spins off a goroutine
  (`go s.checkBudgetAfterExpense(...)`) that updates the budget's spent total and logs
  a warning if the category goes over budget — **without blocking the HTTP response**.
  Watch the server's stdout while hitting the endpoints above to see the log line
  appear asynchronously, right after the response has already gone out.
- `loggingMiddleware` in `main.go` wraps the whole mux — plain function composition,
  Go's equivalent of a FastAPI middleware/decorator.
