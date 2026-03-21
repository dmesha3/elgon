package orm

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/dmesha3/elgon/db"
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
		ptr, ok := dest[i].(*any)
		if !ok {
			return errors.New("unsupported scan destination")
		}
		*ptr = row[i]
	}
	return nil
}
func (r *fakeRows) Columns() ([]string, error) { return r.cols, nil }

type noColumnsRows struct {
	vals [][]any
	i    int
}

func (r *noColumnsRows) Close() error { return nil }
func (r *noColumnsRows) Err() error   { return nil }
func (r *noColumnsRows) Next() bool {
	r.i++
	return r.i <= len(r.vals)
}
func (r *noColumnsRows) Scan(dest ...any) error {
	row := r.vals[r.i-1]
	for i := range dest {
		ptr := dest[i].(*any)
		*ptr = row[i]
	}
	return nil
}

type fakeTx struct {
	parent *fakeAdapter
}

func (tx *fakeTx) ExecContext(ctx context.Context, query string, args ...any) (db.Result, error) {
	return tx.parent.ExecContext(ctx, query, args...)
}
func (tx *fakeTx) QueryContext(ctx context.Context, query string, args ...any) (db.Rows, error) {
	return tx.parent.QueryContext(ctx, query, args...)
}
func (tx *fakeTx) Commit() error {
	tx.parent.commitCount++
	return nil
}
func (tx *fakeTx) Rollback() error {
	tx.parent.rollbackCount++
	return nil
}

type fakeAdapter struct {
	execQueries   []string
	execArgs      [][]any
	queryStmts    []string
	queryArgs     [][]any
	rowsQueue     []db.Rows
	execErr       error
	queryErr      error
	noTx          bool
	commitCount   int
	rollbackCount int
}

func (f *fakeAdapter) ExecContext(_ context.Context, query string, args ...any) (db.Result, error) {
	f.execQueries = append(f.execQueries, query)
	f.execArgs = append(f.execArgs, args)
	if f.execErr != nil {
		return nil, f.execErr
	}
	return fakeResult{affected: 2}, nil
}

func (f *fakeAdapter) QueryContext(_ context.Context, query string, args ...any) (db.Rows, error) {
	f.queryStmts = append(f.queryStmts, query)
	f.queryArgs = append(f.queryArgs, args)
	if f.queryErr != nil {
		return nil, f.queryErr
	}
	if len(f.rowsQueue) == 0 {
		return &fakeRows{}, nil
	}
	row := f.rowsQueue[0]
	f.rowsQueue = f.rowsQueue[1:]
	return row, nil
}

func (f *fakeAdapter) BeginTx(context.Context, *sql.TxOptions) (db.Tx, error) {
	if f.noTx {
		return nil, nil
	}
	return &fakeTx{parent: f}, nil
}
func (f *fakeAdapter) PingContext(context.Context) error { return nil }
func (f *fakeAdapter) Close() error                      { return nil }

