package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strings"

	"github.com/dmesha3/elgon"
	"github.com/dmesha3/elgon/auth"
	"github.com/dmesha3/elgon/config"
	"github.com/dmesha3/elgon/db"
	"github.com/dmesha3/elgon/examples/prod-api/internal/app"
	appdomain "github.com/dmesha3/elgon/examples/prod-api/internal/domain"
	httpx "github.com/dmesha3/elgon/examples/prod-api/internal/http"
	"github.com/dmesha3/elgon/jobs"
	"github.com/dmesha3/elgon/middleware"
	"github.com/dmesha3/elgon/migrate"
	"github.com/dmesha3/elgon/observability"
	"github.com/dmesha3/elgon/openapi"
	_ "github.com/dmesha3/elgon/orm"
	_ "modernc.org/sqlite"
)

type appConfig struct {
	Addr      string `env:"APP_ADDR" default:":8090"`
	AppName   string `env:"APP_NAME" default:"elgon-prod-api"`
	DBDriver  string `env:"APP_DB_DRIVER" default:"sqlite"`
	DBDSN     string `env:"APP_DB_DSN" default:"./prod.db"`
	DBDialect string `env:"APP_DB_DIALECT" default:"sqlite"`
	DBPath    string `env:"APP_DB_PATH" default:""`
	JWTSecret string `env:"APP_JWT_SECRET" default:"change-me"`
}

func main() {
	cfg, err := config.LoadEnv[appConfig]()
	if err != nil {
		log.Fatal(err)
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if cfg.DBPath != "" && cfg.DBDSN == "./prod.db" && strings.EqualFold(cfg.DBDriver, "sqlite") {
		cfg.DBDSN = cfg.DBPath
	}
	adapter, err := db.Open(cfg.DBDriver, cfg.DBDSN)
	if err != nil {
		log.Fatal(err)
	}
	defer adapter.Close()

	migs, err := migrate.Load("migrations", cfg.DBDialect)
	if err != nil {
		log.Fatal(err)
	}
	engine := migrate.NewEngine(adapter, cfg.DBDialect)
	if _, err := engine.Up(context.Background(), migs, 0); err != nil {
		log.Fatal(err)
	}

	queue := jobs.NewSQLBackend(adapter, jobs.SQLBackendConfig{Dialect: cfg.DBDialect, Table: "elgon_jobs"})
	go queue.RunWorker(context.Background(), func(_ context.Context, msg jobs.Message) error {
		logger.Info("processed job", slog.String("name", msg.Name), slog.String("payload", string(msg.Payload)))
		return nil
	})

	jwt := auth.NewJWTManager(cfg.JWTSecret)
	h := &httpx.Handlers{Repo: &app.TodoRepo{DB: adapter, Dialect: cfg.DBDialect}, JWT: jwt, Queue: queue}

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

	// user, _ := appServer.ORM().Table("users").Create(context.Background(), orm.Values{
	// 	"id": 1,
	// 	"name": "Meshack",
	// 	"email": "meshack@example.com",
	// })

	// users, _ := appServer.ORM().Table("users").FindMany(context.Background(), orm.FindOptions{
	// 	Where: orm.Where{
	// 		"OR": []any{
	// 			orm.Where{"email": orm.Where{"endsWith": "gmail.com"}},
	// 			orm.Where{"email": orm.Where{"endsWith": "company.com"}},
	// 		},
	// 	},
	// })

	// fmt.Printf("created user: %+v\n", user)

	appServer.GET("/healthz", h.Health)
	appServer.POST("/auth/login", h.Login)
	todoAPI := appServer.Group("/api/v1", auth.Auth(jwt), auth.RequirePerm("todos:write"))
	todoAPI.GET("/todos", h.ListTodos)
	todoAPI.POST("/todos", h.CreateTodo)
	todoAPI.PATCH("/todos/:id/done", h.MarkDone)
	userAPI := appServer.Group("/api/v1", auth.Auth(jwt), auth.RequirePerm("users:write"))
	userAPI.POST("/users", h.CreateUser)

	docs := openapi.NewGenerator("elgon prod API", elgon.Version)
	docs.Description = "Production-style sample using DB, migrations, auth, observability, and distributed jobs."
	docs.EnableBearerAuth()
	docs.AddOperation("POST", "/api/v1/users", openapi.Operation{
		Summary:       "Create user",
		OperationID:   "createUser",
		Tags:          []string{"users"},
		RequiresAuth:  true,
		RequestModel:  appdomain.CreateUserRequest{},
		ResponseModel: appdomain.User{},
		ResponseCode:  201,
	})
	docs.Register(appServer, "/openapi.json", "/docs")

	fmt.Printf("%s listening on %s\n", cfg.AppName, cfg.Addr)
	if err := appServer.Run(); err != nil {
		log.Fatal(err)
	}
}
