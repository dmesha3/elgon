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

## Roadmap

Next planned modules: auth, db, migrate, jobs, and CLI tooling.
