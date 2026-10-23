package farming

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

type fixture struct {
	svc   *Service
	store *apptest.Store
	owner player.ID
	now   time.Time
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	f := &fixture{
		store: apptest.NewStore(catalog.Default()),
		now:   time.Date(2026, 10, 23, 9, 0, 0, 0, time.UTC),
	}
	guest, _ := player.NewGuest("Ana", f.now)
	f.owner = f.store.AddPlayer(guest).ID
	for _, c := range []farm.CropID{"strawberry", "tomato"} {
		if err := f.store.Give(f.owner, inventory.Seed(c), 2); err != nil {
			t.Fatalf("give seeds: %v", err)
		}
	}
	f.svc = NewService(f.store, catalog.Default(), func() time.Time { return f.now })
	if _, err := f.svc.Farm(t.Context(), f.owner); err != nil {
		t.Fatalf("Farm() = %v", err)
	}
	return f
}

func TestFarmCreatesStarterFarmOnce(t *testing.T) {
	f := newFixture(t)
	first, _ := f.svc.Farm(t.Context(), f.owner)

	got, err := f.svc.Farm(t.Context(), f.owner)
	if err != nil {
		t.Fatalf("Farm() = %v", err)
	}
	if got.Width != farm.StarterSize || got.Height != farm.StarterSize {
		t.Errorf("size = %dx%d", got.Width, got.Height)
	}
	if got.ID != first.ID {
		t.Errorf("second Farm() returned %q, want %q", got.ID, first.ID)
	}
}

func TestFullCycle(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	pos := farm.Position{Col: 2, Row: 3}
	strawberry, _ := catalog.Default().Crop("strawberry")

	if _, err := f.svc.Plow(ctx, f.owner, pos); err != nil {
		t.Fatalf("Plow() = %v", err)
	}
	if _, err := f.svc.Plant(ctx, f.owner, pos, "strawberry"); err != nil {
		t.Fatalf("Plant() = %v", err)
	}
	if _, err := f.svc.Water(ctx, f.owner, pos); err != nil {
		t.Fatalf("Water() = %v", err)
	}

	if _, _, err := f.svc.Harvest(ctx, f.owner, pos); !errors.Is(err, farm.ErrNotRipe) {
		t.Fatalf("early Harvest() = %v, want ErrNotRipe", err)
	}

	f.now = f.now.Add(strawberry.TimeToRipe())
	p, h, err := f.svc.Harvest(ctx, f.owner, pos)
	if err != nil {
		t.Fatalf("Harvest() = %v", err)
	}
	if h.Crop != "strawberry" || h.Quantity != strawberry.Yield {
		t.Errorf("Harvest() = %+v", h)
	}
	if p.State() != farm.PlotWithered {
		t.Errorf("state = %v, want withered", p.State())
	}

	if got := f.store.Quantity(f.owner, inventory.Seed("strawberry")); got != 1 {
		t.Errorf("seeds left = %d, want 1", got)
	}
	if got := f.store.Quantity(f.owner, inventory.Produce("strawberry")); got != strawberry.Yield {
		t.Errorf("produce in barn = %d, want %d", got, strawberry.Yield)
	}
}

func TestPlantWithoutSeeds(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	pos := farm.Position{}
	_, _ = f.svc.Plow(ctx, f.owner, pos)

	if _, err := f.svc.Plant(ctx, f.owner, pos, "corn"); !errors.Is(err, ErrNoSeeds) {
		t.Fatalf("Plant() = %v, want ErrNoSeeds", err)
	}

	got, _ := f.svc.Farm(ctx, f.owner)
	if p, _ := got.Plot(pos); p.State() != farm.PlotPlowed {
		t.Errorf("state = %v, want plowed after a failed plant", p.State())
	}
}

func TestPlantOnOccupiedPlotKeepsSeed(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	pos := farm.Position{}
	_, _ = f.svc.Plow(ctx, f.owner, pos)
	_, _ = f.svc.Plant(ctx, f.owner, pos, "tomato")

	if _, err := f.svc.Plant(ctx, f.owner, pos, "tomato"); !errors.Is(err, farm.ErrPlotOccupied) {
		t.Fatalf("Plant() = %v, want ErrPlotOccupied", err)
	}
	if got := f.store.Quantity(f.owner, inventory.Seed("tomato")); got != 1 {
		t.Errorf("seeds = %d, want 1", got)
	}
}

func TestFailedActionLeavesPlotUntouched(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	pos := farm.Position{}

	if _, err := f.svc.Plant(ctx, f.owner, pos, "tomato"); !errors.Is(err, farm.ErrNotPlowed) {
		t.Fatalf("Plant() on grass = %v, want ErrNotPlowed", err)
	}

	got, _ := f.svc.Farm(ctx, f.owner)
	if p, _ := got.Plot(pos); p.State() != farm.PlotGrass {
		t.Errorf("state = %v, want grass", p.State())
	}
}

func TestPlantUnknownCrop(t *testing.T) {
	f := newFixture(t)
	pos := farm.Position{}
	_, _ = f.svc.Plow(t.Context(), f.owner, pos)

	if _, err := f.svc.Plant(t.Context(), f.owner, pos, "mandrake"); !errors.Is(err, ErrUnknownCrop) {
		t.Errorf("Plant() = %v, want ErrUnknownCrop", err)
	}
}

func TestActionsPropagateDomainErrors(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()

	if _, err := f.svc.Water(ctx, f.owner, farm.Position{}); !errors.Is(err, farm.ErrNothingPlanted) {
		t.Errorf("Water() on grass = %v, want ErrNothingPlanted", err)
	}
	if _, err := f.svc.Plow(ctx, f.owner, farm.Position{Col: 99}); !errors.Is(err, farm.ErrOutOfBounds) {
		t.Errorf("Plow() outside = %v, want ErrOutOfBounds", err)
	}
	if _, err := f.svc.Plow(ctx, "stranger", farm.Position{}); !errors.Is(err, player.ErrNotFound) {
		t.Errorf("Plow() by unknown player = %v, want player.ErrNotFound", err)
	}
}
