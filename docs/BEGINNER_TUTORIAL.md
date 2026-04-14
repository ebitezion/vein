# Vein Framework Complete Beginner Guide

This guide is intentionally detailed. It shows what the framework gives you, how to use each part, and how to build features safely.

## Table Of Contents
1. What Vein Is
2. Project Structure
3. First Run (Local)
4. Environment Variables (Full Reference)
5. Routes And API Surface
6. Authentication And Authorization (JWT + RBAC)
7. Middleware You Already Get
8. Data Layer, Filters, Pagination, Transactions
9. Migrations And Seeding
10. Cache / Queue / Rate Limit / Idempotency Backends
11. Observability (Logging, Metrics, Health)
12. OpenTelemetry Tracing
13. Testing (Unit + Integration/E2E)
14. CI/CD Quality Gates
15. CLI Scaffolding And Generators
16. How To Add A New Protected Feature
17. Troubleshooting

## 1. What Vein Is
Vein is a backend framework skeleton for Go services with:
- secure HTTP defaults
- JWT authentication
- role-based authorization
- PostgreSQL model/repository patterns
- pagination/filter conventions
- Redis-backed shared infra (with in-memory fallback)
- readiness/liveness/metrics
- CI checks for code quality and security

Use it when you want to ship API services quickly without rebuilding backend foundations each time.

## 2. Project Structure
- `cmd/api/`: application entrypoint, routes, middleware, handlers, auth, tracing.
- `internal/data/`: models, filters, transaction helper.
- `internal/validator/`: reusable validation helpers.
- `migrations/`: SQL schema migrations.
- `seeds/`: SQL seed scripts.
- `cmd/seed/`: seed runner.
- `cmd/veincli/`: scaffold and generator CLI.
- `scripts/`: local quality checks (`gofmt`, migration safety).
- `.github/workflows/ci.yml`: CI pipeline.

## 3. First Run (Local)
### 3.1 Prerequisites
- Go `1.25+`
- PostgreSQL `14+`
- `migrate` CLI

Check:

```bash
go version
psql --version
migrate -version
```

### 3.2 Configure

```bash
cp .env.example .env
```

Minimum values in `.env`:

```env
APP_NAME=vein
APP_VERSION=1.0.0
PORT=4000
MY_ENV=development
DB_DSN=postgres://postgres:postgres@localhost:5432/vein?sslmode=disable
TOKEN_SECRET=change-me-now
TOKEN_ISSUER=vein
TOKEN_AUDIENCE=vein-clients
TOKEN_TTL=24h
```

### 3.3 Database setup

```sql
CREATE DATABASE vein;
```

```bash
make migrate-up
make seed
```

Seeded credential examples:
- `admin@vein.dev` / `VeinPass#2026!`
- `manager@vein.dev` / `VeinPass#2026!`

### 3.4 Start server

```bash
make run
```

Server: `http://localhost:4000`

Quick verification:

```bash
curl -s http://localhost:4000/healthcheck | jq
```

Example output:

```json
{
  "healthcheck": {
    "AppName": "vein",
    "Version": "1.0.0",
    "MY_ENV": "development"
  }
}
```

## 4. Environment Variables (Full Reference)
### Core app
- `APP_NAME`: service name.
- `APP_VERSION`: display/version string.
- `PORT`: HTTP port.
- `MY_ENV`: `development|staging|production|test`.

### Database
- `DB_DSN` or `DB_DSN_FILE`.
- `DB_MAX_OPEN_CONNS`.
- `DB_MAX_IDLE_CONNS`.
- `DB_MAX_IDLE_TIME` (duration, e.g. `15m`).

### Auth (JWT)
- `TOKEN_SECRET` or `TOKEN_SECRET_FILE`.
- `TOKEN_ISSUER`.
- `TOKEN_AUDIENCE`.
- `TOKEN_TTL` (duration, e.g. `24h`).

### HTTP security
- `CORS_TRUSTED_ORIGINS` (comma-separated).
- `RATE_LIMIT_RPS`.
- `RATE_LIMIT_BURST`.

