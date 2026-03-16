package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func ApplyMigrations(ctx context.Context, db *sql.DB, migrationsDir string) (int, error) {
	if err := ensureSchemaMigrationsTable(ctx, db); err != nil {
		return 0, err
	}

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return 0, fmt.Errorf("read migrations directory: %w", err)
	}

	applied := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) != ".sql" {
			continue
		}

		appliedThisFile, err := applySingleMigration(ctx, db, migrationsDir, entry.Name())
		if err != nil {
			return applied, err
		}
		if appliedThisFile {
			applied++
		}
	}

	return applied, nil
}

func ensureSchemaMigrationsTable(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(
		ctx,
		`
CREATE TABLE IF NOT EXISTS schema_migrations (
  version TEXT PRIMARY KEY,
  applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
);`,
	)
	if err != nil {
		return fmt.Errorf("create schema_migrations table: %w", err)
	}

	return nil
}

func applySingleMigration(ctx context.Context, db *sql.DB, migrationsDir, filename string) (bool, error) {
	version := strings.TrimSpace(filename)
	if version == "" {
		return false, nil
	}

	var alreadyApplied bool
	err := db.QueryRowContext(
		ctx,
		`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1);`,
		version,
	).Scan(&alreadyApplied)
	if err != nil {
		return false, fmt.Errorf("check migration version %q: %w", version, err)
	}
	if alreadyApplied {
		return false, nil
	}

	content, err := os.ReadFile(filepath.Join(migrationsDir, filename))
	if err != nil {
		return false, fmt.Errorf("read migration file %q: %w", filename, err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("begin migration transaction %q: %w", filename, err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if _, err := tx.ExecContext(ctx, string(content)); err != nil {
		return false, fmt.Errorf("execute migration %q: %w", filename, err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (version) VALUES ($1);`, version); err != nil {
		return false, fmt.Errorf("record migration %q: %w", filename, err)
	}

	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("commit migration %q: %w", filename, err)
	}

	return true, nil
}
