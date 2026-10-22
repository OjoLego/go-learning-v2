package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"dime-api/internal/middleware"
	"dime-api/internal/model"
	"dime-api/internal/repository"
	"dime-api/internal/service"
	"dime-api/internal/validator"
)

// dbTimeout is the maximum duration for database operations
const dbTimeout = 5 * time.Second

type TransactionHandler struct {
	service *service.TransactionService
}

func NewTransactionHandler(s *service.TransactionService) *TransactionHandler {
	return &TransactionHandler{service: s}
}

type createTransactionRequest struct {
	UserID   string  `json:"user_id" validate:"required"`
	Type     string  `json:"type" validate:"required,oneof=income expense"`
	Amount   float64 `json:"amount" validate:"required,gt=0"`
	Category string  `json:"category" validate:"required,min=1,max=50"`
}

func (h *TransactionHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createTransactionRequest
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

	t, err := h.service.RecordTransaction(ctx, req.UserID, model.TransactionType(req.Type), req.Amount, req.Category)
	if err != nil {
		RespondError(w, r, err)
		return
	}

	respondJSON(w, http.StatusCreated, t)
}

func (h *TransactionHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id") // Go 1.22+ path parameter extraction

	// Create a context with timeout for the database operation
	ctx, cancel := context.WithTimeout(r.Context(), dbTimeout)
	defer cancel()

	t, err := h.service.GetTransaction(ctx, id)
	if err != nil {
		RespondError(w, r, err)
		return
	}

	respondJSON(w, http.StatusOK, t)
}

// parsePagination extracts and validates pagination parameters from URL
// Example: ?page=2&page_size=10 → page=2, pageSize=10
func parsePagination(r *http.Request) model.PaginationRequest {
	// Parse page parameter (default: 1)
	pageStr := r.URL.Query().Get("page")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	// Parse page_size parameter (default: 20)
	pageSizeStr := r.URL.Query().Get("page_size")
	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 {
		pageSize = model.DefaultPageSize
	}
	// Enforce maximum page size to prevent abuse
	if pageSize > model.MaxPageSize {
		pageSize = model.MaxPageSize
	}

	return model.NewPaginationRequest(page, pageSize)
}

func (h *TransactionHandler) ListByUser(w http.ResponseWriter, r *http.Request) {
	authUserID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		RespondError(w, r, ErrUnauthorized)
		return
	}

	requestedUserID := r.URL.Query().Get("user_id")
	if requestedUserID == "" {
		requestedUserID = authUserID
	}

	if requestedUserID != authUserID {
		RespondError(w, r, ErrForbidden)
		return
	}

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		RespondError(w, r, repository.ErrInvalidInput)
		return
	}

	// Parse pagination parameters from URL
	pagination := parsePagination(r)

	// Create a context with timeout for the database operation
	ctx, cancel := context.WithTimeout(r.Context(), dbTimeout)
	defer cancel()

	// Get paginated transactions
	transactions, totalCount, err := h.service.GetUserTransactionsPaginated(
		ctx,
		userID,
		pagination.Page,
		pagination.PageSize,
	)
	if err != nil {
		RespondError(w, r, err)
		return
	}

	// Build paginated response with metadata
	response := model.NewPaginatedResponse(
		transactions,
		pagination.Page,
		pagination.PageSize,
		totalCount,
	)

	respondJSON(w, http.StatusOK, response)
}

// --- small helpers, shared across all handlers in this package ---

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}
