package pgstore

import (
	"context"

	"github.com/rhs96/cantinho-do-pomar/services/api/internal/domain/inventory"
	"github.com/rhs96/cantinho-do-pomar/services/api/internal/domain/player"
)

type Inventory struct {
	db DBTX
}

func NewInventory(db DBTX) *Inventory {
	return &Inventory{db: db}
}

func (r *Inventory) List(ctx context.Context, owner player.ID) ([]inventory.Stack, error) {
	rows, err := r.db.Query(ctx, `
		SELECT kind, item_id, quantity
		FROM inventory
		WHERE player_id = $1 AND quantity > 0
		ORDER BY kind, item_id`, owner)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stacks []inventory.Stack
	for rows.Next() {
		var (
			s    inventory.Stack
			kind string
		)
		if err := rows.Scan(&kind, &s.Item.ID, &s.Quantity); err != nil {
			return nil, err
		}
		s.Item.Kind = inventory.Kind(kind)
		stacks = append(stacks, s)
	}
	return stacks, rows.Err()
}

func (r *Inventory) Add(ctx context.Context, owner player.ID, item inventory.Item, qty int) error {
	if err := inventory.ValidateQuantity(qty); err != nil {
		return err
	}
	_, err := r.db.Exec(ctx, `
		INSERT INTO inventory (player_id, kind, item_id, quantity)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (player_id, kind, item_id)
		DO UPDATE SET quantity = inventory.quantity + EXCLUDED.quantity`,
		owner, string(item.Kind), item.ID, qty,
	)
	return err
}

// Remove takes qty items or fails with inventory.ErrNotEnough, never leaving
// a partial change behind.
func (r *Inventory) Remove(ctx context.Context, owner player.ID, item inventory.Item, qty int) error {
	if err := inventory.ValidateQuantity(qty); err != nil {
		return err
	}
	tag, err := r.db.Exec(ctx, `
		UPDATE inventory
		SET quantity = quantity - $4
		WHERE player_id = $1 AND kind = $2 AND item_id = $3 AND quantity >= $4`,
		owner, string(item.Kind), item.ID, qty,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return inventory.ErrNotEnough
	}
	return nil
}
