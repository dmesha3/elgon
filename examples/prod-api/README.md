# elgon Production Sample API

This sample demonstrates a production-oriented `elgon` application with:

- SQLite persistence via `db` adapter
- startup migrations via `migrate`
- JWT authentication and permission guards via `auth`
- SQL-backed distributed jobs via `jobs.SQLBackend`
- metrics and OpenAPI docs

## Run

```bash
cd examples/prod-api
go mod tidy
go run ./cmd/api
```

## Endpoints

- `POST /auth/login`
- `GET /api/v1/todos`
- `POST /api/v1/todos`
- `PATCH /api/v1/todos/:id/done`
- `GET /metrics`
- `GET /openapi.json`
- `GET /docs`

## Quick test

```bash
TOKEN=$(curl -s http://localhost:8090/auth/login -H 'content-type: application/json' -d '{"email":"ops@example.com"}' | jq -r '.access_token')
curl -s http://localhost:8090/api/v1/todos -H "authorization: Bearer $TOKEN"
```
