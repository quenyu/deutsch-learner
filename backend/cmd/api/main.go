package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	catalogapp "deutsch-learner/backend/internal/application/catalog"
	profileapp "deutsch-learner/backend/internal/application/profile"
	progressapp "deutsch-learner/backend/internal/application/progress"
	savedapp "deutsch-learner/backend/internal/application/saved"
	sourceapp "deutsch-learner/backend/internal/application/source"
	"deutsch-learner/backend/internal/platform/config"
	httpapi "deutsch-learner/backend/internal/presentation/http"
)

func main() {
	cfg := config.Load()

	runtimeComponents, err := buildRuntime(cfg)
	if err != nil {
		log.Fatal(err)
	}

	catalogService := catalogapp.NewService(runtimeComponents.catalogRepo)
	profileService := profileapp.NewService(runtimeComponents.profileRepo)
	progressService := progressapp.NewService(runtimeComponents.progressRepo)
	savedService := savedapp.NewService(runtimeComponents.savedRepo)
	sourceService := sourceapp.NewService(runtimeComponents.sourceRepo)

	server := httpapi.NewServer(catalogService, profileService, progressService, savedService, sourceService, httpapi.Options{
		CORSAllowedOrigins:         cfg.CORSAllowedOrigins,
		MaxBodyBytes:               cfg.MaxBodyBytes,
		MaxConcurrentRequests:      cfg.MaxConcurrentRequests,
		RateLimitEnabled:           cfg.RateLimitEnabled,
		RateLimitRequestsPerWindow: cfg.RateLimitRequestsPerWindow,
		RateLimitWindow:            cfg.RateLimitWindow,
		HandlerTimeout:             cfg.HandlerTimeout,
		SlowRequestThreshold:       cfg.SlowRequestThreshold,
		ReadinessTimeout:           cfg.ReadinessTimeout,
		ReadinessChecks:            runtimeComponents.readinessChecks,
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
		if err := runtimeComponents.Close(shutdownCtx); err != nil {
			log.Printf("dependency shutdown error: %v", err)
		}
	}()

	log.Printf("deutsch-learner api listening on %s", addr)
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
	if err := runtimeComponents.Close(context.Background()); err != nil {
		log.Printf("dependency shutdown error: %v", err)
	}
}
