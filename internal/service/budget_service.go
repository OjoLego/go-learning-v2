package service

import (
	"context"
	"errors"
	"fmt"

	"dime-api/internal/model"
	"dime-api/internal/repository"
)

type BudgetService struct {
	Repo repository.BudgetRepository // Exported for transaction support
}

func NewBudgetService(repo repository.BudgetRepository) *BudgetService {
	return &BudgetService{Repo: repo}
}

func (s *BudgetService) CreateBudget(ctx context.Context, b model.Budget) (model.Budget, error) {
	return s.Repo.Create(ctx, b)
}

// CheckBudgetStatus returns "within budget", "near limit" (>80% spent),
// or "over budget" for a given user + category.
func (s *BudgetService) CheckBudgetStatus(ctx context.Context, userID, category string) (string, error) {
	b, err := s.Repo.GetByUserAndCategory(ctx, userID, category)
	if err != nil {
		if errors.Is(err, repository.ErrBudgetNotFound) {
			return "", fmt.Errorf("no budget set for category %q: %w", category, err)
		}
		return "", err
	}

	if b.Limit <= 0 {
		return "", fmt.Errorf("invalid budget limit for category %q", category)
	}

	ratio := b.Spent / b.Limit

	switch {
	case ratio >= 1.0:
		return "over budget", nil
	case ratio > 0.8:
		return "near limit", nil
	default:
		return "within budget", nil
	}
}

// RecordSpend adds to a budget's spent total — called when an expense
// transaction is recorded against that category.
func (s *BudgetService) RecordSpend(ctx context.Context, userID, category string, amount float64) error {
	_, err := s.Repo.AddSpent(ctx, userID, category, amount)
	return err
}
