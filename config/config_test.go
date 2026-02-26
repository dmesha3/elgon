package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

type testCfg struct {
	AppName string        `env:"APP_NAME" default:"elgon"`
	Port    int           `env:"APP_PORT" default:"8080"`
	Debug   bool          `env:"APP_DEBUG" default:"false"`
	Timeout time.Duration `env:"APP_TIMEOUT" default:"2s"`
	Key     string        `env:"APP_KEY" required:"true"`
}

func TestLoadEnv(t *testing.T) {
	t.Setenv("APP_KEY", "secret")
	t.Setenv("APP_PORT", "9090")
	cfg, err := LoadEnv[testCfg]()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.AppName != "elgon" {
		t.Fatalf("expected default name, got %s", cfg.AppName)
	}
	if cfg.Port != 9090 {
		t.Fatalf("expected port 9090, got %d", cfg.Port)
	}
	if cfg.Timeout != 2*time.Second {
		t.Fatalf("unexpected timeout: %s", cfg.Timeout)
	}
}

func TestLoadEnvMissingRequired(t *testing.T) {
	os.Unsetenv("APP_KEY")
	_, err := LoadEnv[testCfg]()
	if err == nil {
		t.Fatal("expected required env error")
	}
}

func TestLoadJSONFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.json")
	content := `{"AppName":"svc","Port":3000,"Debug":true,"Timeout":1000000000,"Key":"x"}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadJSONFile[testCfg](path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AppName != "svc" || cfg.Port != 3000 || cfg.Key != "x" {
		t.Fatalf("unexpected cfg: %+v", cfg)
	}
}
