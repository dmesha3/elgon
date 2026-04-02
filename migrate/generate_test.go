package migrate

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestGenerateFromModelFiles(t *testing.T) {
	d := t.TempDir()
	modelFile := filepath.Join(d, "models.go")
	content := `package models

import (
	"database/sql"
	"time"
)

type User struct {
	ID        int64          ` + "`orm:\"pk,autoincrement\"`" + `
	Email     string         ` + "`orm:\"size:255,notnull,unique\"`" + `
	Nickname  sql.NullString
	CreatedAt time.Time      ` + "`orm:\"default:CURRENT_TIMESTAMP\"`" + `
	Ignored   string         ` + "`orm:\"-\"`" + `
}

type AuditLog struct {
	ID      int64  ` + "`orm:\"pk,autoincrement\"`" + `
	Message string ` + "`orm:\"notnull\"`" + `
}
`
	if err := os.WriteFile(modelFile, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	migrationsDir := filepath.Join(d, "migrations")
	gen, err := GenerateFromModelFiles(migrationsDir, "sqlite", "init users", []string{modelFile})
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	upBase := filepath.Base(gen.UpPath)
	downBase := filepath.Base(gen.DownPath)
	re := regexp.MustCompile(`^\d+_init_users_[0-9a-f]{6}\.(up|down)\.sql$`)
	if !re.MatchString(upBase) {
		t.Fatalf("unexpected up filename: %s", upBase)
	}
	if !re.MatchString(downBase) {
		t.Fatalf("unexpected down filename: %s", downBase)
	}

	upSQL, err := os.ReadFile(gen.UpPath)
	if err != nil {
		t.Fatal(err)
	}
	downSQL, err := os.ReadFile(gen.DownPath)
	if err != nil {
		t.Fatal(err)
	}

	upText := string(upSQL)
	if !strings.Contains(upText, "CREATE TABLE IF NOT EXISTS audit_log") {
		t.Fatalf("expected audit_log table in up migration, got:\n%s", upText)
	}
	if !strings.Contains(upText, "CREATE TABLE IF NOT EXISTS user") {
		t.Fatalf("expected user table in up migration, got:\n%s", upText)
	}
	if !strings.Contains(upText, "id INTEGER PRIMARY KEY AUTOINCREMENT") {
		t.Fatalf("expected sqlite autoincrement id, got:\n%s", upText)
	}

	downText := string(downSQL)
	if !strings.Contains(downText, "DROP TABLE IF EXISTS user;") {
		t.Fatalf("expected user drop in down migration, got:\n%s", downText)
	}
	if !strings.Contains(downText, "DROP TABLE IF EXISTS audit_log;") {
		t.Fatalf("expected audit_log drop in down migration, got:\n%s", downText)
	}
}

func TestGenerateFromModelFilesElgonTags(t *testing.T) {
	d := t.TempDir()
	modelFile := filepath.Join(d, "models.go")
	content := `package models

type baseModel struct{}

type Todo struct {
	baseModel ` + "`elgon:\"table:todos,alias:t\"`" + `
	ID          string ` + "`elgon:\"primary_key\"`" + `
	Title       string ` + "`elgon:\"not_null,text\"`" + `
	IsCompleted bool   ` + "`elgon:\"bool\"`" + `
}
`
	if err := os.WriteFile(modelFile, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	migrationsDir := filepath.Join(d, "migrations")
	gen, err := GenerateFromModelFiles(migrationsDir, "sqlite", "init todos", []string{modelFile})
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	upSQL, err := os.ReadFile(gen.UpPath)
	if err != nil {
		t.Fatal(err)
	}
	downSQL, err := os.ReadFile(gen.DownPath)
	if err != nil {
		t.Fatal(err)
	}

	upText := string(upSQL)
	if !strings.Contains(upText, "CREATE TABLE IF NOT EXISTS todos") {
		t.Fatalf("expected todos table in up migration, got:\n%s", upText)
	}
	if !strings.Contains(upText, "id TEXT PRIMARY KEY NOT NULL") {
		t.Fatalf("expected text primary key in up migration, got:\n%s", upText)
	}
	if !strings.Contains(upText, "is_completed BOOLEAN") {
		t.Fatalf("expected boolean column in up migration, got:\n%s", upText)
	}

	downText := string(downSQL)
	if !strings.Contains(downText, "DROP TABLE IF EXISTS todos;") {
		t.Fatalf("expected todos drop in down migration, got:\n%s", downText)
	}
}

func TestGenerateFromModelFilesErrors(t *testing.T) {
	if _, err := GenerateFromModelFiles("migrations", "sqlite", "x", nil); err == nil {
		t.Fatal("expected error for empty model files")
	}

	d := t.TempDir()
	modelFile := filepath.Join(d, "bad_models.go")
	content := `package models

type User struct {
	Meta map[string]string
}
`
	if err := os.WriteFile(modelFile, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := GenerateFromModelFiles(filepath.Join(d, "migrations"), "sqlite", "x", []string{modelFile})
	if err == nil || !strings.Contains(err.Error(), "unsupported type expression") {
		t.Fatalf("expected unsupported type error, got %v", err)
	}
}
