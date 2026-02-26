package db

import (
	"context"
	"fmt"
	"os"
	"testing"
)

// Integration tests run only when env vars are set:
// ELGON_DB_TEST_DRIVER and ELGON_DB_TEST_DSN.
func TestSQLAdapterIntegration(t *testing.T) {
	driver := os.Getenv("ELGON_DB_TEST_DRIVER")
	dsn := os.Getenv("ELGON_DB_TEST_DSN")
	if driver == "" || dsn == "" {
		t.Skip("set ELGON_DB_TEST_DRIVER and ELGON_DB_TEST_DSN to run integration tests")
	}

	adapter, err := Open(driver, dsn)
	if err != nil {
		t.Fatalf("open failed: %v", err)
	}
	defer adapter.Close()

	ctx := context.Background()
	if err := adapter.PingContext(ctx); err != nil {
		t.Fatalf("ping failed: %v", err)
	}

	table := "elgon_integration_tmp"
	_ = execIgnoreErr(ctx, adapter, fmt.Sprintf("DROP TABLE IF EXISTS %s", table))
	if _, err := adapter.ExecContext(ctx, fmt.Sprintf("CREATE TABLE %s (id INTEGER, name TEXT)", table)); err != nil {
		t.Fatalf("create table failed: %v", err)
	}
	if _, err := adapter.ExecContext(ctx, fmt.Sprintf("INSERT INTO %s (id, name) VALUES (1, 'alpha')", table)); err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	rows, err := adapter.QueryContext(ctx, fmt.Sprintf("SELECT id, name FROM %s", table))
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	defer rows.Close()
	if !rows.Next() {
		t.Fatal("expected one row")
	}
	var id int
	var name string
	if err := rows.Scan(&id, &name); err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	if id != 1 || name != "alpha" {
		t.Fatalf("unexpected row id=%d name=%s", id, name)
	}

	tx, err := adapter.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin tx failed: %v", err)
	}
	if _, err := tx.ExecContext(ctx, fmt.Sprintf("INSERT INTO %s (id, name) VALUES (2, 'beta')", table)); err != nil {
		_ = tx.Rollback()
		t.Fatalf("tx insert failed: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("tx commit failed: %v", err)
	}

	_ = execIgnoreErr(ctx, adapter, fmt.Sprintf("DROP TABLE IF EXISTS %s", table))
}

func execIgnoreErr(ctx context.Context, a *SQLAdapter, query string) error {
	_, err := a.ExecContext(ctx, query)
	return err
}
