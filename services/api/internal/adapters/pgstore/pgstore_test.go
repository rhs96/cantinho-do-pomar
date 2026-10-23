package pgstore_test

import (
	"errors"
	"testing"
	"time"

	"github.com/rhs96/cantinho-do-pomar/services/api/internal/adapters/pgstore"
	"github.com/rhs96/cantinho-do-pomar/services/api/internal/app/auth"
	"github.com/rhs96/cantinho-do-pomar/services/api/internal/domain/player"
	"github.com/rhs96/cantinho-do-pomar/services/api/internal/platform/postgres/pgtest"
)

var now = time.Date(2026, 10, 23, 9, 0, 0, 0, time.UTC)

func TestPlayersCreateAndGet(t *testing.T) {
	db := pgtest.NewMigratedDB(t)
	repo := pgstore.NewPlayers(db)
	ctx := t.Context()

	guest, _ := player.NewGuest("Ana", now)
	created, err := repo.Create(ctx, guest)
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}
	if created.ID == "" {
		t.Fatal("Create() returned no id")
	}

	got, err := repo.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get() = %v", err)
	}
	if got.DisplayName != "Ana" || got.Coins != player.StartingCoins || !got.CreatedAt.Equal(now) {
		t.Errorf("Get() = %+v", got)
	}
}

func TestPlayersGetMissing(t *testing.T) {
	db := pgtest.NewMigratedDB(t)
	repo := pgstore.NewPlayers(db)

	_, err := repo.Get(t.Context(), "6f1c2a4e-0000-4000-8000-000000000000")
	if !errors.Is(err, player.ErrNotFound) {
		t.Errorf("Get() = %v, want ErrNotFound", err)
	}
}

func TestSessionsCreateAndFind(t *testing.T) {
	db := pgtest.NewMigratedDB(t)
	players := pgstore.NewPlayers(db)
	sessions := pgstore.NewSessions(db)
	ctx := t.Context()

	guest, _ := player.NewGuest("Ana", now)
	p, err := players.Create(ctx, guest)
	if err != nil {
		t.Fatalf("create player: %v", err)
	}

	hash := []byte("0123456789abcdef0123456789abcdef")
	want := auth.Session{TokenHash: hash, PlayerID: p.ID, ExpiresAt: now.Add(time.Hour)}
	if err := sessions.Create(ctx, want); err != nil {
		t.Fatalf("Create() = %v", err)
	}

	got, err := sessions.Find(ctx, hash)
	if err != nil {
		t.Fatalf("Find() = %v", err)
	}
	if got.PlayerID != p.ID || !got.ExpiresAt.Equal(want.ExpiresAt) {
		t.Errorf("Find() = %+v, want %+v", got, want)
	}

	if _, err := sessions.Find(ctx, []byte("other")); !errors.Is(err, auth.ErrSessionNotFound) {
		t.Errorf("Find(unknown) = %v, want ErrSessionNotFound", err)
	}
}
