package repository

import (
	"context"
	"sync"

	"dime-api/internal/model"
)

// MemoryTransactionRepo is an in-memory implementation of TransactionRepository for testing
type MemoryTransactionRepo struct {
	mu           sync.RWMutex
	transactions map[string]model.Transaction // key: transaction ID
	userIndex    map[string][]string          // key: userID, value: []transaction IDs
}

// NewMemoryTransactionRepo creates a new in-memory transaction repository
func NewMemoryTransactionRepo() *MemoryTransactionRepo {
	return &MemoryTransactionRepo{
		transactions: make(map[string]model.Transaction),
		userIndex:    make(map[string][]string),
	}
}

// Create implements TransactionRepository
func (m *MemoryTransactionRepo) Create(ctx context.Context, t model.Transaction) (model.Transaction, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.transactions[t.ID] = t
	m.userIndex[t.UserID] = append(m.userIndex[t.UserID], t.ID)
	return t, nil
}

// GetByID implements TransactionRepository
func (m *MemoryTransactionRepo) GetByID(ctx context.Context, id string) (model.Transaction, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	t, ok := m.transactions[id]
	if !ok {
		return model.Transaction{}, ErrNotFound
	}
	return t, nil
}

// ListByUser implements TransactionRepository
func (m *MemoryTransactionRepo) ListByUser(ctx context.Context, userID string) ([]model.Transaction, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ids, ok := m.userIndex[userID]
	if !ok {
		return []model.Transaction{}, nil
	}

	var result []model.Transaction
	for _, id := range ids {
		if t, ok := m.transactions[id]; ok {
			result = append(result, t)
		}
	}
	return result, nil
}

// ListByUserPaginated implements the paginated version
func (m *MemoryTransactionRepo) ListByUserPaginated(
	ctx context.Context,
	userID string,
	offset, limit int,
) ([]model.Transaction, int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ids, ok := m.userIndex[userID]
	if !ok {
		return []model.Transaction{}, 0, nil
	}

	totalCount := len(ids)

	// Get paginated slice
	var result []model.Transaction
	start := offset
	end := offset + limit
	
	if start > totalCount {
		start = totalCount
	}
	if end > totalCount {
		end = totalCount
	}

	for i := start; i < end; i++ {
		if t, ok := m.transactions[ids[i]]; ok {
			result = append(result, t)
		}
	}

	return result, totalCount, nil
}
