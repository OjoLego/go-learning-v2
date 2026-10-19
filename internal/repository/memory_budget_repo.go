package repository

import (
	"context"
	"sync"

	"dime-api/internal/model"
)

// MemoryBudgetRepo is an in-memory implementation of BudgetRepository for testing
type MemoryBudgetRepo struct {
	mu      sync.RWMutex
	budgets map[string]model.Budget // key: "userID|category"
}

// NewMemoryBudgetRepo creates a new in-memory budget repository
func NewMemoryBudgetRepo() *MemoryBudgetRepo {
	return &MemoryBudgetRepo{
		budgets: make(map[string]model.Budget),
	}
}

func (m *MemoryBudgetRepo) makeKey(userID, category string) string {
	return userID + "|" + category
}

// Create implements BudgetRepository
func (m *MemoryBudgetRepo) Create(ctx context.Context, b model.Budget) (model.Budget, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := m.makeKey(b.UserID, b.Category)
	m.budgets[key] = b
	return b, nil
}

// GetByUserAndCategory implements BudgetRepository
func (m *MemoryBudgetRepo) GetByUserAndCategory(ctx context.Context, userID, category string) (model.Budget, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := m.makeKey(userID, category)
	b, ok := m.budgets[key]
	if !ok {
		return model.Budget{}, ErrBudgetNotFound
	}
	return b, nil
}

// AddSpent implements BudgetRepository
func (m *MemoryBudgetRepo) AddSpent(ctx context.Context, userID, category string, amount float64) (model.Budget, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := m.makeKey(userID, category)
	b, ok := m.budgets[key]
	if !ok {
		return model.Budget{}, ErrBudgetNotFound
	}
	b.Spent += amount
	m.budgets[key] = b
	return b, nil
}
