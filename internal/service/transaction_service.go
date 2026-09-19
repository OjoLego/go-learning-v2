package service

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"dime-api/internal/database"
	"dime-api/internal/idgen"
	"dime-api/internal/model"
	"dime-api/internal/repository"
)

type TransactionService struct {
	repo          repository.TransactionRepository // depends on the INTERFACE, not a concrete type
	budgetService *BudgetService                   // used for the async budget check below
	txManager     *database.TransactionManager     // for atomic operations
}

func NewTransactionService(
	repo repository.TransactionRepository,
	budgetService *BudgetService,
	txManager *database.TransactionManager,
) *TransactionService {
	return &TransactionService{
		repo:          repo,
		budgetService: budgetService,
		txManager:     txManager,
	}
}

func (s *TransactionService) RecordTransaction(ctx context.Context, userID string, txType model.TransactionType, amount float64, category string) (model.Transaction, error) {
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

	saved, err := s.repo.Create(ctx, t)
	if err != nil {
		return model.Transaction{}, err
	}

	// Bonus: for expenses, update the budget and check status in the
	// background so the HTTP response doesn't wait on this extra work.
	if txType == model.TypeExpense {
		go s.checkBudgetAfterExpense(ctx, userID, category, amount)
	}

	return saved, nil
}

// checkBudgetAfterExpense runs in its own goroutine (fire-and-forget).
// It never returns an error to the caller — since nothing is waiting on
// it, the only thing it can do with a problem is log it.
func (s *TransactionService) checkBudgetAfterExpense(ctx context.Context, userID, category string, amount float64) {
	if s.budgetService == nil {
		return
	}

	if err := s.budgetService.RecordSpend(ctx, userID, category, amount); err != nil {
		// No budget set for this category is a normal, expected case — not worth logging as a warning.
		return
	}

	status, err := s.budgetService.CheckBudgetStatus(ctx, userID, category)
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

func (s *TransactionService) GetTransaction(ctx context.Context, id string) (model.Transaction, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *TransactionService) GetUserTransactions(ctx context.Context, userID string) ([]model.Transaction, error) {
	return s.repo.ListByUser(ctx, userID)
}

// RecordTransactionAtomic creates a transaction and updates the budget atomically.
// If either operation fails, both are rolled back.
// This demonstrates transaction management for operations that must succeed or fail together.
func (s *TransactionService) RecordTransactionAtomic(
	ctx context.Context,
	userID string,
	txType model.TransactionType,
	amount float64,
	category string,
) (model.Transaction, error) {
	if amount <= 0 {
		return model.Transaction{}, fmt.Errorf("amount must be positive, got %.2f", amount)
	}

	// Create the transaction object
	t := model.Transaction{
		ID:        idgen.New(),
		UserID:    userID,
		Type:      txType,
		Amount:    amount,
		Category:  category,
		CreatedAt: time.Now(),
	}

	// Execute both operations in a transaction
	var saved model.Transaction
	err := s.txManager.RunInTransaction(ctx, func(tx *sql.Tx) error {
		// Get concrete repository to use transaction methods
		txRepo := s.repo.(*repository.PostgresTransactionRepo)

		// Step 1: Create the transaction
		created, err := txRepo.CreateTx(ctx, tx, t)
		if err != nil {
			return fmt.Errorf("failed to create transaction: %w", err)
		}
		saved = created

		// Step 2: If it's an expense, update the budget atomically
		if txType == model.TypeExpense {
			// Get concrete budget repository
			budgetRepo := s.budgetService.Repo.(*repository.PostgresBudgetRepo)

			// Try to update the budget - will fail if budget doesn't exist
			_, err := budgetRepo.AddSpentTx(ctx, tx, userID, category, amount)
			if err != nil {
				// If budget not found, we can choose to:
				// 1. Fail the whole transaction (strict mode)
				// 2. Continue without budget update (lenient mode)
				// For this implementation, we'll be lenient and not fail
				// the transaction if there's no budget set
				if err != repository.ErrBudgetNotFound {
					return fmt.Errorf("failed to update budget: %w", err)
				}
				// Budget not found is OK - just log it
				log.Printf("No budget found for user=%s category=%s, transaction created without budget update",
					userID, category)
			}
		}

		return nil
	})

	if err != nil {
		return model.Transaction{}, fmt.Errorf("atomic transaction failed: %w", err)
	}

	return saved, nil
}
