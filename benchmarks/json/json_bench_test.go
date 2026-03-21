package jsonbench

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dmesha3/elgon"
)

type payload struct {
	ID    int      `json:"id"`
	Name  string   `json:"name"`
	Email string   `json:"email"`
	Tags  []string `json:"tags"`
}

func BenchmarkJSONEncodeResponse(b *testing.B) {
	app := elgon.New(elgon.Config{DisableHealthz: true})
	p := payload{ID: 1, Name: "Alice", Email: "alice@example.com", Tags: []string{"a", "b", "c"}}
	app.GET("/json", func(c *elgon.Ctx) error {
		return c.JSON(http.StatusOK, p)
	})

	req := httptest.NewRequest(http.MethodGet, "/json", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		app.ServeHTTP(rr, req)
	}
}

func BenchmarkJSONBindRequest(b *testing.B) {
	app := elgon.New(elgon.Config{DisableHealthz: true})
	app.POST("/json", func(c *elgon.Ctx) error {
		var in payload
		if err := c.BindJSON(&in); err != nil {
			return err
		}
		return c.Text(http.StatusOK, "ok")
	})

	in := payload{ID: 1, Name: "Alice", Email: "alice@example.com", Tags: []string{"a", "b", "c"}}
	body, _ := json.Marshal(in)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/json", bytes.NewReader(body))
		rr := httptest.NewRecorder()
		app.ServeHTTP(rr, req)
	}
}
