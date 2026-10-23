package auth

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/rhs96/cantinho-do-pomar/services/api/internal/domain/player"
)

type memPlayers struct {
	byID map[player.ID]player.Player
	err  error
}

func (m *memPlayers) Create(_ context.Context, p player.Player) (player.Player, error) {
	if m.err != nil {
		return player.Player{}, m.err
	}
	p.ID = player.ID(fmt.Sprintf("p%d", len(m.byID)+1))
	m.byID[p.ID] = p
	return p, nil
}

func (m *memPlayers) Get(_ context.Context, id player.ID) (player.Player, error) {
	p, ok := m.byID[id]
	if !ok {
		return player.Player{}, player.ErrNotFound
	}
	return p, nil
}

type memSessions struct {
	list []Session
}

func (m *memSessions) Create(_ context.Context, s Session) error {
	m.list = append(m.list, s)
	return nil
}

func (m *memSessions) Find(_ context.Context, hash []byte) (Session, error) {
	for _, s := range m.list {
		if bytes.Equal(s.TokenHash, hash) {
			return s, nil
		}
	}
	return Session{}, ErrSessionNotFound
}

type fixture struct {
	svc      *Service
	players  *memPlayers
	sessions *memSessions
	now      time.Time
}

func newFixture() *fixture {
	f := &fixture{
		players:  &memPlayers{byID: map[player.ID]player.Player{}},
		sessions: &memSessions{},
		now:      time.Date(2026, 10, 23, 9, 0, 0, 0, time.UTC),
	}
	f.svc = NewService(f.players, f.sessions, func() time.Time { return f.now }, 24*time.Hour)
	return f
}

func TestStartGuest(t *testing.T) {
	f := newFixture()

	token, p, err := f.svc.StartGuest(t.Context())
	if err != nil {
		t.Fatalf("StartGuest() = %v", err)
	}
	if token == "" || p.ID == "" {
		t.Fatalf("token = %q, id = %q; want both set", token, p.ID)
	}
	if p.Coins != player.StartingCoins {
		t.Errorf("Coins = %d, want %d", p.Coins, player.StartingCoins)
	}

	if len(f.sessions.list) != 1 {
		t.Fatalf("sessions = %d, want 1", len(f.sessions.list))
	}
	s := f.sessions.list[0]
	if bytes.Contains(s.TokenHash, []byte(token)) {
		t.Error("session stores the raw token")
	}
	if !s.ExpiresAt.Equal(f.now.Add(24 * time.Hour)) {
		t.Errorf("ExpiresAt = %v", s.ExpiresAt)
	}
}

func TestStartGuestGeneratesDistinctTokens(t *testing.T) {
	f := newFixture()

	a, _, _ := f.svc.StartGuest(t.Context())
	b, _, _ := f.svc.StartGuest(t.Context())
	if a == b {
		t.Error("two guests got the same token")
	}
}

func TestStartGuestPropagatesErrors(t *testing.T) {
	f := newFixture()
	f.players.err = errors.New("db down")

	if _, _, err := f.svc.StartGuest(t.Context()); err == nil {
		t.Fatal("expected error")
	}
	if len(f.sessions.list) != 0 {
		t.Error("session created even though the player was not")
	}
}

func TestAuthenticate(t *testing.T) {
	f := newFixture()
	token, guest, err := f.svc.StartGuest(t.Context())
	if err != nil {
		t.Fatalf("StartGuest() = %v", err)
	}

	t.Run("valid token", func(t *testing.T) {
		p, err := f.svc.Authenticate(t.Context(), token)
		if err != nil {
			t.Fatalf("Authenticate() = %v", err)
		}
		if p.ID != guest.ID {
			t.Errorf("player = %q, want %q", p.ID, guest.ID)
		}
	})

	t.Run("unknown token", func(t *testing.T) {
		if _, err := f.svc.Authenticate(t.Context(), "nope"); !errors.Is(err, ErrUnauthenticated) {
			t.Errorf("Authenticate() = %v, want ErrUnauthenticated", err)
		}
	})

	t.Run("empty token", func(t *testing.T) {
		if _, err := f.svc.Authenticate(t.Context(), ""); !errors.Is(err, ErrUnauthenticated) {
			t.Errorf("Authenticate() = %v, want ErrUnauthenticated", err)
		}
	})

	t.Run("expired", func(t *testing.T) {
		f.now = f.now.Add(24 * time.Hour)
		if _, err := f.svc.Authenticate(t.Context(), token); !errors.Is(err, ErrUnauthenticated) {
			t.Errorf("Authenticate() = %v, want ErrUnauthenticated", err)
		}
	})
}

func TestAuthenticateDeletedPlayer(t *testing.T) {
	f := newFixture()
	token, guest, _ := f.svc.StartGuest(t.Context())
	delete(f.players.byID, guest.ID)

	if _, err := f.svc.Authenticate(t.Context(), token); !errors.Is(err, ErrUnauthenticated) {
		t.Errorf("Authenticate() = %v, want ErrUnauthenticated", err)
	}
}
