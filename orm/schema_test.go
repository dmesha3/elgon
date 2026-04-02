package orm

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"
)

type autoUser struct {
	ID        int64  `orm:"pk,autoincrement"`
	Email     string `orm:"size:255,notnull,unique"`
	Name      *string
	Nickname  sql.NullString
	CreatedAt time.Time `orm:"default:CURRENT_TIMESTAMP"`
	Ignored   string    `orm:"-"`
}

func (autoUser) TableName() string { return "users" }

type autoLog struct {
	ID      int64  `orm:"pk,autoincrement"`
	Message string `orm:"notnull"`
}

type schemaMeta struct{}

type elgonTodo struct {
	schemaMeta   `elgon:"table:todos,alias:t"`
	ID           string `elgon:"primary_key"`
	Title        string `elgon:"not_null,text"`
	IsCompleted  bool   `elgon:"bool"`
	Description  string `elgon:"not_null"`
	OptionalNote *string
}

type badEntity struct {
	Meta struct {
		Key string
	}
}

type badTableName struct {
	ID int64
}

func (badTableName) TableName() string { return "bad-name" }

func TestBuildCreateTableSQLSQLite(t *testing.T) {
	stmt, err := BuildCreateTableSQL(autoUser{}, "sqlite")
	if err != nil {
		t.Fatalf("build create table sql failed: %v", err)
	}
	want := "CREATE TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY AUTOINCREMENT, email VARCHAR(255) NOT NULL UNIQUE, name TEXT, nickname TEXT, created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP)"
	if stmt != want {
		t.Fatalf("unexpected sqlite create statement:\nwant: %s\ngot:  %s", want, stmt)
	}
}

func TestBuildCreateTableSQLPostgresAutoIncrement(t *testing.T) {
	stmt, err := BuildCreateTableSQL(autoLog{}, "postgres")
	if err != nil {
		t.Fatalf("build create table sql failed: %v", err)
	}
	if !strings.Contains(stmt, "id BIGSERIAL PRIMARY KEY NOT NULL") {
		t.Fatalf("expected BIGSERIAL pk for postgres, got %s", stmt)
	}
}

func TestBuildCreateTableSQLElgonTags(t *testing.T) {
	stmt, err := BuildCreateTableSQL(elgonTodo{}, "sqlite")
	if err != nil {
		t.Fatalf("build create table sql failed: %v", err)
	}
	want := "CREATE TABLE IF NOT EXISTS todos (id TEXT PRIMARY KEY NOT NULL, title TEXT NOT NULL, is_completed BOOLEAN, description TEXT NOT NULL, optional_note TEXT)"
	if stmt != want {
		t.Fatalf("unexpected elgon create statement:\nwant: %s\ngot:  %s", want, stmt)
	}
}

func TestBuildCreateTableSQLForTableOverride(t *testing.T) {
	entity := struct {
		ID int64 `orm:"pk,autoincrement"`
	}{}
	stmt, err := BuildCreateTableSQLForTable("app_users", entity, "sqlite")
	if err != nil {
		t.Fatalf("build create table sql with override failed: %v", err)
	}
	if !strings.HasPrefix(stmt, "CREATE TABLE IF NOT EXISTS app_users") {
		t.Fatalf("expected override table name, got %s", stmt)
	}
}

func TestBuildCreateTableSQLErrors(t *testing.T) {
	if _, err := BuildCreateTableSQL(nil, ""); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for nil entity, got %v", err)
	}

	if _, err := BuildCreateTableSQL(123, ""); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for non-struct entity, got %v", err)
	}

	if _, err := BuildCreateTableSQL(badEntity{}, ""); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for unsupported field type, got %v", err)
	}

	if _, err := BuildCreateTableSQL(badTableName{}, ""); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for invalid table name, got %v", err)
	}
}

func TestAutoMigrateUsesTransactionWhenAvailable(t *testing.T) {
	adapter := &fakeAdapter{}
	client := NewWithConfig(adapter, Config{Dialect: "sqlite"})

	if err := client.AutoMigrate(context.Background(), autoUser{}, autoLog{}); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}
	if adapter.commitCount != 1 {
		t.Fatalf("expected transaction commit, got %d", adapter.commitCount)
	}
	if adapter.rollbackCount != 0 {
		t.Fatalf("expected no rollback, got %d", adapter.rollbackCount)
	}
	if len(adapter.execQueries) != 2 {
		t.Fatalf("expected 2 create table statements, got %d", len(adapter.execQueries))
	}
	if !strings.HasPrefix(adapter.execQueries[0], "CREATE TABLE IF NOT EXISTS users") {
		t.Fatalf("expected users create statement, got %s", adapter.execQueries[0])
	}
}

func TestAutoMigrateFallbackWhenTxUnavailable(t *testing.T) {
	adapter := &fakeAdapter{noTx: true}
	client := New(adapter)

	if err := client.AutoMigrate(context.Background(), autoLog{}); err != nil {
		t.Fatalf("auto migrate fallback failed: %v", err)
	}
	if adapter.commitCount != 0 || adapter.rollbackCount != 0 {
		t.Fatalf("expected no tx calls in fallback, commits=%d rollbacks=%d", adapter.commitCount, adapter.rollbackCount)
	}
	if len(adapter.execQueries) != 1 {
		t.Fatalf("expected one create statement, got %d", len(adapter.execQueries))
	}
}

func TestAutoMigrateRollsBackOnError(t *testing.T) {
	adapter := &fakeAdapter{execErr: errors.New("boom")}
	client := New(adapter)

	err := client.AutoMigrate(context.Background(), autoLog{})
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected execution error to bubble up, got %v", err)
	}
	if adapter.rollbackCount != 1 {
		t.Fatalf("expected rollback on failure, got %d", adapter.rollbackCount)
	}
	if adapter.commitCount != 0 {
		t.Fatalf("expected no commit on failure, got %d", adapter.commitCount)
	}
}
