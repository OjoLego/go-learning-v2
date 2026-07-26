package model

// Budget tracks a spending limit for a given user + category.
// Spent is updated as expense transactions come in for that category.
type Budget struct {
	ID       string  `json:"id"`
	UserID   string  `json:"user_id"`
	Category string  `json:"category"`
	Limit    float64 `json:"limit"`
	Spent    float64 `json:"spent"`
}
