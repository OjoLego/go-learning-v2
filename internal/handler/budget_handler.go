package handler

import (
	"encoding/json"
	"net/http"

	"dime-api/internal/idgen"
	"dime-api/internal/model"
	"dime-api/internal/service"
)

type BudgetHandler struct {
	service *service.BudgetService
}

func NewBudgetHandler(s *service.BudgetService) *BudgetHandler {
	return &BudgetHandler{service: s}
}

type createBudgetRequest struct {
	UserID   string  `json:"user_id"`
	Category string  `json:"category"`
	Limit    float64 `json:"limit"`
}

// Create lets you set up a budget so /budgets/status has something to check.
// (Not part of the original exercise spec, but needed to exercise it end-to-end.)
func (h *BudgetHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createBudgetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Limit <= 0 {
		respondError(w, http.StatusBadRequest, "limit must be positive")
		return
	}

	b := model.Budget{
		ID:       idgen.New(),
		UserID:   req.UserID,
		Category: req.Category,
		Limit:    req.Limit,
		Spent:    0,
	}

	created, err := h.service.CreateBudget(b)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	respondJSON(w, http.StatusCreated, created)
}

// Status handles GET /budgets/status?user_id=X&category=Y (step 4 of the exercise).
func (h *BudgetHandler) Status(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	category := r.URL.Query().Get("category")

	if userID == "" || category == "" {
		respondError(w, http.StatusBadRequest, "user_id and category query params are required")
		return
	}

	status, err := h.service.CheckBudgetStatus(userID, category)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"user_id":  userID,
		"category": category,
		"status":   status,
	})
}
