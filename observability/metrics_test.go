package observability

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/meshackkazimoto/elgon"
)

func TestMetricsMiddlewareAndEndpoint(t *testing.T) {
	app := elgon.New(elgon.Config{DisableHealthz: true})
	m := NewMetrics()
	app.Use(m.Middleware())
	m.RegisterRoute(app, "/metrics")

	app.GET("/users/:id", func(c *elgon.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	rr1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodGet, "/users/1", nil)
	app.ServeHTTP(rr1, req1)

	rr2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	app.ServeHTTP(rr2, req2)

	body := rr2.Body.String()
	if !strings.Contains(body, "elgon_http_requests_total") {
		t.Fatalf("expected metrics output, got: %s", body)
	}
	if !strings.Contains(body, `route="/users/:id"`) {
		t.Fatalf("expected route label, got: %s", body)
	}
}
