package httpapi

import (
	"context"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/rhs96/cantinho-do-pomar/services/api/internal/domain/farm"
	"github.com/rhs96/cantinho-do-pomar/services/api/internal/domain/player"
)

type Farming interface {
	Farm(ctx context.Context, owner player.ID) (*farm.Farm, error)
	Plow(ctx context.Context, owner player.ID, pos farm.Position) (*farm.Plot, error)
	Plant(ctx context.Context, owner player.ID, pos farm.Position, crop farm.CropID) (*farm.Plot, error)
	Water(ctx context.Context, owner player.ID, pos farm.Position) (*farm.Plot, error)
	Harvest(ctx context.Context, owner player.ID, pos farm.Position) (*farm.Plot, farm.Harvest, error)
}

type farmHandlers struct {
	farming Farming
	now     func() time.Time
	logger  *slog.Logger
}

func (h *farmHandlers) routes(r chi.Router) {
	r.Get("/farm", h.getFarm)
	r.Route("/farm/plots/{col}/{row}", func(r chi.Router) {
		r.Post("/plow", h.plotAction(func(ctx context.Context, owner player.ID, pos farm.Position, _ *http.Request) (*farm.Plot, error) {
			return h.farming.Plow(ctx, owner, pos)
		}))
		r.Post("/water", h.plotAction(func(ctx context.Context, owner player.ID, pos farm.Position, _ *http.Request) (*farm.Plot, error) {
			return h.farming.Water(ctx, owner, pos)
		}))
		r.Post("/plant", h.plotAction(h.plant))
		r.Post("/harvest", h.harvest)
	})
}

type plotResponse struct {
	Col              int        `json:"col"`
	Row              int        `json:"row"`
	State            string     `json:"state"`
	Crop             string     `json:"crop,omitempty"`
	Stage            string     `json:"stage,omitempty"`
	WetUntil         *time.Time `json:"wetUntil,omitempty"`
	RemainingSeconds *int64     `json:"remainingSeconds,omitempty"`
}

func toPlotResponse(p *farm.Plot, now time.Time) plotResponse {
	pos := p.Position()
	res := plotResponse{Col: pos.Col, Row: pos.Row, State: string(p.State())}

	if c, ok := p.Crop(); ok {
		res.Crop = string(c.ID)
	}
	if stage, err := p.Stage(now); err == nil {
		res.Stage = stage.String()
		remaining := int64(math.Ceil(p.Remaining(now).Seconds()))
		res.RemainingSeconds = &remaining
	}
	if p.IsWet(now) {
		until := p.WetUntil().UTC()
		res.WetUntil = &until
	}
	return res
}

func (h *farmHandlers) getFarm(w http.ResponseWriter, r *http.Request) {
	f, err := h.farming.Farm(r.Context(), currentPlayer(r.Context()).ID)
	if err != nil {
		writeAppError(w, r, h.logger, err)
		return
	}

	now := h.now()
	plots := make([]plotResponse, 0, len(f.Plots()))
	for _, p := range f.Plots() {
		plots = append(plots, toPlotResponse(p, now))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"farm": map[string]any{
			"width":  f.Width,
			"height": f.Height,
			"plots":  plots,
		},
		"serverTime": now.UTC(),
	})
}

type plotFunc func(ctx context.Context, owner player.ID, pos farm.Position, r *http.Request) (*farm.Plot, error)

func (h *farmHandlers) plotAction(fn plotFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pos, ok := parsePosition(w, r)
		if !ok {
			return
		}
		p, err := fn(r.Context(), currentPlayer(r.Context()).ID, pos, r)
		if err != nil {
			writeAppError(w, r, h.logger, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"plot": toPlotResponse(p, h.now())})
	}
}

func (h *farmHandlers) plant(ctx context.Context, owner player.ID, pos farm.Position, r *http.Request) (*farm.Plot, error) {
	var body struct {
		Crop string `json:"crop"`
	}
	if err := decodeJSON(r, &body); err != nil || body.Crop == "" {
		return nil, errInvalidBody
	}
	return h.farming.Plant(ctx, owner, pos, farm.CropID(body.Crop))
}

func (h *farmHandlers) harvest(w http.ResponseWriter, r *http.Request) {
	pos, ok := parsePosition(w, r)
	if !ok {
		return
	}
	p, got, err := h.farming.Harvest(r.Context(), currentPlayer(r.Context()).ID, pos)
	if err != nil {
		writeAppError(w, r, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"plot":    toPlotResponse(p, h.now()),
		"harvest": map[string]any{"crop": got.Crop, "quantity": got.Quantity},
	})
}

func parsePosition(w http.ResponseWriter, r *http.Request) (farm.Position, bool) {
	col, errCol := strconv.Atoi(chi.URLParam(r, "col"))
	row, errRow := strconv.Atoi(chi.URLParam(r, "row"))
	if errCol != nil || errRow != nil {
		writeError(w, http.StatusBadRequest, "invalid_position", "col and row must be integers")
		return farm.Position{}, false
	}
	return farm.Position{Col: col, Row: row}, true
}
