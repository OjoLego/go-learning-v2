package config

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/caarlos0/env/v11"
)

// Config holds all application configuration in one place
type Config struct {
	// Application environment: development, test, production
	Environment string `env:"APP_ENV" envDefault:"development"`

	// Database configuration
	Database DatabaseConfig

	// Server configuration
	Server ServerConfig

	// Logging
	LogLevel string `env:"LOG_LEVEL" envDefault:"info"`
}

// DatabaseConfig holds database connection settings
type DatabaseConfig struct {
	Host     string `env:"DB_HOST" envDefault:"localhost"`
	Port     int    `env:"DB_PORT" envDefault:"5432"`
	User     string `env:"DB_USER" envDefault:"dime_user"`
	Password string `env:"DB_PASSWORD" envDefault:""`
	DBName   string `env:"DB_NAME" envDefault:"dime"`
	SSLMode  string `env:"DB_SSL_MODE" envDefault:"disable"`
}

// ServerConfig holds HTTP server settings
type ServerConfig struct {
	Port         string `env:"SERVER_PORT" envDefault:"8080"`
	ReadTimeout  int    `env:"SERVER_READ_TIMEOUT"`  // seconds - environment specific
	WriteTimeout int    `env:"SERVER_WRITE_TIMEOUT"` // seconds - environment specific
}

// Load reads configuration from environment variables and returns a Config struct
func Load() (*Config, error) {
	cfg := &Config{}

	// Use the env library to parse environment variables into the struct
	opts := env.Options{
		RequiredIfNoDef: false,
	}

	if err := env.ParseWithOptions(cfg, opts); err != nil {
		return nil, fmt.Errorf("failed to parse configuration: %w", err)
	}

	// Apply environment-specific defaults
	cfg.applyEnvironmentDefaults()

	// Log loaded configuration (safely - without password)
	slog.Info("Configuration loaded",
		slog.String("environment", cfg.Environment),
		slog.String("db_host", cfg.Database.Host),
		slog.Int("db_port", cfg.Database.Port),
		slog.String("db_name", cfg.Database.DBName),
		slog.String("db_sslmode", cfg.Database.SSLMode),
		slog.String("server_port", cfg.Server.Port),
		slog.Int("read_timeout", cfg.Server.ReadTimeout),
		slog.Int("write_timeout", cfg.Server.WriteTimeout),
		slog.String("log_level", cfg.LogLevel),
		slog.Bool("is_secure", cfg.IsSecure()),
	)

	// Log warnings if not secure
	if !cfg.IsSecure() {
		slog.Warn("Configuration may have security issues - review before deploying to production",
			slog.String("environment", cfg.Environment),
		)
	}

	return cfg, nil
}

// applyEnvironmentDefaults sets sensible defaults based on the environment
func (c *Config) applyEnvironmentDefaults() {
	switch strings.ToLower(c.Environment) {
	case "production":
		// Production: strict timeouts, no default password
		if c.Server.ReadTimeout == 0 {
			c.Server.ReadTimeout = 5 // 5 seconds
		}
		if c.Server.WriteTimeout == 0 {
			c.Server.WriteTimeout = 10 // 10 seconds
		}
		if c.LogLevel == "" || c.LogLevel == "info" {
			c.LogLevel = "warn" // Less verbose in production
		}
		// Ensure SSL is considered in production
		if c.Database.SSLMode == "disable" {
			slog.Warn("Database SSL is disabled in production environment - consider enabling for security")
		}

	case "test":
		// Test: quick timeouts for fast test execution
		if c.Server.ReadTimeout == 0 {
			c.Server.ReadTimeout = 2 // 2 seconds
		}
		if c.Server.WriteTimeout == 0 {
			c.Server.WriteTimeout = 5 // 5 seconds
		}
		if c.LogLevel == "" || c.LogLevel == "info" {
			c.LogLevel = "error" // Minimal logging in tests
		}

	default: // development
		// Development: generous timeouts for debugging
		if c.Server.ReadTimeout == 0 {
			c.Server.ReadTimeout = 30 // 30 seconds - allows for debugging
		}
		if c.Server.WriteTimeout == 0 {
			c.Server.WriteTimeout = 60 // 60 seconds - allows for debugging
		}
		if c.LogLevel == "" {
			c.LogLevel = "debug" // Verbose logging in development
		}
		// Set default password for development convenience
		if c.Database.Password == "" {
			c.Database.Password = "dime_password"
		}
	}
}

