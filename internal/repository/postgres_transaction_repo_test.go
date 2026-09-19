package repository

import (
	"context"
	"testing"
	"time"

	"dime-api/internal/model"
	"dime-api/internal/testutil"
)

func TestPostgresTransactionRepo_Integration(t *testing.T) {
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

	repo := NewPostgresTransactionRepo(db)
	ctx := context.Background()

	t.Run("Create", func(t *testing.T) {
		tx := model.Transaction{
			ID:        "test-tx-1",
			UserID:    "user-1",
			Type:      model.TypeExpense,
			Amount:    50.00,
			Category:  "food",
			CreatedAt: time.Now(),
		}

		created, err := repo.Create(ctx, tx)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		if created.ID != tx.ID {
			t.Errorf("Expected ID %s, got %s", tx.ID, created.ID)
		}
		if created.UserID != tx.UserID {
			t.Errorf("Expected UserID %s, got %s", tx.UserID, created.UserID)
		}
	})

	t.Run("GetByID", func(t *testing.T) {
		// First create a transaction
		tx := model.Transaction{
			ID:        "test-tx-2",
			UserID:    "user-2",
			Type:      model.TypeIncome,
			Amount:    100.00,
			Category:  "salary",
			CreatedAt: time.Now(),
		}

		_, err := repo.Create(ctx, tx)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		// Now retrieve it
		retrieved, err := repo.GetByID(ctx, tx.ID)
		if err != nil {
			t.Fatalf("GetByID failed: %v", err)
		}

		if retrieved.ID != tx.ID {
			t.Errorf("Expected ID %s, got %s", tx.ID, retrieved.ID)
		}
		if retrieved.Amount != tx.Amount {
			t.Errorf("Expected Amount %f, got %f", tx.Amount, retrieved.Amount)
		}
	})

	t.Run("GetByID_NotFound", func(t *testing.T) {
		_, err := repo.GetByID(ctx, "non-existent-id")
		if err != ErrNotFound {
			t.Errorf("Expected ErrNotFound, got %v", err)
		}
	})

	t.Run("ListByUser", func(t *testing.T) {
		userID := "user-3"

		// Create multiple transactions for this user
		transactions := []model.Transaction{
			{
				ID:        "test-tx-3a",
				UserID:    userID,
				Type:      model.TypeExpense,
				Amount:    25.00,
				Category:  "food",
				CreatedAt: time.Now().Add(-1 * time.Hour),
			},
			{
				ID:        "test-tx-3b",
				UserID:    userID,
				Type:      model.TypeExpense,
				Amount:    30.00,
				Category:  "transport",
				CreatedAt: time.Now(),
			},
		}

		for _, tx := range transactions {
			_, err := repo.Create(ctx, tx)
			if err != nil {
				t.Fatalf("Create failed: %v", err)
			}
		}

		// List transactions
		list, err := repo.ListByUser(ctx, userID)
		if err != nil {
			t.Fatalf("ListByUser failed: %v", err)
		}

		if len(list) != 2 {
			t.Errorf("Expected 2 transactions, got %d", len(list))
		}

		// Verify ordering (should be by created_at DESC)
		if list[0].ID != "test-tx-3b" {
			t.Errorf("Expected first transaction to be test-tx-3b, got %s", list[0].ID)
		}
	})

	t.Run("CreateTx", func(t *testing.T) {
		// Test creating within a transaction
		tx := model.Transaction{
			ID:        "test-tx-4",
			UserID:    "user-4",
			Type:      model.TypeExpense,
			Amount:    75.00,
			Category:  "entertainment",
			CreatedAt: time.Now(),
		}

		// Start a transaction
		dbTx, err := db.Begin()
		if err != nil {
			t.Fatalf("Failed to begin transaction: %v", err)
		}

		// Use concrete type to access Tx methods
		concreteRepo := repo.(*PostgresTransactionRepo)
		created, err := concreteRepo.CreateTx(ctx, dbTx, tx)
		if err != nil {
			dbTx.Rollback()
			t.Fatalf("CreateTx failed: %v", err)
		}

		// Commit the transaction
		if err := dbTx.Commit(); err != nil {
			t.Fatalf("Failed to commit transaction: %v", err)
		}

		if created.ID != tx.ID {
			t.Errorf("Expected ID %s, got %s", tx.ID, created.ID)
		}

		// Verify it was actually created
		retrieved, err := repo.GetByID(ctx, tx.ID)
		if err != nil {
			t.Fatalf("GetByID failed: %v", err)
		}
		if retrieved.ID != tx.ID {
			t.Errorf("Expected to retrieve created transaction, got %s", retrieved.ID)
		}
	})

	t.Run("CreateTx_Rollback", func(t *testing.T) {
		// Test that rollback actually works
		tx := model.Transaction{
			ID:        "test-tx-5",
			UserID:    "user-5",
			Type:      model.TypeExpense,
			Amount:    99.00,
			Category:  "test",
			CreatedAt: time.Now(),
		}

		// Start a transaction
		dbTx, err := db.Begin()
		if err != nil {
			t.Fatalf("Failed to begin transaction: %v", err)
		}

		// Use concrete type to access Tx methods
		concreteRepo := repo.(*PostgresTransactionRepo)
		_, err = concreteRepo.CreateTx(ctx, dbTx, tx)
		if err != nil {
			dbTx.Rollback()
			t.Fatalf("CreateTx failed: %v", err)
		}

		// Rollback instead of commit
		if err := dbTx.Rollback(); err != nil {
			t.Fatalf("Failed to rollback transaction: %v", err)
		}

		// Verify the transaction was NOT created
		_, err = repo.GetByID(ctx, tx.ID)
		if err != ErrNotFound {
			t.Errorf("Expected transaction to not exist after rollback, got error: %v", err)
		}
	})
}

func TestPostgresTransactionRepo_ContextTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)
	testutil.CleanupTestDB(t, db)

	repo := NewPostgresTransactionRepo(db)

	t.Run("ContextTimeout", func(t *testing.T) {
		// Create a context with very short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		// Wait for context to expire
		time.Sleep(10 * time.Millisecond)

		tx := model.Transaction{
			ID:        "test-timeout",
			UserID:    "user-timeout",
			Type:      model.TypeExpense,
			Amount:    10.00,
			Category:  "test",
			CreatedAt: time.Now(),
		}

		_, err := repo.Create(ctx, tx)
		if err == nil {
			t.Error("Expected error due to context timeout, got nil")
		}
	})
}
