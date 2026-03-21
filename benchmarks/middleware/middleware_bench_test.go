package middleware

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dmesha3/elgon"
	elgonmw "github.com/dmesha3/elgon/middleware"
)

func BenchmarkMiddlewareChain(b *testing.B) {
	app := elgon.New(elgon.Config{DisableHealthz: true})
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	app.Use(
		elgonmw.Recover(),
		elgonmw.RequestID(),
		elgonmw.SecureHeaders(),
		elgonmw.Logger(logger),
	)
	app.GET("/ok", func(c *elgon.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		app.ServeHTTP(rr, req)
	}
}
