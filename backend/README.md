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
- `/readyz` runs live dependency checks (Postgres `PingContext`, Redis `PING`)
- Startup fail-fast in `DATA_BACKEND=postgres` mode:
  - Postgres connectivity check
  - Required table presence check
  - Redis connectivity check

## Run

```bash
go test ./...
go run ./cmd/migrate
go run ./cmd/seed
go run ./cmd/api
```

## Runtime modes

- `DATA_BACKEND=postgres` (default): Postgres repositories for catalog + saved resources
- `DATA_BACKEND=memory`: in-memory repositories (intended for fast local fallback/tests)
