# Phase 8: Configuration & Environment

## Overview

Phase 8 transforms our hardcoded configuration into a production-ready, environment-aware system. We moved from scattered `getEnv()` calls to a centralized, type-safe configuration struct with validation, security features, and environment-specific defaults.

## Learning Objectives

- Centralize configuration in a struct with type safety
- Use environment variables properly (12-Factor App methodology)
- Implement configuration validation (fail fast on startup)
- Handle different environments (development, test, production)
- Keep sensitive data secure (password masking)
- Build a system that's easy to maintain and extend

---

## Why Configuration Management Matters

### Before (Hardcoded)
```go
// Scattered throughout main.go
host := getEnv("DB_HOST", "localhost")
port := getEnv("DB_PORT", "5432")
// ... repeated 10+ times
```

**Problems:**
- No validation - bad values caught at runtime
- No types - everything is a string
- No documentation - have to read code to find env vars
- No security - easy to accidentally log passwords
- No environment awareness - same settings everywhere

### After (Configuration Struct)
```go
// One centralized location
cfg, err := config.Load()
if err != nil {
    log.Fatal("Invalid config:", err)  // Fail fast!
}

// Type-safe access
dbPort := cfg.Database.Port  // int, not string
timeout := cfg.Server.ReadTimeout  // already validated
```

**Benefits:**
- ✅ Centralized - one place for all config
- ✅ Type-safe - int, string, bool properly typed
- ✅ Validated - errors caught on startup
- ✅ Secure - passwords automatically masked
- ✅ Environment-aware - different settings per environment

---

## Milestone 1: Configuration Struct

### File: `internal/config/config.go`

Created a central configuration struct that holds all application settings:

```go
type Config struct {
    Environment string           // development, test, production
    Database    DatabaseConfig   // DB connection settings
    Server      ServerConfig     // HTTP server settings
    LogLevel    string           // debug, info, warn, error
}
```

**Key Concepts:**

1. **Struct Tags**: `env:"DB_HOST" envDefault:"localhost"`
   - Tells the library which environment variable to read
   - Provides default values if env var not set

2. **Type Safety**: Port is `int`, not `string`
   - Library automatically converts "5432" → 5432
   - Prevents type errors at runtime

3. **Load Function**: `config.Load()`
   - One call to load all configuration
   - Returns fully populated struct or error

---

## Milestone 2: Environment-Specific Config

Different environments need different settings:

| Setting | Development | Test | Production |
|---------|-------------|------|------------|
| Read Timeout | 30s | 2s | 5s |
| Write Timeout | 60s | 5s | 10s |
| Log Level | debug | error | warn |
| Default Password | dime_password | (empty) | (empty) |

**Implementation:**
```go
func (c *Config) applyEnvironmentDefaults() {
    switch c.Environment {
    case "production":
        // Strict timeouts, no default password
    case "test":
        // Quick timeouts for fast tests
    default: // development
        // Generous timeouts for debugging
    }
}
```

**Environment Variable:**
```bash
APP_ENV=production  # Controls which defaults apply
```

---

## Milestone 3: Configuration Validation

**Philosophy: Fail Fast**
- Catch configuration errors immediately on startup
- Don't wait for a user request to discover bad config
- Provide clear, actionable error messages

### Validation Examples

```go
// Port range check
if cfg.Database.Port < 1 || cfg.Database.Port > 65535 {
    return error
}

// Production requirements
if cfg.IsProduction() && cfg.Database.Password == "" {
    return error  // Password required in prod
}

// Timeout validation
if cfg.Server.ReadTimeout <= 0 {
    return error  // Must be positive
}
```

**Error Output:**
```
configuration validation failed:
  - DB_PORT must be between 1 and 65535, got 99999
  - DB_PASSWORD is required in production
  - SERVER_READ_TIMEOUT must be positive, got 0
```

---

## Milestone 4: Security Improvements

### Password Masking

**Never log passwords!**

```go
// String() method automatically masks password
fmt.Println(cfg.String())
// Output: Config{Environment:production Database:{Host:localhost Port:5432 User:admin Password:***REDACTED*** ...}}
```

### Security Checks

```go
func (c *Config) IsSecure() bool {
    // Production must use SSL
    if c.IsProduction() && c.Database.SSLMode == "disable" {
        return false
    }
    
    // No empty passwords in production
    if c.IsProduction() && c.Password == "" {
        return false
    }
    
    return true
}
```

### Password Strength

Warns about weak passwords (not an error, just a warning):
```go
if password == "password" || password == "123456" {
    slog.Warn("Weak database password detected")
}
```

---

## Milestone 5: Integration with Main Application

### Before (main.go)
```go
func main() {
    // Hardcoded config loading
    db, err := connectDB()  // Uses getEnv() internally
    if err != nil {
        log.Fatal(err)
    }
}
```

