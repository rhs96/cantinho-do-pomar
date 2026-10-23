// Package inventory models what a player keeps in the barn: seeds to plant and
// produce to sell.
package inventory

import (
	"errors"

	"github.com/rhs96/cantinho-do-pomar/services/api/internal/domain/farm"
)

type Kind string

const (
	KindSeed    Kind = "seed"
	KindProduce Kind = "produce"
)

var (
	ErrNotEnough       = errors.New("not enough items")
	ErrInvalidQuantity = errors.New("quantity must be positive")
)

type Item struct {
	Kind Kind
	ID   string
}

func Seed(crop farm.CropID) Item    { return Item{Kind: KindSeed, ID: string(crop)} }
func Produce(crop farm.CropID) Item { return Item{Kind: KindProduce, ID: string(crop)} }

type Stack struct {
	Item     Item
	Quantity int
}

func ValidateQuantity(n int) error {
	if n <= 0 {
		return ErrInvalidQuantity
	}
	return nil
}
