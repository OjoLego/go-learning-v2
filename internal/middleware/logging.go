package middleware

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"dime-api/internal/metrics"
)

// StructuredLogger logs every HTTP request with timing and metadata.
func StructuredLogger(m *metrics.Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			requestID := RequestIDFromContext(r.Context())

			wrapped := &responseRecorder{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(wrapped, r)

			m.Inc("requests_total")
			m.Inc("requests_" + strconv.Itoa(wrapped.statusCode))

			slog.Info("http_request",
				slog.String("request_id", requestID),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("remote_addr", r.RemoteAddr),
				slog.Int("status", wrapped.statusCode),
				slog.Duration("duration", time.Since(start)),
			)
		})
	}
}

type responseRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (rr *responseRecorder) WriteHeader(code int) {
	rr.statusCode = code
	rr.ResponseWriter.WriteHeader(code)
}
