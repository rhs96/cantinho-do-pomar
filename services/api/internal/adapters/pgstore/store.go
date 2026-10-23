package pgstore

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rhs96/cantinho-do-pomar/services/api/internal/app"
	"github.com/rhs96/cantinho-do-pomar/services/api/internal/domain/farm"
)

// DBTX is satisfied by both *pgxpool.Pool and pgx.Tx, so repositories work
// the same inside and outside a transaction.
type DBTX interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Begin(ctx context.Context) (pgx.Tx, error)
}

type Store struct {
	pool  *pgxpool.Pool
	crops farm.CropLookup
}

func NewStore(pool *pgxpool.Pool, crops farm.CropLookup) *Store {
	return &Store{pool: pool, crops: crops}
}

func (s *Store) Do(ctx context.Context, fn func(app.Repos) error) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		return fn(repos{db: tx, crops: s.crops})
	})
}

type repos struct {
	db    DBTX
	crops farm.CropLookup
}

func (r repos) Players() app.Players { return NewPlayers(r.db) }
func (r repos) Farms() app.Farms     { return NewFarms(r.db, r.crops) }
