package migrate

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dmesha3/elgon/db"
)

type fakeAdapter struct {
	applied map[int]bool
}

type fakeResult struct{}

func (fakeResult) LastInsertId() (int64, error) { return 0, nil }
func (fakeResult) RowsAffected() (int64, error) { return 1, nil }

type fakeRows struct {
	versions []int
	idx      int
}

func (r *fakeRows) Close() error           { return nil }
func (r *fakeRows) Err() error             { return nil }
func (r *fakeRows) Next() bool             { r.idx++; return r.idx <= len(r.versions) }
func (r *fakeRows) Scan(dest ...any) error { *(dest[0].(*int)) = r.versions[r.idx-1]; return nil }

type fakeTx struct {
	parent *fakeAdapter
}

func (t *fakeTx) ExecContext(_ context.Context, query string, args ...any) (db.Result, error) {
	if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(query)), "INSERT INTO") {
		if len(args) > 0 {
			if v, ok := args[0].(int); ok {
				t.parent.applied[v] = true
			}
		}
	}
	if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(query)), "DELETE FROM") {
		if len(args) > 0 {
			if v, ok := args[0].(int); ok {
				delete(t.parent.applied, v)
			}
		}
	}
	return fakeResult{}, nil
}
func (t *fakeTx) QueryContext(_ context.Context, _ string, _ ...any) (db.Rows, error) {
	return &fakeRows{}, nil
}
func (t *fakeTx) Commit() error   { return nil }
func (t *fakeTx) Rollback() error { return nil }

func (f *fakeAdapter) ExecContext(_ context.Context, _ string, _ ...any) (db.Result, error) {
	if f.applied == nil {
		f.applied = map[int]bool{}
	}
	return fakeResult{}, nil
}
func (f *fakeAdapter) QueryContext(_ context.Context, _ string, _ ...any) (db.Rows, error) {
	versions := make([]int, 0, len(f.applied))
	for v := range f.applied {
		versions = append(versions, v)
	}
	return &fakeRows{versions: versions}, nil
}
func (f *fakeAdapter) BeginTx(_ context.Context, _ *sql.TxOptions) (db.Tx, error) {
	if f.applied == nil {
		f.applied = map[int]bool{}
	}
	return &fakeTx{parent: f}, nil
}
func (f *fakeAdapter) PingContext(_ context.Context) error { return nil }
func (f *fakeAdapter) Close() error                        { return nil }

func TestParseMigrationFileName(t *testing.T) {
	v, name, dirn, ok := ParseMigrationFileName("0001_init.up.sql", "")
	if !ok || v != 1 || name != "init" || dirn != "up" {
		t.Fatalf("unexpected parse result: %v %v %v %v", v, name, dirn, ok)
	}
	_, _, _, ok = ParseMigrationFileName("0001_init.pg.up.sql", "mysql")
	if ok {
		t.Fatal("expected dialect mismatch to skip file")
	}
}

func TestLoadMigrations(t *testing.T) {
	d := t.TempDir()
	up := filepath.Join(d, "0001_init.up.sql")
	down := filepath.Join(d, "0001_init.down.sql")
	if err := osWrite(up, "CREATE TABLE t(x int);"); err != nil {
		t.Fatal(err)
	}
	if err := osWrite(down, "DROP TABLE t;"); err != nil {
		t.Fatal(err)
	}
	migs, err := Load(d, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(migs) != 1 || migs[0].Version != 1 {
		t.Fatalf("unexpected migrations: %+v", migs)
	}
}

func TestEngineUpDownStatus(t *testing.T) {
	adapter := &fakeAdapter{applied: map[int]bool{}}
	engine := NewEngine(adapter, "")
	migs := []Migration{{Version: 1, Name: "init", UpSQL: "CREATE", DownSQL: "DROP"}}

	ctx := context.Background()
	applied, err := engine.Up(ctx, migs, 0)
	if err != nil || applied != 1 {
		t.Fatalf("up failed applied=%d err=%v", applied, err)
	}
	status, err := engine.Status(ctx, migs)
	if err != nil || !status[0].Applied {
		t.Fatalf("status failed: %+v err=%v", status, err)
	}
	done, err := engine.Down(ctx, migs, 1)
	if err != nil || done != 1 {
		t.Fatalf("down failed done=%d err=%v", done, err)
	}
}

func osWrite(path string, content string) error {
	return os.WriteFile(path, []byte(content), 0o600)
}