### Redis shared infra (optional)
- `REDIS_ADDR` (enable Redis when set).
- `REDIS_PASSWORD` or `REDIS_PASSWORD_FILE`.
- `REDIS_DB`.
- `REDIS_QUEUE_KEY`.

### Tracing (optional)
- `OTEL_ENABLED`.
- `OTEL_EXPORTER_OTLP_ENDPOINT` (e.g. `localhost:4317`).
- `OTEL_SAMPLE_RATIO` (`0.0` to `1.0`).

### Integration testing
- `INTEGRATION_TEST_DSN`.

## 5. Routes And API Surface
Public:
- `GET /healthcheck`
- `GET /liveness`
- `GET /readiness`
- `GET /metrics`
- `POST /v1/auth/token`

Protected:
- `GET /v1/users` (requires role `admin` or `manager`)
- `POST /v1/jobs/audit` (requires role `admin` or `manager`)

Example route protection pattern:

```go
routes.Handler(
	http.MethodGet,
	"/v1/users",
	app.authenticate(
		app.requireRoles("admin", "manager")(http.HandlerFunc(app.listUsers)),
	),
)
```

## 6. Authentication And Authorization (JWT + RBAC)
### 6.1 Issue a token

```bash
curl -X POST http://localhost:4000/v1/auth/token \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@vein.dev","password":"VeinPass#2026!"}'
```

Response includes:
- `auth.token`

Example response:

```json
{
  "auth": {
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "expires_in": 3600,
    "role": "admin"
  }
}
```

Security note:
- `role` is **not accepted** from clients.
- Role is derived from the user record in the database after password verification.

### 6.2 Use token on protected endpoints

```bash
TOKEN='your-token'
curl 'http://localhost:4000/v1/users?page=1&page_size=10&sort=created_at' \
  -H "Authorization: Bearer $TOKEN"
```

Example response:

```json
{
  "users": [
    {
      "id": "a9d8...",
      "first_name": "Admin",
      "email": "admin@vein.dev",
      "role": "admin",
      "status": "active"
    }
  ],
  "metadata": {
    "current_page": 1,
    "page_size": 10,
    "first_page": 1,
    "last_page": 1,
    "total_records": 3
  }
}
```

### 6.3 Role behavior
- `admin`, `manager`: allowed on `/v1/users`.
- `user`: forbidden on `/v1/users` (403).

## 7. Middleware You Already Get
Applied globally:
- panic recovery
- request metrics collection
- tracing span creation
- request ID generation/propagation (`X-Request-ID`)
- structured request logging
- security headers
- CORS checks
- rate limiting
- idempotency checks for `POST/PUT/PATCH`

Idempotency example:

```bash
curl -X POST http://localhost:4000/v1/jobs/audit \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: abc-123' \
  -d '{"action":"demo"}'
```

Calling same method/path/idempotency-key again returns conflict.

Conflict response example:

```json
{
  "error": {
    "type": "Conflict",
    "message": "Duplicate request detected",
    "request_id": "7ef9..."
  }
}
```

## 8. Data Layer, Filters, Pagination, Transactions
### 8.1 Filters
Framework filter fields:
- `page`
- `page_size`
- `sort`

Current users safelist:
- `created_at`, `-created_at`
- `email`, `-email`
- `first_name`, `-first_name`

### 8.2 Pagination metadata
Responses include `metadata` with:
- `current_page`
- `page_size`
- `first_page`
- `last_page`
- `total_records`

### 8.3 Transactions
Use `internal/data/tx.go` `WithTransaction(...)` to wrap multi-step writes.

Example:

```go
err := app.model.Tx.WithTransaction(r.Context(), func(tx *sql.Tx) error {
	// insert parent row
	// insert child row
	// return err to rollback both writes
	return nil
})
```

## 9. Migrations And Seeding
Common commands:

```bash
make migrate-up
make migrate-down
make seed
```

Create a migration:

```bash
make migrate-create name=create_orders
```

## 10. Cache / Queue / Rate Limit / Idempotency Backends
The framework supports two backend modes:

1. In-memory (default):
- no Redis needed
- good for local dev/single instance

