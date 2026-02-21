package responder

// Response represents the standard API response structure
type Response struct {
	Data  any    `json:"data,omitempty"`
	Error *Error `json:"error,omitempty"`
	Meta  Meta   `json:"meta"`
}

// Error represents the error structure in API responses
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

// FieldError represents a validation error for a specific field
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Meta represents metadata in API responses
type Meta struct {
	TraceId    string          `json:"traceId,omitempty"`
	Took       int64           `json:"took,omitempty"`
	Pagination *PaginationMeta `json:"pagination,omitempty"`
}

// PaginationMeta represents pagination information
type PaginationMeta struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"pageSize"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"totalPages"`
	HasMore    bool  `json:"hasMore"`
}
