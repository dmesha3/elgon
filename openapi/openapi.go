package openapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"sort"
	"strings"

	"github.com/dmesha3/elgon"
)

// Operation contains OpenAPI operation metadata.
type Operation struct {
	Summary         string   `json:"summary,omitempty"`
	Description     string   `json:"description,omitempty"`
	OperationID     string   `json:"operationId,omitempty"`
	Tags            []string `json:"tags,omitempty"`
	RequiresAuth    bool
	Deprecated      bool
	RequestModel    any
	ResponseModel   any
	ResponseCode    int
	RequestExample  any
	ResponseExample any
}

// Generator builds OpenAPI documents from registered app routes.
type Generator struct {
	Title       string
	Version     string
	Description string
	ServerURLs  []string
	bearerAuth  bool
	opts        map[string]Operation
	schemas     map[string]any
}

func NewGenerator(title, version string) *Generator {
	if title == "" {
		title = "elgon API"
	}
	if version == "" {
		version = "0.1.0"
	}
	return &Generator{
		Title:      title,
		Version:    version,
		ServerURLs: []string{"/"},
		opts:       make(map[string]Operation),
		schemas:    make(map[string]any),
	}
}

// EnableBearerAuth adds HTTP bearer token auth scheme to the generated document.
func (g *Generator) EnableBearerAuth() {
	g.bearerAuth = true
}

// AddOperation attaches metadata to method/path pairs.
func (g *Generator) AddOperation(method, path string, op Operation) {
	g.opts[key(method, path)] = op
}

// RegisterSchema registers a model schema under a specific component name.
func (g *Generator) RegisterSchema(name string, model any) {
	if name == "" || model == nil {
		return
	}
	if _, ok := g.schemas[name]; ok {
		return
	}
	g.schemas[name] = buildSchema(reflect.TypeOf(model), g.schemas)
}

// Register mounts /openapi.json and /docs endpoints.
func (g *Generator) Register(app *elgon.App, jsonPath, docsPath string) {
	if jsonPath == "" {
		jsonPath = "/openapi.json"
	}
	if docsPath == "" {
		docsPath = "/docs"
	}

	app.GET(jsonPath, func(c *elgon.Ctx) error {
		c.Writer.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(c.Writer)
		enc.SetIndent("", "  ")
		return enc.Encode(g.Build(app))
	})

	app.GET(docsPath, func(c *elgon.Ctx) error {
		c.Writer.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, err := c.Writer.Write([]byte(swaggerHTML(jsonPath)))
		return err
	})
}

