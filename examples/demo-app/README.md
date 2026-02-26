# elgon Demo App

This demo shows how developers can use `elgon` to implement a web API with:

- routing + grouped routes
- middleware stack
- JWT auth + RBAC permission checks
- OpenAPI + Swagger docs
- metrics endpoint
- background jobs

## Run

```bash
cd examples/demo-app
go mod tidy
go run ./cmd/api
```

Server defaults to `:8080`.

## Endpoints

- `GET /healthz`
- `POST /auth/login`
- `GET /api/v1/todos` (auth required)
- `POST /api/v1/todos` (auth required)
- `PATCH /api/v1/todos/:id/done` (auth required)
- `GET /metrics`
- `GET /openapi.json`
- `GET /docs`

## Demo flow

1. Get token:
```bash
curl -s http://localhost:8080/auth/login \
  -H 'content-type: application/json' \
  -d '{"email":"admin@example.com"}'
```

2. Create todo:
```bash
curl -s http://localhost:8080/api/v1/todos \
  -H "authorization: Bearer <TOKEN>" \
  -H 'content-type: application/json' \
  -d '{"title":"Ship elgon demo"}'
```

3. List todos:
```bash
curl -s http://localhost:8080/api/v1/todos -H "authorization: Bearer <TOKEN>"
```
