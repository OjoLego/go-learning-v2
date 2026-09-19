package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
)

// TransactionManager handles database transactions with proper error handling and rollback
type TransactionManager struct {
	db *sql.DB
}

// NewTransactionManager creates a new transaction manager
func NewTransactionManager(db *sql.DB) *TransactionManager {
	return &TransactionManager{db: db}
}

// RunInTransaction executes a function within a database transaction.
// It handles commit/rollback automatically and recovers from panics.
//
// Example usage:
//
//	err := tm.RunInTransaction(ctx, func(tx *sql.Tx) error {
//	    // Perform database operations using tx
//	    if err := repo.Create(ctx, tx, item); err != nil {
//	        return err // Automatic rollback
//	    }
//	    return nil // Automatic commit
//	})
func (tm *TransactionManager) RunInTransaction(ctx context.Context, fn func(*sql.Tx) error) error {
	// Begin transaction with context support
	tx, err := tm.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Defer rollback in case of panic or error
	// Rollback after commit is a no-op, so it's safe to always defer
	defer func() {
		if p := recover(); p != nil {
			// Panic occurred - rollback and re-panic
			if rbErr := tx.Rollback(); rbErr != nil {
				slog.Error("transaction rollback failed after panic", slog.String("error", rbErr.Error()))
			}
			panic(p) // Re-panic after rollback
		}
	}()

	// Execute the function
	if err := fn(tx); err != nil {
		// Error occurred - rollback
		if rbErr := tx.Rollback(); rbErr != nil {
			slog.Error("transaction rollback failed", slog.String("error", rbErr.Error()))
			return fmt.Errorf("transaction failed and rollback failed: %v (rollback error: %w)", err, rbErr)
		}
		slog.Debug("transaction rolled back due to error", slog.String("error", err.Error()))
		return fmt.Errorf("transaction rolled back: %w", err)
	}

	// Success - commit
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	slog.Debug("transaction committed successfully")
	return nil
}

// RunInTransactionWithIsolation executes a function within a transaction with a specific isolation level.
//
// Isolation levels:
//   - sql.LevelReadUncommitted: Lowest isolation, may read uncommitted data
//   - sql.LevelReadCommitted:   Prevents dirty reads (PostgreSQL default)
//   - sql.LevelWriteCommitted:  PostgreSQL specific
//   - sql.LevelRepeatableRead:  Prevents non-repeatable reads
//   - sql.LevelSnapshot:        Snapshot isolation
//   - sql.LevelSerializable:    Highest isolation, prevents all anomalies
//
// Use Serializable for critical financial operations where consistency is paramount.
func (tm *TransactionManager) RunInTransactionWithIsolation(
	ctx context.Context,
	isolation sql.IsolationLevel,
	fn func(*sql.Tx) error,
) error {
	// Begin transaction with specific isolation level
	opts := &sql.TxOptions{
		Isolation: isolation,
	}

	tx, err := tm.db.BeginTx(ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to begin transaction with isolation %v: %w", isolation, err)
	}

	// Defer rollback in case of panic or error
	defer func() {
		if p := recover(); p != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				slog.Error("transaction rollback failed after panic",
					slog.String("isolation", isolation.String()),
					slog.String("error", rbErr.Error()))
			}
			panic(p)
		}
	}()

	// Execute the function
	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			slog.Error("transaction rollback failed",
				slog.String("isolation", isolation.String()),
				slog.String("error", rbErr.Error()))
			return fmt.Errorf("transaction failed and rollback failed: %v (rollback error: %w)", err, rbErr)
		}
		slog.Debug("transaction rolled back due to error",
			slog.String("isolation", isolation.String()),
			slog.String("error", err.Error()))
		return fmt.Errorf("transaction rolled back: %w", err)
	}

	// Success - commit
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction with isolation %v: %w", isolation, err)
	}

	slog.Debug("transaction committed successfully", slog.String("isolation", isolation.String()))
	return nil
}

// TxRepository is a helper interface for repositories that support transactions
type TxRepository interface {
	WithTx(tx *sql.Tx) interface{}
}
