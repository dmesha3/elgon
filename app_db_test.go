package elgon

import (
	"context"
	"database/sql"
	"testing"

	"github.com/meshackkazimoto/elgon/db"
)

type noopResult struct{}

func (noopResult) LastInsertId() (int64, error) { return 0, nil }
func (noopResult) RowsAffected() (int64, error) { return 0, nil }

type noopRows struct{}

func (noopRows) Close() error           { return nil }
func (noopRows) Err() error             { return nil }
func (noopRows) Next() bool             { return false }
func (noopRows) Scan(dest ...any) error { return nil }

type noopAdapter struct{}

func (noopAdapter) ExecContext(context.Context, string, ...any) (db.Result, error) {
	return noopResult{}, nil
}
func (noopAdapter) QueryContext(context.Context, string, ...any) (db.Rows, error) {
	return noopRows{}, nil
}
func (noopAdapter) BeginTx(context.Context, *sql.TxOptions) (db.Tx, error) { return nil, nil }
func (noopAdapter) PingContext(context.Context) error                      { return nil }
func (noopAdapter) Close() error                                           { return nil }

func TestAppSQLAndORMAccessors(t *testing.T) {
	app := New(Config{DisableHealthz: true})
	if app.SQL() != nil {
		t.Fatal("expected nil SQL adapter by default")
	}
	if app.ORM() != nil {
		t.Fatal("expected nil ORM client when SQL adapter is not configured")
	}

	adapter := noopAdapter{}
	app.SetSQL(adapter)
	if app.SQL() == nil {
		t.Fatal("expected SQL adapter after SetSQL")
	}

	ormOne := app.ORM()
	if ormOne == nil {
		t.Fatal("expected ORM client after SetSQL")
	}
	ormTwo := app.ORM()
	if ormOne != ormTwo {
		t.Fatal("expected cached ORM client instance")
	}

	app.SetORMDialect("postgres")
	ormThree := app.ORM()
	if ormThree == nil {
		t.Fatal("expected ORM client after dialect update")
	}
	if ormThree == ormOne {
		t.Fatal("expected ORM client to refresh after dialect update")
	}

	app.SetSQL(nil)
	if app.ORM() != nil {
		t.Fatal("expected nil ORM client after clearing SQL adapter")
	}
}
