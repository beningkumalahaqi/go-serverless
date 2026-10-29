// Package lib holds the HTTP building blocks shared by the endpoint files in
// api/: request decoding, the success envelope, the error type, and the
// handler adapter. It depends on the standard library only.
package lib

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
)

// ErrorCode is the stable, machine-readable value in an error response.
type ErrorCode string

const (
	CodeInvalidRequest   ErrorCode = "invalid_request"
	CodeUnauthorized     ErrorCode = "unauthorized"
	CodeForbidden        ErrorCode = "forbidden"
	CodeNotFound         ErrorCode = "not_found"
	CodeMethodNotAllowed ErrorCode = "method_not_allowed"
	CodeConflict         ErrorCode = "conflict"
	CodePayloadTooLarge  ErrorCode = "payload_too_large"
	CodeUnprocessable    ErrorCode = "unprocessable_entity"
	CodeTooManyRequests  ErrorCode = "too_many_requests"
	CodeInternal         ErrorCode = "internal_error"
)

// APIError is an error with an HTTP status and a stable code. Endpoints return
// one when they want a specific client-facing response. Any other error becomes
// a generic 500, and its text is never sent to the client.
type APIError struct {
	Status  int
	Code    ErrorCode
	Message string
	Err     error
}

// Error implements error.
func (e *APIError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

// Unwrap exposes the cause to errors.Is and errors.As.
func (e *APIError) Unwrap() error { return e.Err }

// NewError builds an APIError for a status the helpers do not cover.
func NewError(status int, code ErrorCode, message string) *APIError {
	return &APIError{Status: status, Code: code, Message: message}
}

// BadRequest returns a 400 invalid_request error.
func BadRequest(message string) *APIError {
	return NewError(http.StatusBadRequest, CodeInvalidRequest, message)
}

// Unauthorized returns a 401 error.
func Unauthorized(message string) *APIError {
	return NewError(http.StatusUnauthorized, CodeUnauthorized, message)
}

// Forbidden returns a 403 error.
func Forbidden(message string) *APIError {
	return NewError(http.StatusForbidden, CodeForbidden, message)
}

// NotFound returns a 404 error.
func NotFound(message string) *APIError {
	return NewError(http.StatusNotFound, CodeNotFound, message)
}

// Conflict returns a 409 error.
func Conflict(message string) *APIError {
	return NewError(http.StatusConflict, CodeConflict, message)
}

// PayloadTooLarge returns a 413 error that names the accepted body size.
func PayloadTooLarge(maxBytes int64) *APIError {
	return NewError(http.StatusRequestEntityTooLarge, CodePayloadTooLarge,
		fmt.Sprintf("body must be at most %d bytes", maxBytes))
}

// Unprocessable returns a 422 error.
func Unprocessable(message string) *APIError {
	return NewError(http.StatusUnprocessableEntity, CodeUnprocessable, message)
}

// TooManyRequests returns a 429 error.
func TooManyRequests(message string) *APIError {
	return NewError(http.StatusTooManyRequests, CodeTooManyRequests, message)
}

// Internal wraps an unexpected failure. The cause is logged by WriteError and
// never included in the response.
func Internal(err error) *APIError {
	return &APIError{
		Status:  http.StatusInternalServerError,
		Code:    CodeInternal,
		Message: "internal server error",
		Err:     err,
	}
}

// AsAPIError returns err as an *APIError, or a generic 500 when it is not one.
func AsAPIError(err error) *APIError {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr
	}
	return Internal(err)
}

type errorDetail struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
}

type errorBody struct {
	Error errorDetail `json:"error"`
}

// WriteError writes err in the shared error shape. Errors that are not an
// *APIError become a generic 500, and 5xx causes are logged instead of exposed.
func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	apiErr := AsAPIError(err)
	if apiErr.Status >= http.StatusInternalServerError {
		slog.Error("request failed",
			"method", r.Method,
			"path", r.URL.Path,
			"code", apiErr.Code,
			"err", apiErr,
		)
	}
	writeJSON(w, apiErr.Status, errorBody{
		Error: errorDetail{Code: apiErr.Code, Message: apiErr.Message},
	})
}
