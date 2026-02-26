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
		middleware.SecureHeaders(),
	)

	api := app.Group("/api")
	api.GET("/hello/:name", func(c *elgon.Ctx) error {
		return c.JSON(200, map[string]string{"message": "hello " + c.Param("name")})
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