// IsDevelopment returns true if running in development mode
func (c *Config) IsDevelopment() bool {
	return strings.ToLower(c.Environment) == "development"
}

// IsProduction returns true if running in production mode
func (c *Config) IsProduction() bool {
	return strings.ToLower(c.Environment) == "production"
}

// IsTest returns true if running in test mode
func (c *Config) IsTest() bool {
	return strings.ToLower(c.Environment) == "test"
}

// DatabaseConnectionString returns the PostgreSQL connection string
func (c *Config) DatabaseConnectionString() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.Password,
		c.Database.DBName,
		c.Database.SSLMode,
	)
}

// String returns a safe string representation of the config (passwords hidden)
func (c *Config) String() string {
	return fmt.Sprintf("Config{Environment:%s Database:{Host:%s Port:%d User:%s Password:***REDACTED*** DBName:%s SSLMode:%s} Server:{Port:%s ReadTimeout:%d WriteTimeout:%d} LogLevel:%s}",
		c.Environment,
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.DBName,
		c.Database.SSLMode,
		c.Server.Port,
		c.Server.ReadTimeout,
		c.Server.WriteTimeout,
		c.LogLevel,
	)
}

// SecureString returns a JSON-like representation with all sensitive fields masked
// Safe to use in logs and error messages
func (c *Config) SecureString() string {
	return fmt.Sprintf(`{
  "environment": "%s",
  "database": {
    "host": "%s",
    "port": %d,
    "user": "%s",
    "password": "***REDACTED***",
    "dbname": "%s",
    "sslmode": "%s"
  },
  "server": {
    "port": "%s",
    "read_timeout": %d,
    "write_timeout": %d
  },
  "log_level": "%s"
}`,
		c.Environment,
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.DBName,
		c.Database.SSLMode,
		c.Server.Port,
		c.Server.ReadTimeout,
		c.Server.WriteTimeout,
		c.LogLevel,
	)
}

// IsSecure returns true if the configuration meets basic security standards
func (c *Config) IsSecure() bool {
	// In production, SSL should be enabled
	if c.IsProduction() && c.Database.SSLMode == "disable" {
		return false
	}

	// Password should not be empty (except in development)
	if !c.IsDevelopment() && strings.TrimSpace(c.Database.Password) == "" {
		return false
	}

	// Should not use default/test credentials in production
	if c.IsProduction() {
		if c.Database.User == "postgres" || c.Database.User == "admin" {
			return false
		}
		if c.Database.Password == "password" || c.Database.Password == "123456" {
			return false
		}
	}

	return true
}

// SanitizeForLogging creates a copy of the config safe for logging
func (c *Config) SanitizeForLogging() map[string]interface{} {
	return map[string]interface{}{
		"environment": c.Environment,
		"database": map[string]interface{}{
			"host":     c.Database.Host,
			"port":     c.Database.Port,
			"user":     c.Database.User,
			"password": "***REDACTED***",
			"dbname":   c.Database.DBName,
			"sslmode":  c.Database.SSLMode,
		},
		"server": map[string]interface{}{
			"port":          c.Server.Port,
			"read_timeout":  c.Server.ReadTimeout,
			"write_timeout": c.Server.WriteTimeout,
		},
		"log_level": c.LogLevel,
		"secure":    c.IsSecure(),
	}
}

// Validate checks if the configuration is valid for the current environment
// Returns detailed error messages for all validation failures
func (c *Config) Validate() error {
	var errors []string

	// Validate environment value
	if !c.isValidEnvironment() {
		errors = append(errors, fmt.Sprintf("invalid APP_ENV '%s': must be 'development', 'test', or 'production'", c.Environment))
	}

	// Validate database configuration
	if err := c.validateDatabase(); err != "" {
		errors = append(errors, err)
	}

	// Validate server configuration
	if err := c.validateServer(); err != "" {
		errors = append(errors, err)
	}

	// Production-specific validations
	if c.IsProduction() {
		if err := c.validateProduction(); err != "" {
			errors = append(errors, err)
		}
	}

	// Check password strength (warning only, not error)
	if warning := c.validatePasswordStrength(); warning != "" {
		slog.Warn("Weak database password detected",
			slog.String("warning", warning),
			slog.String("environment", c.Environment),
		)
	}

	// If there are any errors, combine them into one error
	if len(errors) > 0 {
		return fmt.Errorf("configuration validation failed:\n  - %s", strings.Join(errors, "\n  - "))
	}

	return nil
}

