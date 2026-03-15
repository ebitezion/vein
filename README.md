# vein-backend-framework

Technical skeleton for a Go backend service, built as an upgrade to the earlier Bethel framework. It provides a thin HTTP layer, request/response helpers, database access patterns, validation utilities, and migration scripts to bootstrap new services quickly.

## Tech Stack
- Go 1.24
- HTTP router: `github.com/julienschmidt/httprouter`
- Database: PostgreSQL via `github.com/lib/pq`
- Env management: `github.com/joho/godotenv`
- Migrations: `golang-migrate` CLI (used by Makefile targets)

## Architecture
- Entry point: `cmd/api/main.go` wires configuration, database connection pool, and HTTP server.
- Routing: `cmd/api/routes.go` registers handlers; currently exposes a `/healthcheck` endpoint.
- Handlers: `cmd/api/health.go` responds with app name, version, and environment.
- HTTP helpers: `cmd/api/helpers.go` and `cmd/api/error.go` provide JSON encoding, error formatting, and common responses.
- Data layer: `internal/data` contains `Models` factory plus `UserModel` with validation in `internal/validator`.
- Migrations: `migrations/000001_create_users.*.sql` creates the `users` table with UUID primary keys and indexes.

## Configuration
Environment variables (loaded via `.env` at startup):
- `APP_NAME` (string): human-readable service name.
- `APP_VERSION` (string): semantic version string.
- `PORT` (int): HTTP listen port.
- `MY_ENV` (string): runtime environment flag (e.g., `development`, `staging`, `production`).
- `DB_DSN` (string): PostgreSQL DSN, e.g., `postgres://user:pass@localhost:5432/dbname?sslmode=disable`.
- Optional database pool tuning via CLI flags: `-db-max-open-conns`, `-db-max-idle-conns`, `-db-max-idle-time`.

## Running Locally
1) Install dependencies: Go 1.24+, PostgreSQL, and the `migrate` CLI.
2) Create a `.env` file with the variables above (see Makefile for defaults).
3) Apply migrations: `make migrate-up` (uses `DB_DSN`).
4) Start the API: `make run` (or `go run ./cmd/api`).

## API Surface
- `GET /healthcheck` → `{ "healthcheck": { "AppName": "...", "Version": "...", "MY_ENV": "..." } }`.

## Testing
- Run unit tests: `make test` or `go test ./...`.

## Project Layout
- `cmd/api/`: HTTP server wiring, routes, handlers, helpers, and tests.
- `internal/data/`: data access layer and model validation.
- `internal/validator/`: reusable validation helpers.
- `migrations/`: SQL migrations (up/down) for PostgreSQL.
- `Makefile`: common tasks for running, building, testing, and database migrations.

## Production Notes
- Server uses sensible timeouts (idle/read/write) and a configurable connection pool to prevent resource exhaustion.
- JSON responses are wrapped in a consistent `envelope` map to keep response shapes stable for clients.
