# vein-backend-framework

Go backend framework skeleton with secure defaults, observability, versioned APIs, extensibility primitives, and CI quality gates.

Begin here if you are new: [Beginner Tutorial](docs/BEGINNER_TUTORIAL.md)

## Tech Stack
- Go 1.25
- HTTP router: `github.com/julienschmidt/httprouter`
- Database: PostgreSQL via `github.com/lib/pq`
- Env management: `github.com/joho/godotenv`
- Shared infrastructure (optional): Redis via `github.com/redis/go-redis/v9`
- Auth tokens: JWT via `github.com/golang-jwt/jwt/v5`
- Tracing (optional): OpenTelemetry SDK + OTLP exporter

## What Is Included
- Project scaffolding CLI: `cmd/veincli` (`create-app`, `generate module`, `generate endpoint`)
- Typed configuration loading and startup validation
- Secrets support via `*_FILE` env vars (vault/KMS-friendly mounted files)
- Security middleware: CORS allowlist, headers, request ID, panic recovery
- Shared rate limiting + idempotency via Redis (with in-memory fallback)
- JWT auth + authorization middleware with role checks (`admin`, `manager`, `user`)
- Unified JSON error envelope with request correlation IDs
- Observability: structured request logs, in-memory metrics endpoint, health/readiness/liveness endpoints
- DB standards: migrations, seed script support, transaction manager helper, pagination/filtering conventions
- API standards: versioned routes (`/v1`), pagination metadata
- Shared queue + cache abstractions with Redis implementations and in-memory fallback
- DX/extensibility: lifecycle hooks, plugin registry, event bus
- Graceful shutdown handling for server and background workers
- Optional distributed tracing via OpenTelemetry exporter
- CI/CD quality gates: format check, vet, lint, migration safety checks, security scan, vulnerability scan, integration tests

## Framework Completeness

Implemented:
- Scaffolding CLI and generators
- Typed config + startup validation + secret file support
- Security middleware and safer HTTP defaults
- JWT-based auth + role-based authorization
- Standardized JSON errors and request correlation IDs
- Health/readiness/liveness and metrics endpoint
- Pagination/filtering conventions and API versioning
- Transaction helper and seed workflow
- Redis-backed shared cache/queue/rate-limit/idempotency primitives with in-memory fallback
- Lifecycle hooks, plugin/event primitives
- Optional OpenTelemetry tracing exporter support
- CI workflow with lint, gosec, govulncheck, vet, migration checks, and integration test execution

Still needed for full enterprise-grade readiness:
- OAuth2/OIDC provider integration (if you need delegated auth flows)
- External metrics/tracing dashboards and alerting pipelines in deployment
- Expanded e2e/performance/load coverage for high-traffic scenarios

## Endpoints
- `GET /healthcheck`
- `GET /liveness`
- `GET /readiness`
- `GET /metrics`
- `POST /v1/auth/token`
- `GET /v1/users` (requires bearer token and role `admin|manager`)
- `POST /v1/jobs/audit`

## Configuration
Copy `.env.example` and set required values:
- `APP_NAME`, `APP_VERSION`, `PORT`, `MY_ENV`
- `DB_DSN` (or `DB_DSN_FILE`)
- `TOKEN_SECRET` (or `TOKEN_SECRET_FILE`), `TOKEN_ISSUER`, `TOKEN_AUDIENCE`, `TOKEN_TTL`
- `DB_MAX_OPEN_CONNS`, `DB_MAX_IDLE_CONNS`, `DB_MAX_IDLE_TIME`
- `CORS_TRUSTED_ORIGINS`, `RATE_LIMIT_RPS`, `RATE_LIMIT_BURST`
- Optional Redis: `REDIS_ADDR`, `REDIS_PASSWORD` (or `REDIS_PASSWORD_FILE`), `REDIS_DB`, `REDIS_QUEUE_KEY`
- Optional tracing: `OTEL_ENABLED`, `OTEL_EXPORTER_OTLP_ENDPOINT`, `OTEL_SAMPLE_RATIO`
- Integration test DSN for e2e locally/CI: `INTEGRATION_TEST_DSN`

## Commands
- Run API: `make run`
- Build API: `make build`
- Build CLI: `make build-cli`
- Test: `make test`
- Integration test (with DB DSN): `make test-integration`
- Apply migrations: `make migrate-up`
- Rollback migrations: `make migrate-down`
- Seed data: `make seed`
- Format check: `make fmt-check`
- Migration safety check: `make migration-check`
- Vet: `make vet`
- Lint: `make lint`

## CI Pipeline
GitHub Actions workflow is defined in `.github/workflows/ci.yml` and runs:
- format check
- migration safety checks
- `go vet`
- `golangci-lint`
- `go test ./...` (includes integration/e2e when `INTEGRATION_TEST_DSN` is set)
- `gosec`
- `govulncheck`

## CLI Usage
- `go run ./cmd/veincli create-app myservice`
- `go run ./cmd/veincli generate module billing`
- `go run ./cmd/veincli generate endpoint healthz`
