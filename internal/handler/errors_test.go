package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"dime-api/internal/repository"
)

func TestRespondError_MapsNotFound(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	RespondError(rec, req, repository.ErrNotFound)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestRespondError_MasksInternalError(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	RespondError(rec, req, repository.ErrNotFound) // or any generic error

	// For unknown errors, the body should not contain the raw error message.
	// Add an assertion that rec.Body.String() does NOT contain the secret error text.
}
