package orm

import "errors"

var (
	// ErrNotFound is returned when a record does not exist.
	ErrNotFound = errors.New("orm: record not found")
	// ErrInvalidInput is returned when a create/update input is invalid.
	ErrInvalidInput = errors.New("orm: invalid input")
)
