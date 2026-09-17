package repository

import (
	"errors"

	"dime-api/internal/model"
)

// Common errors
var (
	ErrNotFound       = errors.New("transaction not found")
	ErrBudgetNotFound = errors.New("budget not found")
)

// BudgetRepository defines the interface for budget data access
type BudgetRepository interface {
	Create(b model.Budget) (model.Budget, error)
	GetByUserAndCategory(userID, category string) (model.Budget, error)
	AddSpent(userID, category string, amount float64) (model.Budget, error)
}

// TransactionRepository defines the interface for transaction data access
type TransactionRepository interface {
	Create(t model.Transaction) (model.Transaction, error)
	GetByID(id string) (model.Transaction, error)
	ListByUser(userID string) ([]model.Transaction, error)
}
