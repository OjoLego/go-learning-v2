package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"dime-api/internal/config"
	"dime-api/internal/database"
	"dime-api/internal/handler"
	"dime-api/internal/metrics"
	"dime-api/internal/middleware"
	"dime-api/internal/ratelimit"
	"dime-api/internal/repository"
	"dime-api/internal/service"
	"dime-api/internal/swagger"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
	"golang.org/x/time/rate"
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

	// Configure structured logging based on config
	level := new(slog.LevelVar)
	if err := level.UnmarshalText([]byte(cfg.LogLevel)); err != nil {
		level.Set(slog.LevelInfo)
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(logger)

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

	authHandler := handler.NewAuthHandler(cfg)
	rateLimiter := ratelimit.NewMemoryLimiter(rate.Limit(cfg.RateLimitRPS), cfg.RateLimitBurst)

	metricsCollector := metrics.New()
	healthHandler := handler.NewHealthHandler(db)

	// --- Routes ---

	// Public mux
	publicMux := http.NewServeMux()
	publicMux.HandleFunc("POST /auth/token", authHandler.Token)
	publicMux.HandleFunc("GET /health", healthHandler.Check)
	publicMux.HandleFunc("GET /swagger", swagger.UI)
	publicMux.HandleFunc("GET /openapi.yaml", swagger.OpenAPI)
	publicMux.HandleFunc("GET /metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(metricsCollector.Snapshot())
	})

	// Protected mux
	protectedMux := http.NewServeMux()
	protectedMux.HandleFunc("POST /transactions", txHandler.Create)
	protectedMux.HandleFunc("GET /transactions/{id}", txHandler.GetByID)
	protectedMux.HandleFunc("GET /transactions", txHandler.ListByUser)
	protectedMux.HandleFunc("POST /budgets", budgetHandler.Create)
	protectedMux.HandleFunc("GET /budgets/status", budgetHandler.Status)

	// Wrap protected routes with auth middleware
	protectedHandler := middleware.AuthMiddleware(cfg.JWTSecret)(protectedMux)

	// Combine
	mainMux := http.NewServeMux()
	mainMux.Handle("/", publicMux)
	mainMux.Handle("/transactions", protectedHandler)
	mainMux.Handle("/transactions/", protectedHandler)
	mainMux.Handle("/budgets", protectedHandler)
	mainMux.Handle("/budgets/", protectedHandler)

	// Add logging and request ID
	var handler http.Handler = mainMux
	handler = middleware.SecurityHeadersMiddleware(handler)
	handler = middleware.CORSMiddleware(cfg.CORSAllowedOrigins)(handler)
	handler = middleware.RateLimitMiddleware(rateLimiter, middleware.RateLimitKey)(handler)
	handler = middleware.StructuredLogger(metricsCollector)(handler)
	handler = middleware.RequestIDMiddleware(handler)

	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      handler,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	slog.Info("Server starting", slog.String("addr", srv.Addr))

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	sigCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	<-sigCtx.Done()
	slog.Info("Shutdown signal received, gracefully shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("graceful shutdown failed", slog.String("error", err.Error()))
		os.Exit(1)
	}

	slog.Info("Server stopped gracefully")

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
