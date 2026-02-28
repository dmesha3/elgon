package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCmdNew(t *testing.T) {
	d := t.TempDir()
	old, _ := os.Getwd()
	defer func() { _ = os.Chdir(old) }()
	if err := os.Chdir(d); err != nil {
		t.Fatal(err)
	}
	if err := cmdNew([]string{"sample-app"}); err != nil {
		t.Fatal(err)
	}
	mustExist := []string{
		"sample-app/go.mod",
		"sample-app/cmd/api/main.go",
		"sample-app/README.md",
		"sample-app/migrations",
	}
	for _, p := range mustExist {
		if _, err := os.Stat(filepath.Join(d, p)); err != nil {
			t.Fatalf("expected %s: %v", p, err)
		}
	}
}

func TestCmdOpenAPIValidate(t *testing.T) {
	d := t.TempDir()
	file := filepath.Join(d, "openapi.json")
	if err := cmdOpenAPI([]string{"generate", "-file", file}); err != nil {
		t.Fatal(err)
	}
	if err := cmdOpenAPI([]string{"validate", "-file", file}); err != nil {
		t.Fatal(err)
	}
}

func TestCmdMigrateGenerate(t *testing.T) {
	d := t.TempDir()
	old, _ := os.Getwd()
	defer func() { _ = os.Chdir(old) }()
	if err := os.Chdir(d); err != nil {
		t.Fatal(err)
	}

	models := `package models
type User struct {
	ID int64 ` + "`orm:\"pk,autoincrement\"`" + `
	Email string ` + "`orm:\"notnull\"`" + `
}
`
	if err := os.WriteFile("models.go", []byte(models), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := cmdMigrate([]string{"generate", "-models", "models.go", "-dir", "migrations", "-name", "init"}); err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	up, err := filepath.Glob(filepath.Join("migrations", "*.up.sql"))
	if err != nil {
		t.Fatal(err)
	}
	down, err := filepath.Glob(filepath.Join("migrations", "*.down.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if len(up) != 1 || len(down) != 1 {
		t.Fatalf("expected one up/down migration pair, got up=%d down=%d", len(up), len(down))
	}
	if !strings.Contains(filepath.Base(up[0]), "_init_") {
		t.Fatalf("expected generated name to include init, got %s", filepath.Base(up[0]))
	}
}

func TestCmdMigrateGenerateRequiresModels(t *testing.T) {
	err := cmdMigrate([]string{"generate"})
	if err == nil || !strings.Contains(err.Error(), "requires -models") {
		t.Fatalf("expected missing models error, got %v", err)
	}
}
