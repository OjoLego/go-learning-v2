package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"time"

	"dime-api/internal/config"
	"dime-api/internal/database"
	"dime-api/internal/handler"
	"dime-api/internal/repository"
	"dime-api/internal/service"
	"dime-api/internal/swagger"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	slog.Info("Starting dime-api server",
		slog.String("version", "1.0.0"),
		slog.String("environment", cfg.Environment),
	)

	// Connect to PostgreSQL database using config
	db, err := connectDB(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	slog.Info("Connected to PostgreSQL")

	// Check migration status before running
	version, dirty, err := checkMigrationStatus(db)
	if err != nil {
		log.Println("No migrations applied yet or error checking status:", err)
	} else {
		log.Printf("Current migration version: %d, Dirty: %v", version, dirty)
	}

	// Run database migrations
	if err := runMigrations(db); err != nil {
		log.Fatal("Migration failed:", err)
	}
	log.Println("Migrations applied successfully")

	// Check status again after migrations
	version, dirty, err = checkMigrationStatus(db)
	if err != nil {
		log.Println("Error checking final migration status:", err)
	} else {
		log.Printf("Final migration version: %d, Dirty: %v", version, dirty)
	}

	// Log initial pool statistics
	logPoolStats(db)

	// --- Dependency wiring ---
	// Create transaction manager for atomic operations
	txManager := database.NewTransactionManager(db)

	budgetRepo := repository.NewPostgresBudgetRepo(db)
	budgetSvc := service.NewBudgetService(budgetRepo)
	budgetHandler := handler.NewBudgetHandler(budgetSvc)

	txRepo := repository.NewPostgresTransactionRepo(db)
	txSvc := service.NewTransactionService(txRepo, budgetSvc, txManager)
	txHandler := handler.NewTransactionHandler(txSvc)

	// --- Routes ---
	mux := http.NewServeMux()

	mux.HandleFunc("POST /transactions", txHandler.Create)
	mux.HandleFunc("GET /transactions/{id}", txHandler.GetByID)
	mux.HandleFunc("GET /transactions", txHandler.ListByUser)

	mux.HandleFunc("POST /budgets", budgetHandler.Create)
	mux.HandleFunc("GET /budgets/status", budgetHandler.Status)

	mux.HandleFunc("GET /swagger", swagger.UI)
	mux.HandleFunc("GET /openapi.yaml", swagger.OpenAPI)

	// Health check endpoint
	mux.HandleFunc("GET /health", healthCheck(db))

	handlerWithMiddleware := loggingMiddleware(mux)

	fmt.Println("Server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", handlerWithMiddleware))
}

// connectDB establishes connection to PostgreSQL using configuration
func connectDB(cfg *config.Config) (*sql.DB, error) {
	// Use the connection string from config
	connStr := cfg.DatabaseConnectionString()

	// Use pgx driver
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Verify connection with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Configure connection pool using values from config
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(1 * time.Minute)

	slog.Info("Database connection pool configured",
		slog.Int("max_open_conns", 25),
		slog.Int("max_idle_conns", 10),
		slog.Duration("conn_max_lifetime", 5*time.Minute),
		slog.Duration("conn_max_idle_time", 1*time.Minute),
	)

	return db, nil
}

// logPoolStats logs current connection pool statistics
func logPoolStats(db *sql.DB) {
	stats := db.Stats()
	slog.Info("Database pool statistics",
		slog.Int("open_connections", stats.OpenConnections),
		slog.Int("in_use", stats.InUse),
		slog.Int("idle", stats.Idle),
		slog.Int64("wait_count", stats.WaitCount),
		slog.Duration("wait_duration", stats.WaitDuration),
		slog.Int64("max_idle_closed", stats.MaxIdleClosed),
		slog.Int64("max_lifetime_closed", stats.MaxLifetimeClosed),
	)
}

// runMigrations executes all pending migrations
func runMigrations(db *sql.DB) error {
	// Create migrate instance with postgres driver
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	// Create migrate instance pointing to migrations folder
	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations", // Path to your migrations folder
		"postgres",
		driver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	// Run UP migrations
	if err := m.Up(); err != nil {
		// ErrNoChange means no new migrations to apply - that's OK
		if err == migrate.ErrNoChange {
			log.Println("No new migrations to apply")
			return nil
		}
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

func checkMigrationStatus(db *sql.DB) (version uint, dirty bool, err error) {
    driver, err := postgres.WithInstance(db, &postgres.Config{})
    if err != nil {
        return 0, false, err
    }
    
    m, err := migrate.NewWithDatabaseInstance(
        "file://migrations",
        "postgres",
        driver,
    )
    if err != nil {
        return 0, false, err
    }
    
    version, dirty, err = m.Version()
    return version, dirty, err
}

// healthCheck returns a handler that checks database connectivity
func healthCheck(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Log pool stats on each health check
		logPoolStats(db)

		// Check database connectivity with a 5-second timeout
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		if err := db.PingContext(ctx); err != nil {
			slog.Error("Health check failed", slog.String("error", err.Error()))
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, `{"status":"unhealthy","error":"database unreachable"}`)
			return
		}

		// Get pool stats
		stats := db.Stats()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"healthy","open_connections":%d,"in_use":%d,"idle":%d}`,
			stats.OpenConnections, stats.InUse, stats.Idle)
	}
}

// loggingMiddleware wraps a handler with request timing
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s took %v", r.Method, r.URL.Path, time.Since(start))
	})
}
