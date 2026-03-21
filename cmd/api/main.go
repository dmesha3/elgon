package main

import (
	"log"
	"log/slog"
	"os"

	"github.com/dmesha3/elgon"
	"github.com/dmesha3/elgon/config"
	"github.com/dmesha3/elgon/middleware"
	"github.com/dmesha3/elgon/observability"
	"github.com/dmesha3/elgon/openapi"
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
		middleware.SecureHeaders(),
		metrics.Middleware(),
	)
	metrics.RegisterRoute(app, "/metrics")

	docs := openapi.NewGenerator("elgon example API", elgon.Version)
	docs.Description = "Phase 2 example with metrics and OpenAPI endpoints."
	docs.Register(app, "/openapi.json", "/docs")

	api := app.Group("/api")
	api.GET("/hello/:name", func(c *elgon.Ctx) error {
		return c.JSON(200, map[string]string{"message": "hello " + c.Param("name")})
	})
	docs.AddOperation("GET", "/api/hello/:name", openapi.Operation{
		Summary:     "Hello endpoint",
		OperationID: "getHelloByName",
		Tags:        []string{"hello"},
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
