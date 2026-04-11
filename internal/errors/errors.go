package errors

import (
	"fmt"
	"net/http"
)

// Error codes used in API responses.
const (
	CodeValidation    = "VALIDATION_ERROR"
	CodeUnauthorized  = "UNAUTHORIZED"
	CodeForbidden     = "FORBIDDEN"
	CodeNotFound      = "NOT_FOUND"
	CodeConflict      = "CONFLICT"
	CodeRateLimited   = "RATE_LIMITED"
	CodeSetupRequired = "SETUP_REQUIRED"
	CodeInternal      = "INTERNAL_ERROR"
)

// FieldError represents a validation error on a specific field.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// AppError is the application's standard error type.
type AppError struct {
	Code     string       `json:"code"`
	Message  string       `json:"message"`
	Status   int          `json:"-"`
	Details  []FieldError `json:"details,omitempty"`
	Internal error        `json:"-"`
}

// Error implements the error interface.
func (e *AppError) Error() string {
	if e.Internal != nil {
		return fmt.Sprintf("%s: %s (internal: %v)", e.Code, e.Message, e.Internal)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the internal error for errors.Is / errors.As chains.
func (e *AppError) Unwrap() error {
	return e.Internal
}

// NewValidationError creates a 400 error with field-level details.
func NewValidationError(message string, details []FieldError) *AppError {
	return &AppError{
		Code:    CodeValidation,
		Message: message,
		Status:  http.StatusBadRequest,
		Details: details,
	}
}

// NewUnauthorized creates a 401 error.
func NewUnauthorized(message string) *AppError {
	return &AppError{
		Code:    CodeUnauthorized,
		Message: message,
		Status:  http.StatusUnauthorized,
	}
}

// NewForbidden creates a 403 error.
func NewForbidden(message string) *AppError {
	return &AppError{
		Code:    CodeForbidden,
		Message: message,
		Status:  http.StatusForbidden,
	}
}

// NewNotFound creates a 404 error.
func NewNotFound(resource string) *AppError {
	return &AppError{
		Code:    CodeNotFound,
		Message: fmt.Sprintf("%s not found", resource),
		Status:  http.StatusNotFound,
	}
}

// NewConflict creates a 409 error.
func NewConflict(message string) *AppError {
	return &AppError{
		Code:    CodeConflict,
		Message: message,
		Status:  http.StatusConflict,
	}
}

// NewRateLimited creates a 429 error.
func NewRateLimited() *AppError {
	return &AppError{
		Code:    CodeRateLimited,
		Message: "rate limit exceeded",
		Status:  http.StatusTooManyRequests,
	}
}

// NewSetupRequired creates a 503 error.
func NewSetupRequired() *AppError {
	return &AppError{
		Code:    CodeSetupRequired,
		Message: "initial setup required",
		Status:  http.StatusServiceUnavailable,
	}
}

// NewInternal creates a 500 error, wrapping the underlying cause.
func NewInternal(err error) *AppError {
	return &AppError{
		Code:     CodeInternal,
		Message:  "internal server error",
		Status:   http.StatusInternalServerError,
		Internal: err,
	}
}

// NewBadRequest creates a 400 error with a custom message.
func NewBadRequest(message string) *AppError {
	return &AppError{
		Code:    CodeValidation,
		Message: message,
		Status:  http.StatusBadRequest,
	}
}

// ErrorResponse is the JSON envelope sent to clients.
type ErrorResponse struct {
	Error *AppError `json:"error"`
}