func TestFindManyFirstAndFirstOrThrow(t *testing.T) {
	adapter := &fakeAdapter{
		rowsQueue: []db.Rows{
			&fakeRows{
				cols: []string{"id", "email"},
				vals: [][]any{{"u1", []byte("a@b.com")}, {"u2", "b@b.com"}},
			},
			&fakeRows{
				cols: []string{"id", "email"},
				vals: [][]any{{"u1", "a@b.com"}},
			},
			&fakeRows{
				cols: []string{"id"},
				vals: nil,
			},
		},
	}
	table := New(adapter).Table("users")

	rows, err := table.FindMany(context.Background(), FindOptions{
		Columns: []string{"id", "email"},
		Where:   Where{"email": "a@b.com"},
		OrderBy: []OrderBy{{Column: "id"}},
		Limit:   10,
	})
	if err != nil {
		t.Fatalf("find many failed: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[0]["email"] != "a@b.com" {
		t.Fatalf("expected []byte normalization, got %#v", rows[0]["email"])
	}
	if !strings.Contains(adapter.queryStmts[0], "ORDER BY id ASC") {
		t.Fatalf("expected order clause, got %s", adapter.queryStmts[0])
	}

	first, err := table.FindFirst(context.Background(), FindOptions{
		Columns: []string{"id", "email"},
		Where:   Where{"email": "a@b.com"},
	})
	if err != nil {
		t.Fatalf("find first failed: %v", err)
	}
	if first["id"] != "u1" {
		t.Fatalf("unexpected first row: %#v", first)
	}
	if !strings.Contains(adapter.queryStmts[1], "LIMIT ?") {
		t.Fatalf("expected limit clause, got %s", adapter.queryStmts[1])
	}

	_, err = table.FindFirstOrThrow(context.Background(), FindOptions{Columns: []string{"id"}})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestFindUniqueMethods(t *testing.T) {
	adapter := &fakeAdapter{
		rowsQueue: []db.Rows{
			&fakeRows{cols: []string{"id"}, vals: [][]any{{"u1"}}},
			&fakeRows{cols: []string{"id"}, vals: nil},
			&fakeRows{cols: []string{"id"}, vals: [][]any{{"u1"}, {"u2"}}},
			&fakeRows{cols: []string{"id"}, vals: nil},
		},
	}
	table := New(adapter).Table("users")

	row, err := table.FindUnique(context.Background(), Where{"id": "u1"}, "id")
	if err != nil {
		t.Fatalf("find unique failed: %v", err)
	}
	if row["id"] != "u1" {
		t.Fatalf("unexpected unique row: %#v", row)
	}

	row, err = table.FindUnique(context.Background(), Where{"id": "missing"}, "id")
	if err != nil {
		t.Fatalf("find unique missing failed: %v", err)
	}
	if row != nil {
		t.Fatalf("expected nil row for missing unique, got %#v", row)
	}

	_, err = table.FindUnique(context.Background(), Where{"email": "dup@x.com"}, "id")
	if !errors.Is(err, ErrNonUnique) {
		t.Fatalf("expected ErrNonUnique, got %v", err)
	}

	_, err = table.FindUniqueOrThrow(context.Background(), Where{"id": "missing"}, "id")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestCreateCreateManyAndCreateManyAndReturn(t *testing.T) {
	adapter := &fakeAdapter{}
	table := New(adapter).Table("users")

	if _, err := table.Create(context.Background(), Values{"id": "u1", "email": "a@b.com"}); err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if got := adapter.execQueries[0]; got != "INSERT INTO users (email, id) VALUES (?, ?)" {
		t.Fatalf("unexpected create query: %s", got)
	}

	count, err := table.CreateMany(context.Background(), []Values{
		{"id": "u2", "email": "b@b.com"},
		{"id": "u3", "email": "c@b.com"},
	})
	if err != nil {
		t.Fatalf("create many failed: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected create many count 2, got %d", count)
	}
	if adapter.commitCount != 1 {
		t.Fatalf("expected tx commit for create many")
	}

	pgAdapter := &fakeAdapter{
		rowsQueue: []db.Rows{
			&fakeRows{cols: []string{"id", "email"}, vals: [][]any{{"u4", "d@b.com"}}},
			&fakeRows{cols: []string{"id", "email"}, vals: [][]any{{"u5", "e@b.com"}}},
		},
	}
	pgTable := NewWithConfig(pgAdapter, Config{Dialect: "postgres"}).Table("users")
	created, err := pgTable.CreateManyAndReturn(context.Background(), []Values{
		{"id": "u4", "email": "d@b.com"},
		{"id": "u5", "email": "e@b.com"},
	}, []string{"id", "email"})
	if err != nil {
		t.Fatalf("create many and return failed: %v", err)
	}
	if len(created) != 2 {
		t.Fatalf("expected 2 returned rows, got %d", len(created))
	}
	if !strings.Contains(pgAdapter.queryStmts[0], "RETURNING id, email") {
		t.Fatalf("expected returning clause, got %s", pgAdapter.queryStmts[0])
	}
}

func TestUpdateDeleteAndManyVariants(t *testing.T) {
	adapter := &fakeAdapter{
		rowsQueue: []db.Rows{
			&fakeRows{
				cols: []string{"id", "name"},
				vals: [][]any{{"u1", "Updated"}, {"u2", "Updated"}},
			},
		},
	}
	table := NewWithConfig(adapter, Config{Dialect: "postgres"}).Table("users")

	count, err := table.Update(context.Background(), Where{"id": "u1"}, Values{"name": "New"})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected fake affected count 2, got %d", count)
	}
	if !strings.HasPrefix(adapter.execQueries[0], "UPDATE users SET") {
		t.Fatalf("expected update query, got %s", adapter.execQueries[0])
	}

	count, err = table.UpdateMany(context.Background(), Where{"id": "u1"}, Values{"name": "N2"})
	if err != nil {
		t.Fatalf("update many failed: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected update many count 2, got %d", count)
	}

	rows, err := table.UpdateManyAndReturn(context.Background(), Where{"id": "u1"}, Values{"name": "Updated"}, []string{"id", "name"})
	if err != nil {
		t.Fatalf("update many and return failed: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 updated rows, got %d", len(rows))
	}
	if !strings.Contains(adapter.queryStmts[0], "RETURNING id, name") {
		t.Fatalf("expected returning clause, got %s", adapter.queryStmts[0])
	}

	count, err = table.Delete(context.Background(), Where{"id": "u1"})
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected delete count 2, got %d", count)
	}

	count, err = table.DeleteMany(context.Background(), Where{"id": "u2"})
	if err != nil {
		t.Fatalf("delete many failed: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected delete many count 2, got %d", count)
	}
}

func TestUpsert(t *testing.T) {
	existingAdapter := &fakeAdapter{
		rowsQueue: []db.Rows{
			&fakeRows{cols: []string{"id"}, vals: [][]any{{"u1"}}},
			&fakeRows{cols: []string{"id", "name"}, vals: [][]any{{"u1", "Updated"}}},
		},
	}
	table := New(existingAdapter).Table("users")
	row, err := table.Upsert(
		context.Background(),
		Where{"id": "u1"},
		Values{"name": "Created"},
		Values{"name": "Updated"},
	)
	if err != nil {
		t.Fatalf("upsert update-path failed: %v", err)
	}
	if row["name"] != "Updated" {
		t.Fatalf("unexpected upsert updated row: %#v", row)
	}
	if len(existingAdapter.execQueries) == 0 || !strings.Contains(existingAdapter.execQueries[0], "UPDATE users SET") {
		t.Fatalf("expected update in upsert update path, got %#v", existingAdapter.execQueries)
	}

	createAdapter := &fakeAdapter{
		rowsQueue: []db.Rows{
			&fakeRows{cols: []string{"id"}, vals: nil},
			&fakeRows{cols: []string{"id", "name"}, vals: [][]any{{"u2", "Created"}}},
		},
	}
	table = New(createAdapter).Table("users")
	row, err = table.Upsert(
		context.Background(),
		Where{"id": "u2"},
		Values{"name": "Created"},
		Values{"name": "Updated"},
	)
	if err != nil {
		t.Fatalf("upsert create-path failed: %v", err)
	}
	if row["name"] != "Created" {
		t.Fatalf("unexpected upsert created row: %#v", row)
	}
	if len(createAdapter.execQueries) == 0 || !strings.Contains(createAdapter.execQueries[0], "INSERT INTO users") {
		t.Fatalf("expected insert in upsert create path, got %#v", createAdapter.execQueries)
	}
}

func TestFallbackPathsAndValidation(t *testing.T) {
	adapter := &fakeAdapter{noTx: true}
	table := New(adapter).Table("users")

	count, err := table.CreateMany(context.Background(), []Values{{"id": "u1"}, {"id": "u2"}})
	if err != nil {
		t.Fatalf("create many fallback failed: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected create many fallback count 2, got %d", count)
	}

	mysqlAdapter := &fakeAdapter{
		rowsQueue: []db.Rows{
			&noColumnsRows{
				vals: [][]any{{"u1", "Alpha"}},
			},
			&noColumnsRows{
				vals: [][]any{{"u1", "Alpha"}},
			},
		},
	}
	mysql := NewWithConfig(mysqlAdapter, Config{Dialect: "mysql"}).Table("users")
	created, err := mysql.CreateManyAndReturn(context.Background(), []Values{{"id": "u1", "name": "Alpha"}}, []string{"id", "name"})
	if err != nil {
		t.Fatalf("fallback create many and return failed: %v", err)
	}
	if len(created) != 1 || created[0]["id"] != "u1" {
		t.Fatalf("unexpected fallback create many return: %#v", created)
	}

	updated, err := mysql.UpdateManyAndReturn(context.Background(), Where{"id": "u1"}, Values{"name": "Beta"}, []string{"id", "name"})
	if err != nil {
		t.Fatalf("fallback update many and return failed: %v", err)
	}
	if len(updated) != 1 || updated[0]["name"] != "Beta" {
		t.Fatalf("unexpected fallback update many return: %#v", updated)
	}

	_, err = table.FindUnique(context.Background(), nil, "id")
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid input for empty unique where, got %v", err)
	}
	_, err = table.Update(context.Background(), nil, Values{"name": "x"})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid input for update with empty where, got %v", err)
	}
	_, err = table.Delete(context.Background(), nil)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid input for delete with empty where, got %v", err)
	}

	_, err = New(table.db).Table("users;drop").FindMany(context.Background(), FindOptions{Columns: []string{"id"}})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid identifier error, got %v", err)
	}
}

