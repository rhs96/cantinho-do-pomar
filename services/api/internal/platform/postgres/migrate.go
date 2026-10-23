package postgres

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/rhs96/cantinho-do-pomar/services/api/migrations"
)

func newMigrator(pool *pgxpool.Pool) (*goose.Provider, func() error, error) {
	db := stdlib.OpenDBFromPool(pool)
	p, err := goose.NewProvider(goose.DialectPostgres, db, migrations.FS)
	if err != nil {
		_ = db.Close()
		return nil, nil, fmt.Errorf("create migrator: %w", err)
	}
	return p, db.Close, nil
}

func Migrate(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger) error {
	p, closeDB, err := newMigrator(pool)
	if err != nil {
		return err
	}
	defer func() { _ = closeDB() }()

	results, err := p.Up(ctx)
	if err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	for _, r := range results {
		logger.Info("migration applied", "version", r.Source.Version, "file", r.Source.Path, "duration", r.Duration)
	}
	return nil
}

func MigrateDownTo(ctx context.Context, pool *pgxpool.Pool, version int64) error {
	p, closeDB, err := newMigrator(pool)
	if err != nil {
		return err
	}
	defer func() { _ = closeDB() }()

	if _, err := p.DownTo(ctx, version); err != nil {
		return fmt.Errorf("roll back migrations: %w", err)
	}
	return nil
}
