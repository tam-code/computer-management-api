package errors

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"time"
)

// ErrorCode represents a specific error type
type ErrorCode string

const (
	// Business logic errors
	ErrorCodeValidation    ErrorCode = "VALIDATION_ERROR"
	ErrorCodeNotFound      ErrorCode = "NOT_FOUND"
	ErrorCodeAlreadyExists ErrorCode = "ALREADY_EXISTS"
	ErrorCodeUnauthorized  ErrorCode = "UNAUTHORIZED"
	ErrorCodeForbidden     ErrorCode = "FORBIDDEN"
	ErrorCodeConflict      ErrorCode = "CONFLICT"

	// Technical errors
	ErrorCodeInternal        ErrorCode = "INTERNAL_ERROR"
	ErrorCodeDatabase        ErrorCode = "DATABASE_ERROR"
	ErrorCodeExternalService ErrorCode = "EXTERNAL_SERVICE_ERROR"
	ErrorCodeTimeout         ErrorCode = "TIMEOUT_ERROR"
	ErrorCodeRateLimit       ErrorCode = "RATE_LIMIT_ERROR"

	// Request errors
	ErrorCodeBadRequest       ErrorCode = "BAD_REQUEST"
	ErrorCodeInvalidJSON      ErrorCode = "INVALID_JSON"
	ErrorCodeMissingParameter ErrorCode = "MISSING_PARAMETER"
	ErrorCodeInvalidParameter ErrorCode = "INVALID_PARAMETER"
)

// AppError represents a structured application error
type AppError struct {
	Code       ErrorCode              `json:"code"`
	Message    string                 `json:"message"`
	Details    map[string]interface{} `json:"details,omitempty"`
	Cause      error                  `json:"-"`
	Timestamp  time.Time              `json:"timestamp"`
	RequestID  string                 `json:"request_id,omitempty"`
	StackTrace string                 `json:"-"` // Don't expose in JSON
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying cause for error wrapping
func (e *AppError) Unwrap() error {
	return e.Cause
}

// ToJSON converts the error to JSON for API responses
func (e *AppError) ToJSON() []byte {
	data, _ := json.Marshal(map[string]interface{}{
		"error":      e.Message,
		"code":       e.Code,
		"details":    e.Details,
		"timestamp":  e.Timestamp,
		"request_id": e.RequestID,
	})
	return data
}

// GetHTTPStatus returns the appropriate HTTP status code for the error
func (e *AppError) GetHTTPStatus() int {
	switch e.Code {
	case ErrorCodeValidation, ErrorCodeBadRequest, ErrorCodeInvalidJSON,
		ErrorCodeMissingParameter, ErrorCodeInvalidParameter:
		return http.StatusBadRequest
	case ErrorCodeNotFound:
		return http.StatusNotFound
	case ErrorCodeAlreadyExists, ErrorCodeConflict:
		return http.StatusConflict
	case ErrorCodeUnauthorized:
		return http.StatusUnauthorized
	case ErrorCodeForbidden:
		return http.StatusForbidden
	case ErrorCodeTimeout:
		return http.StatusRequestTimeout
	case ErrorCodeRateLimit:
		return http.StatusTooManyRequests
	case ErrorCodeExternalService:
		return http.StatusBadGateway
	default:
		return http.StatusInternalServerError
	}
}

// NewAppError creates a new application error
func NewAppError(code ErrorCode, message string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		Details:    make(map[string]interface{}),
		Timestamp:  time.Now(),
		StackTrace: getStackTrace(),
	}
}

// NewAppErrorWithCause creates a new application error with an underlying cause
func NewAppErrorWithCause(code ErrorCode, message string, cause error) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		Details:    make(map[string]interface{}),
		Cause:      cause,
		Timestamp:  time.Now(),
		StackTrace: getStackTrace(),
	}
}

// WithDetail adds a detail to the error
func (e *AppError) WithDetail(key string, value interface{}) *AppError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// WithRequestID adds a request ID to the error
func (e *AppError) WithRequestID(requestID string) *AppError {
	e.RequestID = requestID
	return e
}

// getStackTrace captures the current stack trace
func getStackTrace() string {
	buf := make([]byte, 2048)
	n := runtime.Stack(buf, false)
	return string(buf[:n])
}

// Predefined error constructors for common cases

// ValidationError creates a validation error
func ValidationError(message string) *AppError {
	return NewAppError(ErrorCodeValidation, message)
}

// ValidationErrorWithDetails creates a validation error with field details
func ValidationErrorWithDetails(message string, fields map[string]string) *AppError {
	err := NewAppError(ErrorCodeValidation, message)
	for field, msg := range fields {
		err.WithDetail(field, msg)
	}
	return err
}

// NotFoundError creates a not found error
func NotFoundError(resource string) *AppError {
	return NewAppError(ErrorCodeNotFound, fmt.Sprintf("%s not found", resource))
}

// AlreadyExistsError creates an already exists error
func AlreadyExistsError(resource string) *AppError {
	return NewAppError(ErrorCodeAlreadyExists, fmt.Sprintf("%s already exists", resource))
}

// DatabaseError creates a database error
func DatabaseError(message string, cause error) *AppError {
	return NewAppErrorWithCause(ErrorCodeDatabase, message, cause)
}

// ExternalServiceError creates an external service error
func ExternalServiceError(service string, cause error) *AppError {
	return NewAppErrorWithCause(ErrorCodeExternalService,
		fmt.Sprintf("external service '%s' error", service), cause)
}

// InternalError creates an internal server error
func InternalError(message string, cause error) *AppError {
	return NewAppErrorWithCause(ErrorCodeInternal, message, cause)
}

// TimeoutError creates a timeout error
func TimeoutError(operation string) *AppError {
	return NewAppError(ErrorCodeTimeout, fmt.Sprintf("timeout during %s", operation))
}

// BadRequestError creates a bad request error
func BadRequestError(message string) *AppError {
	return NewAppError(ErrorCodeBadRequest, message)
}

// InvalidJSONError creates an invalid JSON error
func InvalidJSONError(cause error) *AppError {
	return NewAppErrorWithCause(ErrorCodeInvalidJSON, "Invalid JSON format", cause)
}

// Error handling utilities

// IsAppError checks if an error is an AppError
func IsAppError(err error) bool {
	_, ok := err.(*AppError)
	return ok
}

// AsAppError converts an error to AppError if possible
func AsAppError(err error) (*AppError, bool) {
	appErr, ok := err.(*AppError)
	return appErr, ok
}

// WrapError wraps a generic error as an internal error
func WrapError(err error, message string) *AppError {
	if appErr, ok := AsAppError(err); ok {
		return appErr
	}
	return NewAppErrorWithCause(ErrorCodeInternal, message, err)
}