func TestPostgresPlaceholder(t *testing.T) {
	adapter := &fakeAdapter{
		rowsQueue: []db.Rows{
			&fakeRows{
				cols: []string{"id"},
				vals: [][]any{{"u1"}},
			},
		},
	}
	table := NewWithConfig(adapter, Config{Dialect: "postgres"}).Table("users")

	_, _ = table.FindMany(context.Background(), FindOptions{
		Columns: []string{"id"},
		Where:   Where{"id": "u1"},
		Limit:   1,
	})
	if len(adapter.queryStmts) == 0 || !strings.Contains(adapter.queryStmts[0], "$1") {
		t.Fatalf("expected postgres placeholder in query, got %q", strings.Join(adapter.queryStmts, " | "))
	}
}

func TestScanRecordsErrorsWithoutColumns(t *testing.T) {
	adapter := &fakeAdapter{
		rowsQueue: []db.Rows{&noColumnsRows{vals: [][]any{{"u1"}}}},
	}
	table := New(adapter).Table("users")
	_, err := table.FindMany(context.Background(), FindOptions{})
	if err == nil || !strings.Contains(err.Error(), "does not expose columns") {
		t.Fatalf("expected columns exposure error, got %v", err)
	}
}

func TestReturningClauseValidation(t *testing.T) {
	_, err := buildReturningClause([]string{"id", "bad-column"})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid returning column, got %v", err)
	}
}

