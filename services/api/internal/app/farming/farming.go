package farming

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/rhs96/cantinho-do-pomar/services/api/internal/app"
	"github.com/rhs96/cantinho-do-pomar/services/api/internal/catalog"
	"github.com/rhs96/cantinho-do-pomar/services/api/internal/domain/farm"
	"github.com/rhs96/cantinho-do-pomar/services/api/internal/domain/inventory"
	"github.com/rhs96/cantinho-do-pomar/services/api/internal/domain/player"
)

var (
	ErrUnknownCrop = errors.New("unknown crop")
	ErrNoSeeds     = errors.New("no seeds of this crop in the barn")
)

// XP for the small chores. Harvest XP depends on the crop.
const (
	PlowXP  = 1
	PlantXP = 1
	WaterXP = 1
)

// Outcome is everything an action changed.
type Outcome struct {
	Plot      *farm.Plot
	Player    player.Player
	Harvest   farm.Harvest
	LeveledUp bool
}

type Service struct {
	uow     app.UnitOfWork
	catalog *catalog.Catalog
	now     func() time.Time
}

func NewService(uow app.UnitOfWork, c *catalog.Catalog, now func() time.Time) *Service {
	return &Service{uow: uow, catalog: c, now: now}
}

// Farm returns the player's farm, creating the starter one on first access.
func (s *Service) Farm(ctx context.Context, owner player.ID) (*farm.Farm, error) {
	var f *farm.Farm
	err := s.uow.Do(ctx, func(r app.Repos) error {
		var err error
		f, err = r.Farms().Get(ctx, owner)
		if !errors.Is(err, farm.ErrNotFound) {
			return err
		}

		f, err = farm.New(owner, farm.StarterSize, farm.StarterSize)
		if err != nil {
			return err
		}
		f, err = r.Farms().Create(ctx, f)
		if err != nil {
			return fmt.Errorf("create starter farm: %w", err)
		}
		return nil
	})
	return f, err
}

func (s *Service) Plow(ctx context.Context, owner player.ID, pos farm.Position) (Outcome, error) {
	return s.act(ctx, owner, pos, func(_ app.Repos, p *farm.Plot, _ *Outcome) (int64, error) {
		return PlowXP, p.Plow()
	})
}

func (s *Service) Plant(ctx context.Context, owner player.ID, pos farm.Position, cropID farm.CropID) (Outcome, error) {
	crop, ok := s.catalog.Crop(cropID)
	if !ok {
		return Outcome{}, ErrUnknownCrop
	}

	return s.act(ctx, owner, pos, func(r app.Repos, p *farm.Plot, _ *Outcome) (int64, error) {
		if err := p.Plant(crop, s.now()); err != nil {
			return 0, err
		}
		err := r.Inventory().Remove(ctx, owner, inventory.Seed(cropID), 1)
		if errors.Is(err, inventory.ErrNotEnough) {
			return 0, ErrNoSeeds
		}
		return PlantXP, err
	})
}

func (s *Service) Water(ctx context.Context, owner player.ID, pos farm.Position) (Outcome, error) {
	return s.act(ctx, owner, pos, func(_ app.Repos, p *farm.Plot, _ *Outcome) (int64, error) {
		return WaterXP, p.Water(s.now())
	})
}

func (s *Service) Harvest(ctx context.Context, owner player.ID, pos farm.Position) (Outcome, error) {
	return s.act(ctx, owner, pos, func(r app.Repos, p *farm.Plot, out *Outcome) (int64, error) {
		h, err := p.Harvest(s.now())
		if err != nil {
			return 0, err
		}
		out.Harvest = h

		var xp int64
		if entry, ok := s.catalog.Entry(h.Crop); ok {
			xp = entry.HarvestXP
		}
		return xp, r.Inventory().Add(ctx, owner, inventory.Produce(h.Crop), h.Quantity)
	})
}

type action func(r app.Repos, p *farm.Plot, out *Outcome) (xp int64, err error)

// act locks the player, then the plot, runs fn and credits the XP it earned.
func (s *Service) act(ctx context.Context, owner player.ID, pos farm.Position, fn action) (Outcome, error) {
	var out Outcome
	err := s.uow.Do(ctx, func(r app.Repos) error {
		p, err := r.Players().GetForUpdate(ctx, owner)
		if err != nil {
			return err
		}

		var xp int64
		out.Plot, err = r.Farms().UpdatePlot(ctx, owner, pos, func(plot *farm.Plot) error {
			var err error
			xp, err = fn(r, plot, &out)
			return err
		})
		if err != nil {
			return err
		}

		out.LeveledUp = p.GainXP(xp)
		out.Player = p
		return r.Players().Update(ctx, p)
	})
	if err != nil {
		return Outcome{}, err
	}
	return out, nil
}
