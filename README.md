# elgon

`elgon` is a performance-oriented, API-first Go web framework built on top of `net/http`.
It provides production-focused defaults while keeping APIs explicit, modular, and easy to test.

## Why elgon

- Fast routing with static, param, and wildcard path matching
- Low-overhead middleware pipeline
- Centralized typed error handling
- Built-in health and metrics support
- OpenAPI generation and Swagger UI serving
- Config loading from environment and JSON files
- Auth support (JWT, RBAC guards, OAuth2/OIDC helpers)
- Database adapters, migrations, and background jobs
- CLI commands for common development workflows

## Core Packages

- `elgon`: app lifecycle, router, context, handler/middleware contracts
- `middleware`: recover, request ID, logger, CORS, body limit, secure headers
- `config`: strict typed config loading (`env`, `default`, `required` tags)
- `observability`: metrics middleware/endpoint and tracing interfaces
- `openapi`: route-driven OpenAPI generation with schema/tag annotations
- `auth`: JWT auth, RBAC middleware, OAuth2/OIDC helpers
- `db`: adapter abstractions and SQL adapter helpers
- `orm`: optional typed repositories over `db.Adapter`
- `migrate`: SQL migration loading and engine (`up`, `down`, `status`)
- `jobs`: in-memory queue, SQL distributed backend, Redis/Kafka queue interfaces

## Installation

```bash
go get github.com/dmesha3/elgon
```

## Quick Start

```go
package main

import (
	"log"
	"log/slog"
	"os"

	"github.com/dmesha3/elgon"
	"github.com/dmesha3/elgon/middleware"
	"github.com/dmesha3/elgon/observability"
	"github.com/dmesha3/elgon/openapi"
)

func main() {
	app := elgon.New(elgon.Config{Addr: ":8080"})
	metrics := observability.NewMetrics()

	app.Use(
		middleware.Recover(),
		middleware.RequestID(),
		middleware.Logger(slog.New(slog.NewJSONHandler(os.Stdout, nil))),
		metrics.Middleware(),
	)

	app.GET("/users/:id", func(c *elgon.Ctx) error {
		return c.JSON(200, map[string]string{"id": c.Param("id")})
	})

	metrics.RegisterRoute(app, "/metrics")
	openapi.NewGenerator("Example API", elgon.Version).Register(app, "/openapi.json", "/docs")

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
```

## Demo Application

A complete reference implementation is available at:

- `examples/demo-app`
- `examples/prod-api`

Run it with:

```bash
cd examples/demo-app
go mod tidy
go run ./cmd/api
```

Production sample:

```bash
cd examples/prod-api
go mod tidy
go run ./cmd/api
```

## CLI

`elgon` includes a project CLI under `cmd/elgon`.

```bash
go run ./cmd/elgon --help
```

Available command groups include `new`, `dev`, `test`, `bench`, `migrate`, and `openapi`.

## Developer Run Modes

Standard development run:

```bash
make dev
```

Hot reload (file watcher) with `air`:

```bash
make dev HOT_RELOAD=1
```

`elgon dev --hot-reload` will use local `air` when available, and otherwise falls back to `go run github.com/air-verse/air@latest`.

You can also use the CLI directly:

```bash
go run ./cmd/elgon dev --hot-reload
```

## Testing and Benchmarks

```bash
go test ./...
make bench-ci
make bench
```

DB integration tests are env-driven and can be enabled with:

- `ELGON_DB_TEST_DRIVER`
- `ELGON_DB_TEST_DSN`

## Optional Adapters

Concrete Redis and Kafka adapters are provided behind the `adapters` build tag:

- `jobs/redisadapter` (go-redis)
- `jobs/kafkaadapter` (segmentio/kafka-go)

Build/test with adapters:

```bash
go test -tags adapters ./...
go build -tags adapters ./...
```

## Optional ORM

```go
adapter, _ := db.OpenSQLite(db.SQLiteConfig{})
app := elgon.New(elgon.Config{Addr: ":8080"})
app.SetSQL(adapter)
app.SetORMDialect("sqlite")

_, err := app.ORM().Table("users").Create(context.Background(), orm.Values{
	"id":    "usr_1",
	"email": "kazimoto17@proton.me",
	"name":  "Meshack",
})
if err != nil {
	log.Fatal(err)
}

user, err := app.ORM().Table("users").FindOne(context.Background(), orm.FindOptions{
	Columns: []string{"id", "email", "name"},
	Where:   orm.Where{"id": "usr_1"},
})
if err != nil {
	log.Fatal(err)
}
_ = user["email"]

_, _ = app.SQL().ExecContext(context.Background(), "UPDATE users SET name=? WHERE id=?", "M", "usr_1")
```

## Docs

- API stability: `docs/API_STABILITY.md`
- Module docs:
  - `docs/modules/auth.md`
  - `docs/modules/openapi.md`
  - `docs/modules/jobs.md`
  - `docs/modules/migrate.md`
  - `docs/modules/orm.md`
- Installation: `INSTALL.md`
- Releasing: `docs/RELEASING.md`
