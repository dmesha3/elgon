# Changelog

All notable changes to this project are documented in this file.

## [0.2.0] - 2026-02-27

### Added

- Optional ORM module (`orm`) with generic table client API:
  - `FindMany`, `FindFirst`, `FindFirstOrThrow`, `FindUnique`, `FindUniqueOrThrow`
  - `Create`, `Update`, `Upsert`, `Delete`
  - `CreateMany`, `CreateManyAndReturn`, `UpdateMany`, `UpdateManyAndReturn`, `DeleteMany`
- App-level data access entrypoints:
  - `app.SetSQL(...)`, `app.SQL()`
  - `app.SetORMDialect(...)`, `app.ORM()`
- ORM `where` operator support with backward compatibility for existing equality filters:
  - Logical: `AND`, `OR`, `NOT`
  - Scalar: `equals`, `not`, `in`, `notIn`, `lt`, `lte`, `gt`, `gte`, `contains`, `startsWith`, `endsWith`, `isSet`, `isEmpty`
- ORM errors:
  - `ErrNonUnique`
  - `ErrUnsupportedOperator` (for not-yet-supported composite/list operators such as `some`/`every`/`none`/`has*`)
- ORM module docs: `docs/modules/orm.md`

### Notes

- ORM remains optional and thin over `db.Adapter`; raw SQL remains available via `app.SQL()`.
- Composite/list operators requiring dialect-specific semantics are intentionally deferred and now return `ErrUnsupportedOperator`.

## [0.1.1] - 2026-02-26

### Added

- OpenAPI bearer authentication support:
  - `openapi.Generator.EnableBearerAuth()`
  - `openapi.Operation.RequiresAuth` for per-operation security requirements
- Swagger UI now shows the `Authorize` bearer token flow when bearer auth is enabled.
- Developer hot-reload mode:
  - `make dev HOT_RELOAD=1`
  - `elgon dev --hot-reload`
  - CLI fallback to `go run github.com/air-verse/air@latest` when `air` is not installed locally.

### Fixed

- Kafka adapter producer construction now matches `kafka-go` v0.4.50 (`kafka.NewWriter(kafka.WriterConfig{...})`).
- Adapter integration tests now skip when external Redis/Kafka services are unavailable instead of hard-failing.

## [0.1.0] - 2026-02-26

Initial public release.

### Added

- Core framework runtime (`elgon`): app lifecycle, routing, groups, context helpers, middleware chain, typed error handling.
- Middleware package: recover, request ID, logger, CORS, secure headers, body limits.
- Config module (`config`): strict env and JSON config loading.
- Observability module (`observability`): request metrics middleware and endpoint, tracing interfaces.
- OpenAPI module (`openapi`): route-based OpenAPI document generation and Swagger UI endpoint.
- Auth module (`auth`): JWT auth, RBAC guards, OAuth2/OIDC helpers.
- DB module (`db`): adapter contracts, SQL adapter, DSN helpers.
- Migration module (`migrate`): migration loader and engine (`up`, `down`, `status`).
- Jobs module (`jobs`): in-memory queue, SQL distributed backend, Redis/Kafka queue interfaces.
- Optional adapter implementations (build tag `adapters`):
  - `jobs/redisadapter` using `go-redis`
  - `jobs/kafkaadapter` using `segmentio/kafka-go`
- CLI (`cmd/elgon`): `new`, `dev`, `test`, `bench`, `migrate`, `openapi` commands.
- Benchmark suites and CI guardrails.
- Example applications:
  - `examples/demo-app`
  - `examples/prod-api`

### Notes

- Public API stability policy starts at `v0.1.0` for documented stable surfaces.
- Adapter concrete implementations are optional and excluded from default build unless `-tags adapters` is used.