func (g *Generator) Build(app *elgon.App) map[string]any {
	routes := app.Routes()
	paths := make(map[string]map[string]any)
	usesBearerAuth := g.bearerAuth
	for _, r := range routes {
		tpl := toOpenAPIPath(r.Path)
		if paths[tpl] == nil {
			paths[tpl] = make(map[string]any)
		}
		method := strings.ToLower(r.Method)
		op := g.opts[key(r.Method, r.Path)]
		entry := map[string]any{}
		if op.Summary != "" {
			entry["summary"] = op.Summary
		}
		if op.Description != "" {
			entry["description"] = op.Description
		}
		if op.OperationID != "" {
			entry["operationId"] = op.OperationID
		}
		if len(op.Tags) > 0 {
			entry["tags"] = op.Tags
		}
		if op.Deprecated {
			entry["deprecated"] = true
		}
		if op.RequiresAuth {
			entry["security"] = []map[string][]string{
				{"BearerAuth": []string{}},
			}
			usesBearerAuth = true
		}
		if params := pathParams(tpl); len(params) > 0 {
			entry["parameters"] = params
		}

		if op.RequestModel != nil {
			name := g.ensureSchema(op.RequestModel)
			if name != "" {
				content := map[string]any{
					"schema": map[string]any{"$ref": "#/components/schemas/" + name},
				}
				if op.RequestExample != nil {
					content["example"] = op.RequestExample
				}
				entry["requestBody"] = map[string]any{
					"required": true,
					"content": map[string]any{
						"application/json": content,
					},
				}
			}
		}

		respCode := op.ResponseCode
		if respCode == 0 {
			respCode = 200
		}
		responses := map[string]any{
			fmt.Sprintf("%d", respCode): map[string]any{"description": http.StatusText(respCode)},
		}
		if op.ResponseModel != nil {
			name := g.ensureSchema(op.ResponseModel)
			if name != "" {
				content := map[string]any{
					"schema": map[string]any{"$ref": "#/components/schemas/" + name},
				}
				if op.ResponseExample != nil {
					content["example"] = op.ResponseExample
				}
				responses[fmt.Sprintf("%d", respCode)] = map[string]any{
					"description": http.StatusText(respCode),
					"content": map[string]any{
						"application/json": content,
					},
				}
			}
		}
		entry["responses"] = responses
		paths[tpl][method] = entry
	}

	serverEntries := make([]map[string]string, 0, len(g.ServerURLs))
	for _, u := range g.ServerURLs {
		serverEntries = append(serverEntries, map[string]string{"url": u})
	}
	if len(serverEntries) == 0 {
		serverEntries = append(serverEntries, map[string]string{"url": "/"})
	}

	orderedPaths := make(map[string]map[string]any, len(paths))
	keys := make([]string, 0, len(paths))
	for k := range paths {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		orderedPaths[k] = paths[k]
	}

	components := map[string]any{"schemas": g.schemas}
	if usesBearerAuth {
		components["securitySchemes"] = map[string]any{
			"BearerAuth": map[string]any{
				"type":         "http",
				"scheme":       "bearer",
				"bearerFormat": "JWT",
			},
		}
	}

	return map[string]any{
		"openapi": "3.0.3",
		"info": map[string]string{
			"title":       g.Title,
			"version":     g.Version,
			"description": g.Description,
		},
		"servers":    serverEntries,
		"paths":      orderedPaths,
		"components": components,
	}
}

func (g *Generator) ensureSchema(model any) string {
	t := reflect.TypeOf(model)
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Name() == "" {
		return ""
	}
	if _, ok := g.schemas[t.Name()]; !ok {
		g.schemas[t.Name()] = buildSchema(t, g.schemas)
	}
	return t.Name()
}

func key(method, path string) string {
	return strings.ToUpper(method) + " " + path
}

func toOpenAPIPath(path string) string {
	parts := strings.Split(path, "/")
	for i := range parts {
		if strings.HasPrefix(parts[i], ":") {
			parts[i] = "{" + strings.TrimPrefix(parts[i], ":") + "}"
		}
		if strings.HasPrefix(parts[i], "*") {
			parts[i] = "{" + strings.TrimPrefix(parts[i], "*") + "}"
		}
	}
	out := strings.Join(parts, "/")
	if out == "" {
		return "/"
	}
	return out
}

func pathParams(path string) []map[string]any {
	parts := strings.Split(path, "/")
	params := make([]map[string]any, 0)
	for _, p := range parts {
		if len(p) >= 2 && p[0] == '{' && p[len(p)-1] == '}' {
			name := strings.TrimSuffix(strings.TrimPrefix(p, "{"), "}")
			params = append(params, map[string]any{
				"name":     name,
				"in":       "path",
				"required": true,
				"schema": map[string]any{
					"type": "string",
				},
			})
		}
	}
	return params
}

func swaggerHTML(specPath string) string {
	return `<!doctype html>
<html>
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>elgon docs</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.ui = SwaggerUIBundle({
      url: '` + specPath + `',
      dom_id: '#swagger-ui'
    });
  </script>
</body>
</html>`
}

// SwaggerUIProxy can be used to host Swagger UI assets locally in future iterations.
func SwaggerUIProxy(_ http.Handler) http.Handler { return http.NotFoundHandler() }
