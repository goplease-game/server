package app

import (
	"fmt"
	"net/http"
	"strings"
)

const (
	// CodeBadRequest corresponds to HTTP 400 Bad Request.
	CodeBadRequest = http.StatusBadRequest

	// CodeUnauthorized corresponds to HTTP 401 Unauthorized (Authentication required).
	CodeUnauthorized = http.StatusUnauthorized

	// CodeForbidden corresponds to HTTP 403 Forbidden (Authorization denied).
	CodeForbidden = http.StatusForbidden

	// CodeNotFound corresponds to HTTP 404 Not Found.
	CodeNotFound = http.StatusNotFound

	// CodeUnprocessable corresponds to HTTP 422 Unprocessable Entity (Semantic error in the request body).
	CodeUnprocessable = http.StatusUnprocessableEntity

	// CodeInternal corresponds to HTTP 500 Internal Server Error.
	CodeInternal = http.StatusInternalServerError

	// CodeTooManyRequests corresponds to HTTP 429 Too Many Requests Error.
	CodeTooManyRequests = http.StatusTooManyRequests
)

// Error is a custom application error type that includes an HTTP status code
// for use in API responses, allowing the handler to return a structured error.
type Error struct {
	Code    int
	Message string
}

// Error implements the standard Go error interface.
// It returns the error's descriptive message.
func (e Error) Error() string {
	return e.Message
}

// NewError is a factory function that creates a new structured Error.
func NewError(code int, message string, params ...any) error {
	if len(params) > 0 {
		message = fmt.Sprintf(message, params...)
	}

	return Error{
		Code:    code,
		Message: message,
	}
}

// InputError is a custom error type represented by a map, specifically used to
// report multiple validation failures, mapping input field names to their corresponding error messages.
type InputError map[string]string

// NewInputError creates and returns an empty InputError map.
func NewInputError(kvOpt ...string) InputError {
	e := InputError{}

	if len(kvOpt) >= 2 { //nolint:mnd
		if len(kvOpt) == 2 { //nolint:mnd
			e.Add(kvOpt[0], kvOpt[1])
			return e
		}

		// kvOpt[0] = key, kvOpt[1] = format string, kvOpt[2:] = format args
		raw := kvOpt[2:]
		args := make([]any, len(raw))
		for i, s := range raw {
			args[i] = s
		}

		e.Add(kvOpt[0], kvOpt[1], args...)
	}

	return e
}

// Add inserts a key-value pair (field name and error message) into the InputError map.
func (i InputError) Add(k, v string, args ...any) {
	if len(args) > 0 {
		v = fmt.Sprintf(v, args...)
	}

	i[k] = v
}

// AddIf conditionally inserts a key-value pair into the InputError map
// only if the provided boolean condition is true.
func (i InputError) AddIf(cond bool, k, v string) {
	if cond {
		i[k] = v
	}
}

// Has checks if the InputError map contains any validation errors.
// Returns true if the map's length is greater than zero.
func (i InputError) Has() bool {
	return len(i) > 0
}

// Error implements the standard Go error interface.
// It returns a multi-line string representation of all collected input errors.
func (i InputError) Error() string {
	s := strings.Builder{}
	for k, v := range i {
		s.WriteString(k + ": " + v + "\n")
	}

	return s.String()
}

// ErrUnprocessable is a convenience function to create a new Error with
// the CodeUnprocessable (HTTP 422) status.
func ErrUnprocessable(message string) error {
	return NewError(CodeUnprocessable, message)
}

// ErrNotFound is a convenience function to create a new Error with
// the CodeNotFound (HTTP 404) status.
func ErrNotFound(message string) error {
	return NewError(CodeNotFound, message)
}

// ErrInternal is a convenience function to create a new Error with
// the CodeInternal (HTTP 500) status.
func ErrInternal(message string) error {
	return NewError(CodeInternal, message)
}

// ErrBadRequest is a convenience function to create a new Error with
// the CodeBadRequest (HTTP 400) status.
func ErrBadRequest(message string, args ...any) error {
	return NewError(CodeBadRequest, message, args...)
}

// ErrUnauthorized is a convenience function to create a new Error with
// the CodeUnauthorized (HTTP 401) status and a default message.
func ErrUnauthorized() error {
	return NewError(CodeUnauthorized, "Unauthorized")
}

// ErrForbidden is a convenience function to create a new Error with
// the CodeForbidden (HTTP 403) status.
func ErrForbidden(message string) error {
	return NewError(CodeForbidden, message)
}

// ErrTooManyRequests is a convenience function to create a new Error with
// the CodeTooManyRequests (HTTP 429) status.
func ErrTooManyRequests(message string, params ...any) error {
	return NewError(CodeTooManyRequests, message, params...)
}
