package service

import (
	"fmt"
	"log"
	"time"

	"dime-api/internal/idgen"
	"dime-api/internal/model"
	"dime-api/internal/repository"
)

type TransactionService struct {
	repo          repository.TransactionRepository // depends on the INTERFACE, not a concrete type
	budgetService *BudgetService                   // used for the async budget check below
}

func NewTransactionService(repo repository.TransactionRepository, budgetService *BudgetService) *TransactionService {
	return &TransactionService{repo: repo, budgetService: budgetService}
}

func (s *TransactionService) RecordTransaction(userID string, txType model.TransactionType, amount float64, category string) (model.Transaction, error) {
	if amount <= 0 {
		return model.Transaction{}, fmt.Errorf("amount must be positive, got %.2f", amount)
	}

	t := model.Transaction{
		ID:        idgen.New(),
		UserID:    userID,
		Type:      txType,
		Amount:    amount,
		Category:  category,
		CreatedAt: time.Now(),
	}

	saved, err := s.repo.Create(t)
	if err != nil {
		return model.Transaction{}, err
	}

	// Bonus: for expenses, update the budget and check status in the
	// background so the HTTP response doesn't wait on this extra work.
	if txType == model.TypeExpense {
		go s.checkBudgetAfterExpense(userID, category, amount)
	}

	return saved, nil
}

// checkBudgetAfterExpense runs in its own goroutine (fire-and-forget).
// It never returns an error to the caller — since nothing is waiting on
// it, the only thing it can do with a problem is log it.
func (s *TransactionService) checkBudgetAfterExpense(userID, category string, amount float64) {
	if s.budgetService == nil {
		return
	}

	if err := s.budgetService.RecordSpend(userID, category, amount); err != nil {
		// No budget set for this category is a normal, expected case — not worth logging as a warning.
		return
	}

	status, err := s.budgetService.CheckBudgetStatus(userID, category)
	if err != nil {
		log.Printf("budget check failed for user=%s category=%s: %v", userID, category, err)
		return
	}

	switch status {
	case "over budget":
		log.Printf("⚠️  WARNING: user=%s is over budget for category=%s", userID, category)
	case "near limit":
		log.Printf("user=%s is near their budget limit for category=%s", userID, category)
	}
}

func (s *TransactionService) GetTransaction(id string) (model.Transaction, error) {
	return s.repo.GetByID(id)
}

func (s *TransactionService) GetUserTransactions(userID string) ([]model.Transaction, error) {
	return s.repo.ListByUser(userID)
}
