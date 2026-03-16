package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	catalogapp "deutsch-learner/backend/internal/application/catalog"
	savedapp "deutsch-learner/backend/internal/application/saved"
	"deutsch-learner/backend/internal/infrastructure/memory"
	"deutsch-learner/backend/internal/infrastructure/postgres"
	"deutsch-learner/backend/internal/platform/config"
	httpapi "deutsch-learner/backend/internal/presentation/http"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/redis/go-redis/v9"
)

type runtimeComponents struct {
	catalogRepo     catalogapp.Repository
	savedRepo       savedapp.Repository
	readinessChecks []httpapi.ReadinessCheck
	closeFn         func(context.Context) error
}

func (r runtimeComponents) Close(ctx context.Context) error {
	if r.closeFn == nil {
		return nil
	}
	return r.closeFn(ctx)
}

func buildRuntime(cfg config.Config) (runtimeComponents, error) {
	switch cfg.DataBackend {
	case "memory":
		return runtimeComponents{
			catalogRepo: memory.NewCatalogRepository(memory.DefaultResources()),
			savedRepo:   memory.NewSavedRepository(),
		}, nil
	case "postgres":
		return buildPostgresRuntime(cfg)
	default:
		return runtimeComponents{}, fmt.Errorf("unsupported data backend %q", cfg.DataBackend)
	}
}

func buildPostgresRuntime(cfg config.Config) (runtimeComponents, error) {
	db, err := sql.Open("pgx", cfg.PostgresDSN)
	if err != nil {
		return runtimeComponents{}, fmt.Errorf("open postgres connection: %w", err)
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr: strings.TrimSpace(cfg.RedisAddr),
	})

	closeFn := buildCloseFn(db, redisClient)

	bootCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(bootCtx); err != nil {
		_ = closeFn(context.Background())
		return runtimeComponents{}, fmt.Errorf("postgres connectivity check failed: %w", err)
	}

	if err := ensureRequiredTables(bootCtx, db, []string{
		"catalog_resources",
		"catalog_skills",
		"catalog_topics",
		"catalog_resource_skills",
		"catalog_resource_topics",
		"user_saved_resources",
		"app_users",
	}); err != nil {
		_ = closeFn(context.Background())
		return runtimeComponents{}, err
	}

	if strings.TrimSpace(cfg.RedisAddr) != "" {
		if err := redisClient.Ping(bootCtx).Err(); err != nil {
			_ = closeFn(context.Background())
			return runtimeComponents{}, fmt.Errorf("redis connectivity check failed: %w", err)
		}
	}

	readinessChecks := []httpapi.ReadinessCheck{
		{
			Name: "postgres",
			Check: func(ctx context.Context) error {
				return db.PingContext(ctx)
			},
		},
	}

	if strings.TrimSpace(cfg.RedisAddr) != "" {
		readinessChecks = append(readinessChecks, httpapi.ReadinessCheck{
			Name: "redis",
			Check: func(ctx context.Context) error {
				return redisClient.Ping(ctx).Err()
			},
		})
	}

	return runtimeComponents{
		catalogRepo:     postgres.NewCatalogRepository(db),
		savedRepo:       postgres.NewSavedRepository(db),
		readinessChecks: readinessChecks,
		closeFn:         closeFn,
	}, nil
}

func buildCloseFn(db *sql.DB, redisClient *redis.Client) func(context.Context) error {
	var (
		once     sync.Once
		closeErr error
	)

	return func(_ context.Context) error {
		once.Do(func() {
			errs := make([]error, 0, 2)
			if redisClient != nil {
				if err := redisClient.Close(); err != nil {
					errs = append(errs, fmt.Errorf("close redis: %w", err))
				}
			}
			if db != nil {
				if err := db.Close(); err != nil {
					errs = append(errs, fmt.Errorf("close postgres: %w", err))
				}
			}
			closeErr = errors.Join(errs...)
		})

		return closeErr
	}
}

func ensureRequiredTables(ctx context.Context, db *sql.DB, tables []string) error {
	missing := make([]string, 0, len(tables))

	for _, table := range tables {
		var exists bool
		err := db.QueryRowContext(
			ctx,
			`
SELECT EXISTS (
	SELECT 1
	FROM information_schema.tables
	WHERE table_schema = 'public'
	  AND table_name = $1
);`,
			table,
		).Scan(&exists)
		if err != nil {
			return fmt.Errorf("check required table %q: %w", table, err)
		}

		if !exists {
			missing = append(missing, table)
		}
	}

	if len(missing) == 0 {
		return nil
	}

	return fmt.Errorf("missing required database tables: %s; run migrations first", strings.Join(missing, ", "))
}
