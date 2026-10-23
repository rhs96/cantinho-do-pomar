// Package catalog holds the game content: which crops exist, how they grow and
// what they cost. Balancing changes should only need to touch this package.
package catalog

import (
	"time"

	"github.com/rhs96/cantinho-do-pomar/services/api/internal/domain/farm"
)

// Entry is a crop as the shop and the progression system see it.
type Entry struct {
	farm.Crop
	SeedPrice int64
	// SellPrice is paid per unit of produce.
	SellPrice int64
	MinLevel  int
	HarvestXP int64
}

// Profit is what one plot earns per harvest, ignoring XP.
func (e Entry) Profit() int64 {
	return int64(e.Yield)*e.SellPrice - e.SeedPrice
}

var starter = []Entry{
	{
		Crop:      farm.Crop{ID: "strawberry", Yield: 6, StageDurations: stages(10*time.Minute, 20*time.Minute, 30*time.Minute)},
		SeedPrice: 10, SellPrice: 3, MinLevel: 1, HarvestXP: 4,
	},
	{
		Crop:      farm.Crop{ID: "tomato", Yield: 10, StageDurations: stages(30*time.Minute, time.Hour, 90*time.Minute)},
		SeedPrice: 25, SellPrice: 5, MinLevel: 1, HarvestXP: 8,
	},
	{
		Crop:      farm.Crop{ID: "corn", Yield: 12, StageDurations: stages(time.Hour, 2*time.Hour, 3*time.Hour)},
		SeedPrice: 40, SellPrice: 7, MinLevel: 2, HarvestXP: 14,
	},
	{
		Crop:      farm.Crop{ID: "pepper", Yield: 14, StageDurations: stages(time.Hour, 3*time.Hour, 4*time.Hour)},
		SeedPrice: 60, SellPrice: 8, MinLevel: 3, HarvestXP: 18,
	},
	{
		Crop:      farm.Crop{ID: "pumpkin", Yield: 8, StageDurations: stages(2*time.Hour, 4*time.Hour, 6*time.Hour)},
		SeedPrice: 80, SellPrice: 20, MinLevel: 4, HarvestXP: 26,
	},
	{
		Crop:      farm.Crop{ID: "watermelon", Yield: 6, StageDurations: stages(3*time.Hour, 5*time.Hour, 8*time.Hour)},
		SeedPrice: 120, SellPrice: 40, MinLevel: 5, HarvestXP: 35,
	},
}

func stages(seed, sprout, growing time.Duration) [3]time.Duration {
	return [3]time.Duration{seed, sprout, growing}
}

type Catalog struct {
	entries []Entry
	byID    map[farm.CropID]Entry
}

func New(entries []Entry) *Catalog {
	c := &Catalog{entries: entries, byID: make(map[farm.CropID]Entry, len(entries))}
	for _, e := range entries {
		c.byID[e.ID] = e
	}
	return c
}

func Default() *Catalog {
	return New(starter)
}

// Crop implements farm.CropLookup.
func (c *Catalog) Crop(id farm.CropID) (farm.Crop, bool) {
	e, ok := c.byID[id]
	return e.Crop, ok
}

func (c *Catalog) Entry(id farm.CropID) (Entry, bool) {
	e, ok := c.byID[id]
	return e, ok
}

// Entries are ordered by unlock level.
func (c *Catalog) Entries() []Entry {
	return c.entries
}
