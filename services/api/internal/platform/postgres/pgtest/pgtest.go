// Package pgtest provides throwaway databases for integration tests.
package pgtest

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rhs96/cantinho-do-pomar/services/api/internal/platform/postgres"
)

var seq atomic.Int64

// NewDB creates an empty database and drops it when the test ends. It skips the
// test unless TEST_DATABASE_URL points to a server where the user can create
// databases.
func NewDB(t testing.TB) *pgxpool.Pool {
	t.Helper()

	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	ctx := t.Context()
	admin, err := pgx.Connect(ctx, url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { _ = admin.Close(context.Background()) })

	name := fmt.Sprintf("test_%d_%d", time.Now().UnixNano(), seq.Add(1))
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatalf("create database: %v", err)
	}

	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	cfg.ConnConfig.Database = name

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("open pool: %v", err)
	}
	t.Cleanup(func() {
		pool.Close()
		_, _ = admin.Exec(context.Background(), "DROP DATABASE IF EXISTS "+name+" WITH (FORCE)")
	})

	return pool
}

// NewMigratedDB is NewDB with all migrations applied.
func NewMigratedDB(t testing.TB) *pgxpool.Pool {
	t.Helper()

	pool := NewDB(t)
	if err := postgres.Migrate(t.Context(), pool, slog.New(slog.NewTextHandler(io.Discard, nil))); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return pool
}
