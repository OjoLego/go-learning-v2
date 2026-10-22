package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"dime-api/internal/auth"
)

type authContextKey string

const userIDKey authContextKey = "userID"

// AuthMiddleware validates JWT and injects userID into context.
func AuthMiddleware(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				RespondAuthError(w)
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				RespondAuthError(w)
				return
			}

			claims, err := auth.ValidateToken(parts[1], secret)
			if err != nil {
				RespondAuthError(w)
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, claims.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UserIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(userIDKey).(string)
	return id, ok
}

// WithUserID returns a context with the given user ID injected.
// Useful for tests that call handlers directly without routing through AuthMiddleware.
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

func RespondAuthError(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{
			"code":    "UNAUTHORIZED",
			"message": "missing or invalid authorization token",
		},
	})
}
