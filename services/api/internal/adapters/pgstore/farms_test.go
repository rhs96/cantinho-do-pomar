package pgstore_test

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rhs96/cantinho-do-pomar/services/api/internal/adapters/pgstore"
	"github.com/rhs96/cantinho-do-pomar/services/api/internal/catalog"
	"github.com/rhs96/cantinho-do-pomar/services/api/internal/domain/farm"
	"github.com/rhs96/cantinho-do-pomar/services/api/internal/domain/player"
	"github.com/rhs96/cantinho-do-pomar/services/api/internal/platform/postgres/pgtest"
)

var strawberry, _ = catalog.Default().Crop("strawberry")

func newFarmFixture(t *testing.T) (*pgstore.Farms, player.ID) {
	t.Helper()
	db := pgtest.NewMigratedDB(t)
	return pgstore.NewFarms(db, catalog.Default()), createPlayer(t, db)
}

func createPlayer(t *testing.T, db *pgxpool.Pool) player.ID {
	t.Helper()
	guest, _ := player.NewGuest("Ana", now)
	p, err := pgstore.NewPlayers(db).Create(t.Context(), guest)
	if err != nil {
		t.Fatalf("create player: %v", err)
	}
	return p.ID
}

func createFarm(t *testing.T, repo *pgstore.Farms, owner player.ID) *farm.Farm {
	t.Helper()
	f, _ := farm.New(owner, farm.StarterSize, farm.StarterSize)
	created, err := repo.Create(t.Context(), f)
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}
	return created
}

func TestFarmsCreateAndGet(t *testing.T) {
	repo, owner := newFarmFixture(t)

	if _, err := repo.Get(t.Context(), owner); !errors.Is(err, farm.ErrNotFound) {
		t.Fatalf("Get() before create = %v, want ErrNotFound", err)
	}

	created := createFarm(t, repo, owner)
	if created.ID == "" || created.Owner != owner {
		t.Fatalf("Create() = %+v", created)
	}
	if got := len(created.Plots()); got != farm.StarterSize*farm.StarterSize {
		t.Fatalf("plots = %d", got)
	}
	for _, p := range created.Plots() {
		if p.State() != farm.PlotGrass {
			t.Fatalf("plot %v state = %v, want grass", p.Position(), p.State())
		}
	}
}

func TestFarmsCreateIsIdempotentPerOwner(t *testing.T) {
	repo, owner := newFarmFixture(t)

	first := createFarm(t, repo, owner)
	second := createFarm(t, repo, owner)
	if first.ID != second.ID {
		t.Errorf("second Create() made a new farm %q, want %q", second.ID, first.ID)
	}
}

func TestFarmsUpdatePlotPersistsChanges(t *testing.T) {
	repo, owner := newFarmFixture(t)
	createFarm(t, repo, owner)
	ctx := t.Context()
	pos := farm.Position{Col: 2, Row: 4}

	_, err := repo.UpdatePlot(ctx, owner, pos, func(p *farm.Plot) error {
		if err := p.Plow(); err != nil {
			return err
		}
		if err := p.Plant(strawberry, now); err != nil {
			return err
		}
		return p.Water(now)
	})
	if err != nil {
		t.Fatalf("UpdatePlot() = %v", err)
	}

	f, err := repo.Get(ctx, owner)
	if err != nil {
		t.Fatalf("Get() = %v", err)
	}
	p, _ := f.Plot(pos)
	if p.State() != farm.PlotPlanted {
		t.Fatalf("state = %v, want planted", p.State())
	}
	if c, _ := p.Crop(); c.ID != strawberry.ID {
		t.Errorf("crop = %q", c.ID)
	}
	if !p.IsWet(now.Add(time.Minute)) {
		t.Error("watering was not persisted")
	}
	if stage, _ := p.Stage(now.Add(strawberry.TimeToRipe())); stage != farm.StageRipe {
		t.Errorf("stage at ripe time = %v", stage)
	}
}

func TestFarmsUpdatePlotRollsBackOnError(t *testing.T) {
	repo, owner := newFarmFixture(t)
	createFarm(t, repo, owner)
	pos := farm.Position{Col: 0, Row: 0}

	_, err := repo.UpdatePlot(t.Context(), owner, pos, func(p *farm.Plot) error {
		_ = p.Plow()
		return farm.ErrNotPlowed
	})
	if !errors.Is(err, farm.ErrNotPlowed) {
		t.Fatalf("UpdatePlot() = %v, want the callback error", err)
	}

	f, _ := repo.Get(t.Context(), owner)
	if p, _ := f.Plot(pos); p.State() != farm.PlotGrass {
		t.Errorf("state = %v, want grass after rollback", p.State())
	}
}

func TestFarmsUpdatePlotMissing(t *testing.T) {
	repo, owner := newFarmFixture(t)
	noop := func(*farm.Plot) error { return nil }

	if _, err := repo.UpdatePlot(t.Context(), owner, farm.Position{}, noop); !errors.Is(err, farm.ErrNotFound) {
		t.Errorf("UpdatePlot() without farm = %v, want ErrNotFound", err)
	}

	createFarm(t, repo, owner)
	outside := farm.Position{Col: farm.StarterSize, Row: 0}
	if _, err := repo.UpdatePlot(t.Context(), owner, outside, noop); !errors.Is(err, farm.ErrOutOfBounds) {
		t.Errorf("UpdatePlot() outside grid = %v, want ErrOutOfBounds", err)
	}
}

func TestFarmsConcurrentHarvestOnlySucceedsOnce(t *testing.T) {
	repo, owner := newFarmFixture(t)
	createFarm(t, repo, owner)
	ctx := t.Context()
	pos := farm.Position{Col: 1, Row: 1}

	_, err := repo.UpdatePlot(ctx, owner, pos, func(p *farm.Plot) error {
		_ = p.Plow()
		_ = p.Plant(strawberry, now)
		return p.Water(now)
	})
	if err != nil {
		t.Fatalf("prepare plot: %v", err)
	}

	ripe := now.Add(strawberry.TimeToRipe())
	var (
		wg        sync.WaitGroup
		mu        sync.Mutex
		harvested int
	)
	for range 8 {
		wg.Go(func() {
			_, err := repo.UpdatePlot(ctx, owner, pos, func(p *farm.Plot) error {
				_, err := p.Harvest(ripe)
				return err
			})
			if err == nil {
				mu.Lock()
				harvested++
				mu.Unlock()
			}
		})
	}
	wg.Wait()

	if harvested != 1 {
		t.Errorf("harvested %d times, want exactly 1", harvested)
	}
}
