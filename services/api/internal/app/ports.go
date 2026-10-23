// Package app holds what the use cases share: the repositories they need and
// the unit of work that runs them in a single transaction.
package app

import (
	"context"

	"github.com/rhs96/cantinho-do-pomar/services/api/internal/domain/farm"
	"github.com/rhs96/cantinho-do-pomar/services/api/internal/domain/inventory"
	"github.com/rhs96/cantinho-do-pomar/services/api/internal/domain/player"
)

type Players interface {
	Get(ctx context.Context, id player.ID) (player.Player, error)
	// GetForUpdate locks the player until the unit of work ends. Every unit
	// of work that changes a player's state takes this lock first, which
	// keeps the lock order fixed and serializes actions of the same player.
	GetForUpdate(ctx context.Context, id player.ID) (player.Player, error)
	// Update saves coins and xp.
	Update(ctx context.Context, p player.Player) error
}

type Farms interface {
	// Get returns farm.ErrNotFound when the player has no farm yet.
	Get(ctx context.Context, owner player.ID) (*farm.Farm, error)
	// Create returns the existing farm if the owner already has one.
	Create(ctx context.Context, f *farm.Farm) (*farm.Farm, error)
	// UpdatePlot runs fn on the plot and saves it. If fn fails nothing is saved.
	UpdatePlot(ctx context.Context, owner player.ID, pos farm.Position, fn func(*farm.Plot) error) (*farm.Plot, error)
}

type Inventory interface {
	List(ctx context.Context, owner player.ID) ([]inventory.Stack, error)
	Add(ctx context.Context, owner player.ID, item inventory.Item, qty int) error
	// Remove fails with inventory.ErrNotEnough without changing anything.
	Remove(ctx context.Context, owner player.ID, item inventory.Item, qty int) error
}

type Repos interface {
	Players() Players
	Farms() Farms
	Inventory() Inventory
}

type UnitOfWork interface {
	// Do runs fn in a transaction, committing if it returns nil.
	Do(ctx context.Context, fn func(Repos) error) error
}
