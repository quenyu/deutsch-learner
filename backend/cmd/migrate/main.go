package main

import (
	"context"
	"database/sql"
	"log"
	"time"

	"deutsch-learner/backend/internal/infrastructure/postgres"
	"deutsch-learner/backend/internal/platform/config"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	cfg := config.Load()

	db, err := sql.Open("pgx", cfg.PostgresDSN)
	if err != nil {
		log.Fatalf("open postgres connection: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("postgres connectivity check failed: %v", err)
	}

	applied, err := postgres.ApplyMigrations(ctx, db, "db/migrations")
	if err != nil {
		log.Fatalf("apply migrations: %v", err)
	}

	log.Printf("migration complete: %d migration(s) applied", applied)
}
