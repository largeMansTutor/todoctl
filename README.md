# Todo Service

This repository contains a production‑grade Todo API implemented in Go. It exposes endpoints to create, update and list todo items in bulk. The service is instrumented with structured logging, idempotency for safe retries, optional API key authentication and sensible defaults for security and performance.

## Features

- **Bulk operations**: Create and update multiple todos in a single request.
- **Single fetch**: Retrieve a todo by `id` or `title`.
- **Pagination**: Offset (`page`/`limit`) and keyset (`cursor`/`limit`) pagination for listing.
- **Idempotency**: Safe retries on POST and PATCH via `Idempotency-Key` header.
- **MySQL persistence**: Efficient queries with indices on due date and completion status.
- **Configurable via env**: Port, timeouts, API key and database DSN come from environment variables.
- **API key auth**: Optional static API key enforced via `X-API-Key` header.
- **Structured logging**: Zap logger with per‑request fields and request IDs.
- **Graceful shutdown**: Automatic startup and teardown managed by Fx.
- **Docker**: Multi‑stage Dockerfile with distroless runtime and compose for local development.
- **OpenAPI spec**: Documented API in `openapi/openapi.yaml`.

## Getting Started

### Prerequisites

- Go 1.22+
- MySQL 8.x (or Docker)

### Running with Docker Compose

Clone this repository and run:

```bash
docker compose up --build
```

This starts a MySQL database and the API server. The API listens on `localhost:8080` with the default API key `dev-key`. Use this key in the `X-API-Key` header when making requests.

### Running Locally

Ensure a MySQL instance is available and create a database named `todos`.

1. Install dependencies and build the binary:

   ```bash
   go install ./cmd/todoctl
   ```

2. Run migrations against your database (adjust `DB_DSN` accordingly):

   ```bash
   DB_DSN="root:password@tcp(localhost:3306)/todos?parseTime=true&charset=utf8mb4&loc=UTC" \
   todoctl migrate up
   ```

3. Start the API:

   ```bash
   APP_PORT=8080 \
   DB_DSN="root:password@tcp(localhost:3306)/todos?parseTime=true&charset=utf8mb4&loc=UTC" \
   APP_API_KEY=mysecretkey \
   todoctl api
   ```

### Example Usage

Create todos:

```bash
curl -X POST http://localhost:8080/v1/todos \
  -H 'Content-Type: application/json' \
  -H 'X-API-Key: dev-key' \
  -d '{"todos":[{"title":"Buy milk","description":"2% fat"},{"title":"Learn Go","due_date":"2025-01-01T12:00:00Z"}]}'
```

Update todos:

```bash
curl -X PATCH http://localhost:8080/v1/todos \
  -H 'Content-Type: application/json' \
  -H 'X-API-Key: dev-key' \
  -d '{"todos":[{"id":1,"complete":true}]}'
```

List todos (page 1):

```bash
curl -X GET 'http://localhost:8080/v1/todos?page=1&limit=20' \
  -H 'X-API-Key: dev-key'
```

List todos using cursors:

```bash
# First page (no cursor)
curl -X GET 'http://localhost:8080/v1/todos?limit=20' -H 'X-API-Key: dev-key'
# Use the next_cursor returned in the previous response
curl -X GET 'http://localhost:8080/v1/todos?limit=20&cursor=eyJpZCI6...==' -H 'X-API-Key: dev-key'
```

Get a todo by id or title:

```bash
curl -X GET 'http://localhost:8080/v1/todos/1' -H 'X-API-Key: dev-key'
curl -X GET 'http://localhost:8080/v1/todos/0?title=Buy%20milk' -H 'X-API-Key: dev-key'
```

### Migrations

Migrations are stored in the `migrations/` directory. Run them via `todoctl migrate up`. Each `.sql` file is read and split by semicolons into individual statements.

## Security Notes

- Secrets and credentials are provided via environment variables rather than committed into source control.
- API key authentication can be enabled or disabled without code changes.
- Request bodies are size‑limited and parsed carefully to prevent panics and memory exhaustion.
- All SQL queries use prepared statements to avoid injection.
- Errors are sanitized and do not leak sensitive information.
- The code uses Fx for dependency injection and lifecycle management, ensuring proper resource cleanup.

