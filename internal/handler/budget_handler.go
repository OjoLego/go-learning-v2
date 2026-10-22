package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"dime-api/internal/idgen"
	"dime-api/internal/middleware"
	"dime-api/internal/model"
	"dime-api/internal/repository"
	"dime-api/internal/service"
	"dime-api/internal/validator"
)

type BudgetHandler struct {
	service *service.BudgetService
}

func NewBudgetHandler(s *service.BudgetService) *BudgetHandler {
	return &BudgetHandler{service: s}
}

type createBudgetRequest struct {
	UserID   string  `json:"user_id" validate:"required"`
	Category string  `json:"category" validate:"required,min=1,max=50"`
	Limit    float64 `json:"limit" validate:"required,gt=0"`
}

// Create lets you set up a budget so /budgets/status has something to check.
// (Not part of the original exercise spec, but needed to exercise it end-to-end.)
func (h *BudgetHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createBudgetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, r, err)
		return
	}

	// Validate
	if fieldErrors := validator.ValidateStruct(req); fieldErrors != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"code":    "VALIDATION_ERROR",
				"message": "request validation failed",
				"fields":  fieldErrors,
			},
		})
		return
	}

	// Authorize
	authUserID, ok := middleware.UserIDFromContext(r.Context())
	if !ok || authUserID != req.UserID {
		RespondError(w, r, ErrForbidden)
		return
	}

	// Create a context with timeout for the database operation
	ctx, cancel := context.WithTimeout(r.Context(), dbTimeout)
	defer cancel()

	created, err := h.service.CreateBudget(ctx, model.Budget{
		ID:       idgen.New(),
		UserID:   req.UserID,
		Category: req.Category,
		Limit:    req.Limit,
		Spent:    0,
	})
	if err != nil {
		RespondError(w, r, err)
		return
	}

	respondJSON(w, http.StatusCreated, created)
}

// Status handles GET /budgets/status?user_id=X&category=Y (step 4 of the exercise).
func (h *BudgetHandler) Status(w http.ResponseWriter, r *http.Request) {
	// Authenticate
	authUserID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		RespondError(w, r, ErrUnauthorized)
		return
	}

	userID := r.URL.Query().Get("user_id")
	category := r.URL.Query().Get("category")

	// Default to authenticated user if no user_id provided
	if userID == "" {
		userID = authUserID
	}

	// Authorize
	if userID != authUserID {
		RespondError(w, r, ErrForbidden)
		return
	}

	if category == "" {
		RespondError(w, r, repository.ErrInvalidInput)
		return
	}

	// Create a context with timeout for the database operation
	ctx, cancel := context.WithTimeout(r.Context(), dbTimeout)
	defer cancel()

	status, err := h.service.CheckBudgetStatus(ctx, userID, category)
	if err != nil {
		RespondError(w, r, err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"user_id":  userID,
		"category": category,
		"status":   status,
	})
}
