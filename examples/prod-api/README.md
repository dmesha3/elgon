# elgon Production Sample API

This sample demonstrates a production-oriented `elgon` application with:

- SQLite persistence via `db` adapter
- optional PostgreSQL runtime via `pgx` driver
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
- `POST /api/v1/users`
- `GET /metrics`
- `GET /openapi.json`
- `GET /docs`

## Quick test

```bash
TOKEN=$(curl -s http://localhost:8090/auth/login -H 'content-type: application/json' -d '{"email":"ops@example.com"}' | jq -r '.access_token')
curl -s http://localhost:8090/api/v1/todos -H "authorization: Bearer $TOKEN"
curl -s http://localhost:8090/api/v1/users -H "authorization: Bearer $TOKEN" -H 'content-type: application/json' -d '{"email":"ada@example.com","name":"Ada Lovelace"}'
```

## PostgreSQL demo

Start PostgreSQL (example):

```bash
docker run --name elgon-pg -e POSTGRES_PASSWORD=postgres -e POSTGRES_USER=postgres -e POSTGRES_DB=elgon -p 5432:5432 -d postgres:16
```

Enable Postgres driver and run:

```bash
cd examples/prod-api
go get github.com/jackc/pgx/v5/stdlib
APP_DB_DRIVER=pgx \
APP_DB_DIALECT=pg \
APP_DB_DSN='postgres://postgres:postgres@localhost:5432/elgon?sslmode=disable' \
go run -tags postgres ./cmd/api
```
