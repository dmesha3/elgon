package middleware

import (
	"log/slog"
	"time"

	"github.com/meshackkazimoto/elgon"
)

// Logger logs basic request metadata.
func Logger(logger *slog.Logger) elgon.Middleware {
	if logger == nil {
		logger = slog.Default()
	}
	return func(next elgon.HandlerFunc) elgon.HandlerFunc {
		return func(c *elgon.Ctx) error {
			start := time.Now()
			err := next(c)
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
