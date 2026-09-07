package service

import (
	"testing"

	"dime-api/internal/model"
	"dime-api/internal/repository"
)

// mockBudgetRepo is a test double that implements BudgetRepository
type mockBudgetRepo struct {
	budgets map[string]model.Budget // key: "userID|category"
}

func newMockBudgetRepo() *mockBudgetRepo {
	return &mockBudgetRepo{
		budgets: make(map[string]model.Budget),
	}
}

func (m *mockBudgetRepo) Create(b model.Budget) (model.Budget, error) {
	key := b.UserID + "|" + b.Category
	m.budgets[key] = b
	return b, nil
}

func (m *mockBudgetRepo) GetByUserAndCategory(userID, category string) (model.Budget, error) {
	key := userID + "|" + category
	b, ok := m.budgets[key]
	if !ok {
		return model.Budget{}, repository.ErrBudgetNotFound
	}
	return b, nil
}

func (m *mockBudgetRepo) AddSpent(userID, category string, amount float64) (model.Budget, error) {
	key := userID + "|" + category
	b, ok := m.budgets[key]
	if !ok {
		return model.Budget{}, repository.ErrBudgetNotFound
	}
	b.Spent += amount
	m.budgets[key] = b
	return b, nil
}

func TestCheckBudgetStatus(t *testing.T) {
	// Create mock repository and service
	repo := newMockBudgetRepo()
	svc := NewBudgetService(repo)

	// Setup: Create budgets with different spent amounts
	repo.Create(model.Budget{UserID: "user1", Category: "food", Limit: 100, Spent: 50})   // 50% spent
	repo.Create(model.Budget{UserID: "user1", Category: "rent", Limit: 100, Spent: 85})   // 85% spent - near limit
	repo.Create(model.Budget{UserID: "user1", Category: "fun", Limit: 100, Spent: 100})   // 100% spent - over budget
	repo.Create(model.Budget{UserID: "user1", Category: "bad", Limit: 100, Spent: 150})   // 150% spent - over budget
	repo.Create(model.Budget{UserID: "user1", Category: "zero", Limit: 0, Spent: 10})     // Zero limit - error case

	// Table-driven tests
	tests := []struct {
		name     string
		userID   string
		category string
		expected string
		wantErr  bool
	}{
		{"within budget at 50%", "user1", "food", "within budget", false},
		{"near limit at 85%", "user1", "rent", "near limit", false},
		{"exactly at limit", "user1", "fun", "over budget", false},
		{"over budget at 150%", "user1", "bad", "over budget", false},
		{"budget not found", "user1", "missing", "", true},
		{"zero limit error", "user1", "zero", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := svc.CheckBudgetStatus(tt.userID, tt.category)

			// Check error expectation
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckBudgetStatus() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Check result (only if we don't expect an error)
			if !tt.wantErr && got != tt.expected {
				t.Errorf("CheckBudgetStatus() = %v, want %v", got, tt.expected)
			}
		})
	}
}