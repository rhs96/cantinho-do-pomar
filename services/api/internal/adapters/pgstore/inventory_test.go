package pgstore_test

import (
	"errors"
	"testing"

	"github.com/rhs96/cantinho-do-pomar/services/api/internal/adapters/pgstore"
	"github.com/rhs96/cantinho-do-pomar/services/api/internal/domain/inventory"
	"github.com/rhs96/cantinho-do-pomar/services/api/internal/domain/player"
	"github.com/rhs96/cantinho-do-pomar/services/api/internal/platform/postgres/pgtest"
)

func TestInventoryAddRemoveList(t *testing.T) {
	db := pgtest.NewMigratedDB(t)
	owner := createPlayer(t, db)
	repo := pgstore.NewInventory(db)
	ctx := t.Context()

	seeds := inventory.Seed("tomato")
	produce := inventory.Produce("tomato")

	if err := repo.Add(ctx, owner, seeds, 3); err != nil {
		t.Fatalf("Add() = %v", err)
	}
	if err := repo.Add(ctx, owner, seeds, 2); err != nil {
		t.Fatalf("second Add() = %v", err)
	}
	if err := repo.Add(ctx, owner, produce, 10); err != nil {
		t.Fatalf("Add(produce) = %v", err)
	}
	if err := repo.Remove(ctx, owner, seeds, 4); err != nil {
		t.Fatalf("Remove() = %v", err)
	}

	if err := repo.Remove(ctx, owner, seeds, 2); !errors.Is(err, inventory.ErrNotEnough) {
		t.Fatalf("Remove() more than held = %v, want ErrNotEnough", err)
	}
	if err := repo.Remove(ctx, owner, inventory.Seed("corn"), 1); !errors.Is(err, inventory.ErrNotEnough) {
		t.Fatalf("Remove() missing item = %v, want ErrNotEnough", err)
	}
	if err := repo.Add(ctx, owner, seeds, 0); !errors.Is(err, inventory.ErrInvalidQuantity) {
		t.Fatalf("Add(0) = %v, want ErrInvalidQuantity", err)
	}

	got, err := repo.List(ctx, owner)
	if err != nil {
		t.Fatalf("List() = %v", err)
	}
	want := []inventory.Stack{
		{Item: seeds, Quantity: 1},
		{Item: produce, Quantity: 10},
	}
	if len(got) != len(want) {
		t.Fatalf("List() = %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("List()[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestInventoryEmptyStacksAreHidden(t *testing.T) {
	db := pgtest.NewMigratedDB(t)
	owner := createPlayer(t, db)
	repo := pgstore.NewInventory(db)
	ctx := t.Context()

	_ = repo.Add(ctx, owner, inventory.Seed("corn"), 1)
	_ = repo.Remove(ctx, owner, inventory.Seed("corn"), 1)

	got, err := repo.List(ctx, owner)
	if err != nil || len(got) != 0 {
		t.Errorf("List() = %+v, %v; want empty", got, err)
	}
}

func TestPlayersUpdate(t *testing.T) {
	db := pgtest.NewMigratedDB(t)
	repo := pgstore.NewPlayers(db)
	ctx := t.Context()

	p, _ := repo.Get(ctx, createPlayer(t, db))
	p.Coins, p.XP = 999, 42
	if err := repo.Update(ctx, p); err != nil {
		t.Fatalf("Update() = %v", err)
	}

	got, _ := repo.Get(ctx, p.ID)
	if got.Coins != 999 || got.XP != 42 {
		t.Errorf("after Update() coins, xp = %d, %d", got.Coins, got.XP)
	}

	p.Coins = -1
	if err := repo.Update(ctx, p); err == nil {
		t.Error("Update() with negative coins should violate the check constraint")
	}

	missing := player.Player{ID: "6f1c2a4e-0000-4000-8000-000000000000"}
	if err := repo.Update(ctx, missing); !errors.Is(err, player.ErrNotFound) {
		t.Errorf("Update(missing) = %v, want ErrNotFound", err)
	}
}
