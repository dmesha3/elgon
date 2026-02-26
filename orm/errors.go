package orm

import "errors"

var (
	// ErrNotFound is returned when a record does not exist.
	ErrNotFound = errors.New("orm: record not found")
	// ErrInvalidInput is returned when a create/update input is invalid.
	ErrInvalidInput = errors.New("orm: invalid input")
	// ErrNonUnique is returned when a unique lookup matches multiple rows.
	ErrNonUnique = errors.New("orm: non-unique result")
)
