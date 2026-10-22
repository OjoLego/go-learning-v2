package handler

import (
	"dime-api/internal/middleware"
	"dime-api/internal/repository"
	"dime-api/internal/service"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
)

var ErrForbidden = errors.New("forbidden")
var ErrUnauthorized = errors.New("unauthorized")

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type errorResponse struct {
	Error APIError `json:"error"`
}

// Common error codes
const (
	ErrCodeInvalidRequest = "INVALID_REQUEST"
	ErrCodeNotFound       = "NOT_FOUND"
	ErrCodeUnauthorized   = "UNAUTHORIZED"
	ErrCodeForbidden      = "FORBIDDEN"
	ErrCodeRateLimited    = "RATE_LIMITED"
	ErrCodeInternal       = "INTERNAL_ERROR"
)

func RespondError(w http.ResponseWriter, r *http.Request, err error) {
	requestID := middleware.RequestIDFromContext(r.Context())

	var status int
	var code string
	var message string

	switch {
	case errors.Is(err, repository.ErrNotFound) || errors.Is(err, repository.ErrBudgetNotFound):
		status = http.StatusNotFound
		code = ErrCodeNotFound
		message = "resource not found"
	case errors.Is(err, repository.ErrInvalidInput),
		errors.Is(err, service.ErrInvalidAmount),
		errors.Is(err, service.ErrInvalidLimit):
		status = http.StatusBadRequest
		code = ErrCodeInvalidRequest
		message = err.Error()
	case errors.Is(err, ErrForbidden):
		status = http.StatusForbidden
		code = ErrCodeForbidden
		message = "forbidden"
	case errors.Is(err, ErrUnauthorized):
		status = http.StatusUnauthorized
		code = ErrCodeUnauthorized
		message = "unauthorized"
	default:
		status = http.StatusInternalServerError
		code = ErrCodeInternal
		message = "internal server error"
		// Log the real error with request ID
		slog.Error("internal_error",
			slog.String("request_id", requestID),
			slog.String("error", err.Error()),
		)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(errorResponse{Error: APIError{Code: code, Message: message}})
}
