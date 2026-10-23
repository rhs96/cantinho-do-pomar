package farming

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rhs96/cantinho-do-pomar/services/api/internal/catalog"
	"github.com/rhs96/cantinho-do-pomar/services/api/internal/domain/farm"
	"github.com/rhs96/cantinho-do-pomar/services/api/internal/domain/player"
)

type memFarms struct {
	byOwner map[player.ID]*farm.Farm
	creates int
}

func (m *memFarms) Get(_ context.Context, owner player.ID) (*farm.Farm, error) {
	f, ok := m.byOwner[owner]
	if !ok {
		return nil, farm.ErrNotFound
	}
	return f, nil
}

func (m *memFarms) Create(_ context.Context, f *farm.Farm) (*farm.Farm, error) {
	if existing, ok := m.byOwner[f.Owner]; ok {
		return existing, nil
	}
	m.creates++
	f.ID = "f1"
	m.byOwner[f.Owner] = f
	return f, nil
}

func (m *memFarms) UpdatePlot(ctx context.Context, owner player.ID, pos farm.Position, fn func(*farm.Plot) error) (*farm.Plot, error) {
	f, err := m.Get(ctx, owner)
	if err != nil {
		return nil, err
	}
	p, err := f.Plot(pos)
	if err != nil {
		return nil, err
	}

	before := p.Snapshot()
	if err := fn(p); err != nil {
		restored, _ := farm.RestorePlot(before, catalog.Default())
		*p = *restored
		return nil, err
	}
	return p, nil
}

type fixture struct {
	svc   *Service
	farms *memFarms
	now   time.Time
}

const owner player.ID = "p1"

func newFixture(t *testing.T) *fixture {
	t.Helper()
	f := &fixture{
		farms: &memFarms{byOwner: map[player.ID]*farm.Farm{}},
		now:   time.Date(2026, 10, 23, 9, 0, 0, 0, time.UTC),
	}
	f.svc = NewService(f.farms, catalog.Default(), func() time.Time { return f.now })
	if _, err := f.svc.Farm(t.Context(), owner); err != nil {
		t.Fatalf("Farm() = %v", err)
	}
	return f
}

func TestFarmCreatesStarterFarmOnce(t *testing.T) {
	f := newFixture(t)

	got, err := f.svc.Farm(t.Context(), owner)
	if err != nil {
		t.Fatalf("Farm() = %v", err)
	}
	if got.Width != farm.StarterSize || got.Height != farm.StarterSize {
		t.Errorf("size = %dx%d", got.Width, got.Height)
	}
	if f.farms.creates != 1 {
		t.Errorf("creates = %d, want 1", f.farms.creates)
	}
}

func TestFullCycle(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	pos := farm.Position{Col: 2, Row: 3}
	strawberry, _ := catalog.Default().Crop("strawberry")

	if _, err := f.svc.Plow(ctx, owner, pos); err != nil {
		t.Fatalf("Plow() = %v", err)
	}
	if _, err := f.svc.Plant(ctx, owner, pos, "strawberry"); err != nil {
		t.Fatalf("Plant() = %v", err)
	}
	if _, err := f.svc.Water(ctx, owner, pos); err != nil {
		t.Fatalf("Water() = %v", err)
	}

	if _, _, err := f.svc.Harvest(ctx, owner, pos); !errors.Is(err, farm.ErrNotRipe) {
		t.Fatalf("early Harvest() = %v, want ErrNotRipe", err)
	}

	f.now = f.now.Add(strawberry.TimeToRipe())
	p, h, err := f.svc.Harvest(ctx, owner, pos)
	if err != nil {
		t.Fatalf("Harvest() = %v", err)
	}
	if h.Crop != "strawberry" || h.Quantity != strawberry.Yield {
		t.Errorf("Harvest() = %+v", h)
	}
	if p.State() != farm.PlotWithered {
		t.Errorf("state = %v, want withered", p.State())
	}
}

func TestPlantUnknownCrop(t *testing.T) {
	f := newFixture(t)
	pos := farm.Position{}
	_, _ = f.svc.Plow(t.Context(), owner, pos)

	if _, err := f.svc.Plant(t.Context(), owner, pos, "mandrake"); !errors.Is(err, ErrUnknownCrop) {
		t.Errorf("Plant() = %v, want ErrUnknownCrop", err)
	}
}

func TestActionsPropagateDomainErrors(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()

	if _, err := f.svc.Plant(ctx, owner, farm.Position{}, "tomato"); !errors.Is(err, farm.ErrNotPlowed) {
		t.Errorf("Plant() on grass = %v, want ErrNotPlowed", err)
	}
	if _, err := f.svc.Water(ctx, owner, farm.Position{}); !errors.Is(err, farm.ErrNothingPlanted) {
		t.Errorf("Water() on grass = %v, want ErrNothingPlanted", err)
	}
	if _, err := f.svc.Plow(ctx, owner, farm.Position{Col: 99}); !errors.Is(err, farm.ErrOutOfBounds) {
		t.Errorf("Plow() outside = %v, want ErrOutOfBounds", err)
	}
	if _, err := f.svc.Plow(ctx, "stranger", farm.Position{}); !errors.Is(err, farm.ErrNotFound) {
		t.Errorf("Plow() without farm = %v, want ErrNotFound", err)
	}
}
