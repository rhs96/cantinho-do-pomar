package pgstore

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/rhs96/cantinho-do-pomar/services/api/internal/domain/player"
)

type Players struct {
	db DBTX
}

func NewPlayers(db DBTX) *Players {
	return &Players{db: db}
}

func (r *Players) Create(ctx context.Context, p player.Player) (player.Player, error) {
	err := r.db.QueryRow(ctx, `
		INSERT INTO players (display_name, coins, xp, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $4)
		RETURNING id`,
		p.DisplayName, p.Coins, p.XP, p.CreatedAt,
	).Scan(&p.ID)
	return p, err
}

func (r *Players) Get(ctx context.Context, id player.ID) (player.Player, error) {
	return r.get(ctx, id, "")
}

func (r *Players) GetForUpdate(ctx context.Context, id player.ID) (player.Player, error) {
	return r.get(ctx, id, "FOR UPDATE")
}

func (r *Players) get(ctx context.Context, id player.ID, lock string) (player.Player, error) {
	var p player.Player
	err := r.db.QueryRow(ctx, `
		SELECT id, display_name, coins, xp, created_at
		FROM players
		WHERE id = $1 `+lock, id,
	).Scan(&p.ID, &p.DisplayName, &p.Coins, &p.XP, &p.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return player.Player{}, player.ErrNotFound
	}
	return p, err
}
