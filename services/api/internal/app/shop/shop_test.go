package shop

import (
	"errors"
	"testing"
	"time"

	"github.com/rhs96/cantinho-do-pomar/services/api/internal/app/apptest"
	"github.com/rhs96/cantinho-do-pomar/services/api/internal/catalog"
	"github.com/rhs96/cantinho-do-pomar/services/api/internal/domain/farm"
	"github.com/rhs96/cantinho-do-pomar/services/api/internal/domain/inventory"
	"github.com/rhs96/cantinho-do-pomar/services/api/internal/domain/player"
)

func newShop(t *testing.T, coins int64) (*Service, *apptest.Store, player.ID) {
	t.Helper()
	store := apptest.NewStore(catalog.Default())
	guest, _ := player.NewGuest("Ana", time.Now())
	guest.Coins = coins
	id := store.AddPlayer(guest).ID
	return NewService(store, catalog.Default()), store, id
}

func TestBuySeeds(t *testing.T) {
	svc, store, owner := newShop(t, 100)
	tomato, _ := catalog.Default().Entry("tomato")

	res, err := svc.BuySeeds(t.Context(), owner, "tomato", 3)
	if err != nil {
		t.Fatalf("BuySeeds() = %v", err)
	}
	if want := 100 - 3*tomato.SeedPrice; res.Player.Coins != want {
		t.Errorf("coins = %d, want %d", res.Player.Coins, want)
	}
	if got := store.Quantity(owner, inventory.Seed("tomato")); got != 3 {
		t.Errorf("seeds = %d, want 3", got)
	}
	if len(res.Inventory) != 1 || res.Inventory[0].Quantity != 3 {
		t.Errorf("inventory = %+v", res.Inventory)
	}
}

func TestBuySeedsWithoutEnoughCoins(t *testing.T) {
	svc, store, owner := newShop(t, 20)

	if _, err := svc.BuySeeds(t.Context(), owner, "tomato", 1); !errors.Is(err, player.ErrNotEnoughCoins) {
		t.Fatalf("BuySeeds() = %v, want ErrNotEnoughCoins", err)
	}
	if store.Player(owner).Coins != 20 {
		t.Error("coins changed after a failed purchase")
	}
	if store.Quantity(owner, inventory.Seed("tomato")) != 0 {
		t.Error("seeds added after a failed purchase")
	}
}

func TestSell(t *testing.T) {
	svc, store, owner := newShop(t, 0)
	tomato, _ := catalog.Default().Entry("tomato")
	seedProduce(t, store, owner, "tomato", 10)

	res, err := svc.Sell(t.Context(), owner, "tomato", 4)
	if err != nil {
		t.Fatalf("Sell() = %v", err)
	}
	if want := 4 * tomato.SellPrice; res.Player.Coins != want {
		t.Errorf("coins = %d, want %d", res.Player.Coins, want)
	}
	if got := store.Quantity(owner, inventory.Produce("tomato")); got != 6 {
		t.Errorf("produce = %d, want 6", got)
	}

	if _, err := svc.Sell(t.Context(), owner, "tomato", 7); !errors.Is(err, inventory.ErrNotEnough) {
		t.Errorf("Sell() more than held = %v, want ErrNotEnough", err)
	}
	if store.Player(owner).Coins != 4*tomato.SellPrice {
		t.Error("coins changed after a failed sale")
	}
}

func TestValidation(t *testing.T) {
	svc, _, owner := newShop(t, 1000)

	if _, err := svc.BuySeeds(t.Context(), owner, "mandrake", 1); !errors.Is(err, ErrUnknownCrop) {
		t.Errorf("unknown crop = %v", err)
	}
	for _, qty := range []int{0, -3, MaxQuantity + 1} {
		if _, err := svc.BuySeeds(t.Context(), owner, "tomato", qty); !errors.Is(err, ErrInvalidQuantity) {
			t.Errorf("BuySeeds(qty=%d) = %v, want ErrInvalidQuantity", qty, err)
		}
		if _, err := svc.Sell(t.Context(), owner, "tomato", qty); !errors.Is(err, ErrInvalidQuantity) {
			t.Errorf("Sell(qty=%d) = %v, want ErrInvalidQuantity", qty, err)
		}
	}
}

func seedProduce(t *testing.T, store *apptest.Store, owner player.ID, crop string, qty int) {
	t.Helper()
	if err := store.Give(owner, inventory.Produce(farm.CropID(crop)), qty); err != nil {
		t.Fatalf("give produce: %v", err)
	}
}

func TestBuyLockedSeeds(t *testing.T) {
	svc, store, owner := newShop(t, 10_000)
	corn, _ := catalog.Default().Entry("corn")

	if _, err := svc.BuySeeds(t.Context(), owner, corn.ID, 1); !errors.Is(err, ErrCropLocked) {
		t.Fatalf("BuySeeds() at level 1 = %v, want ErrCropLocked", err)
	}

	p := store.Player(owner)
	p.XP = player.XPForLevel(corn.MinLevel)
	store.SetPlayer(p)

	if _, err := svc.BuySeeds(t.Context(), owner, corn.ID, 1); err != nil {
		t.Errorf("BuySeeds() at level %d = %v", corn.MinLevel, err)
	}
}
