# elgon

`elgon` is a performance-focused, API-first Go web framework designed to stay close to `net/http` while offering production defaults.

## Current Scope (MVP Core)

Implemented from the framework spec draft:

- `App` lifecycle with graceful shutdown
- High-performance path router (static, param, wildcard)
- Group routing and per-route middleware
- `Ctx` helpers: params, query, headers, JSON binding, JSON/text response
- Centralized typed error handling
- Built-in health endpoints: `/health`, `/ready`, `/live`
- Metrics endpoint stub: `/metrics` (optional)
- Middleware package with `Recover`, `RequestID`, `Logger`, `SecureHeaders`, `BodyLimit`, `CORS`
- Version constant in `version.go`

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
    "github.com/meshackkazimoto/elgon/middleware"
)

func main() {
    app := elgon.New(elgon.Config{Addr: ":8080", EnableMetricsStub: true})
    app.Use(
        middleware.Recover(),
        middleware.RequestID(),
        middleware.Logger(slog.New(slog.NewJSONHandler(os.Stdout, nil))),
    )

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

## Project status

This is an initial implementation aligned with Phase 1 MVP of the spec. Modules like auth, db, migrate, jobs, and openapi are planned but not yet implemented.
