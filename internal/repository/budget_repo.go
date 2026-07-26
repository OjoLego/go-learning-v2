package repository

import (
	"errors"
	"sync"

	"dime-api/internal/model"
)

var ErrBudgetNotFound = errors.New("budget not found")

// BudgetRepository — same shape as TransactionRepository: an interface the
// service layer depends on, with a swappable in-memory implementation.
type BudgetRepository interface {
	Create(b model.Budget) (model.Budget, error)
	GetByUserAndCategory(userID, category string) (model.Budget, error)
	// AddSpent increments the Spent amount on a budget and returns the updated budget.
	AddSpent(userID, category string, amount float64) (model.Budget, error)
}

type InMemoryBudgetRepo struct {
	mu sync.Mutex
	// keyed by "userID|category" for quick lookup — a slice would need a
	// linear scan on every read, this keeps GetByUserAndCategory O(1)
	data map[string]model.Budget
}

func NewInMemoryBudgetRepo() *InMemoryBudgetRepo {
	return &InMemoryBudgetRepo{
		data: make(map[string]model.Budget),
	}
}

func budgetKey(userID, category string) string {
	return userID + "|" + category
}

func (r *InMemoryBudgetRepo) Create(b model.Budget) (model.Budget, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data[budgetKey(b.UserID, b.Category)] = b
	return b, nil
}

func (r *InMemoryBudgetRepo) GetByUserAndCategory(userID, category string) (model.Budget, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	b, ok := r.data[budgetKey(userID, category)]
	if !ok {
		return model.Budget{}, ErrBudgetNotFound
	}
	return b, nil
}

func (r *InMemoryBudgetRepo) AddSpent(userID, category string, amount float64) (model.Budget, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := budgetKey(userID, category)
	b, ok := r.data[key]
	if !ok {
		return model.Budget{}, ErrBudgetNotFound
	}

	b.Spent += amount
	r.data[key] = b
	return b, nil
}
