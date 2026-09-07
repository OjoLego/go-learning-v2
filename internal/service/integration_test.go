package service

import (
	"testing"
	"time"

	"dime-api/internal/model"
	"dime-api/internal/repository"
)

// TestFullBudgetWorkflow is an INTEGRATION TEST that verifies
// BudgetService and TransactionService work correctly together.
//
// Why this is an integration test:
// - Uses real service implementations (not mocks)
// - Tests interaction between multiple components
// - Verifies the complete business workflow
// - Uses in-memory repos (fast, but real code paths)
//
// When to use integration tests vs unit tests:
// - Integration tests: Verify components work together (like this one)
// - Unit tests: Test single component in isolation (like our earlier tests)
func TestFullBudgetWorkflow(t *testing.T) {
	// Setup: Create real repositories and services
	// We're using the actual implementations, not mocks!
	budgetRepo := repository.NewInMemoryBudgetRepo()
	budgetSvc := NewBudgetService(budgetRepo)

	// TransactionService needs both transaction repo AND budget service
	// This is the key integration point!
	transactionRepo := repository.NewInMemoryTransactionRepo()
	transactionSvc := NewTransactionService(transactionRepo, budgetSvc)

	// Test scenario: User sets a budget, makes purchases, checks status
	userID := "user123"
	category := "groceries"

	// Step 1: Create a budget with $100 limit
	t.Run("create budget", func(t *testing.T) {
		budget := model.Budget{
			ID:       "budget-001",
			UserID:   userID,
			Category: category,
			Limit:    100.0,
			Spent:    0,
		}

		created, err := budgetSvc.CreateBudget(budget)
		if err != nil {
			t.Fatalf("failed to create budget: %v", err)
		}

		if created.Limit != 100.0 {
			t.Errorf("expected limit 100.0, got %.2f", created.Limit)
		}
	})

	// Step 2: Record a $60 expense
	// This should update the budget's spent amount in the background (async goroutine)
	t.Run("record first expense", func(t *testing.T) {
		tx, err := transactionSvc.RecordTransaction(userID, model.TypeExpense, 60.0, category)
		if err != nil {
			t.Fatalf("failed to record transaction: %v", err)
		}

		if tx.Amount != 60.0 {
			t.Errorf("expected amount 60.0, got %.2f", tx.Amount)
		}
		if tx.Type != model.TypeExpense {
			t.Errorf("expected type expense, got %s", tx.Type)
		}

		// Wait for the async goroutine to update the budget
		// In production this is fine, but tests need to wait
		time.Sleep(50 * time.Millisecond)
	})

	// Step 3: Check budget status (should be "within budget" - 60% spent)
	t.Run("check status after first expense", func(t *testing.T) {
		status, err := budgetSvc.CheckBudgetStatus(userID, category)
		if err != nil {
			t.Fatalf("failed to check budget status: %v", err)
		}

		// 60/100 = 60%, which is < 80%, so should be "within budget"
		if status != "within budget" {
			t.Errorf("expected 'within budget', got %q", status)
		}
	})

	// Step 4: Record another $30 expense (total spent: $90)
	t.Run("record second expense", func(t *testing.T) {
		tx, err := transactionSvc.RecordTransaction(userID, model.TypeExpense, 30.0, category)
		if err != nil {
			t.Fatalf("failed to record transaction: %v", err)
		}

		if tx.Amount != 30.0 {
			t.Errorf("expected amount 30.0, got %.2f", tx.Amount)
		}

		// Wait for async goroutine
		time.Sleep(50 * time.Millisecond)
	})

	// Step 5: Check budget status (should be "near limit" - 90% spent)
	t.Run("check status after second expense", func(t *testing.T) {
		status, err := budgetSvc.CheckBudgetStatus(userID, category)
		if err != nil {
			t.Fatalf("failed to check budget status: %v", err)
		}

		// 90/100 = 90%, which is > 80%, so should be "near limit"
		if status != "near limit" {
			t.Errorf("expected 'near limit', got %q", status)
		}
	})

	// Step 6: Record final $15 expense (total spent: $105, over budget!)
	t.Run("record third expense that exceeds budget", func(t *testing.T) {
		tx, err := transactionSvc.RecordTransaction(userID, model.TypeExpense, 15.0, category)
		if err != nil {
			t.Fatalf("failed to record transaction: %v", err)
		}

		if tx.Amount != 15.0 {
			t.Errorf("expected amount 15.0, got %.2f", tx.Amount)
		}

		// Wait for async goroutine
		time.Sleep(50 * time.Millisecond)
	})

	// Step 7: Check budget status (should be "over budget" - 105% spent)
	t.Run("check status after going over budget", func(t *testing.T) {
		status, err := budgetSvc.CheckBudgetStatus(userID, category)
		if err != nil {
			t.Fatalf("failed to check budget status: %v", err)
		}

		// 105/100 = 105%, which is >= 100%, so should be "over budget"
		if status != "over budget" {
			t.Errorf("expected 'over budget', got %q", status)
		}
	})
}

