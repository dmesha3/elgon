package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/meshackkazimoto/elgon"
	"github.com/meshackkazimoto/elgon/db"
	"github.com/meshackkazimoto/elgon/migrate"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	var err error
	switch os.Args[1] {
	case "new":
		err = cmdNew(os.Args[2:])
	case "dev":
		err = cmdDev(os.Args[2:])
	case "test":
		err = runCmd("go", "test", "./...")
	case "bench":
		err = runCmd("go", "test", "./...", "-bench=.", "-benchmem")
	case "migrate":
		err = cmdMigrate(os.Args[2:])
	case "openapi":
		err = cmdOpenAPI(os.Args[2:])
	default:
		printUsage()
		err = fmt.Errorf("unknown command: %s", os.Args[1])
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("elgon CLI")
	fmt.Println("usage: elgon <command>")
	fmt.Println("commands: new, dev, test, bench, migrate, openapi")
}

func cmdNew(args []string) error {
	if len(args) < 1 {
		return errors.New("usage: elgon new <app>")
	}
	name := args[0]
	if name == "" {
		return errors.New("app name cannot be empty")
	}
	root := filepath.Clean(name)
	if err := os.MkdirAll(filepath.Join(root, "cmd", "api"), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(root, "internal", "http", "handlers"), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(root, "migrations"), 0o755); err != nil {
		return err
	}
	goMod := "module " + name + "\n\ngo 1.24.1\n"
	if err := writeIfMissing(filepath.Join(root, "go.mod"), goMod); err != nil {
		return err
	}
	mainGo := `package main

import (
	"log"

	"github.com/meshackkazimoto/elgon"
)

func main() {
	app := elgon.New(elgon.Config{Addr: ":8080"})
	app.GET("/healthz", func(c *elgon.Ctx) error {
		return c.JSON(200, map[string]string{"status": "ok"})
	})
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
`
	if err := writeIfMissing(filepath.Join(root, "cmd", "api", "main.go"), mainGo); err != nil {
		return err
	}
	readme := "# " + name + "\n\nGenerated with elgon CLI.\n"
	if err := writeIfMissing(filepath.Join(root, "README.md"), readme); err != nil {
		return err
	}
	fmt.Println("created", root)
	return nil
}

func writeIfMissing(path, content string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func cmdDev(args []string) error {
	fs := flag.NewFlagSet("dev", flag.ContinueOnError)
	hotReload := fs.Bool("hot-reload", false, "enable hot reload with air")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return errors.New("usage: elgon dev [--hot-reload]")
	}
	if *hotReload {
		if _, err := exec.LookPath("air"); err != nil {
			return runCmd("go", "run", "github.com/air-verse/air@latest", "-c", ".air.toml")
		}
		return runCmd("air", "-c", ".air.toml")
	}
	return runCmd("go", "run", "./cmd/api")
}

