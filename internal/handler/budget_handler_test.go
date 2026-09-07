package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"dime-api/internal/model"
	"dime-api/internal/repository"
	"dime-api/internal/service"
)

// mockBudgetRepoForHandler is a stub repository for handler tests
type mockBudgetRepoForHandler struct {
	data map[string]model.Budget
}

func newMockBudgetRepoForHandler() *mockBudgetRepoForHandler {
	return &mockBudgetRepoForHandler{
		data: make(map[string]model.Budget),
	}
}

func (m *mockBudgetRepoForHandler) Create(b model.Budget) (model.Budget, error) {
	key := b.UserID + "|" + b.Category
	m.data[key] = b
	return b, nil
}

func (m *mockBudgetRepoForHandler) GetByUserAndCategory(userID, category string) (model.Budget, error) {
	key := userID + "|" + category
	b, ok := m.data[key]
	if !ok {
		return model.Budget{}, repository.ErrBudgetNotFound
	}
	return b, nil
}

func (m *mockBudgetRepoForHandler) AddSpent(userID, category string, amount float64) (model.Budget, error) {
	key := userID + "|" + category
	b, ok := m.data[key]
	if !ok {
		return model.Budget{}, repository.ErrBudgetNotFound
	}
	b.Spent += amount
	m.data[key] = b
	return b, nil
}

func setupTestHandler() (*BudgetHandler, *mockBudgetRepoForHandler) {
	repo := newMockBudgetRepoForHandler()
	svc := service.NewBudgetService(repo)
	handler := NewBudgetHandler(svc)
	return handler, repo
}

func TestBudgetHandler_Status(t *testing.T) {
	handler, repo := setupTestHandler()

	// Setup test data
	repo.Create(model.Budget{UserID: "user1", Category: "food", Limit: 100, Spent: 50})

	tests := []struct {
		name           string
		url            string
		expectedStatus int
		checkBody      bool
		expectedBody   map[string]string
	}{
		{
			name:           "missing user_id param",
			url:            "/budgets/status?category=food",
			expectedStatus: http.StatusBadRequest,
			checkBody:      false,
		},
		{
			name:           "missing category param",
			url:            "/budgets/status?user_id=user1",
			expectedStatus: http.StatusBadRequest,
			checkBody:      false,
		},
		{
			name:           "budget not found",
			url:            "/budgets/status?user_id=user1&category=nonexistent",
			expectedStatus: http.StatusNotFound,
			checkBody:      false,
		},
		{
			name:           "valid request - within budget",
			url:            "/budgets/status?user_id=user1&category=food",
			expectedStatus: http.StatusOK,
			checkBody:      true,
			expectedBody: map[string]string{
				"user_id":  "user1",
				"category": "food",
				"status":   "within budget",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			
			// Create response recorder
			rec := httptest.NewRecorder()

			// Call the handler
			handler.Status(rec, req)

			// Check status code
			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			// Check response body if needed
			if tt.checkBody {
				var got map[string]string
				if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
					t.Fatalf("failed to parse response body: %v", err)
				}

				for key, want := range tt.expectedBody {
					if got[key] != want {
						t.Errorf("expected %s=%s, got %s", key, want, got[key])
					}
				}
			}
		})
	}
}