package repository

import (
	"context"
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
	Create(ctx context.Context, b model.Budget) (model.Budget, error)
	GetByUserAndCategory(ctx context.Context, userID, category string) (model.Budget, error)
	AddSpent(ctx context.Context, userID, category string, amount float64) (model.Budget, error)
}

// TransactionRepository defines the interface for transaction data access
type TransactionRepository interface {
	Create(ctx context.Context, t model.Transaction) (model.Transaction, error)
	GetByID(ctx context.Context, id string) (model.Transaction, error)
	ListByUser(ctx context.Context, userID string) ([]model.Transaction, error)
}
