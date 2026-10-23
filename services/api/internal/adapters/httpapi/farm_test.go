package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/rhs96/cantinho-do-pomar/services/api/internal/app/farming"
	"github.com/rhs96/cantinho-do-pomar/services/api/internal/domain/farm"
	"github.com/rhs96/cantinho-do-pomar/services/api/internal/domain/player"
)

var (
	farmNow    = time.Date(2026, 10, 23, 9, 0, 0, 0, time.UTC)
	strawberry = farm.Crop{ID: "strawberry", Yield: 6, StageDurations: [3]time.Duration{time.Minute, time.Minute, time.Minute}}
)

type fakeFarming struct {
	err      error
	lastPos  farm.Position
	lastCrop farm.CropID
}

func (f *fakeFarming) Farm(_ context.Context, owner player.ID) (*farm.Farm, error) {
	if f.err != nil {
		return nil, f.err
	}
	return farm.New(owner, 2, 2)
}

func (f *fakeFarming) outcome(pos farm.Position) (farming.Outcome, error) {
	f.lastPos = pos
	if f.err != nil {
		return farming.Outcome{}, f.err
	}
	p := farm.NewPlot(pos)
	_ = p.Plow()
	_ = p.Plant(strawberry, farmNow)
	_ = p.Water(farmNow)
	return farming.Outcome{Plot: p, Player: player.Player{ID: "p1", XP: 60}, LeveledUp: true}, nil
}

func (f *fakeFarming) Plow(_ context.Context, _ player.ID, pos farm.Position) (farming.Outcome, error) {
	return f.outcome(pos)
}

func (f *fakeFarming) Water(_ context.Context, _ player.ID, pos farm.Position) (farming.Outcome, error) {
	return f.outcome(pos)
}

func (f *fakeFarming) Plant(_ context.Context, _ player.ID, pos farm.Position, crop farm.CropID) (farming.Outcome, error) {
	f.lastCrop = crop
	return f.outcome(pos)
}

func (f *fakeFarming) Harvest(_ context.Context, _ player.ID, pos farm.Position) (farming.Outcome, error) {
	out, err := f.outcome(pos)
	out.Harvest = farm.Harvest{Crop: strawberry.ID, Quantity: strawberry.Yield}
	return out, err
}

func newFarmRouter(f Farming) http.Handler {
	return NewRouter(Deps{
		Logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		DB:      fakePinger{},
		Auth:    fakeAuth{token: "secret", player: ana},
		Farming: f,
		Now:     func() time.Time { return farmNow.Add(30 * time.Second) },
	})
}

func authed(body string) func(*http.Request) {
	return func(r *http.Request) {
		r.Header.Set("Authorization", "Bearer secret")
		if body != "" {
			r.Body = io.NopCloser(strings.NewReader(body))
		}
	}
}

func TestGetFarm(t *testing.T) {
	rec := do(t, newFarmRouter(&fakeFarming{}), http.MethodGet, "/v1/farm", authed(""))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}

	var body struct {
		Farm struct {
			Width  int            `json:"width"`
			Height int            `json:"height"`
			Plots  []plotResponse `json:"plots"`
		} `json:"farm"`
		ServerTime time.Time `json:"serverTime"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Farm.Width != 2 || len(body.Farm.Plots) != 4 {
		t.Errorf("farm = %+v", body.Farm)
	}
	if !body.ServerTime.Equal(farmNow.Add(30 * time.Second)) {
		t.Errorf("serverTime = %v", body.ServerTime)
	}
}

func TestGetFarmRequiresSession(t *testing.T) {
	rec := do(t, newFarmRouter(&fakeFarming{}), http.MethodGet, "/v1/farm", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestPlotActionResponse(t *testing.T) {
	f := &fakeFarming{}
	rec := do(t, newFarmRouter(f), http.MethodPost, "/v1/farm/plots/1/0/water", authed(""))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	if f.lastPos != (farm.Position{Col: 1, Row: 0}) {
		t.Errorf("position = %v", f.lastPos)
	}

	var body struct {
		Plot      plotResponse   `json:"plot"`
		Player    playerResponse `json:"player"`
		LeveledUp bool           `json:"leveledUp"`
		Harvest   *struct{}      `json:"harvest"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Player.XP != 60 || !body.LeveledUp {
		t.Errorf("player = %+v, leveledUp = %v", body.Player, body.LeveledUp)
	}
	if body.Harvest != nil {
		t.Error("harvest should only be present on harvest")
	}
	p := body.Plot
	if p.State != "planted" || p.Crop != "strawberry" || p.Stage != "seed" {
		t.Errorf("plot = %+v", p)
	}
	if p.RemainingSeconds == nil || *p.RemainingSeconds != 150 {
		t.Errorf("remainingSeconds = %v, want 150", p.RemainingSeconds)
	}
	if p.WetUntil == nil || !p.WetUntil.Equal(farmNow.Add(farm.SoilMoisture)) {
		t.Errorf("wetUntil = %v", p.WetUntil)
	}
}

func TestPlant(t *testing.T) {
	f := &fakeFarming{}
	h := newFarmRouter(f)

	rec := do(t, h, http.MethodPost, "/v1/farm/plots/0/1/plant", authed(`{"crop":"strawberry"}`))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	if f.lastCrop != "strawberry" {
		t.Errorf("crop = %q", f.lastCrop)
	}

	for _, body := range []string{"", "{", `{"crop":""}`, `{"crop":"` + strings.Repeat("a", 2048) + `"}`} {
		rec := do(t, h, http.MethodPost, "/v1/farm/plots/0/1/plant", authed(body))
		if rec.Code != http.StatusBadRequest {
			t.Errorf("body %.20q: status = %d, want 400", body, rec.Code)
		}
	}
}

func TestHarvestResponse(t *testing.T) {
	rec := do(t, newFarmRouter(&fakeFarming{}), http.MethodPost, "/v1/farm/plots/0/0/harvest", authed(""))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	var body struct {
		Harvest struct {
			Crop     string `json:"crop"`
			Quantity int    `json:"quantity"`
		} `json:"harvest"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Harvest.Crop != "strawberry" || body.Harvest.Quantity != 6 {
		t.Errorf("harvest = %+v", body.Harvest)
	}
}

func TestInvalidPosition(t *testing.T) {
	rec := do(t, newFarmRouter(&fakeFarming{}), http.MethodPost, "/v1/farm/plots/a/0/plow", authed(""))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestFarmErrorMapping(t *testing.T) {
	tests := []struct {
		err    error
		status int
		code   string
	}{
		{farm.ErrNotFound, http.StatusNotFound, "farm_not_found"},
		{farm.ErrOutOfBounds, http.StatusNotFound, "plot_not_found"},
		{farming.ErrUnknownCrop, http.StatusUnprocessableEntity, "unknown_crop"},
		{farm.ErrNotPlowed, http.StatusConflict, "not_plowed"},
		{farm.ErrNotRipe, http.StatusConflict, "not_ripe"},
		{farm.ErrAlreadyWet, http.StatusConflict, "already_wet"},
		{errors.New("boom"), http.StatusInternalServerError, "internal"},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			h := newFarmRouter(&fakeFarming{err: tt.err})
			rec := do(t, h, http.MethodPost, "/v1/farm/plots/0/0/plow", authed(""))
			if rec.Code != tt.status {
				t.Fatalf("status = %d, want %d", rec.Code, tt.status)
			}

			var body struct {
				Error errorBody `json:"error"`
			}
			if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if body.Error.Code != tt.code {
				t.Errorf("code = %q, want %q", body.Error.Code, tt.code)
			}
		})
	}
}
