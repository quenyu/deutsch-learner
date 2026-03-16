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

	err = postgres.SeedCuratedCatalog(ctx, db, postgres.SeedOptions{
		Enabled:       cfg.SeedEnabled,
		DemoUserID:    cfg.SeedDemoUserID,
		DemoUserEmail: cfg.SeedDemoUserEmail,
		DemoUserName:  cfg.SeedDemoUserName,
	})
	if err != nil {
		log.Fatalf("seed curated catalog: %v", err)
	}

	if !cfg.SeedEnabled {
		log.Printf("seed step skipped because SEED_ENABLED=false")
		return
	}

	log.Printf("seed complete")
}
