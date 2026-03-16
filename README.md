# Deutsch Learner

German-learning platform focused on curated third-party resources.

## Product scope

- We do not host lesson content.
- We store metadata, categorization, paths, and user state.
- External learning content stays external via trusted outbound links.

## Repository layout

- `backend` Go API with explicit layered architecture.
- `frontend` SvelteKit client with Tailwind and reusable UI primitives.
- `docs` product and architecture source of truth.

## Quick start

1. Start infrastructure:
   - `docker compose up -d`
2. Backend:
   - `cd backend`
   - `go test ./...`
   - `go run ./cmd/migrate`
   - `go run ./cmd/seed`
   - `go run ./cmd/api`
3. Frontend:
   - `cd frontend`
   - `npm install`
   - `npm run dev`

## Docker (full stack)

- Start app + infra:
  - `docker compose --profile app up -d --build`
- Start infra only:
  - `docker compose up -d`
- Run backend migrations and seed in app image:
  - `docker compose --profile app run --rm backend migrate`
  - `docker compose --profile app run --rm backend seed`
- Infra guide:
  - `docs/infrastructure.md`

## Notes

- Backend defaults to `DATA_BACKEND=postgres` for app/runtime profiles.
- Memory repositories remain available via `DATA_BACKEND=memory` for focused local fallback/testing.
- PostgreSQL schema and persistence adapters live under `backend/db/migrations` and `backend/internal/infrastructure/postgres`.
- Redis is reserved for caching and short-lived state.
