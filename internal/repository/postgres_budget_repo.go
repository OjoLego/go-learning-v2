package repository

import (
	"context"
	"database/sql"
	"dime-api/internal/model"
	"fmt"
)

// PostgresBudgetRepo implements BudgetRepository using PostgreSQL
type PostgresBudgetRepo struct {
	db *sql.DB
}

// NewPostgresBudgetRepo creates a new PostgreSQL budget repository
func NewPostgresBudgetRepo(db *sql.DB) BudgetRepository {
	return &PostgresBudgetRepo{db: db}
}

// Create inserts a new budget into the database
func (r *PostgresBudgetRepo) Create(ctx context.Context, b model.Budget) (model.Budget, error) {
	query := `
		INSERT INTO budgets (id, user_id, category, limit_amount, spent)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, category) DO UPDATE SET
			limit_amount = EXCLUDED.limit_amount
		RETURNING id, user_id, category, limit_amount, spent
	`

	err := r.db.QueryRowContext(ctx,
		query,
		b.ID,
		b.UserID,
		b.Category,
		b.Limit,
		b.Spent,
	).Scan(&b.ID, &b.UserID, &b.Category, &b.Limit, &b.Spent)

	if err != nil {
		return model.Budget{}, fmt.Errorf("failed to create budget: %w", err)
	}

	return b, nil
}

// GetByUserAndCategory retrieves a budget by user ID and category
func (r *PostgresBudgetRepo) GetByUserAndCategory(ctx context.Context, userID, category string) (model.Budget, error) {
	query := `
		SELECT id, user_id, category, limit_amount, spent
		FROM budgets
		WHERE user_id = $1 AND category = $2
	`

	var b model.Budget
	err := r.db.QueryRowContext(ctx, query, userID, category).Scan(
		&b.ID,
		&b.UserID,
		&b.Category,
		&b.Limit,
		&b.Spent,
	)

	if err == sql.ErrNoRows {
		return model.Budget{}, ErrBudgetNotFound
	}
	if err != nil {
		return model.Budget{}, fmt.Errorf("failed to get budget: %w", err)
	}

	return b, nil
}

// AddSpent increments the spent amount on a budget
func (r *PostgresBudgetRepo) AddSpent(ctx context.Context, userID, category string, amount float64) (model.Budget, error) {
	query := `
		UPDATE budgets
		SET spent = spent + $3
		WHERE user_id = $1 AND category = $2
		RETURNING id, user_id, category, limit_amount, spent
	`

	var b model.Budget
	err := r.db.QueryRowContext(ctx, query, userID, category, amount).Scan(
		&b.ID,
		&b.UserID,
		&b.Category,
		&b.Limit,
		&b.Spent,
	)

	if err == sql.ErrNoRows {
		return model.Budget{}, ErrBudgetNotFound
	}
	if err != nil {
		return model.Budget{}, fmt.Errorf("failed to add spent: %w", err)
	}

	return b, nil
}

// CreateTx inserts a new budget within an existing transaction
func (r *PostgresBudgetRepo) CreateTx(ctx context.Context, tx *sql.Tx, b model.Budget) (model.Budget, error) {
	query := `
		INSERT INTO budgets (id, user_id, category, limit_amount, spent)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, category) DO UPDATE SET
			limit_amount = EXCLUDED.limit_amount
		RETURNING id, user_id, category, limit_amount, spent
	`

	err := tx.QueryRowContext(ctx,
		query,
		b.ID,
		b.UserID,
		b.Category,
		b.Limit,
		b.Spent,
	).Scan(&b.ID, &b.UserID, &b.Category, &b.Limit, &b.Spent)

	if err != nil {
		return model.Budget{}, fmt.Errorf("failed to create budget: %w", err)
	}

	return b, nil
}

// GetByUserAndCategoryTx retrieves a budget by user ID and category within an existing transaction
func (r *PostgresBudgetRepo) GetByUserAndCategoryTx(ctx context.Context, tx *sql.Tx, userID, category string) (model.Budget, error) {
	query := `
		SELECT id, user_id, category, limit_amount, spent
		FROM budgets
		WHERE user_id = $1 AND category = $2
	`

	var b model.Budget
	err := tx.QueryRowContext(ctx, query, userID, category).Scan(
		&b.ID,
		&b.UserID,
		&b.Category,
		&b.Limit,
		&b.Spent,
	)

	if err == sql.ErrNoRows {
		return model.Budget{}, ErrBudgetNotFound
	}
	if err != nil {
		return model.Budget{}, fmt.Errorf("failed to get budget: %w", err)
	}

	return b, nil
}

// AddSpentTx increments the spent amount on a budget within an existing transaction
func (r *PostgresBudgetRepo) AddSpentTx(ctx context.Context, tx *sql.Tx, userID, category string, amount float64) (model.Budget, error) {
	query := `
		UPDATE budgets
		SET spent = spent + $3
		WHERE user_id = $1 AND category = $2
		RETURNING id, user_id, category, limit_amount, spent
	`

	var b model.Budget
	err := tx.QueryRowContext(ctx, query, userID, category, amount).Scan(
		&b.ID,
		&b.UserID,
		&b.Category,
		&b.Limit,
		&b.Spent,
	)

	if err == sql.ErrNoRows {
		return model.Budget{}, ErrBudgetNotFound
	}
	if err != nil {
		return model.Budget{}, fmt.Errorf("failed to add spent: %w", err)
	}

	return b, nil
}
