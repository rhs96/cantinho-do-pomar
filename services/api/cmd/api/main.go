package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rhs96/cantinho-do-pomar/services/api/internal/adapters/httpapi"
	"github.com/rhs96/cantinho-do-pomar/services/api/internal/adapters/pgstore"
	"github.com/rhs96/cantinho-do-pomar/services/api/internal/app/auth"
	"github.com/rhs96/cantinho-do-pomar/services/api/internal/app/farming"
	"github.com/rhs96/cantinho-do-pomar/services/api/internal/catalog"
	"github.com/rhs96/cantinho-do-pomar/services/api/internal/config"
	"github.com/rhs96/cantinho-do-pomar/services/api/internal/platform/postgres"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := postgres.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	if cfg.MigrateOnStart {
		if err := postgres.Migrate(ctx, db, logger); err != nil {
			return err
		}
	}

	crops := catalog.Default()
	authSvc := auth.NewService(pgstore.NewPlayers(db), pgstore.NewSessions(db), time.Now, cfg.SessionTTL)
	farmingSvc := farming.NewService(pgstore.NewFarms(db, crops), crops, time.Now)

	router := httpapi.NewRouter(httpapi.Deps{
		Logger:  logger,
		DB:      db,
		Auth:    authSvc,
		Farming: farmingSvc,
		Now:     time.Now,
		Session: httpapi.SessionConfig{
			TTL:          cfg.SessionTTL,
			SecureCookie: cfg.CookieSecure,
		},
	})

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("api listening", "addr", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	logger.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}
