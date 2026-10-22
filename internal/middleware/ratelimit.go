package middleware

import (
	"encoding/json"
	"net/http"
	"strings"

	"dime-api/internal/ratelimit"
)

func RateLimitMiddleware(limiter ratelimit.Limiter, keyFunc func(r *http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := keyFunc(r)
			if !limiter.Allow(key) {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", "60")
				w.WriteHeader(http.StatusTooManyRequests)
				json.NewEncoder(w).Encode(map[string]any{
					"error": map[string]any{
						"code":    "RATE_LIMITED",
						"message": "rate limit exceeded, please slow down",
					},
				})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func RateLimitKey(r *http.Request) string {
	if userID, ok := UserIDFromContext(r.Context()); ok {
		return "user:" + userID
	}
	// Fall back to IP for unauthenticated routes
	ip := r.RemoteAddr
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		ip = strings.Split(fwd, ",")[0]
	}
	return "ip:" + ip
}
