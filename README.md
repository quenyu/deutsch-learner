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
- Infra guide:
  - `docs/infrastructure.md`

## Notes

- Backend currently defaults to in-memory repositories for local bootstrap.
- PostgreSQL schema and repository scaffolding are included under `backend/db/migrations` and `backend/internal/infrastructure/postgres`.
- Redis is reserved for caching and short-lived state.
