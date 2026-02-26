package e2e

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/meshackkazimoto/elgon"
	"github.com/meshackkazimoto/elgon/middleware"
	"github.com/meshackkazimoto/elgon/observability"
)

func BenchmarkE2EStack(b *testing.B) {
	app := elgon.New(elgon.Config{DisableHealthz: true})
	metrics := observability.NewMetrics()
	app.Use(
		middleware.Recover(),
		middleware.RequestID(),
		middleware.SecureHeaders(),
		metrics.Middleware(),
	)
	app.GET("/api/v1/users/:id", func(c *elgon.Ctx) error {
		return c.JSON(http.StatusOK, map[string]string{"id": c.Param("id")})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/42", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		app.ServeHTTP(rr, req)
	}
}
