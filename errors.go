package elgon

import (
	stdErrors "errors"
)

const (
	CodeBadRequest   = "BAD_REQUEST"
	CodeUnauthorized = "UNAUTHORIZED"
	CodeForbidden    = "FORBIDDEN"
	CodeNotFound     = "NOT_FOUND"
	CodeConflict     = "CONFLICT"
	CodeInternal     = "INTERNAL_ERROR"
)

// HTTPError is a typed application error mapped by the central handler.
type HTTPError struct {
	Code    string
	Message string
	Details any
	Status  int
}

func (e *HTTPError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func newHTTPError(code, message string, details any, status int) error {
	return &HTTPError{Code: code, Message: message, Details: details, Status: status}
}

func ErrBadRequest(msg string, details any) error {
	return newHTTPError(CodeBadRequest, msg, details, 400)
}

func ErrUnauthorized(msg string) error {
	return newHTTPError(CodeUnauthorized, msg, nil, 401)
}

func ErrForbidden(msg string) error {
	return newHTTPError(CodeForbidden, msg, nil, 403)
}

func ErrNotFound(msg string) error {
	return newHTTPError(CodeNotFound, msg, nil, 404)
}

func ErrConflict(msg string) error {
	return newHTTPError(CodeConflict, msg, nil, 409)
}

func ErrInternal(msg string) error {
	return newHTTPError(CodeInternal, msg, nil, 500)
}

func asHTTPError(err error) (*HTTPError, bool) {
	var he *HTTPError
	ok := stdErrors.As(err, &he)
	return he, ok
}
