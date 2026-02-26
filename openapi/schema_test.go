package openapi

import (
	"net/http"
	"testing"
	"time"

	"github.com/meshackkazimoto/elgon"
)

type profile struct {
	Bio string `json:"bio"`
}

type userReq struct {
	Name  string   `json:"name"`
	Email string   `json:"email,omitempty"`
	Tags  []string `json:"tags,omitempty"`
}

type userResp struct {
	ID        int       `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	Profile   profile   `json:"profile"`
}

func TestBuildSchemasFromModels(t *testing.T) {
	app := elgon.New(elgon.Config{DisableHealthz: true})
	app.POST("/users", func(c *elgon.Ctx) error { return c.Text(http.StatusOK, "ok") })

	gen := NewGenerator("Test", "1.0.0")
	gen.AddOperation(http.MethodPost, "/users", Operation{
		RequestModel:  userReq{},
		ResponseModel: userResp{},
		ResponseCode:  201,
	})
	doc := gen.Build(app)

	components := doc["components"].(map[string]any)
	schemas := components["schemas"].(map[string]any)
	if _, ok := schemas["userReq"]; !ok {
		t.Fatalf("expected userReq schema, got %#v", schemas)
	}
	if _, ok := schemas["userResp"]; !ok {
		t.Fatalf("expected userResp schema")
	}
	if _, ok := schemas["profile"]; !ok {
		t.Fatalf("expected nested profile schema")
	}

	paths := doc["paths"].(map[string]map[string]any)
	post := paths["/users"]["post"].(map[string]any)
	if _, ok := post["requestBody"]; !ok {
		t.Fatal("expected requestBody in operation")
	}
}
