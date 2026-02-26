# elgon

`elgon` is a performance-focused, API-first Go web framework designed to stay close to `net/http` while offering production defaults.

## Current Scope

### Phase 1 (Core)

- `App` lifecycle with graceful shutdown
- High-performance path router (static, param, wildcard)
- Group routing and per-route middleware
- `Ctx` helpers: params, query, headers, JSON binding, JSON/text response
- Centralized typed error handling
- Built-in health endpoints: `/health`, `/ready`, `/live`
- Middleware package with `Recover`, `RequestID`, `Logger`, `SecureHeaders`, `BodyLimit`, `CORS`

### Phase 2 (Production Batteries)

- `config` module:
  - Strict env loading with tags (`env`, `default`, `required`)
  - Strict JSON file loading (`DisallowUnknownFields`)
- `observability` module:
  - Metrics collector middleware (request count + duration)
  - Prometheus text endpoint registration (`/metrics`)
  - Optional tracing middleware interface (`Tracer`/`Span`)
- `openapi` module:
  - OpenAPI 3.0.3 generation from registered routes
  - Endpoint registration for `/openapi.json` and `/docs` (Swagger UI)

### Phase 3 (Platform Features)

- `db` module:
  - Adapter interfaces (`ExecContext`, `QueryContext`, `BeginTx`)
  - `database/sql` wrapper and DSN helpers for Postgres/MySQL/SQLite
- `migrate` module:
  - SQL migration loader (`*.up.sql`, `*.down.sql`, optional dialect suffix)
  - Migration engine with `up`, `down`, and `status`
- `jobs` module:
  - In-memory queue (`Enqueue`, `RunWorker`)
  - Interval scheduler (`@every <duration>`/duration specs)
- `elgon` CLI:
  - `elgon new <app>`
  - `elgon dev`
  - `elgon test`
  - `elgon bench`
  - `elgon migrate up|down|status`
  - `elgon openapi generate|validate`
- Benchmarks and CI guardrails:
  - `benchmarks/` module for router, middleware, json, and e2e benchmarks
  - `benchmarks/compare` for elgon vs stdlib baseline microbenchmarks
  - `scripts/bench_guard.sh` threshold checks for PR smoke benchmarks
  - GitHub Actions workflows for CI smoke and nightly full benchmarks

### Phase 4 (Auth and Plugins)

- `auth` module:
  - HS256 JWT signing and verification
  - Authentication middleware (`auth.Auth`)
  - RBAC guards (`auth.RequireRole`, `auth.RequirePerm`)
- Plugin system:
  - App plugin lifecycle via `RegisterPlugins`
  - Duplicate plugin protection and plugin registry access (`Plugins`)

## Install

```bash
go get github.com/meshackkazimoto/elgon
```

## Quick Start

```go
package main

import (
    "log"
    "log/slog"
    "os"

    "github.com/meshackkazimoto/elgon"
    "github.com/meshackkazimoto/elgon/config"
    "github.com/meshackkazimoto/elgon/middleware"
    "github.com/meshackkazimoto/elgon/openapi"
    "github.com/meshackkazimoto/elgon/observability"
)

type appConfig struct {
    Addr string `env:"APP_ADDR" default:":8080"`
}

func main() {
    cfg, err := config.LoadEnv[appConfig]()
    if err != nil {
        log.Fatal(err)
    }

    app := elgon.New(elgon.Config{Addr: cfg.Addr})
    metrics := observability.NewMetrics()

    app.Use(
        middleware.Recover(),
        middleware.RequestID(),
        middleware.Logger(slog.New(slog.NewJSONHandler(os.Stdout, nil))),
        metrics.Middleware(),
    )

    metrics.RegisterRoute(app, "/metrics")

    docs := openapi.NewGenerator("Example API", elgon.Version)
    docs.Register(app, "/openapi.json", "/docs")

    app.GET("/users/:id", func(c *elgon.Ctx) error {
        return c.JSON(200, map[string]string{"id": c.Param("id")})
    })

    if err := app.Run(); err != nil {
        log.Fatal(err)
    }
}
```

## Run tests

```bash
go test ./...
```

## CLI usage

```bash
go run ./cmd/elgon --help
go run ./cmd/elgon migrate status -dir ./migrations -driver sqlite -dsn 'file::memory:?cache=shared'
```

## Roadmap

Next planned modules: DB driver integration tests, richer OpenAPI schema generation, and multi-node distributed job backends.
