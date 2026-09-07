package service

import (
	"strings"
	"testing"

	"dime-api/internal/model"
	"dime-api/internal/repository"
)

// mockTransactionRepo is a test double for TransactionRepository
type mockTransactionRepo struct {
	data map[string]model.Transaction
}

func newMockTransactionRepo() *mockTransactionRepo {
	return &mockTransactionRepo{
		data: make(map[string]model.Transaction),
	}
}

func (m *mockTransactionRepo) Create(t model.Transaction) (model.Transaction, error) {
	m.data[t.ID] = t
	return t, nil
}

func (m *mockTransactionRepo) GetByID(id string) (model.Transaction, error) {
	t, ok := m.data[id]
	if !ok {
		return model.Transaction{}, repository.ErrNotFound
	}
	return t, nil
}

func (m *mockTransactionRepo) ListByUser(userID string) ([]model.Transaction, error) {
	var results []model.Transaction
	for _, t := range m.data {
		if t.UserID == userID {
			results = append(results, t)
		}
	}
	return results, nil
}

func setupTransactionService() (*TransactionService, *mockTransactionRepo) {
	repo := newMockTransactionRepo()
	// For these tests, we don't need budget service functionality
	// so we pass nil (the code handles this gracefully in checkBudgetAfterExpense)
	svc := NewTransactionService(repo, nil)
	return svc, repo
}

func TestRecordTransaction_Validation(t *testing.T) {
	svc, _ := setupTransactionService()

	tests := []struct {
		name      string
		userID    string
		txType    model.TransactionType
		amount    float64
		category  string
		wantErr   bool
		errMsg    string
	}{
		{
			name:     "valid income transaction",
			userID:   "user1",
			txType:   model.TypeIncome,
			amount:   100.50,
			category: "salary",
			wantErr:  false,
		},
		{
			name:     "valid expense transaction",
			userID:   "user1",
			txType:   model.TypeExpense,
			amount:   50.25,
			category: "groceries",
			wantErr:  false,
		},
		{
			name:     "invalid: zero amount",
			userID:   "user1",
			txType:   model.TypeExpense,
			amount:   0,
			category: "food",
			wantErr:  true,
			errMsg:   "amount must be positive",
		},
		{
			name:     "invalid: negative amount",
			userID:   "user1",
			txType:   model.TypeExpense,
			amount:   -10.00,
			category: "food",
			wantErr:  true,
			errMsg:   "amount must be positive",
		},
		{
			name:     "invalid: large negative amount",
			userID:   "user1",
			txType:   model.TypeIncome,
			amount:   -9999.99,
			category: "refund",
			wantErr:  true,
			errMsg:   "amount must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := svc.RecordTransaction(tt.userID, tt.txType, tt.amount, tt.category)

			// Check error expectation
			if (err != nil) != tt.wantErr {
				t.Errorf("RecordTransaction() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Check error message contains expected text
			if tt.wantErr && err != nil {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error message to contain %q, got %q", tt.errMsg, err.Error())
				}
				// For error cases, we don't need to check the transaction
				return
			}

			// For success cases, verify the transaction was created correctly
			if !tt.wantErr {
				if got.UserID != tt.userID {
					t.Errorf("expected UserID %q, got %q", tt.userID, got.UserID)
				}
				if got.Type != tt.txType {
					t.Errorf("expected Type %q, got %q", tt.txType, got.Type)
				}
				if got.Amount != tt.amount {
					t.Errorf("expected Amount %.2f, got %.2f", tt.amount, got.Amount)
				}
				if got.Category != tt.category {
					t.Errorf("expected Category %q, got %q", tt.category, got.Category)
				}
				// Verify ID was generated (should be 32 hex chars)
				if len(got.ID) != 32 {
					t.Errorf("expected ID length 32, got %d", len(got.ID))
				}
				// Verify CreatedAt was set (should not be zero)
				if got.CreatedAt.IsZero() {
					t.Error("expected CreatedAt to be set, got zero value")
				}
			}
		})
	}
}

func TestRecordTransaction_SuccessFields(t *testing.T) {
	svc, repo := setupTransactionService()

	// Record a transaction
	tx, err := svc.RecordTransaction("user123", model.TypeExpense, 75.50, "dining")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify it was saved to the repository
	saved, err := repo.GetByID(tx.ID)
	if err != nil {
		t.Fatalf("failed to retrieve saved transaction: %v", err)
	}

	// Verify all fields match
	if saved.ID != tx.ID {
		t.Errorf("ID mismatch: got %q, want %q", saved.ID, tx.ID)
	}
	if saved.UserID != "user123" {
		t.Errorf("UserID mismatch: got %q, want %q", saved.UserID, "user123")
	}
	if saved.Type != model.TypeExpense {
		t.Errorf("Type mismatch: got %q, want %q", saved.Type, model.TypeExpense)
	}
	if saved.Amount != 75.50 {
		t.Errorf("Amount mismatch: got %.2f, want %.2f", saved.Amount, 75.50)
	}
	if saved.Category != "dining" {
		t.Errorf("Category mismatch: got %q, want %q", saved.Category, "dining")
	}
}

func TestGetTransaction(t *testing.T) {
	svc, _ := setupTransactionService()

	// First, record a transaction
	tx, err := svc.RecordTransaction("user1", model.TypeIncome, 500.00, "freelance")
	if err != nil {
		t.Fatalf("failed to create transaction: %v", err)
	}

	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{
			name:    "existing transaction",
			id:      tx.ID,
			wantErr: false,
		},
		{
			name:    "non-existent transaction",
			id:      "nonexistent-id-1234567890abcd",
			wantErr: true,
		},
		{
			name:    "empty id",
			id:      "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := svc.GetTransaction(tt.id)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetTransaction() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if got.ID != tt.id {
					t.Errorf("expected ID %q, got %q", tt.id, got.ID)
				}
			}
		})
	}
}
