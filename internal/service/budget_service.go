package service

import (
	"errors"
	"fmt"

	"dime-api/internal/model"
	"dime-api/internal/repository"
)

type BudgetService struct {
	repo repository.BudgetRepository
}

func NewBudgetService(repo repository.BudgetRepository) *BudgetService {
	return &BudgetService{repo: repo}
}

func (s *BudgetService) CreateBudget(b model.Budget) (model.Budget, error) {
	return s.repo.Create(b)
}

// CheckBudgetStatus returns "within budget", "near limit" (>80% spent),
// or "over budget" for a given user + category.
func (s *BudgetService) CheckBudgetStatus(userID, category string) (string, error) {
	b, err := s.repo.GetByUserAndCategory(userID, category)
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
func (s *BudgetService) RecordSpend(userID, category string, amount float64) error {
	_, err := s.repo.AddSpent(userID, category, amount)
	return err
}
