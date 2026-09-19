package repository

import (
	"context"
	"testing"

	"dime-api/internal/model"
	"dime-api/internal/testutil"
)

func TestPostgresBudgetRepo_Integration(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Setup test database
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)

	// Cleanup before and after tests
	testutil.CleanupTestDB(t, db)
	defer testutil.CleanupTestDB(t, db)

	repo := NewPostgresBudgetRepo(db)
	ctx := context.Background()

	t.Run("Create", func(t *testing.T) {
		budget := model.Budget{
			ID:       "test-budget-1",
			UserID:   "user-1",
			Category: "food",
			Limit:    500.00,
			Spent:    0,
		}

		created, err := repo.Create(ctx, budget)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		if created.ID != budget.ID {
			t.Errorf("Expected ID %s, got %s", budget.ID, created.ID)
		}
		if created.Limit != budget.Limit {
			t.Errorf("Expected Limit %f, got %f", budget.Limit, created.Limit)
		}
	})

	t.Run("Create_Upsert", func(t *testing.T) {
		// Create initial budget
		budget := model.Budget{
			ID:       "test-budget-2",
			UserID:   "user-2",
			Category: "transport",
			Limit:    200.00,
			Spent:    50.00,
		}

		_, err := repo.Create(ctx, budget)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		// Create again with same user/category - should update limit
		budget2 := model.Budget{
			ID:       "test-budget-2-new",
			UserID:   "user-2",
			Category: "transport",
			Limit:    300.00, // Updated limit
			Spent:    0,      // Should not update spent (0)
		}

		updated, err := repo.Create(ctx, budget2)
		if err != nil {
			t.Fatalf("Create (upsert) failed: %v", err)
		}

		if updated.Limit != 300.00 {
			t.Errorf("Expected updated Limit 300.00, got %f", updated.Limit)
		}
	})

	t.Run("GetByUserAndCategory", func(t *testing.T) {
		// Create a budget
		budget := model.Budget{
			ID:       "test-budget-3",
			UserID:   "user-3",
			Category: "entertainment",
			Limit:    100.00,
			Spent:    25.00,
		}

		_, err := repo.Create(ctx, budget)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		// Retrieve it
		retrieved, err := repo.GetByUserAndCategory(ctx, budget.UserID, budget.Category)
		if err != nil {
			t.Fatalf("GetByUserAndCategory failed: %v", err)
		}

		if retrieved.ID != budget.ID {
			t.Errorf("Expected ID %s, got %s", budget.ID, retrieved.ID)
		}
		if retrieved.Spent != budget.Spent {
			t.Errorf("Expected Spent %f, got %f", budget.Spent, retrieved.Spent)
		}
	})

	t.Run("GetByUserAndCategory_NotFound", func(t *testing.T) {
		_, err := repo.GetByUserAndCategory(ctx, "non-existent-user", "non-existent-category")
		if err != ErrBudgetNotFound {
			t.Errorf("Expected ErrBudgetNotFound, got %v", err)
		}
	})

	t.Run("AddSpent", func(t *testing.T) {
		// Create a budget
		budget := model.Budget{
			ID:       "test-budget-4",
			UserID:   "user-4",
			Category: "utilities",
			Limit:    150.00,
			Spent:    0,
		}

		_, err := repo.Create(ctx, budget)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		// Add spent amount
		updated, err := repo.AddSpent(ctx, budget.UserID, budget.Category, 45.00)
		if err != nil {
			t.Fatalf("AddSpent failed: %v", err)
		}

		if updated.Spent != 45.00 {
			t.Errorf("Expected Spent 45.00, got %f", updated.Spent)
		}

		// Add more spent
		updated2, err := repo.AddSpent(ctx, budget.UserID, budget.Category, 30.00)
		if err != nil {
			t.Fatalf("AddSpent failed: %v", err)
		}

		if updated2.Spent != 75.00 {
			t.Errorf("Expected Spent 75.00, got %f", updated2.Spent)
		}
	})

	t.Run("AddSpent_NotFound", func(t *testing.T) {
		_, err := repo.AddSpent(ctx, "non-existent-user", "non-existent-category", 50.00)
		if err != ErrBudgetNotFound {
			t.Errorf("Expected ErrBudgetNotFound, got %v", err)
		}
	})

	t.Run("CreateTx", func(t *testing.T) {
		budget := model.Budget{
			ID:       "test-budget-tx",
			UserID:   "user-tx",
			Category: "health",
			Limit:    1000.00,
			Spent:    0,
		}

		// Start a transaction
		dbTx, err := db.Begin()
		if err != nil {
			t.Fatalf("Failed to begin transaction: %v", err)
		}

		// Use concrete type to access Tx methods
		concreteRepo := repo.(*PostgresBudgetRepo)
		created, err := concreteRepo.CreateTx(ctx, dbTx, budget)
		if err != nil {
			dbTx.Rollback()
			t.Fatalf("CreateTx failed: %v", err)
		}

		// Commit
		if err := dbTx.Commit(); err != nil {
			t.Fatalf("Failed to commit: %v", err)
		}

		if created.ID != budget.ID {
			t.Errorf("Expected ID %s, got %s", budget.ID, created.ID)
		}
	})

	t.Run("AddSpentTx", func(t *testing.T) {
		// Create a budget first
		budget := model.Budget{
			ID:       "test-budget-tx2",
			UserID:   "user-tx2",
			Category: "shopping",
			Limit:    500.00,
			Spent:    100.00,
		}

		_, err := repo.Create(ctx, budget)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		// Add spent within a transaction
		dbTx, err := db.Begin()
		if err != nil {
			t.Fatalf("Failed to begin transaction: %v", err)
		}

		// Use concrete type to access Tx methods
		concreteRepo := repo.(*PostgresBudgetRepo)
		updated, err := concreteRepo.AddSpentTx(ctx, dbTx, budget.UserID, budget.Category, 50.00)
		if err != nil {
			dbTx.Rollback()
			t.Fatalf("AddSpentTx failed: %v", err)
		}

		// Commit
		if err := dbTx.Commit(); err != nil {
			t.Fatalf("Failed to commit: %v", err)
		}

		if updated.Spent != 150.00 {
			t.Errorf("Expected Spent 150.00, got %f", updated.Spent)
		}
	})
}

func TestPostgresBudgetRepo_TransactionRollback(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)
	testutil.CleanupTestDB(t, db)

	repo := NewPostgresBudgetRepo(db)
	ctx := context.Background()

	t.Run("Transaction_Rollback", func(t *testing.T) {
		budget := model.Budget{
			ID:       "test-rollback",
			UserID:   "user-rollback",
			Category: "rollback-test",
			Limit:    100.00,
			Spent:    0,
		}

		// Start transaction
		dbTx, err := db.Begin()
		if err != nil {
			t.Fatalf("Failed to begin transaction: %v", err)
		}

		// Create within transaction using concrete type
		concreteRepo := repo.(*PostgresBudgetRepo)
		_, err = concreteRepo.CreateTx(ctx, dbTx, budget)
		if err != nil {
			dbTx.Rollback()
			t.Fatalf("CreateTx failed: %v", err)
		}

		// Rollback
		if err := dbTx.Rollback(); err != nil {
			t.Fatalf("Failed to rollback: %v", err)
		}

		// Verify it doesn't exist
		_, err = repo.GetByUserAndCategory(ctx, budget.UserID, budget.Category)
		if err != ErrBudgetNotFound {
			t.Errorf("Expected budget to not exist after rollback, got error: %v", err)
		}
	})
}
