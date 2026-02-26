package orm

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	"github.com/meshackkazimoto/elgon/db"
)

type fakeResult struct {
	affected int64
}

func (r fakeResult) LastInsertId() (int64, error) { return 0, nil }
func (r fakeResult) RowsAffected() (int64, error) { return r.affected, nil }

type fakeRows struct {
	cols []string
	vals [][]any
	i    int
}

func (r *fakeRows) Close() error { return nil }
func (r *fakeRows) Err() error   { return nil }
func (r *fakeRows) Next() bool {
	r.i++
	return r.i <= len(r.vals)
}
func (r *fakeRows) Scan(dest ...any) error {
	row := r.vals[r.i-1]
	for i := range dest {
		switch d := dest[i].(type) {
		case *any:
			*d = row[i]
		default:
			return errors.New("unsupported scan destination")
		}
	}
	return nil
}
func (r *fakeRows) Columns() ([]string, error) {
	return r.cols, nil
}

type fakeAdapter struct {
	execQuery string
	execArgs  []any
	queryStmt string
	queryArgs []any
	rows      db.Rows
	execErr   error
	queryErr  error
}

func (f *fakeAdapter) ExecContext(_ context.Context, query string, args ...any) (db.Result, error) {
	f.execQuery = query
	f.execArgs = args
	if f.execErr != nil {
		return nil, f.execErr
	}
	return fakeResult{affected: 2}, nil
}

func (f *fakeAdapter) QueryContext(_ context.Context, query string, args ...any) (db.Rows, error) {
	f.queryStmt = query
	f.queryArgs = args
	if f.queryErr != nil {
		return nil, f.queryErr
	}
	if f.rows == nil {
		return &fakeRows{}, nil
	}
	return f.rows, nil
}

func (f *fakeAdapter) BeginTx(context.Context, *sql.TxOptions) (db.Tx, error) { return nil, nil }
func (f *fakeAdapter) PingContext(context.Context) error                      { return nil }
func (f *fakeAdapter) Close() error                                           { return nil }

func TestTableCreateBuildsInsert(t *testing.T) {
	adapter := &fakeAdapter{}
	table := New(adapter).Table("users")

	_, err := table.Create(context.Background(), Values{
		"name":  "Meshack",
		"email": "meshack@example.com",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	want := "INSERT INTO users (email, name) VALUES (?, ?)"
	if adapter.execQuery != want {
		t.Fatalf("unexpected query\nwant: %s\ngot:  %s", want, adapter.execQuery)
	}
	if len(adapter.execArgs) != 2 {
		t.Fatalf("expected 2 args, got %d", len(adapter.execArgs))
	}
}

func TestTableFindManyMapsRows(t *testing.T) {
	adapter := &fakeAdapter{
		rows: &fakeRows{
			cols: []string{"id", "email", "name"},
			vals: [][]any{
				{"u1", []byte("a@b.com"), "Alpha"},
				{"u2", "c@d.com", "Beta"},
			},
		},
	}
	table := New(adapter).Table("users")

	rows, err := table.FindMany(context.Background(), FindOptions{
		Columns: []string{"id", "email", "name"},
		Where:   Where{"name": "Alpha"},
		OrderBy: []OrderBy{{Column: "id"}},
		Limit:   10,
		Offset:  0,
	})
	if err != nil {
		t.Fatalf("find many failed: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[0]["email"] != "a@b.com" {
		t.Fatalf("expected byte slice email normalization, got %#v", rows[0]["email"])
	}
	if !strings.Contains(adapter.queryStmt, "ORDER BY id ASC") {
		t.Fatalf("expected order by in query, got %s", adapter.queryStmt)
	}
}

func TestTableFindOne(t *testing.T) {
	adapter := &fakeAdapter{
		rows: &fakeRows{
			cols: []string{"id", "name"},
			vals: [][]any{{"u1", "Alpha"}},
		},
	}
	table := New(adapter).Table("users")

	row, err := table.FindOne(context.Background(), FindOptions{
		Columns: []string{"id", "name"},
		Where:   Where{"id": "u1"},
	})
	if err != nil {
		t.Fatalf("find one failed: %v", err)
	}
	if row["id"] != "u1" {
		t.Fatalf("unexpected row: %#v", row)
	}
	if !strings.Contains(adapter.queryStmt, "LIMIT ?") {
		t.Fatalf("expected limit 1 query, got %s", adapter.queryStmt)
	}
}

func TestTableFindOneNotFound(t *testing.T) {
	adapter := &fakeAdapter{rows: &fakeRows{cols: []string{"id"}, vals: nil}}
	table := New(adapter).Table("users")

	_, err := table.FindOne(context.Background(), FindOptions{Columns: []string{"id"}})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestTableUpdateAndDelete(t *testing.T) {
	adapter := &fakeAdapter{}
	table := New(adapter).Table("users")

	affected, err := table.Update(context.Background(), Where{"id": "u1"}, Values{"name": "New"})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if affected != 2 {
		t.Fatalf("expected affected rows from fake result, got %d", affected)
	}
	if !strings.Contains(adapter.execQuery, "UPDATE users SET") {
		t.Fatalf("expected update query, got %s", adapter.execQuery)
	}

	affected, err = table.Delete(context.Background(), Where{"id": "u1"})
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if affected != 2 {
		t.Fatalf("expected affected rows from fake result, got %d", affected)
	}
	if !strings.HasPrefix(adapter.execQuery, "DELETE FROM users") {
		t.Fatalf("expected delete query, got %s", adapter.execQuery)
	}
}

func TestTableRejectsUnsafeWrites(t *testing.T) {
	table := New(&fakeAdapter{}).Table("users")

	if _, err := table.Update(context.Background(), nil, Values{"name": "N"}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid input for empty update where, got %v", err)
	}
	if _, err := table.Delete(context.Background(), nil); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid input for empty delete where, got %v", err)
	}
}

func TestTablePostgresPlaceholders(t *testing.T) {
	adapter := &fakeAdapter{}
	table := NewWithConfig(adapter, Config{Dialect: "postgres"}).Table("users")

	_, _ = table.FindMany(context.Background(), FindOptions{
		Columns: []string{"id"},
		Where:   Where{"id": "u1"},
		Limit:   1,
	})
	if !strings.Contains(adapter.queryStmt, "$1") {
		t.Fatalf("expected postgres placeholder in query, got %q", adapter.queryStmt)
	}
}

func TestTableRejectsInvalidIdentifier(t *testing.T) {
	table := New(&fakeAdapter{}).Table("users; DROP TABLE users")
	if _, err := table.FindMany(context.Background(), FindOptions{Columns: []string{"id"}}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid identifier error, got %v", err)
	}
}
