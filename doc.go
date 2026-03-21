// Package elgon is a performance-oriented, API-first web framework for Go.
//
// Elgon is built on top of the standard net/http package and focuses on
// delivering production-ready features with minimal overhead. It provides
// a clean and explicit API for building scalable backend services.
//
// Features:
//
//   - Fast and flexible HTTP router (static, param, wildcard routes)
//   - Composable middleware pipeline
//   - Typed request context (Ctx)
//   - Centralized error handling
//   - Built-in observability (metrics & tracing interfaces)
//   - OpenAPI generation with Swagger UI
//   - Authentication utilities (JWT, RBAC, OAuth2/OIDC)
//   - Database adapters, ORM helpers, and migrations
//   - Background jobs with pluggable queue backends
//
// Basic Usage:
//
//	app := elgon.New(elgon.Config{Addr: ":8080"})
//
//	app.GET("/", func(c *elgon.Ctx) error {
//	    return c.JSON(200, map[string]string{"message": "hello"})
//	})
//
//	if err := app.Run(); err != nil {
//	    log.Fatal(err)
//	}
//
// Middleware:
//
//	app.Use(
//	    middleware.Recover(),
//	    middleware.RequestID(),
//	    middleware.Logger(...),
//	)
//
// Observability:
//
//	metrics := observability.NewMetrics()
//	app.Use(metrics.Middleware())
//	metrics.RegisterRoute(app, "/metrics")
//
// OpenAPI:
//
//	openapi.NewGenerator("My API", elgon.Version).
//	    Register(app, "/openapi.json", "/docs")
//
// Project Structure:
//
//   - elgon: core application, router, and context
//   - middleware: HTTP middleware implementations
//   - config: configuration loading utilities
//   - observability: metrics and tracing
//   - openapi: OpenAPI generation and docs UI
//   - auth: authentication and authorization helpers
//   - db: database adapters
//   - orm: higher-level database abstractions
//   - migrate: migration engine
//   - jobs: background job processing
//
// For more examples and guides, see the README and examples directory.
package elgon