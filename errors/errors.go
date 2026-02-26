package errors

import "github.com/meshackkazimoto/elgon"

func BadRequest(msg string, details any) error { return elgon.ErrBadRequest(msg, details) }
func Unauthorized(msg string) error            { return elgon.ErrUnauthorized(msg) }
func Forbidden(msg string) error               { return elgon.ErrForbidden(msg) }
func NotFound(msg string) error                { return elgon.ErrNotFound(msg) }
func Conflict(msg string) error                { return elgon.ErrConflict(msg) }
func Internal(msg string) error                { return elgon.ErrInternal(msg) }
