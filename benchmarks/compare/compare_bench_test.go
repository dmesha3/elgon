package compare

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/meshackkazimoto/elgon"
)

func BenchmarkCompareElgonStatic(b *testing.B) {
	app := elgon.New(elgon.Config{DisableHealthz: true})
	app.GET("/ping", func(c *elgon.Ctx) error { return c.Text(http.StatusOK, "pong") })
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		app.ServeHTTP(rr, req)
	}
}

func BenchmarkCompareStdHTTPStatic(b *testing.B) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /ping", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("pong"))
	})
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)
	}
}
