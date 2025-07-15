package handler

import (
	"context"
	"net/http"
	"strconv"
	"time"
)

// ResponseHelper provides common response utilities and context management
type ResponseHelper struct{}

// NewResponseHelper creates a new ResponseHelper instance
func NewResponseHelper() *ResponseHelper {
	return &ResponseHelper{}
}

// ContextKey type for context keys to avoid collisions
type ContextKey string

const (
	RequestIDKey ContextKey = "request_id"
)

// PaginationParams holds pagination parameters
type PaginationParams struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
	Offset   int `json:"offset"`
	Limit    int `json:"limit"`
}

// PaginationMeta holds pagination metadata for responses
type PaginationMeta struct {
	Page         int  `json:"page"`
	PageSize     int  `json:"page_size"`
	TotalItems   int  `json:"total_items"`
	TotalPages   int  `json:"total_pages"`
	HasNext      bool `json:"has_next"`
	HasPrevious  bool `json:"has_previous"`
	NextPage     *int `json:"next_page,omitempty"`
	PreviousPage *int `json:"previous_page,omitempty"`
}

// Default pagination constants
const (
	DefaultPageSize = 10
	MaxPageSize     = 100
	MinPageSize     = 1
)

func (rh *ResponseHelper) ParsePaginationParams(r *http.Request) PaginationParams {
	query := r.URL.Query()

	// Parse page (default to 1)
	page := 1
	if pageStr := query.Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	// Parse page_size (default to DefaultPageSize)
	pageSize := DefaultPageSize
	if pageSizeStr := query.Get("page_size"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil {
			if ps >= MinPageSize && ps <= MaxPageSize {
				pageSize = ps
			}
		}
	}

	// Calculate offset and limit
	offset := (page - 1) * pageSize
	limit := pageSize

	return PaginationParams{
		Page:     page,
		PageSize: pageSize,
		Offset:   offset,
		Limit:    limit,
	}
}

// CalculatePaginationMeta calculates pagination metadata
func (rh *ResponseHelper) CalculatePaginationMeta(params PaginationParams, totalItems int) PaginationMeta {
	totalPages := (totalItems + params.PageSize - 1) / params.PageSize // Ceiling division
	if totalPages == 0 {
		totalPages = 1
	}

	hasNext := params.Page < totalPages
	hasPrevious := params.Page > 1

	var nextPage, previousPage *int
	if hasNext {
		next := params.Page + 1
		nextPage = &next
	}
	if hasPrevious {
		prev := params.Page - 1
		previousPage = &prev
	}

	return PaginationMeta{
		Page:         params.Page,
		PageSize:     params.PageSize,
		TotalItems:   totalItems,
		TotalPages:   totalPages,
		HasNext:      hasNext,
		HasPrevious:  hasPrevious,
		NextPage:     nextPage,
		PreviousPage: previousPage,
	}
}

// CreateRequestContext creates a context with timeout and optional request ID
func (rh *ResponseHelper) CreateRequestContext(r *http.Request, timeout time.Duration) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(r.Context(), timeout)

	// Add request ID if available from headers (common in production)
	if requestID := r.Header.Get("X-Request-ID"); requestID != "" {
		ctx = context.WithValue(ctx, RequestIDKey, requestID)
	}

	return ctx, cancel
}

// GetRequestIDFromContext extracts request ID from context
func (rh *ResponseHelper) GetRequestIDFromContext(ctx context.Context) string {
	if requestID, ok := ctx.Value(RequestIDKey).(string); ok {
		return requestID
	}
	return ""
}

// SetCommonHeaders sets common HTTP headers for all responses
func (rh *ResponseHelper) SetCommonHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-API-Version", "v1")
	// Add other common headers as needed
}

// CreateComputerSuccessData creates success response data for computer operations
func (rh *ResponseHelper) CreateComputerSuccessData(computerID, macAddress string) map[string]interface{} {
	data := make(map[string]interface{})
	if computerID != "" {
		data["id"] = computerID
	}
	if macAddress != "" {
		data["mac_address"] = macAddress
	}
	return data
}

// CreateListResponseData creates response data for list operations with metadata
func (rh *ResponseHelper) CreateListResponseData(items interface{}, count int, additionalData map[string]interface{}) map[string]interface{} {
	data := map[string]interface{}{
		"items": items,
		"count": count,
	}

	// Add any additional metadata
	for key, value := range additionalData {
		data[key] = value
	}

	return data
}

// CreatePaginatedListResponseData creates response data for paginated list operations
func (rh *ResponseHelper) CreatePaginatedListResponseData(items interface{}, pagination PaginationMeta, additionalData map[string]interface{}) map[string]interface{} {
	data := map[string]interface{}{
		"items":      items,
		"pagination": pagination,
	}

	// Add any additional metadata
	for key, value := range additionalData {
		data[key] = value
	}

	return data
}

// CreateHealthCheckData creates health check response data
func (rh *ResponseHelper) CreateHealthCheckData() map[string]interface{} {
	return map[string]interface{}{
		"timestamp": time.Now().UTC(),
		"service":   "computer-management-api",
		"status":    "healthy",
	}
}
