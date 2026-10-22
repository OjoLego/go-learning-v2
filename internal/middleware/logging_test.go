package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"dime-api/internal/metrics"
)

func TestStructuredLogger_CapturesStatusCode(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})

	metricsCollector := metrics.New()
	handler := StructuredLogger(metricsCollector)(inner)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTeapot {
		t.Errorf("expected status %d, got %d", http.StatusTeapot, rec.Code)
	}
}
