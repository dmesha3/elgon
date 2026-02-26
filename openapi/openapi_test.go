package openapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/meshackkazimoto/elgon"
)

func TestBuildAndServe(t *testing.T) {
	app := elgon.New(elgon.Config{DisableHealthz: true})
	app.GET("/users/:id", func(c *elgon.Ctx) error { return c.Text(http.StatusOK, "ok") })

	gen := NewGenerator("Test API", "1.0.0")
	gen.AddOperation(http.MethodGet, "/users/:id", Operation{Summary: "Get user"})
	gen.Register(app, "/openapi.json", "/docs")

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	app.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var doc map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &doc); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	paths, ok := doc["paths"].(map[string]any)
	if !ok {
		t.Fatalf("paths missing: %#v", doc)
	}
	if _, ok := paths["/users/{id}"]; !ok {
		t.Fatalf("expected templated path, got %#v", paths)
	}

	rrDocs := httptest.NewRecorder()
	reqDocs := httptest.NewRequest(http.MethodGet, "/docs", nil)
	app.ServeHTTP(rrDocs, reqDocs)
	if rrDocs.Code != http.StatusOK {
		t.Fatalf("expected docs 200, got %d", rrDocs.Code)
	}
}
