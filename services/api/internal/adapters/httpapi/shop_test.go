package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"testing"

	"github.com/rhs96/cantinho-do-pomar/services/api/internal/app/shop"
	"github.com/rhs96/cantinho-do-pomar/services/api/internal/catalog"
	"github.com/rhs96/cantinho-do-pomar/services/api/internal/domain/farm"
	"github.com/rhs96/cantinho-do-pomar/services/api/internal/domain/inventory"
	"github.com/rhs96/cantinho-do-pomar/services/api/internal/domain/player"
)

type fakeShop struct {
	err      error
	lastCrop farm.CropID
	lastQty  int
}

func (f *fakeShop) Catalog() []catalog.Entry { return catalog.Default().Entries() }

func (f *fakeShop) Inventory(context.Context, player.ID) ([]inventory.Stack, error) {
	return []inventory.Stack{
		{Item: inventory.Seed("tomato"), Quantity: 3},
		{Item: inventory.Produce("corn"), Quantity: 12},
	}, f.err
}

func (f *fakeShop) result(crop farm.CropID, qty int) (shop.Result, error) {
	f.lastCrop, f.lastQty = crop, qty
	if f.err != nil {
		return shop.Result{}, f.err
	}
	return shop.Result{
		Player:    player.Player{ID: "p1", Coins: 125},
		Inventory: []inventory.Stack{{Item: inventory.Seed(crop), Quantity: qty}},
	}, nil
}

func (f *fakeShop) BuySeeds(_ context.Context, _ player.ID, crop farm.CropID, qty int) (shop.Result, error) {
	return f.result(crop, qty)
}

func (f *fakeShop) Sell(_ context.Context, _ player.ID, crop farm.CropID, qty int) (shop.Result, error) {
	return f.result(crop, qty)
}

func newShopRouter(s Shop) http.Handler {
	return NewRouter(Deps{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		DB:     fakePinger{},
		Auth:   fakeAuth{token: "secret", player: ana},
		Shop:   s,
	})
}

func TestShopCatalog(t *testing.T) {
	rec := do(t, newShopRouter(&fakeShop{}), http.MethodGet, "/v1/shop", authed(""))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	var body struct {
		Crops []cropResponse `json:"crops"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Crops) != len(catalog.Default().Entries()) {
		t.Fatalf("crops = %d", len(body.Crops))
	}
	first := body.Crops[0]
	if first.ID != "strawberry" || first.GrowSeconds != 3600 || first.SeedPrice == 0 {
		t.Errorf("first crop = %+v", first)
	}
}

func TestInventoryGroupsByKind(t *testing.T) {
	rec := do(t, newShopRouter(&fakeShop{}), http.MethodGet, "/v1/inventory", authed(""))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	var body struct {
		Inventory inventoryResponse `json:"inventory"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	inv := body.Inventory
	if len(inv.Seeds) != 1 || inv.Seeds[0] != (stackResponse{Crop: "tomato", Quantity: 3}) {
		t.Errorf("seeds = %+v", inv.Seeds)
	}
	if len(inv.Produce) != 1 || inv.Produce[0] != (stackResponse{Crop: "corn", Quantity: 12}) {
		t.Errorf("produce = %+v", inv.Produce)
	}
}

func TestBuySeedsEndpoint(t *testing.T) {
	s := &fakeShop{}
	rec := do(t, newShopRouter(s), http.MethodPost, "/v1/shop/seeds", authed(`{"crop":"corn","quantity":5}`))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	if s.lastCrop != "corn" || s.lastQty != 5 {
		t.Errorf("called with %q x%d", s.lastCrop, s.lastQty)
	}

	var body struct {
		Player    playerResponse    `json:"player"`
		Inventory inventoryResponse `json:"inventory"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Player.Coins != 125 || len(body.Inventory.Seeds) != 1 {
		t.Errorf("body = %+v", body)
	}
}

func TestTradeErrors(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		body   string
		err    error
		status int
		code   string
	}{
		{"bad body", "/v1/shop/seeds", `{`, nil, http.StatusBadRequest, "invalid_body"},
		{"missing crop", "/v1/barn/sell", `{"quantity":1}`, nil, http.StatusBadRequest, "invalid_body"},
		{"bad quantity", "/v1/shop/seeds", `{"crop":"corn","quantity":0}`, shop.ErrInvalidQuantity, http.StatusBadRequest, "invalid_quantity"},
		{"poor", "/v1/shop/seeds", `{"crop":"corn","quantity":1}`, player.ErrNotEnoughCoins, http.StatusConflict, "not_enough_coins"},
		{"empty barn", "/v1/barn/sell", `{"crop":"corn","quantity":1}`, inventory.ErrNotEnough, http.StatusConflict, "not_enough_items"},
		{"unknown crop", "/v1/shop/seeds", `{"crop":"x","quantity":1}`, shop.ErrUnknownCrop, http.StatusUnprocessableEntity, "unknown_crop"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := do(t, newShopRouter(&fakeShop{err: tt.err}), http.MethodPost, tt.path, authed(tt.body))
			if rec.Code != tt.status {
				t.Fatalf("status = %d, want %d", rec.Code, tt.status)
			}
			var body struct {
				Error errorBody `json:"error"`
			}
			_ = json.NewDecoder(rec.Body).Decode(&body)
			if body.Error.Code != tt.code {
				t.Errorf("code = %q, want %q", body.Error.Code, tt.code)
			}
		})
	}
}
