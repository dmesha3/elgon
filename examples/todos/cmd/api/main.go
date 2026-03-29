package main

import (
	"context"
	"log"
	"log/slog"

	"github.com/dmesha3/elgon"
	"github.com/dmesha3/elgon/db"
	"github.com/dmesha3/elgon/middleware"
	"github.com/dmesha3/elgon/migrate"
	"github.com/dmesha3/elgon/openapi"
	"github.com/dmesha3/todos/internal/handlers"
	"github.com/dmesha3/todos/internal/repositories"
	"github.com/dmesha3/todos/internal/routes"
	"github.com/dmesha3/todos/internal/services"
	_ "github.com/lib/pq"
)

func main() {
	cfg := elgon.Config{
		Addr: ":8000",
	}

	logger := slog.Default()
	app := elgon.New(cfg)

	adapter, err := db.Open("postgres", "postgresql://koinet_user:S3cret134@localhost:5432/todos_db")
	if err != nil {
		log.Fatal(err)
	}
	defer adapter.Close()

	migs, err := migrate.Load("migrations", "postgres")
	if err != nil {
		log.Fatal(err)
	}

	engine := migrate.NewEngine(adapter, "postgres")
	if _, err := engine.Up(context.Background(), migs, 0); err != nil {
		log.Fatal(err)
	}

	app.Use(
		middleware.Recover(),
		middleware.Logger(logger),
		middleware.RequestID(),
	)

	todoRepo := repositories.NewTodoRepository(adapter)
	todoService := services.NewTodoService(todoRepo)

	app.GET("/testing", func(c *elgon.Ctx) error {
		return c.JSON(200, map[string]string{"status": "ok"})
	})

	docs := openapi.NewGenerator("Todo API", "v0.0.1")
	docs.Description = "API for todo application - Elgon"
	docs.Register(app, "/openapi.json", "/docs")

	todoHandler := handlers.NewTodoHandler(todoService, docs)
	routes.RegisterTodoRoutes(app, todoHandler)

	logger.Info("Server starting...", "port", cfg.Addr)

	if err := app.Run(); err != nil {
		logger.Error("Failed to run server", "error", err)
	}
}
