package middleware

import (
	"log/slog"
	"os"
	"time"

	"github.com/dmesha3/elgon"
	"github.com/lmittmann/tint"
)

// Logger logs basic request metadata.
func Logger(logger *slog.Logger) elgon.Middleware {
	logger = slog.New(
		tint.NewHandler(os.Stdout, &tint.Options{
			Level:      slog.LevelDebug,
			TimeFormat: time.Kitchen,
			NoColor:    false,
		}),
	)

	return func(next elgon.HandlerFunc) elgon.HandlerFunc {
		return func(c *elgon.Ctx) error {
			start := time.Now()
			err := next(c)

			if err != nil {
				logger.Error("request failed",
					slog.String("method", c.Request.Method),
					slog.String("path", c.Request.URL.Path),
					slog.Duration("latency", time.Since(start)),
					slog.String("request_id", requestID(c)),
					slog.Any("error", err),
				)
				return err
			}

			logger.Info("request",
				slog.String("method", c.Request.Method),
				slog.String("path", c.Request.URL.Path),
				slog.Duration("latency", time.Since(start)),
				slog.String("request_id", requestID(c)),
			)
			return err
		}
	}
}

func requestID(c *elgon.Ctx) string {
	v, _ := c.Get(requestIDKey)
	s, _ := v.(string)
	return s
}
