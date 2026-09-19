package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// TestDBConfig holds configuration for test database
var TestDBConfig = struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
}{
	Host:     getEnvOrDefault("TEST_DB_HOST", "localhost"),
	Port:     getEnvOrDefault("TEST_DB_PORT", "5433"),
	User:     getEnvOrDefault("TEST_DB_USER", "test_user"),
	Password: getEnvOrDefault("TEST_DB_PASSWORD", "test_password"),
	DBName:   getEnvOrDefault("TEST_DB_NAME", "dime_test"),
}

// getEnvOrDefault returns environment variable value or default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// SetupTestDB creates a test database connection with migrations applied
func SetupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	// Build connection string
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		TestDBConfig.Host,
		TestDBConfig.Port,
		TestDBConfig.User,
		TestDBConfig.Password,
		TestDBConfig.DBName,
	)

	// Open connection
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Verify connection with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("Failed to ping test database: %v", err)
	}

	// Configure connection pool for tests
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Run migrations
	if err := runMigrations(db); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	log.Println("Test database setup complete")
	return db
}

// runMigrations executes all pending migrations
func runMigrations(db *sql.DB) error {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://../../migrations", // Relative path from test file
		"postgres",
		driver,
	)
	if err != nil {
		// Try alternative path
		m, err = migrate.NewWithDatabaseInstance(
			"file://./migrations",
			"postgres",
			driver,
		)
		if err != nil {
			return fmt.Errorf("failed to create migrate instance: %w", err)
		}
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// CleanupTestDB truncates all tables to ensure test isolation
func CleanupTestDB(t *testing.T, db *sql.DB) {
	t.Helper()

	tables := []string{"transactions", "budgets"}

	for _, table := range tables {
		_, err := db.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
		if err != nil {
			t.Logf("Warning: failed to truncate table %s: %v", table, err)
		}
	}
}

// TeardownTestDB closes the database connection
func TeardownTestDB(t *testing.T, db *sql.DB) {
	t.Helper()

	if err := db.Close(); err != nil {
		t.Logf("Warning: failed to close test database: %v", err)
	}
}

// WithTransaction runs a test function within a transaction that gets rolled back
// This provides automatic test isolation without truncating tables
func WithTransaction(t *testing.T, db *sql.DB, fn func(*sql.Tx)) {
	t.Helper()

	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	// Ensure rollback happens even if test panics
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	// Run the test function
	fn(tx)

	// Always rollback to maintain test isolation
	if err := tx.Rollback(); err != nil {
		t.Logf("Warning: failed to rollback transaction: %v", err)
	}
}

// WaitForTestDB waits for the test database to be ready
func WaitForTestDB(timeout time.Duration) error {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		TestDBConfig.Host,
		TestDBConfig.Port,
		TestDBConfig.User,
		TestDBConfig.Password,
		TestDBConfig.DBName,
	)

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		db, err := sql.Open("pgx", connStr)
		if err != nil {
			time.Sleep(500 * time.Millisecond)
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		err = db.PingContext(ctx)
		cancel()
		db.Close()

		if err == nil {
			return nil
		}

		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("test database not ready after %v", timeout)
}
