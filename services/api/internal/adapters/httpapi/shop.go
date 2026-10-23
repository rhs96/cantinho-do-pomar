package httpapi

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/rhs96/cantinho-do-pomar/services/api/internal/app/shop"
	"github.com/rhs96/cantinho-do-pomar/services/api/internal/catalog"
	"github.com/rhs96/cantinho-do-pomar/services/api/internal/domain/farm"
	"github.com/rhs96/cantinho-do-pomar/services/api/internal/domain/inventory"
	"github.com/rhs96/cantinho-do-pomar/services/api/internal/domain/player"
)

type Shop interface {
	Catalog() []catalog.Entry
	Inventory(ctx context.Context, owner player.ID) ([]inventory.Stack, error)
	BuySeeds(ctx context.Context, owner player.ID, crop farm.CropID, qty int) (shop.Result, error)
	Sell(ctx context.Context, owner player.ID, crop farm.CropID, qty int) (shop.Result, error)
}

type shopHandlers struct {
	shop   Shop
	logger *slog.Logger
}

func (h *shopHandlers) routes(r chi.Router) {
	r.Get("/shop", h.catalog)
	r.Get("/inventory", h.inventory)
	r.Post("/shop/seeds", h.trade(func(ctx context.Context, owner player.ID, crop farm.CropID, qty int) (shop.Result, error) {
		return h.shop.BuySeeds(ctx, owner, crop, qty)
	}))
	r.Post("/barn/sell", h.trade(func(ctx context.Context, owner player.ID, crop farm.CropID, qty int) (shop.Result, error) {
		return h.shop.Sell(ctx, owner, crop, qty)
	}))
}

type cropResponse struct {
	ID          farm.CropID `json:"id"`
	SeedPrice   int64       `json:"seedPrice"`
	SellPrice   int64       `json:"sellPrice"`
	Yield       int         `json:"yield"`
	GrowSeconds int64       `json:"growSeconds"`
	MinLevel    int         `json:"minLevel"`
	HarvestXP   int64       `json:"harvestXp"`
}

func (h *shopHandlers) catalog(w http.ResponseWriter, _ *http.Request) {
	entries := h.shop.Catalog()
	crops := make([]cropResponse, 0, len(entries))
	for _, e := range entries {
		crops = append(crops, cropResponse{
			ID:          e.ID,
			SeedPrice:   e.SeedPrice,
			SellPrice:   e.SellPrice,
			Yield:       e.Yield,
			GrowSeconds: int64(e.TimeToRipe().Seconds()),
			MinLevel:    e.MinLevel,
			HarvestXP:   e.HarvestXP,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"crops": crops})
}

type stackResponse struct {
	Crop     string `json:"crop"`
	Quantity int    `json:"quantity"`
}

type inventoryResponse struct {
	Seeds   []stackResponse `json:"seeds"`
	Produce []stackResponse `json:"produce"`
}

func toInventoryResponse(stacks []inventory.Stack) inventoryResponse {
	res := inventoryResponse{Seeds: []stackResponse{}, Produce: []stackResponse{}}
	for _, s := range stacks {
		item := stackResponse{Crop: s.Item.ID, Quantity: s.Quantity}
		switch s.Item.Kind {
		case inventory.KindSeed:
			res.Seeds = append(res.Seeds, item)
		case inventory.KindProduce:
			res.Produce = append(res.Produce, item)
		}
	}
	return res
}

func (h *shopHandlers) inventory(w http.ResponseWriter, r *http.Request) {
	stacks, err := h.shop.Inventory(r.Context(), currentPlayer(r.Context()).ID)
	if err != nil {
		writeAppError(w, r, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"inventory": toInventoryResponse(stacks)})
}

type tradeFunc func(ctx context.Context, owner player.ID, crop farm.CropID, qty int) (shop.Result, error)

func (h *shopHandlers) trade(fn tradeFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Crop     string `json:"crop"`
			Quantity int    `json:"quantity"`
		}
		if err := decodeJSON(r, &body); err != nil || body.Crop == "" {
			writeAppError(w, r, h.logger, errInvalidBody)
			return
		}

		res, err := fn(r.Context(), currentPlayer(r.Context()).ID, farm.CropID(body.Crop), body.Quantity)
		if err != nil {
			writeAppError(w, r, h.logger, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"player":    toPlayerResponse(res.Player),
			"inventory": toInventoryResponse(res.Inventory),
		})
	}
}
