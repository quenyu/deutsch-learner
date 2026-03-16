package main

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	catalogapp "deutsch-learner/backend/internal/application/catalog"
	savedapp "deutsch-learner/backend/internal/application/saved"
	"deutsch-learner/backend/internal/infrastructure/memory"
	"deutsch-learner/backend/internal/platform/config"
	httpapi "deutsch-learner/backend/internal/presentation/http"
)

func main() {
	cfg := config.Load()

	catalogRepo := memory.NewCatalogRepository(memory.DefaultResources())
	savedRepo := memory.NewSavedRepository()

	catalogService := catalogapp.NewService(catalogRepo)
	savedService := savedapp.NewService(savedRepo)
	readinessChecks := buildReadinessChecks(cfg)

	server := httpapi.NewServer(catalogService, savedService, httpapi.Options{
		CORSAllowedOrigins:         cfg.CORSAllowedOrigins,
		MaxBodyBytes:               cfg.MaxBodyBytes,
		MaxConcurrentRequests:      cfg.MaxConcurrentRequests,
		RateLimitEnabled:           cfg.RateLimitEnabled,
		RateLimitRequestsPerWindow: cfg.RateLimitRequestsPerWindow,
		RateLimitWindow:            cfg.RateLimitWindow,
		HandlerTimeout:             cfg.HandlerTimeout,
		SlowRequestThreshold:       cfg.SlowRequestThreshold,
		ReadinessTimeout:           cfg.ReadinessTimeout,
		ReadinessChecks:            readinessChecks,
	})

	addr := ":" + cfg.Port
	httpServer := &http.Server{
		Addr:              addr,
		Handler:           server.Routes(),
		ReadTimeout:       cfg.ReadTimeout,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		MaxHeaderBytes:    cfg.MaxHeaderBytes,
	}

	stopSignal := make(chan os.Signal, 1)
	signal.Notify(stopSignal, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-stopSignal
		log.Printf("shutdown signal received, stopping server")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()

		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			log.Printf("graceful shutdown error: %v", err)
		}
	}()

	log.Printf("deutsch-learner api listening on %s", addr)
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func buildReadinessChecks(cfg config.Config) []httpapi.ReadinessCheck {
	checks := make([]httpapi.ReadinessCheck, 0, 2)

	postgresAddress := postgresAddressFromDSN(cfg.PostgresDSN)
	if postgresAddress != "" {
		checks = append(checks, tcpDialReadinessCheck("postgres_tcp", postgresAddress))
	}

	redisAddress := strings.TrimSpace(cfg.RedisAddr)
	if redisAddress != "" {
		checks = append(checks, tcpDialReadinessCheck("redis_tcp", redisAddress))
	}

	return checks
}

func tcpDialReadinessCheck(name, address string) httpapi.ReadinessCheck {
	return httpapi.ReadinessCheck{
		Name: name,
		Check: func(ctx context.Context) error {
			dialer := net.Dialer{Timeout: 800 * time.Millisecond}
			conn, err := dialer.DialContext(ctx, "tcp", address)
			if err != nil {
				return err
			}
			_ = conn.Close()
			return nil
		},
	}
}

func postgresAddressFromDSN(dsn string) string {
	parsed, err := url.Parse(strings.TrimSpace(dsn))
	if err != nil {
		return ""
	}
	return parsed.Host
}
