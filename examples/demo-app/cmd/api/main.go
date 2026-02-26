package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/meshackkazimoto/elgon"
	"github.com/meshackkazimoto/elgon/auth"
	"github.com/meshackkazimoto/elgon/config"
	"github.com/meshackkazimoto/elgon/examples/demo-app/internal/app"
	"github.com/meshackkazimoto/elgon/examples/demo-app/internal/http/handlers"
	"github.com/meshackkazimoto/elgon/examples/demo-app/internal/http/routes"
	"github.com/meshackkazimoto/elgon/jobs"
	"github.com/meshackkazimoto/elgon/middleware"
	"github.com/meshackkazimoto/elgon/observability"
)

type appConfig struct {
	Addr      string `env:"APP_ADDR" default:":8080"`
	AppName   string `env:"APP_NAME" default:"elgon-demo"`
	JWTSecret string `env:"JWT_SECRET" default:"change-me"`
}

func main() {
	cfg, err := config.LoadEnv[appConfig]()
	if err != nil {
		log.Fatal(err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	metrics := observability.NewMetrics()
	queue := jobs.NewInMemoryQueue(64)
	jwt := auth.NewJWTManager(cfg.JWTSecret)

	go queue.RunWorker(context.Background(), func(_ context.Context, msg jobs.Message) error {
		logger.Info("background job", slog.String("name", msg.Name), slog.String("payload", string(msg.Payload)))
		return nil
	})

	a := elgon.New(elgon.Config{Addr: cfg.Addr})
	a.Use(
		middleware.Recover(),
		middleware.RequestID(),
		middleware.Logger(logger),
		middleware.SecureHeaders(),
		metrics.Middleware(),
	)
	metrics.RegisterRoute(a, "/metrics")

	api := &handlers.API{
		Todos: app.NewTodoService(),
		JWT:   jwt,
		Queue: queue,
	}
	routes.Register(a, api, jwt)

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Stop(sig)
		<-sig
		queue.Close()
	}()

	fmt.Printf("%s listening on %s\n", cfg.AppName, cfg.Addr)
	if err := a.Run(); err != nil {
		log.Fatal(err)
	}
}
