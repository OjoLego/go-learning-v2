package model

import "time"

type TransactionType string

const (
	TypeIncome  TransactionType = "income"
	TypeExpense TransactionType = "expense"
)

type Transaction struct {
	ID        string          `json:"id"`
	UserID    string          `json:"user_id"`
	Type      TransactionType `json:"type"`
	Amount    float64         `json:"amount"`
	Category  string          `json:"category"`
	CreatedAt time.Time       `json:"created_at"`
}
