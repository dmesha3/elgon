# openapi Module

OpenAPI 3.0 generation from registered routes.

## Provides

- Route-to-path document generation
- Struct-driven schema generation (`components.schemas`)
- Request/response model references
- Swagger UI route serving
- Field annotations via tags:
  - `description` and `example`
  - `openapi` tag (`format`, `enum`, `minimum`, `maximum`, `minLength`, `maxLength`, `pattern`)

## Primary API

- `NewGenerator(title, version string) *Generator`
- `func (g *Generator) AddOperation(method, path string, op Operation)`
- `func (g *Generator) Register(app *elgon.App, jsonPath, docsPath string)`
- `func (g *Generator) Build(app *elgon.App) map[string]any`