func runCmd(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func cmdMigrate(args []string) error {
	if len(args) < 1 {
		return errors.New("usage: elgon migrate up|down|status|generate [flags]")
	}
	action := args[0]
	fs := flag.NewFlagSet("migrate", flag.ContinueOnError)
	dir := fs.String("dir", "migrations", "migrations directory")
	driver := fs.String("driver", "sqlite", "database/sql driver")
	dsn := fs.String("dsn", "file::memory:?cache=shared", "database dsn")
	dialect := fs.String("dialect", "", "migration dialect suffix (pg/mysql/sqlite)")
	steps := fs.Int("steps", 1, "number of steps (0 for all on up)")
	models := fs.String("models", "", "comma-separated model file paths or globs for generate")
	name := fs.String("name", "autogen", "migration name for generate")
	apply := fs.Bool("apply", false, "apply generated migration immediately")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	if action == "generate" {
		modelFiles, err := resolveModelFiles(*models)
		if err != nil {
			return err
		}
		gen, err := migrate.GenerateFromModelFiles(*dir, *dialect, *name, modelFiles)
		if err != nil {
			return err
		}
		fmt.Println("generated", gen.UpPath)
		fmt.Println("generated", gen.DownPath)
		if !*apply {
			return nil
		}
		adapter, err := db.Open(*driver, *dsn)
		if err != nil {
			return err
		}
		defer adapter.Close()
		engine := migrate.NewEngine(adapter, *dialect)
		applied, err := engine.Up(context.Background(), []migrate.Migration{{
			Version: gen.Version,
			Name:    gen.Name,
			UpSQL:   gen.UpSQL,
			DownSQL: gen.DownSQL,
		}}, 0)
		if err != nil {
			return err
		}
		fmt.Println("applied", applied, "migrations")
		return nil
	}

	adapter, err := db.Open(*driver, *dsn)
	if err != nil {
		return err
	}
	defer adapter.Close()

	migs, err := migrate.Load(*dir, *dialect)
	if err != nil {
		return err
	}
	engine := migrate.NewEngine(adapter, *dialect)

	ctx := context.Background()
	switch action {
	case "up":
		upSteps := *steps
		if upSteps < 0 {
			upSteps = 0
		}
		applied, err := engine.Up(ctx, migs, upSteps)
		if err != nil {
			return err
		}
		fmt.Println("applied", applied, "migrations")
		return nil
	case "down":
		done, err := engine.Down(ctx, migs, *steps)
		if err != nil {
			return err
		}
		fmt.Println("rolled back", done, "migrations")
		return nil
	case "status":
		st, err := engine.Status(ctx, migs)
		if err != nil {
			return err
		}
		for _, s := range st {
			state := "pending"
			if s.Applied {
				state = "applied"
			}
			fmt.Printf("%04d %-20s %s\n", s.Version, s.Name, state)
		}
		return nil
	default:
		return fmt.Errorf("unknown migrate action: %s", action)
	}
}

func resolveModelFiles(raw string) ([]string, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, errors.New("migrate generate requires -models")
	}
	parts := strings.Split(raw, ",")
	found := map[string]struct{}{}
	out := make([]string, 0)
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		matches, err := filepath.Glob(part)
		if err != nil {
			return nil, err
		}
		if len(matches) == 0 {
			if _, err := os.Stat(part); err == nil {
				matches = []string{part}
			}
		}
		for _, match := range matches {
			if strings.ToLower(filepath.Ext(match)) != ".go" {
				continue
			}
			if _, ok := found[match]; ok {
				continue
			}
			found[match] = struct{}{}
			out = append(out, match)
		}
	}
	sort.Strings(out)
	if len(out) == 0 {
		return nil, fmt.Errorf("no model files matched %q", raw)
	}
	return out, nil
}

func cmdOpenAPI(args []string) error {
	if len(args) < 1 {
		return errors.New("usage: elgon openapi generate|validate [flags]")
	}
	action := args[0]
	fs := flag.NewFlagSet("openapi", flag.ContinueOnError)
	file := fs.String("file", "openapi.json", "openapi file path")
	title := fs.String("title", "elgon API", "api title")
	version := fs.String("version", elgon.Version, "api version")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	switch action {
	case "generate":
		doc := map[string]any{
			"openapi": "3.0.3",
			"info":    map[string]any{"title": *title, "version": *version},
			"paths":   map[string]any{},
		}
		b, err := json.MarshalIndent(doc, "", "  ")
		if err != nil {
			return err
		}
		return os.WriteFile(*file, b, 0o644)
	case "validate":
		b, err := os.ReadFile(*file)
		if err != nil {
			return err
		}
		var doc map[string]any
		if err := json.Unmarshal(b, &doc); err != nil {
			return err
		}
		if _, ok := doc["openapi"].(string); !ok {
			return errors.New("openapi field missing or invalid")
		}
		if _, ok := doc["info"].(map[string]any); !ok {
			return errors.New("info field missing or invalid")
		}
		if _, ok := doc["paths"].(map[string]any); !ok {
			return errors.New("paths field missing or invalid")
		}
		fmt.Println("openapi document is valid")
		return nil
	default:
		return fmt.Errorf("unknown openapi action: %s", action)
	}
}
