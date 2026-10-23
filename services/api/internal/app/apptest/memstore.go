// Package apptest provides an in-memory unit of work for use case tests.
package apptest

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"

	"github.com/rhs96/cantinho-do-pomar/services/api/internal/app"
	"github.com/rhs96/cantinho-do-pomar/services/api/internal/domain/farm"
	"github.com/rhs96/cantinho-do-pomar/services/api/internal/domain/inventory"
	"github.com/rhs96/cantinho-do-pomar/services/api/internal/domain/player"
)

type farmRecord struct {
	id            farm.ID
	width, height int
	plots         []farm.PlotSnapshot
}

// Store keeps everything in maps. Do holds a single lock for the whole unit of
// work and restores the previous state if fn fails, like a rolled back
// transaction.
type Store struct {
	mu      sync.Mutex
	crops   farm.CropLookup
	seq     int
	players map[player.ID]player.Player
	farms   map[player.ID]farmRecord
	items   map[player.ID]map[inventory.Item]int
}

func NewStore(crops farm.CropLookup) *Store {
	return &Store{
		crops:   crops,
		players: map[player.ID]player.Player{},
		farms:   map[player.ID]farmRecord{},
		items:   map[player.ID]map[inventory.Item]int{},
	}
}

func (s *Store) Do(_ context.Context, fn func(app.Repos) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	players := maps.Clone(s.players)
	farms := make(map[player.ID]farmRecord, len(s.farms))
	for k, v := range s.farms {
		v.plots = slices.Clone(v.plots)
		farms[k] = v
	}

	items := make(map[player.ID]map[inventory.Item]int, len(s.items))
	for k, v := range s.items {
		items[k] = maps.Clone(v)
	}

	if err := fn(repos{s}); err != nil {
		s.players, s.farms, s.items = players, farms, items
		return err
	}
	return nil
}

// AddPlayer stores p with a fresh ID and returns it.
func (s *Store) AddPlayer(p player.Player) player.Player {
	s.mu.Lock()
	defer s.mu.Unlock()
	p.ID = player.ID(s.nextID("player"))
	s.players[p.ID] = p
	return p
}

// SetPlayer overwrites a stored player.
func (s *Store) SetPlayer(p player.Player) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.players[p.ID] = p
}

func (s *Store) Player(id player.ID) player.Player {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.players[id]
}

// Quantity reports how many of item the player holds.
func (s *Store) Quantity(owner player.ID, item inventory.Item) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.items[owner][item]
}

// Give puts items straight into the player's inventory.
func (s *Store) Give(owner player.ID, item inventory.Item, qty int) error {
	return s.Do(context.Background(), func(r app.Repos) error {
		return r.Inventory().Add(context.Background(), owner, item, qty)
	})
}

func (s *Store) nextID(prefix string) string {
	s.seq++
	return fmt.Sprintf("%s-%d", prefix, s.seq)
}

type repos struct{ s *Store }

func (r repos) Players() app.Players     { return players(r) }
func (r repos) Farms() app.Farms         { return farms(r) }
func (r repos) Inventory() app.Inventory { return items(r) }

type players repos

func (r players) Get(_ context.Context, id player.ID) (player.Player, error) {
	p, ok := r.s.players[id]
	if !ok {
		return player.Player{}, player.ErrNotFound
	}
	return p, nil
}

func (r players) GetForUpdate(ctx context.Context, id player.ID) (player.Player, error) {
	return r.Get(ctx, id)
}

func (r players) Update(_ context.Context, p player.Player) error {
	if _, ok := r.s.players[p.ID]; !ok {
		return player.ErrNotFound
	}
	r.s.players[p.ID] = p
	return nil
}

type farms repos

func (r farms) Get(_ context.Context, owner player.ID) (*farm.Farm, error) {
	rec, ok := r.s.farms[owner]
	if !ok {
		return nil, farm.ErrNotFound
	}

	plots := make([]*farm.Plot, 0, len(rec.plots))
	for _, snap := range rec.plots {
		p, err := farm.RestorePlot(snap, r.s.crops)
		if err != nil {
			return nil, err
		}
		plots = append(plots, p)
	}
	return farm.Restore(rec.id, owner, rec.width, rec.height, plots)
}

func (r farms) Create(ctx context.Context, f *farm.Farm) (*farm.Farm, error) {
	if _, ok := r.s.farms[f.Owner]; !ok {
		rec := farmRecord{id: farm.ID(r.s.nextID("farm")), width: f.Width, height: f.Height}
		for _, p := range f.Plots() {
			rec.plots = append(rec.plots, p.Snapshot())
		}
		r.s.farms[f.Owner] = rec
	}
	return r.Get(ctx, f.Owner)
}

func (r farms) UpdatePlot(_ context.Context, owner player.ID, pos farm.Position, fn func(*farm.Plot) error) (*farm.Plot, error) {
	rec, ok := r.s.farms[owner]
	if !ok {
		return nil, farm.ErrNotFound
	}
	i := slices.IndexFunc(rec.plots, func(s farm.PlotSnapshot) bool { return s.Pos == pos })
	if i < 0 {
		return nil, farm.ErrOutOfBounds
	}

	p, err := farm.RestorePlot(rec.plots[i], r.s.crops)
	if err != nil {
		return nil, err
	}
	if err := fn(p); err != nil {
		return nil, err
	}
	rec.plots[i] = p.Snapshot()
	return p, nil
}

type items repos

func (r items) List(_ context.Context, owner player.ID) ([]inventory.Stack, error) {
	var stacks []inventory.Stack
	for item, qty := range r.s.items[owner] {
		if qty > 0 {
			stacks = append(stacks, inventory.Stack{Item: item, Quantity: qty})
		}
	}
	slices.SortFunc(stacks, func(a, b inventory.Stack) int {
		if a.Item.Kind != b.Item.Kind {
			return strings.Compare(string(a.Item.Kind), string(b.Item.Kind))
		}
		return strings.Compare(a.Item.ID, b.Item.ID)
	})
	return stacks, nil
}

func (r items) Add(_ context.Context, owner player.ID, item inventory.Item, qty int) error {
	if err := inventory.ValidateQuantity(qty); err != nil {
		return err
	}
	if r.s.items[owner] == nil {
		r.s.items[owner] = map[inventory.Item]int{}
	}
	r.s.items[owner][item] += qty
	return nil
}

func (r items) Remove(_ context.Context, owner player.ID, item inventory.Item, qty int) error {
	if err := inventory.ValidateQuantity(qty); err != nil {
		return err
	}
	if r.s.items[owner][item] < qty {
		return inventory.ErrNotEnough
	}
	r.s.items[owner][item] -= qty
	return nil
}