// TestIncomeDoesNotAffectBudget verifies that income transactions
// do NOT update the budget spent amount.
func TestIncomeDoesNotAffectBudget(t *testing.T) {
	// Setup
	budgetRepo := repository.NewInMemoryBudgetRepo()
	budgetSvc := NewBudgetService(budgetRepo)
	transactionRepo := repository.NewInMemoryTransactionRepo()
	transactionSvc := NewTransactionService(transactionRepo, budgetSvc)

	userID := "user456"
	category := "salary"

	// Create budget
	budget := model.Budget{
		ID:       "budget-002",
		UserID:   userID,
		Category: category,
		Limit:    1000.0,
		Spent:    0,
	}
	budgetSvc.CreateBudget(budget)

	// Record an income transaction
	_, err := transactionSvc.RecordTransaction(userID, model.TypeIncome, 5000.0, category)
	if err != nil {
		t.Fatalf("failed to record income: %v", err)
	}

	// Check budget - should still be "within budget" (0% spent)
	// because income doesn't count against the budget
	status, err := budgetSvc.CheckBudgetStatus(userID, category)
	if err != nil {
		t.Fatalf("failed to check budget status: %v", err)
	}

	if status != "within budget" {
		t.Errorf("income should not affect budget; expected 'within budget', got %q", status)
	}

	// Verify spent is still 0 by checking the budget directly
	b, _ := budgetRepo.GetByUserAndCategory(userID, category)
	if b.Spent != 0 {
		t.Errorf("income incorrectly updated budget spent; expected 0, got %.2f", b.Spent)
	}
}

// TestMultipleCategoriesIndependent verifies that budgets for different
// categories are tracked independently.
func TestMultipleCategoriesIndependent(t *testing.T) {
	// Setup
	budgetRepo := repository.NewInMemoryBudgetRepo()
	budgetSvc := NewBudgetService(budgetRepo)
	transactionRepo := repository.NewInMemoryTransactionRepo()
	transactionSvc := NewTransactionService(transactionRepo, budgetSvc)

	userID := "user789"

	// Create two budgets for different categories
	budgetSvc.CreateBudget(model.Budget{
		ID:       "budget-food",
		UserID:   userID,
		Category: "food",
		Limit:    200.0,
		Spent:    0,
	})

	budgetSvc.CreateBudget(model.Budget{
		ID:       "budget-entertainment",
		UserID:   userID,
		Category: "entertainment",
		Limit:    100.0,
		Spent:    0,
	})

	// Spend on food only
	transactionSvc.RecordTransaction(userID, model.TypeExpense, 150.0, "food")

	// Wait for async goroutine to update budget
	time.Sleep(50 * time.Millisecond)

	// Check food budget (should be 75% - within budget)
	foodStatus, _ := budgetSvc.CheckBudgetStatus(userID, "food")
	if foodStatus != "within budget" {
		t.Errorf("food budget should be 'within budget', got %q", foodStatus)
	}

	// Check entertainment budget (should still be 0% - within budget)
	entStatus, _ := budgetSvc.CheckBudgetStatus(userID, "entertainment")
	if entStatus != "within budget" {
		t.Errorf("entertainment budget should be 'within budget', got %q", entStatus)
	}

	// Verify the actual spent amounts
	foodBudget, _ := budgetRepo.GetByUserAndCategory(userID, "food")
	entBudget, _ := budgetRepo.GetByUserAndCategory(userID, "entertainment")

	if foodBudget.Spent != 150.0 {
		t.Errorf("food spent should be 150.0, got %.2f", foodBudget.Spent)
	}
	if entBudget.Spent != 0 {
		t.Errorf("entertainment spent should be 0, got %.2f", entBudget.Spent)
	}
}

// TestBudgetNotFoundScenario verifies behavior when no budget exists
// for a given category.
func TestBudgetNotFoundScenario(t *testing.T) {
	// Setup
	budgetRepo := repository.NewInMemoryBudgetRepo()
	budgetSvc := NewBudgetService(budgetRepo)
	transactionRepo := repository.NewInMemoryTransactionRepo()
	transactionSvc := NewTransactionService(transactionRepo, budgetSvc)

	userID := "user999"

	// Don't create a budget - just record an expense
	// This should NOT fail - the expense is recorded, but budget check is skipped
	_, err := transactionSvc.RecordTransaction(userID, model.TypeExpense, 50.0, "nonexistent-category")
	if err != nil {
		t.Fatalf("recording expense without budget should not fail: %v", err)
	}

	// But checking budget status should fail
	_, err = budgetSvc.CheckBudgetStatus(userID, "nonexistent-category")
	if err == nil {
		t.Error("expected error when checking status of non-existent budget")
	}
}
