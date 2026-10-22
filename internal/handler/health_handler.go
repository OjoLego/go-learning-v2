package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"
)

type HealthHandler struct {
    db *sql.DB
}

func NewHealthHandler(db *sql.DB) *HealthHandler {
    return &HealthHandler{db: db}
}

func (h *HealthHandler) Check(w http.ResponseWriter, r *http.Request) {
    ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
    defer cancel()

    status := "healthy"
    code := http.StatusOK

    if err := h.db.PingContext(ctx); err != nil {
        status = "unhealthy"
        code = http.StatusServiceUnavailable
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(code)
    json.NewEncoder(w).Encode(map[string]any{
        "status": status,
        "time":   time.Now().UTC().Format(time.RFC3339),
    })
}