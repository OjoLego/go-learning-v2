package repository

import (
	"context"
	"database/sql"
	"dime-api/internal/model"
	"fmt"
)

// PostgresTransactionRepo implements TransactionRepository using PostgreSQL
type PostgresTransactionRepo struct {
	db *sql.DB
}

// NewPostgresTransactionRepo creates a new PostgreSQL transaction repository
func NewPostgresTransactionRepo(db *sql.DB) TransactionRepository {
	return &PostgresTransactionRepo{db: db}
}

// Create inserts a new transaction into the database
func (r *PostgresTransactionRepo) Create(ctx context.Context, t model.Transaction) (model.Transaction, error) {
	query := `
		INSERT INTO transactions (id, user_id, type, amount, category, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, user_id, type, amount, category, created_at
	`

	err := r.db.QueryRowContext(ctx,
		query,
		t.ID,
		t.UserID,
		t.Type,
		t.Amount,
		t.Category,
		t.CreatedAt,
	).Scan(&t.ID, &t.UserID, &t.Type, &t.Amount, &t.Category, &t.CreatedAt)

	if err != nil {
		return model.Transaction{}, fmt.Errorf("failed to create transaction: %w", err)
	}

	return t, nil
}

// GetByID retrieves a transaction by its ID
func (r *PostgresTransactionRepo) GetByID(ctx context.Context, id string) (model.Transaction, error) {
	query := `
		SELECT id, user_id, type, amount, category, created_at
		FROM transactions
		WHERE id = $1
	`

	var t model.Transaction
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&t.ID,
		&t.UserID,
		&t.Type,
		&t.Amount,
		&t.Category,
		&t.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return model.Transaction{}, ErrNotFound
	}
	if err != nil {
		return model.Transaction{}, fmt.Errorf("failed to get transaction: %w", err)
	}

	return t, nil
}

// ListByUser retrieves all transactions for a specific user
func (r *PostgresTransactionRepo) ListByUser(ctx context.Context, userID string) ([]model.Transaction, error) {
	query := `
		SELECT id, user_id, type, amount, category, created_at
		FROM transactions
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list transactions: %w", err)
	}
	defer rows.Close()

	var transactions []model.Transaction
	for rows.Next() {
		var t model.Transaction
		err := rows.Scan(
			&t.ID,
			&t.UserID,
			&t.Type,
			&t.Amount,
			&t.Category,
			&t.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction: %w", err)
		}
		transactions = append(transactions, t)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating transactions: %w", err)
	}

	return transactions, nil
}

// CreateTx inserts a new transaction within an existing transaction
func (r *PostgresTransactionRepo) CreateTx(ctx context.Context, tx *sql.Tx, t model.Transaction) (model.Transaction, error) {
	query := `
		INSERT INTO transactions (id, user_id, type, amount, category, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, user_id, type, amount, category, created_at
	`

	err := tx.QueryRowContext(ctx,
		query,
		t.ID,
		t.UserID,
		t.Type,
		t.Amount,
		t.Category,
		t.CreatedAt,
	).Scan(&t.ID, &t.UserID, &t.Type, &t.Amount, &t.Category, &t.CreatedAt)

	if err != nil {
		return model.Transaction{}, fmt.Errorf("failed to create transaction: %w", err)
	}

	return t, nil
}

// GetByIDTx retrieves a transaction by its ID within an existing transaction
func (r *PostgresTransactionRepo) GetByIDTx(ctx context.Context, tx *sql.Tx, id string) (model.Transaction, error) {
	query := `
		SELECT id, user_id, type, amount, category, created_at
		FROM transactions
		WHERE id = $1
	`

	var t model.Transaction
	err := tx.QueryRowContext(ctx, query, id).Scan(
		&t.ID,
		&t.UserID,
		&t.Type,
		&t.Amount,
		&t.Category,
		&t.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return model.Transaction{}, ErrNotFound
	}
	if err != nil {
		return model.Transaction{}, fmt.Errorf("failed to get transaction: %w", err)
	}

	return t, nil
}