// isValidEnvironment checks if the environment value is allowed
func (c *Config) isValidEnvironment() bool {
	env := strings.ToLower(c.Environment)
	return env == "development" || env == "test" || env == "production"
}

// validateDatabase checks database configuration
func (c *Config) validateDatabase() string {
	// Port must be valid
	if c.Database.Port < 1 || c.Database.Port > 65535 {
		return fmt.Sprintf("DB_PORT must be between 1 and 65535, got %d", c.Database.Port)
	}

	// Host cannot be empty
	if strings.TrimSpace(c.Database.Host) == "" {
		return "DB_HOST cannot be empty"
	}

	// Database name cannot be empty
	if strings.TrimSpace(c.Database.DBName) == "" {
		return "DB_NAME cannot be empty"
	}

	// User cannot be empty
	if strings.TrimSpace(c.Database.User) == "" {
		return "DB_USER cannot be empty"
	}

	return ""
}

// validateServer checks server configuration
func (c *Config) validateServer() string {
	// Read timeout must be positive
	if c.Server.ReadTimeout <= 0 {
		return fmt.Sprintf("SERVER_READ_TIMEOUT must be positive, got %d", c.Server.ReadTimeout)
	}

	// Write timeout must be positive
	if c.Server.WriteTimeout <= 0 {
		return fmt.Sprintf("SERVER_WRITE_TIMEOUT must be positive, got %d", c.Server.WriteTimeout)
	}

	// Port must be valid
	port := 0
	fmt.Sscanf(c.Server.Port, "%d", &port)
	if port < 1 || port > 65535 {
		return fmt.Sprintf("SERVER_PORT must be between 1 and 65535, got %s", c.Server.Port)
	}

	return ""
}

// validateProduction checks production-specific requirements
func (c *Config) validateProduction() string {
	var issues []string

	// Password is required
	if strings.TrimSpace(c.Database.Password) == "" {
		issues = append(issues, "DB_PASSWORD is required in production")
	}

	// SSL should be enabled
	if c.Database.SSLMode == "disable" {
		issues = append(issues, "DB_SSL_MODE should not be 'disable' in production (use 'require' or 'verify-full')")
	}

	// Timeouts should be reasonable (not too long)
	if c.Server.ReadTimeout > 30 {
		issues = append(issues, fmt.Sprintf("SERVER_READ_TIMEOUT (%ds) is too long for production (recommend <= 30s)", c.Server.ReadTimeout))
	}

	if c.Server.WriteTimeout > 30 {
		issues = append(issues, fmt.Sprintf("SERVER_WRITE_TIMEOUT (%ds) is too long for production (recommend <= 30s)", c.Server.WriteTimeout))
	}

	if len(issues) > 0 {
		return strings.Join(issues, "; ")
	}

	return ""
}

// validatePasswordStrength checks if password meets basic requirements
// Returns a warning message if password is weak, empty string if OK
func (c *Config) validatePasswordStrength() string {
	password := c.Database.Password

	// Skip validation for development
	if c.IsDevelopment() {
		return ""
	}

	// Check minimum length
	if len(password) < 8 {
		return "password should be at least 8 characters long"
	}

	// Check for common weak passwords
	weakPasswords := []string{"password", "123456", "qwerty", "admin", "dime_password"}
	for _, weak := range weakPasswords {
		if strings.ToLower(password) == weak {
			return fmt.Sprintf("password is too common (matches '%s')", weak)
		}
	}

	return ""
}

// ValidateWithWarnings validates config and returns warnings for non-critical issues
func (c *Config) ValidateWithWarnings() (error, []string) {
	err := c.Validate()
	
	var warnings []string
	
	// Add warnings for recommended but not required settings
	if !c.IsProduction() {
		if c.Database.SSLMode != "disable" {
			warnings = append(warnings, "SSL is enabled for non-production environment (this is fine but may slow down local development)")
		}
	}
	
	if c.LogLevel == "debug" && c.IsProduction() {
		warnings = append(warnings, "LOG_LEVEL is 'debug' in production (may impact performance and log sensitive data)")
	}
	
	return err, warnings
}