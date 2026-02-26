package elgon

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStaticRoute(t *testing.T) {
	app := New(Config{DisableHealthz: true})
	app.GET("/users", func(c *Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	app.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
}

func TestParamRoute(t *testing.T) {
	app := New(Config{DisableHealthz: true})
	app.GET("/users/:id", func(c *Ctx) error {
		return c.Text(http.StatusOK, c.Param("id"))
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/users/42", nil)
	app.ServeHTTP(rr, req)

	if rr.Body.String() != "42" {
		t.Fatalf("expected body 42, got %q", rr.Body.String())
	}
}

func TestWildcardRoute(t *testing.T) {
	app := New(Config{DisableHealthz: true})
	app.GET("/files/*path", func(c *Ctx) error {
		return c.Text(http.StatusOK, c.Param("path"))
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/files/a/b/c.txt", nil)
	app.ServeHTTP(rr, req)

	if rr.Body.String() != "a/b/c.txt" {
		t.Fatalf("expected wildcard capture, got %q", rr.Body.String())
	}
}

func TestNotFound(t *testing.T) {
	app := New(Config{DisableHealthz: true})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/missing", nil)
	app.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}

	var payload ErrorResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if payload.Error.Code != CodeNotFound {
		t.Fatalf("expected code %s got %s", CodeNotFound, payload.Error.Code)
	}
}

func TestGroupRoute(t *testing.T) {
	app := New(Config{DisableHealthz: true})
	api := app.Group("/api")
	api.GET("/v1/ping", func(c *Ctx) error {
		return c.Text(http.StatusOK, "pong")
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil)
	app.ServeHTTP(rr, req)

	if rr.Body.String() != "pong" {
		t.Fatalf("expected pong, got %q", rr.Body.String())
	}
}
