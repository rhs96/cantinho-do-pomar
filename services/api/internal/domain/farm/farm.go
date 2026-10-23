package farm

import (
	"errors"
	"fmt"

	"github.com/rhs96/cantinho-do-pomar/services/api/internal/domain/player"
)

type ID string

const (
	StarterSize = 6
	MaxSize     = 24
)

var (
	ErrNotFound    = errors.New("farm not found")
	ErrOutOfBounds = errors.New("position is outside the farm")
	ErrInvalidSize = fmt.Errorf("farm size must be between 1 and %d", MaxSize)
)

type Farm struct {
	ID     ID
	Owner  player.ID
	Width  int
	Height int

	plots []*Plot
}

// New lays out a farm covered in grass. It has no ID until persisted.
func New(owner player.ID, width, height int) (*Farm, error) {
	if width < 1 || height < 1 || width > MaxSize || height > MaxSize {
		return nil, ErrInvalidSize
	}

	f := &Farm{Owner: owner, Width: width, Height: height, plots: make([]*Plot, 0, width*height)}
	for row := range height {
		for col := range width {
			f.plots = append(f.plots, NewPlot(Position{Col: col, Row: row}))
		}
	}
	return f, nil
}

// Restore rebuilds a farm from storage. Plots are expected to cover the whole
// grid; missing ones are filled with grass.
func Restore(id ID, owner player.ID, width, height int, plots []*Plot) (*Farm, error) {
	f, err := New(owner, width, height)
	if err != nil {
		return nil, err
	}
	f.ID = id

	for _, p := range plots {
		if !f.contains(p.Position()) {
			return nil, fmt.Errorf("restore plot %v: %w", p.Position(), ErrOutOfBounds)
		}
		f.plots[f.index(p.Position())] = p
	}
	return f, nil
}

func (f *Farm) Plot(pos Position) (*Plot, error) {
	if !f.contains(pos) {
		return nil, ErrOutOfBounds
	}
	return f.plots[f.index(pos)], nil
}

// Plots returns plots in row-major order.
func (f *Farm) Plots() []*Plot {
	return f.plots
}

func (f *Farm) contains(p Position) bool {
	return p.Col >= 0 && p.Row >= 0 && p.Col < f.Width && p.Row < f.Height
}

func (f *Farm) index(p Position) int {
	return p.Row*f.Width + p.Col
}
