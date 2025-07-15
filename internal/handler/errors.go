package handler

import (
	"computer-management-api/internal/repository"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/google/uuid"
)

// ErrorHandler provides centralized error handling functionality for handlers
type ErrorHandler struct {
	Logger *log.Logger
}

// NewErrorHandler creates a new ErrorHandler instance
func NewErrorHandler(logger *log.Logger) *ErrorHandler {
	if logger == nil {
		logger = log.Default()
	}
	return &ErrorHandler{
		Logger: logger,
	}
}

// SendErrorResponse sends a structured error response
func (e *ErrorHandler) SendErrorResponse(w http.ResponseWriter, statusCode int, message, code string, details map[string]string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := ErrorResponse{
		Error:   message,
		Code:    code,
		Details: details,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		e.Logger.Printf("Failed to encode error response: %v", err)
	}
}

// SendSuccessResponse sends a structured success response
func (e *ErrorHandler) SendSuccessResponse(w http.ResponseWriter, statusCode int, message string, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := SuccessResponse{
		Message: message,
		Data:    data,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		e.Logger.Printf("Failed to encode success response: %v", err)
	}
}

// SendJSONResponse sends a generic JSON response
func (e *ErrorHandler) SendJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		e.Logger.Printf("Failed to encode JSON response: %v", err)
		e.SendErrorResponse(w, http.StatusInternalServerError, "Failed to encode response", "ENCODING_ERROR", nil)
	}
}

// HandleRepositoryError handles repository-specific errors and maps them to HTTP responses
func (e *ErrorHandler) HandleRepositoryError(w http.ResponseWriter, err error, operation string) {
	e.Logger.Printf("Repository error during %s: %v", operation, err)

	switch {
	case errors.Is(err, repository.ErrComputerNotFound):
		e.SendErrorResponse(w, http.StatusNotFound, "Computer not found", "COMPUTER_NOT_FOUND", nil)
	case errors.Is(err, repository.ErrDuplicateMAC):
		e.SendErrorResponse(w, http.StatusConflict, "Computer with this MAC address already exists", "DUPLICATE_MAC", nil)
	case errors.Is(err, repository.ErrInvalidMACFormat):
		e.SendErrorResponse(w, http.StatusBadRequest, "Invalid MAC address format", "INVALID_MAC_FORMAT", nil)
	case errors.Is(err, context.DeadlineExceeded):
		e.SendErrorResponse(w, http.StatusRequestTimeout, "Operation timed out", "TIMEOUT", nil)
	default:
		e.SendErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to %s computer", operation), "INTERNAL_ERROR", nil)
	}
}

// HandleValidationErrors handles validation errors and sends appropriate response
func (e *ErrorHandler) HandleValidationErrors(w http.ResponseWriter, validationErrors map[string]string) {
	if len(validationErrors) > 0 {
		e.SendErrorResponse(w, http.StatusBadRequest, "Validation failed", "VALIDATION_ERROR", validationErrors)
	}
}

// HandleJSONDecodeError handles JSON decoding errors
func (e *ErrorHandler) HandleJSONDecodeError(w http.ResponseWriter, err error) {
	e.Logger.Printf("JSON decode error: %v", err)
	e.SendErrorResponse(w, http.StatusBadRequest, "Invalid JSON format", "INVALID_JSON", nil)
}

// HandleUUIDParseError handles UUID parsing errors
func (e *ErrorHandler) HandleUUIDParseError(w http.ResponseWriter, err error) {
	e.Logger.Printf("UUID parse error: %v", err)
	e.SendErrorResponse(w, http.StatusBadRequest, "Invalid UUID format", "INVALID_UUID", nil)
}

// ParseAndValidateUUID parses and validates UUID from string
func (e *ErrorHandler) ParseAndValidateUUID(w http.ResponseWriter, idStr string) (uuid.UUID, bool) {
	if idStr == "" {
		e.SendErrorResponse(w, http.StatusBadRequest, "ID is required", "INVALID_UUID", nil)
		return uuid.Nil, false
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		e.HandleUUIDParseError(w, err)
		return uuid.Nil, false
	}

	return id, true
}

// HandleEmployeeAbbreviationError handles employee abbreviation validation errors
func (e *ErrorHandler) HandleEmployeeAbbreviationError(w http.ResponseWriter, err error) {
	e.SendErrorResponse(w, http.StatusBadRequest, err.Error(), "INVALID_EMPLOYEE_ABBREV", nil)
}
