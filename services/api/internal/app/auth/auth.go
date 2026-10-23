package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	mrand "math/rand/v2"
	"time"

	"github.com/rhs96/cantinho-do-pomar/services/api/internal/domain/player"
)

var (
	ErrUnauthenticated = errors.New("unauthenticated")
	ErrSessionNotFound = errors.New("session not found")
)

type Session struct {
	TokenHash []byte
	PlayerID  player.ID
	ExpiresAt time.Time
}

type Players interface {
	Create(ctx context.Context, p player.Player) (player.Player, error)
	Get(ctx context.Context, id player.ID) (player.Player, error)
}

type Sessions interface {
	Create(ctx context.Context, s Session) error
	// Find returns ErrSessionNotFound when no session has the given hash.
	Find(ctx context.Context, tokenHash []byte) (Session, error)
}

type Service struct {
	players  Players
	sessions Sessions
	now      func() time.Time
	ttl      time.Duration
}

func NewService(players Players, sessions Sessions, now func() time.Time, ttl time.Duration) *Service {
	return &Service{players: players, sessions: sessions, now: now, ttl: ttl}
}

// StartGuest creates a new guest player and a session for it. The returned
// token is the only copy of the secret; only its hash is stored.
func (s *Service) StartGuest(ctx context.Context) (string, player.Player, error) {
	now := s.now()

	guest, err := player.NewGuest(guestName(), now)
	if err != nil {
		return "", player.Player{}, err
	}
	guest, err = s.players.Create(ctx, guest)
	if err != nil {
		return "", player.Player{}, fmt.Errorf("create guest: %w", err)
	}

	token, err := newToken()
	if err != nil {
		return "", player.Player{}, err
	}
	err = s.sessions.Create(ctx, Session{
		TokenHash: hashToken(token),
		PlayerID:  guest.ID,
		ExpiresAt: now.Add(s.ttl),
	})
	if err != nil {
		return "", player.Player{}, fmt.Errorf("create session: %w", err)
	}

	return token, guest, nil
}

func (s *Service) Authenticate(ctx context.Context, token string) (player.Player, error) {
	if token == "" {
		return player.Player{}, ErrUnauthenticated
	}

	sess, err := s.sessions.Find(ctx, hashToken(token))
	if errors.Is(err, ErrSessionNotFound) {
		return player.Player{}, ErrUnauthenticated
	}
	if err != nil {
		return player.Player{}, fmt.Errorf("find session: %w", err)
	}
	if !s.now().Before(sess.ExpiresAt) {
		return player.Player{}, ErrUnauthenticated
	}

	p, err := s.players.Get(ctx, sess.PlayerID)
	if errors.Is(err, player.ErrNotFound) {
		return player.Player{}, ErrUnauthenticated
	}
	if err != nil {
		return player.Player{}, fmt.Errorf("load player: %w", err)
	}
	return p, nil
}

func newToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func hashToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

func guestName() string {
	return fmt.Sprintf("Fazendeiro %04d", mrand.IntN(10000))
}