2. Redis-backed (shared):
- set `REDIS_ADDR` and restart
- supports multi-instance consistency for:
  - cache
  - queue
  - rate-limiting counters
  - idempotency keys

Example `.env` for Redis mode:

```env
REDIS_ADDR=localhost:6379
REDIS_DB=0
REDIS_QUEUE_KEY=vein:jobs
```

Trusted proxy example (only needed behind reverse proxies/load balancers):

```env
TRUSTED_PROXIES=10.0.0.0/8,192.168.0.0/16
```

Auth endpoint-specific rate limiting:

```env
AUTH_RATE_LIMIT_RPS=1
AUTH_RATE_LIMIT_BURST=3
```

## 11. Observability (Logging, Metrics, Health)
### 11.1 Logs
Request logs are emitted in structured JSON style with:
- method, path, status_code, duration_ms, request_id

Example log line:

```json
{"ts":"2026-04-13T17:00:00Z","method":"GET","path":"/v1/users","status_code":200,"duration_ms":8,"request_id":"7f0c..."}
```

### 11.2 Health
- `/healthcheck`: app metadata
- `/liveness`: process is alive
- `/readiness`: dependencies are ready (DB ping)

### 11.3 Metrics
- `/metrics`: request totals, status groups, average latency, per-endpoint counts

Example:

```bash
curl -s http://localhost:4000/metrics | jq
```

```json
{
  "metrics": {
    "total_requests": 12,
    "total_2xx": 10,
    "total_4xx": 2
  }
}
```

## 12. OpenTelemetry Tracing
Enable:

```env
OTEL_ENABLED=true
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317
OTEL_SAMPLE_RATIO=1.0
```

Then restart service. Every HTTP request creates a trace span.

## 13. Testing (Unit + Integration/E2E)
### 13.1 Unit tests

```bash
make test
```

### 13.2 Integration/E2E tests
Requires test DB DSN:

```bash
export DB_DSN='postgres://postgres:postgres@localhost:5432/vein?sslmode=disable'
export INTEGRATION_TEST_DSN="$DB_DSN"
make test-integration
```

E2E test verifies:
- migrations and seed application
- token issuance
- protected endpoint access
- idempotency behavior

## 14. CI/CD Quality Gates
CI workflow runs:
- format check
- migration safety check
- `go vet`
- `golangci-lint`
- `go test ./...`
- `gosec`
- `govulncheck`

Local equivalents:

```bash
make fmt-check
make migration-check
make vet
make lint
```

## 15. CLI Scaffolding And Generators
Build CLI:

```bash
make build-cli
```

Create a new app skeleton:

```bash
go run ./cmd/veincli create-app myservice
```

Generate module:

```bash
go run ./cmd/veincli generate module billing
```

Generate endpoint stub:

```bash
go run ./cmd/veincli generate endpoint invoices
```

Example generated file:
- `cmd/api/invoices.go`

## 16. How To Add A New Protected Feature
Recommended flow:
1. Add handler file in `cmd/api/`.
2. Parse/validate request with existing helper + validator.
3. Add data operation in `internal/data/`.
4. Register route in `cmd/api/routes.go`.
5. Protect route using `app.authenticate(...)` and `app.requireRoles(...)`.
6. Add unit test.
7. Add e2e scenario if DB or middleware-sensitive.

## 17. Troubleshooting
- `readiness` fails: invalid DB DSN or DB down.
- `401 Unauthorized`: missing/invalid/expired JWT.
- `403 Forbidden`: role not allowed for route.
- `429 Too Many Requests`: adjust rate limits.
- duplicate request conflicts: idempotency key reused.
- Redis not used: check `REDIS_ADDR` is set and reachable.
- traces missing: verify `OTEL_ENABLED=true` and OTLP endpoint.

Quick debug checklist:
1. `cat .env` and confirm required vars are present.
2. `curl -i http://localhost:4000/healthcheck`.
3. `curl -i http://localhost:4000/readiness`.
4. issue token, then call protected route with `Authorization: Bearer ...`.
5. check server logs for `request_id`.
