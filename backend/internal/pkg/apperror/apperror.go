// Package apperror defines typed errors that handlers map to the unified
// business-code envelope.
package apperror

import (
	"errors"
	"fmt"
)

// Business codes shared across the API (0 = ok).
const (
	CodeOK           = 0
	CodeBadRequest   = 400
	CodeUnauthorized = 401
	CodeForbidden    = 403
	CodeNotFound     = 404
	CodeConflict     = 409
	CodeTooMany      = 429
	CodeInternal     = 500
	CodeUnavailable  = 503
)

// Error is a typed application error carrying a business code.
type Error struct {
	Code int
	Msg  string
	Err  error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Msg, e.Err)
	}
	return e.Msg
}

// Unwrap allows errors.Is/As compatibility.
func (e *Error) Unwrap() error { return e.Err }

// New builds an Error with the given business code and message.
func New(code int, msg string) *Error {
	return &Error{Code: code, Msg: msg}
}

// Wrap annotates an error with a code and message.
func Wrap(code int, msg string, err error) *Error {
	return &Error{Code: code, Msg: msg, Err: err}
}

// BadRequest returns a 400 error.
func BadRequest(msg string) *Error { return New(CodeBadRequest, msg) }

// Unauthorized returns a 401 error.
func Unauthorized(msg string) *Error { return New(CodeUnauthorized, msg) }

// Forbidden returns a 403 error.
func Forbidden(msg string) *Error { return New(CodeForbidden, msg) }

// NotFound returns a 404 error for a resource name.
func NotFound(resource string) *Error {
	return New(CodeNotFound, resource+" not found")
}

// Conflict returns a 409 error.
func Conflict(msg string) *Error { return New(CodeConflict, msg) }

// TooMany returns a 429 error.
func TooMany(msg string) *Error { return New(CodeTooMany, msg) }

// Unavailable returns a 503 error.
func Unavailable(msg string) *Error { return New(CodeUnavailable, msg) }

// Internal returns a 500 error, logging the underlying cause separately.
func Internal(msg string, err error) *Error {
	return Wrap(CodeInternal, msg, err)
}

// CodeOf extracts the business code from an error, defaulting to 500.
func CodeOf(err error) int {
	var ae *Error
	if errors.As(err, &ae) {
		return ae.Code
	}
	return CodeInternal
}

// MessageOf extracts a safe client-facing message, defaulting to a generic
// message for internal errors so internals never leak.
func MessageOf(err error) string {
	var ae *Error
	if errors.As(err, &ae) {
		if ae.Code >= 500 {
			return "internal server error"
		}
		return ae.Msg
	}
	return "internal server error"
}
