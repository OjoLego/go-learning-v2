package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"dime-api/internal/handler"
	"dime-api/internal/repository"
	"dime-api/internal/service"
	"dime-api/internal/swagger"
)

func main() {
	// --- Dependency wiring — bottom-up, like constructing a FastAPI dependency graph ---
	budgetRepo := repository.NewInMemoryBudgetRepo()
	budgetSvc := service.NewBudgetService(budgetRepo)
	budgetHandler := handler.NewBudgetHandler(budgetSvc)

	txRepo := repository.NewInMemoryTransactionRepo()
	txSvc := service.NewTransactionService(txRepo, budgetSvc) // budgetSvc injected for the async check
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

// loggingMiddleware wraps a handler with request timing — plain function
// composition, no decorators or annotations needed.
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s took %v", r.Method, r.URL.Path, time.Since(start))
	})
}
