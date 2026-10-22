package postgres

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// newTestDB creates an empty database for the test and drops it afterwards.
// Set TEST_DATABASE_URL to a server where the user can create databases.
func newTestDB(t *testing.T) *pgxpool.Pool {
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

	name := fmt.Sprintf("test_%d", time.Now().UnixNano())
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

func TestMigrateUpDownUp(t *testing.T) {
	pool := newTestDB(t)
	ctx := t.Context()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	if err := Migrate(ctx, pool, logger); err != nil {
		t.Fatalf("Migrate() = %v", err)
	}
	for _, table := range []string{"players", "farms", "plots"} {
		var exists bool
		err := pool.QueryRow(ctx, "SELECT to_regclass($1) IS NOT NULL", table).Scan(&exists)
		if err != nil || !exists {
			t.Errorf("table %s missing after migrate (err=%v)", table, err)
		}
	}

	m, closeDB, err := newMigrator(pool)
	if err != nil {
		t.Fatalf("newMigrator() = %v", err)
	}
	defer func() { _ = closeDB() }()

	if _, err := m.DownTo(ctx, 0); err != nil {
		t.Fatalf("DownTo(0) = %v", err)
	}
	if err := Migrate(ctx, pool, logger); err != nil {
		t.Fatalf("Migrate() after rollback = %v", err)
	}
}

func TestPlotConstraints(t *testing.T) {
	pool := newTestDB(t)
	ctx := t.Context()

	if err := Migrate(ctx, pool, slog.New(slog.NewTextHandler(io.Discard, nil))); err != nil {
		t.Fatalf("Migrate() = %v", err)
	}

	var farmID string
	err := pool.QueryRow(ctx, `
		WITH p AS (INSERT INTO players (display_name) VALUES ('Ana') RETURNING id)
		INSERT INTO farms (player_id, width, height) SELECT id, 6, 6 FROM p RETURNING id
	`).Scan(&farmID)
	if err != nil {
		t.Fatalf("seed farm: %v", err)
	}

	tests := []struct {
		name    string
		state   string
		cropID  *string
		wantErr bool
	}{
		{"grass without crop", "grass", nil, false},
		{"planted with crop", "planted", ptr("tomato"), false},
		{"planted without crop", "planted", nil, true},
		{"plowed with crop", "plowed", ptr("tomato"), true},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := pool.Exec(ctx,
				`INSERT INTO plots (farm_id, col, "row", state, crop_id) VALUES ($1, $2, 0, $3, $4)`,
				farmID, i, tt.state, tt.cropID)
			if (err != nil) != tt.wantErr {
				t.Errorf("insert err = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func ptr[T any](v T) *T { return &v }
