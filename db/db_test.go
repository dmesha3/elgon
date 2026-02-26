package db

import (
	"strings"
	"testing"
)

func TestPostgresDSN(t *testing.T) {
	cfg := PostgresConfig{
		Host: "localhost", Port: 5432, User: "user", Password: "pass", DBName: "elgon", SSLMode: "disable",
	}
	dsn := cfg.DSN()
	if !strings.Contains(dsn, "postgres://") || !strings.Contains(dsn, "sslmode=disable") {
		t.Fatalf("unexpected postgres dsn: %s", dsn)
	}
}

func TestMySQLDSN(t *testing.T) {
	cfg := MySQLConfig{User: "u", Password: "p", Host: "127.0.0.1", Port: 3306, DBName: "elgon", Params: map[string]string{"parseTime": "true"}}
	dsn := cfg.DSN()
	if !strings.Contains(dsn, "@tcp(127.0.0.1:3306)/elgon") {
		t.Fatalf("unexpected mysql dsn: %s", dsn)
	}
	if !strings.Contains(dsn, "parseTime=true") {
		t.Fatalf("missing mysql params: %s", dsn)
	}
}

func TestSQLiteDSN(t *testing.T) {
	cfg := SQLiteConfig{}
	if cfg.DSN() == "" {
		t.Fatal("expected sqlite memory dsn")
	}
}
