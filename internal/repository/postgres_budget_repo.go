package repository

import (
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
func (r *PostgresBudgetRepo) Create(b model.Budget) (model.Budget, error) {
	query := `
		INSERT INTO budgets (id, user_id, category, limit_amount, spent)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, category) DO UPDATE SET
			limit_amount = EXCLUDED.limit_amount
		RETURNING id, user_id, category, limit_amount, spent
	`

	err := r.db.QueryRow(
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
func (r *PostgresBudgetRepo) GetByUserAndCategory(userID, category string) (model.Budget, error) {
	query := `
		SELECT id, user_id, category, limit_amount, spent
		FROM budgets
		WHERE user_id = $1 AND category = $2
	`

	var b model.Budget
	err := r.db.QueryRow(query, userID, category).Scan(
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
func (r *PostgresBudgetRepo) AddSpent(userID, category string, amount float64) (model.Budget, error) {
	query := `
		UPDATE budgets
		SET spent = spent + $3
		WHERE user_id = $1 AND category = $2
		RETURNING id, user_id, category, limit_amount, spent
	`

	var b model.Budget
	err := r.db.QueryRow(query, userID, category, amount).Scan(
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