func TestProjectRecordsValidation(t *testing.T) {
	_, err := projectRecords([]Record{{"id": "u1"}}, []string{"id", "bad-column"})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid projection column, got %v", err)
	}
}

func TestCreateAndReturnNoRow(t *testing.T) {
	adapter := &fakeAdapter{
		rowsQueue: []db.Rows{
			&fakeRows{cols: []string{"id"}, vals: nil},
		},
	}
	table := NewWithConfig(adapter, Config{Dialect: "postgres"}).Table("users")
	_, err := table.CreateManyAndReturn(context.Background(), []Values{{"id": "u1"}}, []string{"id"})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound for no returning row, got %v", err)
	}
}

func TestBuildQueriesDeterministicColumns(t *testing.T) {
	adapter := &fakeAdapter{}
	table := New(adapter).Table("users")
	_, err := table.Create(context.Background(), Values{"z": 1, "a": 2})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if got := adapter.execQueries[0]; got != "INSERT INTO users (a, z) VALUES (?, ?)" {
		t.Fatalf("unexpected deterministic query order: %s", got)
	}
}

func TestQueryErrorPassthrough(t *testing.T) {
	adapter := &fakeAdapter{queryErr: fmt.Errorf("boom")}
	table := New(adapter).Table("users")
	_, err := table.FindMany(context.Background(), FindOptions{Columns: []string{"id"}})
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected passthrough query error, got %v", err)
	}
}

