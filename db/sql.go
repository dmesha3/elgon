package db

import (
	"context"
	"database/sql"
	"fmt"
)

// SQLAdapter wraps *sql.DB to satisfy Adapter.
type SQLAdapter struct {
	db *sql.DB
}

func NewSQLAdapter(db *sql.DB) *SQLAdapter {
	return &SQLAdapter{db: db}
}

// Open opens a DB adapter for a registered database/sql driver.
func Open(driver, dsn string) (*SQLAdapter, error) {
	sqldb, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, err
	}
	return NewSQLAdapter(sqldb), nil
}

func (a *SQLAdapter) ExecContext(ctx context.Context, query string, args ...any) (Result, error) {
	return a.db.ExecContext(ctx, query, args...)
}

func (a *SQLAdapter) QueryContext(ctx context.Context, query string, args ...any) (Rows, error) {
	return a.db.QueryContext(ctx, query, args...)
}

func (a *SQLAdapter) BeginTx(ctx context.Context, opts *sql.TxOptions) (Tx, error) {
	tx, err := a.db.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &sqlTx{tx: tx}, nil
}

func (a *SQLAdapter) PingContext(ctx context.Context) error {
	return a.db.PingContext(ctx)
}

func (a *SQLAdapter) Close() error {
	return a.db.Close()
}

// SetPool configures SQL connection pool settings.
func (a *SQLAdapter) SetPool(maxOpen, maxIdle int) {
	a.db.SetMaxOpenConns(maxOpen)
	a.db.SetMaxIdleConns(maxIdle)
}

func (a *SQLAdapter) DB() *sql.DB {
	return a.db
}

func MustOpen(driver, dsn string) *SQLAdapter {
	a, err := Open(driver, dsn)
	if err != nil {
		panic(fmt.Errorf("db: open failed: %w", err))
	}
	return a
}

type sqlTx struct {
	tx *sql.Tx
}

func (t *sqlTx) ExecContext(ctx context.Context, query string, args ...any) (Result, error) {
	return t.tx.ExecContext(ctx, query, args...)
}

func (t *sqlTx) QueryContext(ctx context.Context, query string, args ...any) (Rows, error) {
	return t.tx.QueryContext(ctx, query, args...)
}

func (t *sqlTx) Commit() error {
	return t.tx.Commit()
}

func (t *sqlTx) Rollback() error {
	return t.tx.Rollback()
}
