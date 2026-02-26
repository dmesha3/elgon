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
	Name  string   `json:"name" description:"Display name" openapi:"minLength=2,maxLength=50,example=Alice"`
	Email string   `json:"email,omitempty" openapi:"format=email,example=alice@example.com"`
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
		RequestExample: map[string]any{
			"name":  "Alice",
			"email": "alice@example.com",
		},
		ResponseExample: map[string]any{
			"id": 1,
		},
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
	reqBody := post["requestBody"].(map[string]any)
	content := reqBody["content"].(map[string]any)["application/json"].(map[string]any)
	if content["example"] == nil {
		t.Fatal("expected request example")
	}
	resp := post["responses"].(map[string]any)["201"].(map[string]any)
	respContent := resp["content"].(map[string]any)["application/json"].(map[string]any)
	if respContent["example"] == nil {
		t.Fatal("expected response example")
	}

	reqSchema := schemas["userReq"].(map[string]any)
	props := reqSchema["properties"].(map[string]any)
	nameProp := props["name"].(map[string]any)
	if nameProp["description"] != "Display name" {
		t.Fatalf("missing description annotation: %#v", nameProp)
	}
	if nameProp["minLength"] != 2 {
		t.Fatalf("missing minLength annotation: %#v", nameProp)
	}
}
