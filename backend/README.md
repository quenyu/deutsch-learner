# Backend

Go API scaffold for the curated German-learning platform.

## Layers

- `internal/domain`: core entities and invariants
- `internal/application`: use-case services
- `internal/infrastructure`: adapters (Postgres + in-memory bootstrap)
- `internal/presentation`: HTTP transport
- `db/migrations`: PostgreSQL schema

## API surface (initial)

- `GET /healthz`
- `GET /readyz`
- `GET /api/v1/resources`
- `GET /api/v1/resources/{slug}`
- `GET /api/v1/me/saved-resources`
- `POST /api/v1/me/saved-resources`
- `DELETE /api/v1/me/saved-resources/{resourceId}`

## Runtime guardrails

- Request ID propagation (`X-Request-ID`)
- Structured request logging
- Panic recovery
- Security response headers
- Body size limit for write methods
- Configurable CORS allow-list
- Per-IP fixed-window rate limiting
- Concurrency limiting
- Handler timeout
- Slow-request logging
- HTTP server read/write/header/idle timeouts
- Graceful shutdown on `SIGINT` / `SIGTERM`
- `/readyz` includes dependency TCP checks for Postgres/Redis addresses

## Run

```bash
go test ./...
go run ./cmd/api
```
