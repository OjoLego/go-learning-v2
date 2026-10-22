package middleware

import (
    "context"
    "net/http"

    "github.com/google/uuid"
)

type contextKey string

const RequestIDHeader = "X-Request-ID"
const requestIDKey contextKey = "requestID"

// RequestIDMiddleware adds a unique request ID to every request.
func RequestIDMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        requestID := r.Header.Get(RequestIDHeader)
        if requestID == "" {
            requestID = uuid.NewString()
        }

        ctx := context.WithValue(r.Context(), requestIDKey, requestID)
        w.Header().Set(RequestIDHeader, requestID)

        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// RequestIDFromContext extracts the request ID from context.
func RequestIDFromContext(ctx context.Context) string {
    id, _ := ctx.Value(requestIDKey).(string)
    return id
}