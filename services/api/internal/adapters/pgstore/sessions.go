package pgstore

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rhs96/cantinho-do-pomar/services/api/internal/app/auth"
)

type Sessions struct {
	db *pgxpool.Pool
}

func NewSessions(db *pgxpool.Pool) *Sessions {
	return &Sessions{db: db}
}

func (r *Sessions) Create(ctx context.Context, s auth.Session) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO sessions (token_hash, player_id, expires_at)
		VALUES ($1, $2, $3)`,
		s.TokenHash, s.PlayerID, s.ExpiresAt,
	)
	return err
}

func (r *Sessions) Find(ctx context.Context, tokenHash []byte) (auth.Session, error) {
	s := auth.Session{TokenHash: tokenHash}
	err := r.db.QueryRow(ctx, `
		SELECT player_id, expires_at
		FROM sessions
		WHERE token_hash = $1`, tokenHash,
	).Scan(&s.PlayerID, &s.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return auth.Session{}, auth.ErrSessionNotFound
	}
	return s, err
}
