package model

// PaginationRequest holds query parameters from the URL
// Example: ?page=2&page_size=10
type PaginationRequest struct {
	Page     int // Page number (1-based)
	PageSize int // Number of items per page
}

// Default values for pagination
const (
	DefaultPageSize = 20
	MaxPageSize     = 100
	MinPageSize     = 1
)

// NewPaginationRequest creates a pagination request with defaults
func NewPaginationRequest(page, pageSize int) PaginationRequest {
	// Ensure page is at least 1
	if page < 1 {
		page = 1
	}

	// Ensure pageSize is within bounds
	if pageSize < MinPageSize {
		pageSize = DefaultPageSize
	}
	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}

	return PaginationRequest{
		Page:     page,
		PageSize: pageSize,
	}
}

// Offset calculates the SQL OFFSET value
// Formula: (page - 1) * pageSize
func (p PaginationRequest) Offset() int {
	return (p.Page - 1) * p.PageSize
}

// Limit returns the SQL LIMIT value
func (p PaginationRequest) Limit() int {
	return p.PageSize
}

// PaginatedResponse wraps any data list with pagination metadata
type PaginatedResponse struct {
	Data       interface{} `json:"data"`        // The actual items
	Page       int         `json:"page"`        // Current page number
	PageSize   int         `json:"page_size"`   // Items per page
	TotalCount int         `json:"total_count"` // Total items in database
	TotalPages int         `json:"total_pages"` // Total number of pages
	HasNext    bool        `json:"has_next"`    // Is there a next page?
	HasPrev    bool        `json:"has_prev"`    // Is there a previous page?
}

// NewPaginatedResponse creates a paginated response with metadata
func NewPaginatedResponse(
	data interface{},
	page, pageSize, totalCount int,
) PaginatedResponse {
	// Calculate total pages
	totalPages := totalCount / pageSize
	if totalCount%pageSize > 0 {
		totalPages++ // Add one more page for remainder
	}

	// Handle edge case: no items
	if totalCount == 0 {
		totalPages = 0
	}

	return PaginatedResponse{
		Data:       data,
		Page:       page,
		PageSize:   pageSize,
		TotalCount: totalCount,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}
}