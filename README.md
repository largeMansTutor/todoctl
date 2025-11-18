# Todo Service

Todo Service is a Go API that exposes bulk operations for managing todos (create, update, list, fetch single) backed by MySQL. It is built with Cobra for the CLI, Uber Fx for dependency injection, versioned with Git, and ships with Docker/Docker Compose for local and containerized runs. Optional API key authentication, structured logging, idempotent writes, and CI workflows are included to keep the service reliable and secure.

## Architecture and Stack

- Go 1.25, Git for version control, Cobra CLI (`todoctl`) for operational commands.
- Uber Fx wires dependencies (config, logger, DB, HTTP router) and manages lifecycle.
- MySQL 8.x persistence with prepared statements, indices, and idempotency store.
- REST API served via chi with middleware for tracing, rate limiting, body size limits, security headers, and request IDs.
- Optional API key auth via `X-API-Key`; OpenAPI spec in `openapi/openapi.yaml`.
- Dockerfile for the API image, `docker-compose.yml` for API + MySQL (and observability stack).
- GitHub Actions workflow `.github/workflows/ci.yml` runs lint, build, unit, and tagged integration tests.

## CLI (Cobra)

Install the CLI:

```bash
go install ./cmd/todoctl
```

Available commands:

- `todoctl api` — run the API server using environment-driven config.
- `todoctl migrate --direction up|down [--steps N] [--path migrations]` — apply database migrations forward/backward.
- Additional helper subcommands are attached in `internal/cli/commands`.

## Running the Service

### With Docker Compose

```bash
docker compose up --build
```

This starts MySQL and the API on `localhost:8080` with the default API key `dev-key`. Migrations run via the `migrator` service before the API starts.

### Locally (without Docker)

Ensure you have a MySQL database named `todos`.

```bash
# 1) Install the CLI
go install ./cmd/todoctl

# 2) Run migrations
DB_DSN="root:password@tcp(localhost:3306)/todos?parseTime=true&charset=utf8mb4&loc=UTC" \
todoctl migrate --direction up

# 3) Start the API
APP_PORT=8080 \
DB_DSN="root:password@tcp(localhost:3306)/todos?parseTime=true&charset=utf8mb4&loc=UTC" \
APP_API_KEY=mysecretkey \
todoctl api
```

## Configuration

Set environment variables to tune behavior (defaults follow `internal/platform/config/config.go`):

| Variable | Default | Description |
| --- | --- | --- |
| `APP_PORT` | `8080` | HTTP listen port. Supports TLS via `APP_TLS_CERT_FILE`/`APP_TLS_KEY_FILE`. |
| `APP_READ_TIMEOUT` / `APP_WRITE_TIMEOUT` / `APP_IDLE_TIMEOUT` | `10s` / `10s` / `60s` | HTTP timeouts. |
| `APP_MAX_BODY_BYTES` | `1048576` | Max request body size (bytes). |
| `DB_DSN` | `root:password@tcp(localhost:3306)/todos?parseTime=true&charset=utf8mb4&loc=UTC` | MySQL DSN. |
| `APP_API_KEY` | `` | Static API key value; if empty, auth is disabled. |
| `APP_ALLOWED_API_KEYS`, `APP_ALLOWED_API_KEYS_HASHED` | `` | Labeled or hashed keys; see `config.Config`. |
| `APP_RATE_LIMIT_PER_SECOND` / `APP_RATE_LIMIT_BURST` | `20` / `40` | Token-bucket rate limiting. |
| `APP_OPENAPI_PATH` | `openapi/openapi.yaml` | Path served at `/docs`. |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `` | OTLP endpoint for traces/metrics; auto-instrumented with chi middleware. |
| Vault variables | `` | `APP_SECRETS_PROVIDER=vault` with `VAULT_ADDR`, `VAULT_TOKEN`, `VAULT_MOUNT`, `VAULT_KV_PATH` to source secrets. |

## API Endpoints

All endpoints accept/return JSON. When `APP_API_KEY` is set, include `X-API-Key: <value>`.

- `POST /todos` — bulk create todos. Body: `{"todos":[{"title":"Buy milk","description":"2%","due_date":"2025-01-01T12:00:00Z"}]}` (title required, unique). Returns created todos with IDs. Supports `Idempotency-Key` header for safe retries.
- `PATCH /todos` — bulk partial update. Body: `{"todos":[{"id":1,"title":"new","complete":true,"due_date":"2025-06-01T12:00:00Z"}]}`. Returns updated todos; idempotent with `Idempotency-Key`.
- `GET /todos` — list todos with offset or cursor pagination. Query: `page` (default 1), `limit` (default 10), `cursor` (optional). Returns todos plus `next_cursor` when available.
- `GET /todos/{id}` — fetch by ID; optional `title` query param for lookup by title.
- `GET /healthz` — health probe.

Quick examples (using defaults):

```bash
curl -X POST http://localhost:8080/todos \
  -H 'Content-Type: application/json' \
  -H 'X-API-Key: dev-key' \
  -H 'Idempotency-Key: create-123' \
  -d '{"todos":[{"title":"Buy milk","description":"2% fat"}]}'

curl -X PATCH http://localhost:8080/todos \
  -H 'Content-Type: application/json' \
  -H 'X-API-Key: dev-key' \
  -H 'Idempotency-Key: update-123' \
  -d '{"todos":[{"id":1,"complete":true}]}'

curl "http://localhost:8080/todos?page=1&limit=20" -H 'X-API-Key: dev-key'
```

## Migrations

Migration files live in `migrations/`. They are applied by `todoctl migrate` using [`golang-migrate`](https://github.com/golang-migrate/migrate) and are also run automatically in Docker Compose via the `migrator` service.

## Testing and CI

- Run unit tests locally: `go test ./...`.
- Integration tests (tagged) spin up MySQL via Testcontainers: `go test -tags=integration ./internal/repository/mysql ./internal/web`. The existing integration suite is skipped by default until a stable container runtime is available; unskip when running in CI or a reliable Docker environment.
- GitHub Actions workflow `.github/workflows/ci.yml` runs linting (golangci-lint), build, unit tests with coverage, and tagged integration tests.

## Docker and Deployment

- `Dockerfile` builds a multi-stage image with a distroless runtime.
- `docker-compose.yml` provides API + MySQL + observability (Prometheus, Grafana, Loki, Jaeger). Override env vars in the compose file for non-default credentials or ports.
- Helm chart scaffold is available under `deploy/chart/` for Kubernetes packaging.

## Design Decisions and Trade-offs

- **Idempotent writes**: `Idempotency-Key` is persisted alongside request bodies to safely retry `POST /todos` and `PATCH /todos` without duplicate work.
- **API key auth**: Simple header-based auth keeps the surface minimal; multiple/hashed keys supported for gradual hardening.
- **Fx-managed lifecycle**: Using Uber Fx centralizes construction and teardown (DB, HTTP server, tracer), reducing manual wiring.
- **Rate limiting and body limits**: Defaults protect the service from abuse; override per environment.
- **Testing strategy**: Unit tests run fast; integration tests rely on Docker and are tagged/skipped to avoid local flakiness while still covered in CI.

## Version Control

This project is Git version-controlled. Commit frequently with clear messages; GitHub Actions validate each push/PR on `main`.
