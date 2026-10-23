package shop

import (
	"context"
	"errors"

	"github.com/rhs96/cantinho-do-pomar/services/api/internal/app"
	"github.com/rhs96/cantinho-do-pomar/services/api/internal/catalog"
	"github.com/rhs96/cantinho-do-pomar/services/api/internal/domain/farm"
	"github.com/rhs96/cantinho-do-pomar/services/api/internal/domain/inventory"
	"github.com/rhs96/cantinho-do-pomar/services/api/internal/domain/player"
)

const MaxQuantity = 99

var (
	ErrUnknownCrop     = errors.New("unknown crop")
	ErrInvalidQuantity = errors.New("quantity must be between 1 and 99")
	ErrCropLocked      = errors.New("player level too low for this crop")
)

// Result is the player's wallet and barn after a trade.
type Result struct {
	Player    player.Player
	Inventory []inventory.Stack
}

type Service struct {
	uow     app.UnitOfWork
	catalog *catalog.Catalog
}

func NewService(uow app.UnitOfWork, c *catalog.Catalog) *Service {
	return &Service{uow: uow, catalog: c}
}

func (s *Service) Catalog() []catalog.Entry {
	return s.catalog.Entries()
}

func (s *Service) Inventory(ctx context.Context, owner player.ID) ([]inventory.Stack, error) {
	var stacks []inventory.Stack
	err := s.uow.Do(ctx, func(r app.Repos) error {
		var err error
		stacks, err = r.Inventory().List(ctx, owner)
		return err
	})
	return stacks, err
}

func (s *Service) BuySeeds(ctx context.Context, owner player.ID, crop farm.CropID, qty int) (Result, error) {
	entry, err := s.validate(crop, qty)
	if err != nil {
		return Result{}, err
	}

	return s.trade(ctx, owner, func(r app.Repos, p *player.Player) error {
		if p.Level() < entry.MinLevel {
			return ErrCropLocked
		}
		if err := p.Spend(entry.SeedPrice * int64(qty)); err != nil {
			return err
		}
		return r.Inventory().Add(ctx, owner, inventory.Seed(crop), qty)
	})
}

func (s *Service) Sell(ctx context.Context, owner player.ID, crop farm.CropID, qty int) (Result, error) {
	entry, err := s.validate(crop, qty)
	if err != nil {
		return Result{}, err
	}

	return s.trade(ctx, owner, func(r app.Repos, p *player.Player) error {
		if err := r.Inventory().Remove(ctx, owner, inventory.Produce(crop), qty); err != nil {
			return err
		}
		return p.Earn(entry.SellPrice * int64(qty))
	})
}

func (s *Service) validate(crop farm.CropID, qty int) (catalog.Entry, error) {
	entry, ok := s.catalog.Entry(crop)
	if !ok {
		return catalog.Entry{}, ErrUnknownCrop
	}
	if qty < 1 || qty > MaxQuantity {
		return catalog.Entry{}, ErrInvalidQuantity
	}
	return entry, nil
}

func (s *Service) trade(ctx context.Context, owner player.ID, fn func(app.Repos, *player.Player) error) (Result, error) {
	var res Result
	err := s.uow.Do(ctx, func(r app.Repos) error {
		p, err := r.Players().GetForUpdate(ctx, owner)
		if err != nil {
			return err
		}
		if err := fn(r, &p); err != nil {
			return err
		}
		if err := r.Players().Update(ctx, p); err != nil {
			return err
		}

		res.Player = p
		res.Inventory, err = r.Inventory().List(ctx, owner)
		return err
	})
	return res, err
}
