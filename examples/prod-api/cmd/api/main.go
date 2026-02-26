package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/meshackkazimoto/elgon"
	"github.com/meshackkazimoto/elgon/auth"
	"github.com/meshackkazimoto/elgon/config"
	"github.com/meshackkazimoto/elgon/db"
	"github.com/meshackkazimoto/elgon/examples/prod-api/internal/app"
	httpx "github.com/meshackkazimoto/elgon/examples/prod-api/internal/http"
	"github.com/meshackkazimoto/elgon/jobs"
	"github.com/meshackkazimoto/elgon/middleware"
	"github.com/meshackkazimoto/elgon/migrate"
	"github.com/meshackkazimoto/elgon/observability"
	"github.com/meshackkazimoto/elgon/openapi"
	_ "modernc.org/sqlite"
)

type appConfig struct {
	Addr      string `env:"APP_ADDR" default:":8090"`
	AppName   string `env:"APP_NAME" default:"elgon-prod-api"`
	DBPath    string `env:"APP_DB_PATH" default:"./prod.db"`
	JWTSecret string `env:"APP_JWT_SECRET" default:"change-me"`
}

func main() {
	cfg, err := config.LoadEnv[appConfig]()
	if err != nil {
		log.Fatal(err)
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	adapter, err := db.Open("sqlite", cfg.DBPath)
	if err != nil {
		log.Fatal(err)
	}
	defer adapter.Close()

	migs, err := migrate.Load("migrations", "")
	if err != nil {
		log.Fatal(err)
	}
	engine := migrate.NewEngine(adapter, "sqlite")
	if _, err := engine.Up(context.Background(), migs, 0); err != nil {
		log.Fatal(err)
	}

	queue := jobs.NewSQLBackend(adapter, jobs.SQLBackendConfig{Dialect: "sqlite", Table: "elgon_jobs"})
	go queue.RunWorker(context.Background(), func(_ context.Context, msg jobs.Message) error {
		logger.Info("processed job", slog.String("name", msg.Name), slog.String("payload", string(msg.Payload)))
		return nil
	})

	jwt := auth.NewJWTManager(cfg.JWTSecret)
	h := &httpx.Handlers{Repo: &app.TodoRepo{DB: adapter}, JWT: jwt, Queue: queue}

	appServer := elgon.New(elgon.Config{Addr: cfg.Addr})
	metrics := observability.NewMetrics()
	appServer.Use(
		middleware.Recover(),
		middleware.RequestID(),
		middleware.Logger(logger),
		middleware.SecureHeaders(),
		metrics.Middleware(),
	)
	metrics.RegisterRoute(appServer, "/metrics")

	appServer.GET("/healthz", h.Health)
	appServer.POST("/auth/login", h.Login)
	api := appServer.Group("/api/v1", auth.Auth(jwt), auth.RequirePerm("todos:write"))
	api.GET("/todos", h.ListTodos)
	api.POST("/todos", h.CreateTodo)
	api.PATCH("/todos/:id/done", h.MarkDone)

	docs := openapi.NewGenerator("elgon prod API", elgon.Version)
	docs.Description = "Production-style sample using DB, migrations, auth, observability, and distributed jobs."
	docs.Register(appServer, "/openapi.json", "/docs")

	fmt.Printf("%s listening on %s\n", cfg.AppName, cfg.Addr)
	if err := appServer.Run(); err != nil {
		log.Fatal(err)
	}
}
