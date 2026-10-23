package farming

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/rhs96/cantinho-do-pomar/services/api/internal/domain/farm"
	"github.com/rhs96/cantinho-do-pomar/services/api/internal/domain/player"
)

var ErrUnknownCrop = errors.New("unknown crop")

type Farms interface {
	// Get returns farm.ErrNotFound when the player has no farm yet.
	Get(ctx context.Context, owner player.ID) (*farm.Farm, error)
	// Create returns the existing farm if the owner already has one.
	Create(ctx context.Context, f *farm.Farm) (*farm.Farm, error)
	// UpdatePlot runs fn on the plot atomically. If fn fails nothing is saved.
	UpdatePlot(ctx context.Context, owner player.ID, pos farm.Position, fn func(*farm.Plot) error) (*farm.Plot, error)
}

type Service struct {
	farms Farms
	crops farm.CropLookup
	now   func() time.Time
}

func NewService(farms Farms, crops farm.CropLookup, now func() time.Time) *Service {
	return &Service{farms: farms, crops: crops, now: now}
}

// Farm returns the player's farm, creating the starter one on first access.
func (s *Service) Farm(ctx context.Context, owner player.ID) (*farm.Farm, error) {
	f, err := s.farms.Get(ctx, owner)
	if !errors.Is(err, farm.ErrNotFound) {
		return f, err
	}

	f, err = farm.New(owner, farm.StarterSize, farm.StarterSize)
	if err != nil {
		return nil, err
	}
	f, err = s.farms.Create(ctx, f)
	if err != nil {
		return nil, fmt.Errorf("create starter farm: %w", err)
	}
	return f, nil
}

func (s *Service) Plow(ctx context.Context, owner player.ID, pos farm.Position) (*farm.Plot, error) {
	return s.farms.UpdatePlot(ctx, owner, pos, func(p *farm.Plot) error {
		return p.Plow()
	})
}

func (s *Service) Plant(ctx context.Context, owner player.ID, pos farm.Position, cropID farm.CropID) (*farm.Plot, error) {
	crop, ok := s.crops.Crop(cropID)
	if !ok {
		return nil, ErrUnknownCrop
	}

	return s.farms.UpdatePlot(ctx, owner, pos, func(p *farm.Plot) error {
		return p.Plant(crop, s.now())
	})
}

func (s *Service) Water(ctx context.Context, owner player.ID, pos farm.Position) (*farm.Plot, error) {
	return s.farms.UpdatePlot(ctx, owner, pos, func(p *farm.Plot) error {
		return p.Water(s.now())
	})
}

func (s *Service) Harvest(ctx context.Context, owner player.ID, pos farm.Position) (*farm.Plot, farm.Harvest, error) {
	var h farm.Harvest
	p, err := s.farms.UpdatePlot(ctx, owner, pos, func(p *farm.Plot) error {
		var err error
		h, err = p.Harvest(s.now())
		return err
	})
	if err != nil {
		return nil, farm.Harvest{}, err
	}
	return p, h, nil
}
