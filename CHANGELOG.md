# Changelog

All notable changes to this project are documented in this file.

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
