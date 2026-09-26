# Configuration Reference

Quick reference guide for all configuration options in dime-api.

---

## Quick Start

### Development (Default)
```bash
# Uses all defaults - no env vars needed
go run ./cmd/api
```

### Production
```bash
$env:APP_ENV="production"
$env:DB_PASSWORD="your_secure_password"
$env:DB_SSL_MODE="require"
go run ./cmd/api
```

---

## Environment Variables

### Application

| Variable | Default | Options | Description |
|----------|---------|---------|-------------|
| `APP_ENV` | `development` | `development`, `test`, `production` | Application environment |
| `LOG_LEVEL` | *environment-specific* | `debug`, `info`, `warn`, `error` | Logging verbosity |

**Environment-Specific Defaults:**
- **Development**: `LOG_LEVEL=debug`
- **Test**: `LOG_LEVEL=error`
- **Production**: `LOG_LEVEL=warn`

---

### Database

| Variable | Default | Required | Description |
|----------|---------|----------|-------------|
| `DB_HOST` | `localhost` | No | PostgreSQL server hostname |
| `DB_PORT` | `5432` | No | PostgreSQL server port (1-65535) |
| `DB_USER` | `dime_user` | No | Database username |
| `DB_PASSWORD` | *(empty)* | In production | Database password |
| `DB_NAME` | `dime` | No | Database name |
| `DB_SSL_MODE` | `disable` | No | SSL/TLS mode |

**SSL Modes:**
- `disable` - No SSL (development only)
- `require` - SSL required but don't verify certificate
- `verify-ca` - SSL required and verify CA certificate
- `verify-full` - SSL required and verify CA + hostname

**Production Requirements:**
- `DB_PASSWORD` must be set (cannot be empty)
- `DB_SSL_MODE` should be `require` or higher (not `disable`)

---

### Server

| Variable | Default | Environment | Description |
|----------|---------|-------------|-------------|
| `SERVER_PORT` | `8080` | All | HTTP server port |
| `SERVER_READ_TIMEOUT` | `30` (dev), `5` (prod), `2` (test) | All | HTTP read timeout in seconds |
| `SERVER_WRITE_TIMEOUT` | `60` (dev), `10` (prod), `5` (test) | All | HTTP write timeout in seconds |

**Timeout Guidelines:**
- **Development**: 30s/60s (generous for debugging)
- **Production**: 5s/10s (strict for performance)
- **Test**: 2s/5s (fast test execution)

---

## Configuration Examples

### Minimal Development Setup
```bash
# No env vars needed - uses all defaults
go run ./cmd/api
```

### Custom Database
```bash
$env:DB_HOST="my-db-server"
$env:DB_PORT="5433"
$env:DB_USER="myuser"
$env:DB_PASSWORD="mypassword"
$env:DB_NAME="myapp"
go run ./cmd/api
```

### Production Deployment
```bash
$env:APP_ENV="production"
$env:DB_HOST="prod-db.example.com"
$env:DB_PASSWORD="secure_random_password"
$env:DB_SSL_MODE="require"
$env:SERVER_PORT="80"
$env:SERVER_READ_TIMEOUT="10"
$env:SERVER_WRITE_TIMEOUT="15"
$env:LOG_LEVEL="warn"
go run ./cmd/api
```

### Docker Compose
```yaml
services:
  api:
    build: .
    environment:
      - APP_ENV=production
      - DB_HOST=postgres
      - DB_PASSWORD=${DB_PASSWORD}
      - DB_SSL_MODE=require
    ports:
      - "8080:8080"
```

---

## Validation Rules

Configuration is validated on startup. The app will **fail to start** if validation fails.

### Always Validated

- **Port ranges**: Must be 1-65535 (both DB_PORT and SERVER_PORT)
- **Timeouts**: Must be positive (> 0)
- **Environment**: Must be `development`, `test`, or `production`
- **Required fields**: DB_HOST, DB_NAME, DB_USER cannot be empty

### Production-Only Validations

- **DB_PASSWORD**: Required (cannot be empty)
- **DB_SSL_MODE**: Should not be `disable` (warning if disabled)
- **Timeouts**: Recommended ≤30 seconds (warning if longer)

### Validation Errors

Example error output:
```
2026/09/26 10:30:15 Invalid configuration: configuration validation failed:
  - DB_PORT must be between 1 and 65535, got 99999
  - DB_PASSWORD is required in production
```

---

## Security Best Practices

### ✅ Do

- Use strong passwords (8+ characters, not common words)
- Enable SSL in production (`DB_SSL_MODE=require`)
- Use environment variables (not hardcoded values)
- Keep `.env` files out of version control (add to `.gitignore`)
- Use different passwords for different environments

### ❌ Don't

- Commit `.env` files with real passwords
- Use default passwords in production
- Disable SSL in production
- Log configuration with passwords (use `cfg.String()` which auto-masks)
- Use `admin`, `postgres`, or `root` as username in production

---

## Troubleshooting

### App won't start

Check the error message:
```bash
$env:APP_ENV="production"
go run ./cmd/api
# FATAL: Invalid configuration: DB_PASSWORD is required in production
```

**Solution:** Set required environment variables.

### Database connection fails

```
Failed to connect to database: failed to ping database: ...
```

**Check:**
- Is PostgreSQL running? `docker-compose ps`
- Are DB_HOST, DB_PORT correct?
- Is DB_PASSWORD correct?
- Is the database name correct?

### Timeout errors

```
context deadline exceeded
```

**Solution:** Increase timeouts for your environment:
```bash
$env:SERVER_READ_TIMEOUT="30"
$env:SERVER_WRITE_TIMEOUT="60"
```

---

## Advanced Usage

### Programmatic Access

```go
import "dime-api/internal/config"

cfg, err := config.Load()
if err != nil {
    log.Fatal(err)
}

// Access configuration
fmt.Println(cfg.Database.Host)
fmt.Println(cfg.Server.Port)
fmt.Println(cfg.IsProduction())

// Safe logging (passwords masked)
log.Println(cfg.String())
```

### Custom Validation

```go
if err := cfg.Validate(); err != nil {
    log.Fatal("Config invalid:", err)
}

if !cfg.IsSecure() {
    log.Println("Warning: Configuration has security issues")
}
```

### Configuration with Warnings

```go
err, warnings := cfg.ValidateWithWarnings()
if err != nil {
    log.Fatal(err)
}
for _, warning := range warnings {
    log.Println("Warning:", warning)
}
```

---

## Files

- `internal/config/config.go` - Configuration implementation
- `.env` - Local development configuration (gitignored)
- `.env.example` - Template for new developers
- `docs/backend-learning/phase-08-notes.md` - Detailed learning notes

---

## See Also

- [Phase 8 Learning Notes](backend-learning/phase-08-notes.md) - Detailed explanation
- [12-Factor App Config](https://12factor.net/config) - Best practices
- [PostgreSQL SSL Modes](https://www.postgresql.org/docs/current/libpq-ssl.html) - SSL documentation
