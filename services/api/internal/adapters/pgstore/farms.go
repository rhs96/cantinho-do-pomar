package pgstore

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rhs96/cantinho-do-pomar/services/api/internal/domain/farm"
	"github.com/rhs96/cantinho-do-pomar/services/api/internal/domain/player"
)

type Farms struct {
	db    *pgxpool.Pool
	crops farm.CropLookup
}

func NewFarms(db *pgxpool.Pool, crops farm.CropLookup) *Farms {
	return &Farms{db: db, crops: crops}
}

func (r *Farms) Get(ctx context.Context, owner player.ID) (*farm.Farm, error) {
	var (
		id            farm.ID
		width, height int
	)
	err := r.db.QueryRow(ctx, `SELECT id, width, height FROM farms WHERE player_id = $1`, owner).
		Scan(&id, &width, &height)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, farm.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	rows, err := r.db.Query(ctx, `
		SELECT col, "row", state, crop_id, growth_ms, growth_at, wet_until
		FROM plots
		WHERE farm_id = $1`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var plots []*farm.Plot
	for rows.Next() {
		p, err := r.scanPlot(rows)
		if err != nil {
			return nil, err
		}
		plots = append(plots, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return farm.Restore(id, owner, width, height, plots)
}

// Create stores a new farm with its plots. If the owner already has one, the
// existing farm is returned instead.
func (r *Farms) Create(ctx context.Context, f *farm.Farm) (*farm.Farm, error) {
	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		var id farm.ID
		err := tx.QueryRow(ctx, `
			INSERT INTO farms (player_id, width, height)
			VALUES ($1, $2, $3)
			ON CONFLICT (player_id) DO NOTHING
			RETURNING id`,
			f.Owner, f.Width, f.Height,
		).Scan(&id)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}

		plots := f.Plots()
		cols := make([]int, len(plots))
		rowsIdx := make([]int, len(plots))
		states := make([]string, len(plots))
		for i, p := range plots {
			cols[i], rowsIdx[i], states[i] = p.Position().Col, p.Position().Row, string(p.State())
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO plots (farm_id, col, "row", state)
			SELECT $1, t.col, t.row, t.state::plot_state
			FROM unnest($2::int[], $3::int[], $4::text[]) AS t(col, row, state)`,
			id, cols, rowsIdx, states,
		)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("create farm: %w", err)
	}
	return r.Get(ctx, f.Owner)
}

// UpdatePlot locks a single plot, hands it to fn and saves the result in the
// same transaction. If fn returns an error nothing is written.
func (r *Farms) UpdatePlot(ctx context.Context, owner player.ID, pos farm.Position, fn func(*farm.Plot) error) (*farm.Plot, error) {
	var plot *farm.Plot

	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		var farmID farm.ID
		row := tx.QueryRow(ctx, `
			SELECT p.farm_id, p.col, p."row", p.state, p.crop_id, p.growth_ms, p.growth_at, p.wet_until
			FROM plots p
			JOIN farms f ON f.id = p.farm_id
			WHERE f.player_id = $1 AND p.col = $2 AND p."row" = $3
			FOR UPDATE OF p`,
			owner, pos.Col, pos.Row,
		)

		var err error
		plot, err = r.scanPlot(row, &farmID)
		if errors.Is(err, pgx.ErrNoRows) {
			return r.missingPlotError(ctx, tx, owner)
		}
		if err != nil {
			return err
		}

		if err := fn(plot); err != nil {
			return err
		}

		s := plot.Snapshot()
		_, err = tx.Exec(ctx, `
			UPDATE plots
			SET state = $4, crop_id = $5, growth_ms = $6, growth_at = $7, wet_until = $8
			WHERE farm_id = $1 AND col = $2 AND "row" = $3`,
			farmID, s.Pos.Col, s.Pos.Row,
			string(s.State), nullString(string(s.CropID)), s.Growth.Milliseconds(),
			nullTime(s.GrowthAt), nullTime(s.WetUntil),
		)
		return err
	})
	if err != nil {
		return nil, err
	}
	return plot, nil
}

func (r *Farms) missingPlotError(ctx context.Context, tx pgx.Tx, owner player.ID) error {
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM farms WHERE player_id = $1)`, owner).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return farm.ErrOutOfBounds
	}
	return farm.ErrNotFound
}

// scanPlot reads the plot columns in the order used by the queries above,
// optionally preceded by extra destinations.
func (r *Farms) scanPlot(row pgx.Row, prefix ...any) (*farm.Plot, error) {
	var (
		s                  farm.PlotSnapshot
		state              string
		cropID             *string
		growthMS           int64
		growthAt, wetUntil *time.Time
	)
	dest := make([]any, 0, len(prefix)+7)
	dest = append(dest, prefix...)
	dest = append(dest, &s.Pos.Col, &s.Pos.Row, &state, &cropID, &growthMS, &growthAt, &wetUntil)
	if err := row.Scan(dest...); err != nil {
		return nil, err
	}

	s.State = farm.PlotState(state)
	s.Growth = time.Duration(growthMS) * time.Millisecond
	if cropID != nil {
		s.CropID = farm.CropID(*cropID)
	}
	if growthAt != nil {
		s.GrowthAt = *growthAt
	}
	if wetUntil != nil {
		s.WetUntil = *wetUntil
	}
	return farm.RestorePlot(s, r.crops)
}

func nullString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func nullTime(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}
