package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"dime-api/internal/model"
	"dime-api/internal/repository"
	"dime-api/internal/service"
)

type TransactionHandler struct {
	service *service.TransactionService
}

func NewTransactionHandler(s *service.TransactionService) *TransactionHandler {
	return &TransactionHandler{service: s}
}

type createTransactionRequest struct {
	UserID   string  `json:"user_id"`
	Type     string  `json:"type"`
	Amount   float64 `json:"amount"`
	Category string  `json:"category"`
}

func (h *TransactionHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	t, err := h.service.RecordTransaction(req.UserID, model.TransactionType(req.Type), req.Amount, req.Category)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, t)
}

func (h *TransactionHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id") // Go 1.22+ path parameter extraction

	t, err := h.service.GetTransaction(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			respondError(w, http.StatusNotFound, "transaction not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	respondJSON(w, http.StatusOK, t)
}

func (h *TransactionHandler) ListByUser(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		respondError(w, http.StatusBadRequest, "user_id query param is required")
		return
	}

	txns, err := h.service.GetUserTransactions(userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	respondJSON(w, http.StatusOK, txns)
}

// --- small helpers, shared across all handlers in this package ---

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}
