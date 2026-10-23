package postgres_test

import (
	"io"
	"log/slog"
	"testing"

	"github.com/rhs96/cantinho-do-pomar/services/api/internal/platform/postgres"
	"github.com/rhs96/cantinho-do-pomar/services/api/internal/platform/postgres/pgtest"
)

func TestMigrateUpDownUp(t *testing.T) {
	pool := pgtest.NewDB(t)
	ctx := t.Context()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	if err := postgres.Migrate(ctx, pool, logger); err != nil {
		t.Fatalf("Migrate() = %v", err)
	}
	for _, table := range []string{"players", "farms", "plots", "sessions", "inventory"} {
		var exists bool
		err := pool.QueryRow(ctx, "SELECT to_regclass($1) IS NOT NULL", table).Scan(&exists)
		if err != nil || !exists {
			t.Errorf("table %s missing after migrate (err=%v)", table, err)
		}
	}

	if err := postgres.MigrateDownTo(ctx, pool, 0); err != nil {
		t.Fatalf("MigrateDownTo(0) = %v", err)
	}
	if err := postgres.Migrate(ctx, pool, logger); err != nil {
		t.Fatalf("Migrate() after rollback = %v", err)
	}
}

func TestPlotConstraints(t *testing.T) {
	pool := pgtest.NewMigratedDB(t)
	ctx := t.Context()

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
