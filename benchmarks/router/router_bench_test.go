package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dmesha3/elgon"
)

func setupRouterApp() *elgon.App {
	app := elgon.New(elgon.Config{DisableHealthz: true})
	app.GET("/health", func(c *elgon.Ctx) error { return c.Text(http.StatusOK, "ok") })
	app.GET("/users/:id", func(c *elgon.Ctx) error { return c.Text(http.StatusOK, c.Param("id")) })
	app.GET("/files/*path", func(c *elgon.Ctx) error { return c.Text(http.StatusOK, c.Param("path")) })
	return app
}

func benchRoute(b *testing.B, app *elgon.App, method, target string) {
	req := httptest.NewRequest(method, target, nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		app.ServeHTTP(rr, req)
	}
}

func BenchmarkRouterStatic(b *testing.B) {
	app := setupRouterApp()
	benchRoute(b, app, http.MethodGet, "/health")
}

func BenchmarkRouterParam(b *testing.B) {
	app := setupRouterApp()
	benchRoute(b, app, http.MethodGet, "/users/123")
}

func BenchmarkRouterWildcard(b *testing.B) {
	app := setupRouterApp()
	benchRoute(b, app, http.MethodGet, "/files/a/b/c.txt")
}
