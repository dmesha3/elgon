package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dmesha3/elgon"
)

func TestJWTSignVerify(t *testing.T) {
	mgr := NewJWTManager("secret")
	tok, err := mgr.Sign(Claims{Sub: "u1", Roles: []string{"admin"}}, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := mgr.Verify(tok)
	if err != nil {
		t.Fatal(err)
	}
	if claims.Sub != "u1" {
		t.Fatalf("expected sub u1, got %s", claims.Sub)
	}
}

func TestAuthAndRequireRole(t *testing.T) {
	mgr := NewJWTManager("secret")
	tok, err := mgr.Sign(Claims{Sub: "u1", Roles: []string{"admin"}, Perms: []string{"users:write"}}, time.Minute)
	if err != nil {
		t.Fatal(err)
	}

	app := elgon.New(elgon.Config{DisableHealthz: true})
	app.Use(Auth(mgr), RequireRole("admin"), RequirePerm("users:write"))
	app.GET("/secure", func(c *elgon.Ctx) error {
		p, ok := FromCtx(c)
		if !ok || p.ID != "u1" {
			t.Fatalf("principal missing: %+v", p)
		}
		return c.Text(http.StatusOK, "ok")
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	app.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestRequireRoleForbidden(t *testing.T) {
	mgr := NewJWTManager("secret")
	tok, err := mgr.Sign(Claims{Sub: "u1", Roles: []string{"user"}}, time.Minute)
	if err != nil {
		t.Fatal(err)
	}

	app := elgon.New(elgon.Config{DisableHealthz: true})
	app.Use(Auth(mgr), RequireRole("admin"))
	app.GET("/secure", func(c *elgon.Ctx) error { return c.Text(http.StatusOK, "ok") })

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	app.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rr.Code)
	}
}
