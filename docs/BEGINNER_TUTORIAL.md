# Beginner Tutorial: Build Your First API with Vein

This guide walks you from zero to a running API with:
- a healthy server
- seeded users in PostgreSQL
- JWT authentication
- a protected endpoint

## 1. Prerequisites
Install:
- Go 1.25+
- PostgreSQL 14+
- `migrate` CLI (golang-migrate)

Confirm tools:

```bash
go version
psql --version
migrate -version
```

## 2. Clone and Configure
From project root:

```bash
cp .env.example .env
```

Open `.env` and set at least:

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

## 3. Create the Database
In PostgreSQL:

```sql
CREATE DATABASE vein;
```

## 4. Run Migrations and Seed Data
From project root:

```bash
make migrate-up
make seed
```

This creates the `users` table and inserts starter users:
- `admin@vein.dev`
- `manager@vein.dev`
- `user@vein.dev`

## 5. Start the API

```bash
make run
```

Server starts on `http://localhost:4000`.

## 6. Test Public Endpoints
Health:

```bash
curl http://localhost:4000/healthcheck
```

Liveness:

```bash
curl http://localhost:4000/liveness
```

Readiness:

```bash
curl http://localhost:4000/readiness
```

Metrics:

```bash
curl http://localhost:4000/metrics
```

## 7. Get a JWT Token
Request token as admin:

```bash
curl -X POST http://localhost:4000/v1/auth/token \
  -H 'Content-Type: application/json' \
  -d '{"subject":"my-first-user","role":"admin"}'
```

Copy the token from response:

```json
{"auth":{"token":"..."}}
```

Save it in your shell:

```bash
TOKEN='paste-token-here'
```

## 8. Call a Protected Endpoint
List users:

```bash
curl 'http://localhost:4000/v1/users?page=1&page_size=10&sort=created_at' \
  -H "Authorization: Bearer $TOKEN"
```

You should receive JSON with `users` and `metadata`.

## 9. Try Idempotency Protection
First request should pass:

```bash
curl -X POST http://localhost:4000/v1/jobs/audit \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: demo-1' \
  -d '{"action":"first-run"}'
```

Second request with same key should return conflict:

```bash
curl -X POST http://localhost:4000/v1/jobs/audit \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: demo-1' \
  -d '{"action":"first-run"}'
```

## 10. Run Tests and Quality Checks

```bash
make test
make fmt-check
make migration-check
make vet
```

If you have a test database DSN:

```bash
export DB_DSN='postgres://postgres:postgres@localhost:5432/vein?sslmode=disable'
make test-integration
```

## 11. Optional: Enable Redis Shared Backends
Set these in `.env`:

```env
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0
REDIS_QUEUE_KEY=vein:jobs
```

Restart the app. The framework will auto-switch cache/queue/rate-limit/idempotency to Redis.

## 12. Optional: Enable OpenTelemetry
Set:

```env
OTEL_ENABLED=true
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317
OTEL_SAMPLE_RATIO=1.0
```

Restart the app to emit traces.

## 13. Generate New Code Quickly
Use built-in CLI:

```bash
go run ./cmd/veincli generate module billing
go run ./cmd/veincli generate endpoint invoices
```

## Common Beginner Issues
- `readiness` fails: DB is not reachable from `DB_DSN`.
- `401 Unauthorized`: token missing/expired/invalid issuer-audience.
- `403 Forbidden`: role is not `admin` or `manager` for `/v1/users`.
- migration errors: ensure `migrate` CLI is installed and DB exists.

## Next Step
After this tutorial, build your first feature by adding a new endpoint in `cmd/api/`, then protect it with `app.authenticate` and `app.requireRoles`.
