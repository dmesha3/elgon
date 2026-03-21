package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dmesha3/elgon"
)

func TestRecoverMiddleware(t *testing.T) {
	app := elgon.New(elgon.Config{DisableHealthz: true})
	app.Use(Recover())
	app.GET("/panic", func(c *elgon.Ctx) error {
		panic("boom")
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	app.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 got %d", rr.Code)
	}
}

func TestRequestIDMiddleware(t *testing.T) {
	app := elgon.New(elgon.Config{DisableHealthz: true})
	app.Use(RequestID())
	app.GET("/ok", func(c *elgon.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	app.ServeHTTP(rr, req)

	if rr.Header().Get("X-Request-Id") == "" {
		t.Fatal("expected X-Request-Id header")
	}
}

func TestBodyLimitMiddleware(t *testing.T) {
	app := elgon.New(elgon.Config{DisableHealthz: true})
	app.Use(BodyLimit(2))
	app.POST("/bind", func(c *elgon.Ctx) error {
		var payload map[string]any
		err := c.BindJSON(&payload)
		if err != nil {
			return err
		}
		return c.Text(http.StatusOK, "ok")
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/bind", strings.NewReader(`{"x":1}`))
	app.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 from limited body got %d", rr.Code)
	}
}