### After (main.go)
```go
func main() {
    // Load and validate config
    cfg, err := config.Load()
    if err != nil {
        log.Fatalf("Failed to load configuration: %v", err)
    }
    
    // Validate before starting
    if err := cfg.Validate(); err != nil {
        log.Fatalf("Invalid configuration: %v", err)
    }
    
    // Use config
    db, err := connectDB(cfg)
    if err != nil {
        log.Fatalf("Failed to connect to database: %v", err)
    }
}
```

**Benefits:**
- App refuses to start with bad config
- Clear error messages at startup
- Config available throughout application
- Removed old `getEnv()` helper function

---

## File Structure

```
dime-api/
├── cmd/api/main.go                    # Updated to use config
├── internal/
│   └── config/
│       └── config.go                  # All configuration logic
├── .env                               # Local dev config (gitignored)
├── .env.example                       # Template for developers
└── docs/
    ├── backend-learning/
    │   └── phase-08-notes.md          # This file
    └── configuration.md               # Quick reference
```

---

## Configuration Reference

### Required Environment Variables

None are strictly required - everything has defaults for development.

### Optional Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `APP_ENV` | development | Environment: development, test, production |
| `DB_HOST` | localhost | PostgreSQL host |
| `DB_PORT` | 5432 | PostgreSQL port |
| `DB_USER` | dime_user | Database user |
| `DB_PASSWORD` | (empty) | Database password |
| `DB_NAME` | dime | Database name |
| `DB_SSL_MODE` | disable | SSL mode: disable, require, verify-full |
| `SERVER_PORT` | 8080 | HTTP server port |
| `SERVER_READ_TIMEOUT` | (env-specific) | HTTP read timeout in seconds |
| `SERVER_WRITE_TIMEOUT` | (env-specific) | HTTP write timeout in seconds |
| `LOG_LEVEL` | (env-specific) | Logging level: debug, info, warn, error |

### Production Requirements

In production (`APP_ENV=production`), the following are required:
- `DB_PASSWORD` must be set (not empty)
- `DB_SSL_MODE` should not be `disable` (security risk)
- Timeouts should be reasonable (recommend ≤30s)

---

## Testing Different Environments

### Development (Default)
```bash
go run ./cmd/api
# Uses: 30s timeouts, debug logging, default password
```

### Production
```bash
$env:APP_ENV="production"
$env:DB_PASSWORD="secure_password"
go run ./cmd/api
# Uses: 5s timeouts, warn logging, requires password
```

### Test
```bash
$env:APP_ENV="test"
go run ./cmd/api
# Uses: 2s timeouts, error logging only
```

---

## Common Issues & Solutions

### Issue: "DB_PASSWORD is required in production"
**Solution:** Set the password environment variable:
```bash
$env:DB_PASSWORD="your_secure_password"
```

### Issue: "Invalid configuration" on startup
**Solution:** Check the error message - it lists all validation failures. Common causes:
- Port out of range (must be 1-65535)
- Empty required field
- Invalid environment name

### Issue: App won't start in production
**Solution:** Production has stricter requirements:
- Must set DB_PASSWORD
- Should enable SSL (DB_SSL_MODE=require)
- Use reasonable timeouts

---

## Key Concepts Learned

### 1. 12-Factor App Methodology
- Store config in environment variables
- Keep config out of code
- Separate config per environment

### 2. Fail Fast Principle
- Validate on startup, not at runtime
- Provide clear error messages
- Don't start with bad config

### 3. Security by Default
- Never log passwords
- Mask sensitive data automatically
- Production has stricter requirements

### 4. Type Safety
- Use proper types (int, bool, not just string)
- Catch type errors at compile time
- Automatic conversion from env vars

---

## Next Phase Preview

**Phase 9: Pagination & Performance**

Will cover:
- Paginating large result sets
- Database query optimization
- Connection pool tuning
- Caching strategies
- Rate limiting

---

## Summary

Phase 8 transforms our application from a learning project to a production-ready system:

✅ **Centralized config** - One struct holds all settings  
✅ **Environment awareness** - Different defaults per environment  
✅ **Validation** - Catch errors on startup  
✅ **Security** - Passwords never logged  
✅ **Type safety** - Proper types, not strings  
✅ **Documentation** - Clear reference for all options  

The configuration system is now:
- **Maintainable**: Add new settings in one place
- **Testable**: Easy to inject test config
- **Secure**: Sensitive data protected
- **Flexible**: Works in any environment

**Phase 8 Complete!** 🎉

---

*Part of: Backend Learning Project - Phase 8*  
*Prerequisite: Phase 7 (Integration Testing)*  
*Next: Phase 9 (Pagination & Performance)*
