package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"dime-api/internal/handler"
	"dime-api/internal/repository"
	"dime-api/internal/service"
	"dime-api/internal/swagger"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

func main() {
	// Connect to PostgreSQL database
	db, err := connectDB()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()
	log.Println("Connected to PostgreSQL")

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

	// --- Dependency wiring ---
	budgetRepo := repository.NewPostgresBudgetRepo(db)
	budgetSvc := service.NewBudgetService(budgetRepo)
	budgetHandler := handler.NewBudgetHandler(budgetSvc)

	txRepo := repository.NewPostgresTransactionRepo(db)
	txSvc := service.NewTransactionService(txRepo, budgetSvc)
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

	handlerWithMiddleware := loggingMiddleware(mux)

	fmt.Println("Server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", handlerWithMiddleware))
}

// connectDB establishes connection to PostgreSQL
func connectDB() (*sql.DB, error) {
	// Get connection details from environment or use defaults
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "dime_user")
	password := getEnv("DB_PASSWORD", "dime_password")
	dbname := getEnv("DB_NAME", "dime")

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
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

// getEnv retrieves environment variable or returns default
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// loggingMiddleware wraps a handler with request timing
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s took %v", r.Method, r.URL.Path, time.Since(start))
	})
}
