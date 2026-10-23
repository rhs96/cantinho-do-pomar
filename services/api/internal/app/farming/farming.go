package farming

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/rhs96/cantinho-do-pomar/services/api/internal/app"
	"github.com/rhs96/cantinho-do-pomar/services/api/internal/domain/farm"
	"github.com/rhs96/cantinho-do-pomar/services/api/internal/domain/player"
)

var ErrUnknownCrop = errors.New("unknown crop")

type Service struct {
	uow   app.UnitOfWork
	crops farm.CropLookup
	now   func() time.Time
}

func NewService(uow app.UnitOfWork, crops farm.CropLookup, now func() time.Time) *Service {
	return &Service{uow: uow, crops: crops, now: now}
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

func (s *Service) Plow(ctx context.Context, owner player.ID, pos farm.Position) (*farm.Plot, error) {
	return s.updatePlot(ctx, owner, pos, func(_ app.Repos, p *farm.Plot) error {
		return p.Plow()
	})
}

func (s *Service) Plant(ctx context.Context, owner player.ID, pos farm.Position, cropID farm.CropID) (*farm.Plot, error) {
	crop, ok := s.crops.Crop(cropID)
	if !ok {
		return nil, ErrUnknownCrop
	}

	return s.updatePlot(ctx, owner, pos, func(_ app.Repos, p *farm.Plot) error {
		return p.Plant(crop, s.now())
	})
}

func (s *Service) Water(ctx context.Context, owner player.ID, pos farm.Position) (*farm.Plot, error) {
	return s.updatePlot(ctx, owner, pos, func(_ app.Repos, p *farm.Plot) error {
		return p.Water(s.now())
	})
}

func (s *Service) Harvest(ctx context.Context, owner player.ID, pos farm.Position) (*farm.Plot, farm.Harvest, error) {
	var h farm.Harvest
	p, err := s.updatePlot(ctx, owner, pos, func(_ app.Repos, p *farm.Plot) error {
		var err error
		h, err = p.Harvest(s.now())
		return err
	})
	if err != nil {
		return nil, farm.Harvest{}, err
	}
	return p, h, nil
}

// updatePlot locks the player, then the plot, and runs fn on it.
func (s *Service) updatePlot(ctx context.Context, owner player.ID, pos farm.Position, fn func(app.Repos, *farm.Plot) error) (*farm.Plot, error) {
	var plot *farm.Plot
	err := s.uow.Do(ctx, func(r app.Repos) error {
		if _, err := r.Players().GetForUpdate(ctx, owner); err != nil {
			return err
		}

		var err error
		plot, err = r.Farms().UpdatePlot(ctx, owner, pos, func(p *farm.Plot) error {
			return fn(r, p)
		})
		return err
	})
	return plot, err
}
