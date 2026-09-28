package errors

import (
	stderrors "errors"
	"fmt"
	"net/http"
	"strings"
)

const (
	CodeBadRequest          = "BAD_REQUEST"
	CodeValidation          = "VALIDATION_ERROR"
	CodeUnauthorized        = "UNAUTHORIZED"
	CodeForbidden           = "FORBIDDEN"
	CodeNotFound            = "NOT_FOUND"
	CodeConflict            = "CONFLICT"
	CodeUnprocessableEntity = "UNPROCESSABLE_ENTITY"
	CodeTooManyRequests     = "TOO_MANY_REQUESTS"
	CodeInternal            = "INTERNAL_ERROR"
	CodeServiceUnavailable  = "SERVICE_UNAVAILABLE"
)

// APIError represents an error that may safely cross the HTTP boundary.
//
// Cause is deliberately kept private so database/internal error details
// are never accidentally serialized to clients.
type APIError struct {
	Status  int
	Code    string
	Message string
	Details any

	cause error
}

func New(
	status int,
	code string,
	message string,
) *APIError {
	status = normalizeStatus(
		status,
	)

	code = strings.TrimSpace(
		code,
	)
	if code == "" {
		code = defaultCodeForStatus(
			status,
		)
	}

	message = strings.TrimSpace(
		message,
	)
	if message == "" {
		message = http.StatusText(
			status,
		)
	}

	return &APIError{
		Status:  status,
		Code:    code,
		Message: message,
	}
}

func Wrap(
	err error,
	status int,
	code string,
	message string,
) *APIError {
	result := New(
		status,
		code,
		message,
	)

	result.cause = err

	return result
}

func (e *APIError) Error() string {
	if e == nil {
		return ""
	}

	if e.cause != nil {
		return fmt.Sprintf(
			"%s: %v",
			e.Code,
			e.cause,
		)
	}

	if e.Message != "" {
		return e.Message
	}

	return e.Code
}

func (e *APIError) Unwrap() error {
	if e == nil {
		return nil
	}

	return e.cause
}

func (e *APIError) Cause() error {
	if e == nil {
		return nil
	}

	return e.cause
}

// WithDetails returns a copy rather than modifying e.
//
// This makes shared/sentinel API errors safe to reuse without leaking
// request-specific validation details between requests.
func (e *APIError) WithDetails(
	details any,
) *APIError {
	if e == nil {
		return nil
	}

	result := *e
	result.Details = details

	return &result
}

func As(
	err error,
) (*APIError, bool) {
	if err == nil {
		return nil, false
	}

	var target *APIError

	if !stderrors.As(
		err,
		&target,
	) {
		return nil, false
	}

	return target, true
}

// Normalize preserves an existing APIError.
//
// Unknown errors become a safe generic 500 response. Their original
// error remains available through Unwrap/Cause for structured logging.
func Normalize(
	err error,
) *APIError {
	if err == nil {
		return nil
	}

	if apiErr, ok := As(
		err,
	); ok {
		return apiErr
	}

	return Internal(
		err,
	)
}

func IsCode(
	err error,
	code string,
) bool {
	apiErr, ok := As(
		err,
	)
	if !ok {
		return false
	}

	return apiErr.Code == strings.TrimSpace(
		code,
	)
}

func BadRequest(
	code string,
	message string,
) *APIError {
	return New(
		http.StatusBadRequest,
		code,
		message,
	)
}

func Validation(
	message string,
	details any,
) *APIError {
	return New(
		http.StatusBadRequest,
		CodeValidation,
		message,
	).WithDetails(
		details,
	)
}

func Unauthorized(
	code string,
	message string,
) *APIError {
	return New(
		http.StatusUnauthorized,
		code,
		message,
	)
}

func Forbidden(
	code string,
	message string,
) *APIError {
	return New(
		http.StatusForbidden,
		code,
		message,
	)
}

func NotFound(
	code string,
	message string,
) *APIError {
	return New(
		http.StatusNotFound,
		code,
		message,
	)
}

func Conflict(
	code string,
	message string,
) *APIError {
	return New(
		http.StatusConflict,
		code,
		message,
	)
}

func Unprocessable(
	code string,
	message string,
) *APIError {
	return New(
		http.StatusUnprocessableEntity,
		code,
		message,
	)
}

func TooManyRequests(
	code string,
	message string,
) *APIError {
	return New(
		http.StatusTooManyRequests,
		code,
		message,
	)
}

func Internal(
	err error,
) *APIError {
	return Wrap(
		err,
		http.StatusInternalServerError,
		CodeInternal,
		"Internal server error",
	)
}

func ServiceUnavailable(
	err error,
) *APIError {
	return Wrap(
		err,
		http.StatusServiceUnavailable,
		CodeServiceUnavailable,
		"Service temporarily unavailable",
	)
}

func normalizeStatus(
	status int,
) int {
	if status < 400 ||
		status > 599 {
		return http.StatusInternalServerError
	}

	return status
}

func defaultCodeForStatus(
	status int,
) string {
	switch status {
	case http.StatusBadRequest:
		return CodeBadRequest

	case http.StatusUnauthorized:
		return CodeUnauthorized

	case http.StatusForbidden:
		return CodeForbidden

	case http.StatusNotFound:
		return CodeNotFound

	case http.StatusConflict:
		return CodeConflict

	case http.StatusUnprocessableEntity:
		return CodeUnprocessableEntity

	case http.StatusTooManyRequests:
		return CodeTooManyRequests

	case http.StatusServiceUnavailable:
		return CodeServiceUnavailable

	default:
		return CodeInternal
	}
}
