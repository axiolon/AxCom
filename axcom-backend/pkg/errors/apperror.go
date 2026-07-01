// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package errors

import "fmt"

// AppError represents a domain-specific error that carries HTTP status and user context.
type AppError struct {
	Code    int    // HTTP Status Code
	Message string // User-friendly error message
	Err     error  // The underlying error (not sent to user, for logging)
	Type    string // RFC 7807 problem type URI (defaults to "about:blank")
}

// Error returns the string representation of the AppError.
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("code=%d message=%q err=%v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("code=%d message=%q", e.Code, e.Message)
}

// Unwrap returns the wrapped internal error.
func (e *AppError) Unwrap() error {
	return e.Err
}

// WithType sets the RFC 7807 problem type URI and returns the AppError for chaining.
func (e *AppError) WithType(t string) *AppError {
	e.Type = t
	return e
}

// NewAppError creates a generic AppError.
func NewAppError(code int, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
		Type:    "about:blank",
	}
}

// NewBadRequest creates a 400 Bad Request AppError.
func NewBadRequest(message string, err error) *AppError {
	return &AppError{Code: 400, Message: message, Err: err, Type: "about:blank"}
}

// NewUnauthorized creates a 401 Unauthorized AppError.
func NewUnauthorized(message string, err error) *AppError {
	return &AppError{Code: 401, Message: message, Err: err, Type: "about:blank"}
}

// NewForbidden creates a 403 Forbidden AppError.
func NewForbidden(message string, err error) *AppError {
	return &AppError{Code: 403, Message: message, Err: err, Type: "about:blank"}
}

// NewNotFound creates a 404 Not Found AppError.
func NewNotFound(message string, err error) *AppError {
	return &AppError{Code: 404, Message: message, Err: err, Type: "about:blank"}
}

// NewConflict creates a 409 Conflict AppError.
func NewConflict(message string, err error) *AppError {
	return &AppError{Code: 409, Message: message, Err: err, Type: "about:blank"}
}

// NewInternal creates a 500 Internal Server Error AppError.
func NewInternal(message string, err error) *AppError {
	return &AppError{Code: 500, Message: message, Err: err, Type: "about:blank"}
}

// NewTooManyRequests creates a 429 Too Many Requests AppError.
func NewTooManyRequests(message string, err error) *AppError {
	return &AppError{Code: 429, Message: message, Err: err, Type: "about:blank"}
}

// Note on error unwrapping/compatibility:
// AppError implements Unwrap() error, allowing standard library functions
// like errors.Is and errors.As to inspect the underlying wrapped error.
// Example:
//
//	var appErr *AppError
//	if errors.As(err, &appErr) {
//	    log.Printf("HTTP Code: %d", appErr.Code)
//	}
