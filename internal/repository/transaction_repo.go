package repository

import (
	"errors"
	"sync"

	"dime-api/internal/model"
)

var ErrNotFound = errors.New("transaction not found")

// TransactionRepository is what the SERVICE layer depends on.
// Swap InMemoryTransactionRepo for a Postgres/Firestore implementation
// later without touching the service layer at all.
type TransactionRepository interface {
	Create(t model.Transaction) (model.Transaction, error)
	GetByID(id string) (model.Transaction, error)
	ListByUser(userID string) ([]model.Transaction, error)
}

type InMemoryTransactionRepo struct {
	mu   sync.Mutex
	data map[string]model.Transaction
}

func NewInMemoryTransactionRepo() *InMemoryTransactionRepo {
	return &InMemoryTransactionRepo{
		data: make(map[string]model.Transaction),
	}
}

func (r *InMemoryTransactionRepo) Create(t model.Transaction) (model.Transaction, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data[t.ID] = t
	return t, nil
}

func (r *InMemoryTransactionRepo) GetByID(id string) (model.Transaction, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.data[id]
	if !ok {
		return model.Transaction{}, ErrNotFound
	}
	return t, nil
}

func (r *InMemoryTransactionRepo) ListByUser(userID string) ([]model.Transaction, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var results []model.Transaction
	for _, t := range r.data {
		if t.UserID == userID {
			results = append(results, t)
		}
	}
	return results, nil
}
