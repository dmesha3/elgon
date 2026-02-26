package db

import (
	"context"
	"database/sql"
)

// Result mirrors database/sql Result.
type Result interface {
	LastInsertId() (int64, error)
	RowsAffected() (int64, error)
}

// Rows mirrors database/sql Rows scan behavior.
type Rows interface {
	Close() error
	Err() error
	Next() bool
	Scan(dest ...any) error
}

// Tx represents a database transaction.
type Tx interface {
	ExecContext(ctx context.Context, query string, args ...any) (Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (Rows, error)
	Commit() error
	Rollback() error
}

// Adapter abstracts database operations without locking to one client.
type Adapter interface {
	ExecContext(ctx context.Context, query string, args ...any) (Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (Rows, error)
	BeginTx(ctx context.Context, opts *sql.TxOptions) (Tx, error)
	PingContext(ctx context.Context) error
	Close() error
}
