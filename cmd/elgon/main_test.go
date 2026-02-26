package main

import (
	"os"
	"path/filepath"
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