func TestWhereOperatorsAndLogicalCombinators(t *testing.T) {
	adapter := &fakeAdapter{
		rowsQueue: []db.Rows{
			&fakeRows{cols: []string{"id"}, vals: [][]any{{"u1"}}},
		},
	}
	table := New(adapter).Table("users")

	_, err := table.FindMany(context.Background(), FindOptions{
		Columns: []string{"id"},
		Where: Where{
			"OR": []any{
				Where{"email": map[string]any{"endsWith": "gmail.com"}},
				Where{"email": map[string]any{"endsWith": "company.com"}},
			},
			"NOT": Where{
				"email": map[string]any{"endsWith": "admin.company.com"},
			},
			"age": map[string]any{"gte": 18},
		},
	})
	if err != nil {
		t.Fatalf("find many with logical operators failed: %v", err)
	}
	q := adapter.queryStmts[0]
	if !strings.Contains(q, "NOT (email LIKE ?)") {
		t.Fatalf("expected NOT expression in query, got %s", q)
	}
	if !strings.Contains(q, "(email LIKE ? OR email LIKE ?)") {
		t.Fatalf("expected OR expression in query, got %s", q)
	}
	if !strings.Contains(q, "age >= ?") {
		t.Fatalf("expected gte expression in query, got %s", q)
	}
	if len(adapter.queryArgs[0]) != 4 {
		t.Fatalf("expected 4 args for logical operator query, got %d", len(adapter.queryArgs[0]))
	}
}

func TestWhereScalarOperators(t *testing.T) {
	adapter := &fakeAdapter{
		rowsQueue: []db.Rows{
			&fakeRows{cols: []string{"id"}, vals: [][]any{{"u1"}}},
		},
	}
	table := New(adapter).Table("users")

	_, err := table.FindMany(context.Background(), FindOptions{
		Columns: []string{"id"},
		Where: Where{
			"id": map[string]any{
				"in":    []string{"u1", "u2"},
				"notIn": []string{"u3"},
			},
			"email": map[string]any{"contains": "@example.com"},
			"name":  map[string]any{"startsWith": "Mes"},
		},
	})
	if err != nil {
		t.Fatalf("find many with scalar operators failed: %v", err)
	}
	q := adapter.queryStmts[0]
	if !strings.Contains(q, "id IN (?, ?)") {
		t.Fatalf("expected IN expression in query, got %s", q)
	}
	if !strings.Contains(q, "id NOT IN (?)") {
		t.Fatalf("expected NOT IN expression in query, got %s", q)
	}
	if !strings.Contains(q, "email LIKE ?") || !strings.Contains(q, "name LIKE ?") {
		t.Fatalf("expected LIKE expressions in query, got %s", q)
	}
}

func TestWhereMapEqualityBackwardsCompatible(t *testing.T) {
	adapter := &fakeAdapter{
		rowsQueue: []db.Rows{
			&fakeRows{cols: []string{"id"}, vals: [][]any{{"u1"}}},
		},
	}
	table := New(adapter).Table("users")
	jsonValue := map[string]any{"theme": "dark"}

	_, err := table.FindMany(context.Background(), FindOptions{
		Columns: []string{"id"},
		Where:   Where{"profile_json": jsonValue},
	})
	if err != nil {
		t.Fatalf("find many with map equality should still work: %v", err)
	}
	if !strings.Contains(adapter.queryStmts[0], "profile_json = ?") {
		t.Fatalf("expected equality fallback for map value, got %s", adapter.queryStmts[0])
	}
	if got := adapter.queryArgs[0][0]; !reflect.DeepEqual(got, jsonValue) {
		t.Fatalf("expected original map arg, got %#v", got)
	}
}

func TestUnsupportedAndInvalidOperators(t *testing.T) {
	adapter := &fakeAdapter{}
	table := New(adapter).Table("users")

	_, err := table.FindMany(context.Background(), FindOptions{
		Columns: []string{"id"},
		Where: Where{
			"photos": map[string]any{
				"some": map[string]any{"url": "2.jpg"},
			},
		},
	})
	if !errors.Is(err, ErrUnsupportedOperator) {
		t.Fatalf("expected ErrUnsupportedOperator for composite/list operator, got %v", err)
	}

	_, err = table.FindMany(context.Background(), FindOptions{
		Columns: []string{"id"},
		Where:   Where{"id": map[string]any{"unknown": 1}},
	})
	if err != nil {
		t.Fatalf("unknown nested key should fallback to equality, got %v", err)
	}

	_, err = table.FindMany(context.Background(), FindOptions{
		Columns: []string{"id"},
		Where:   Where{"id": map[string]any{"in": []string{}}},
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for empty in list, got %v", err)
	}
}
