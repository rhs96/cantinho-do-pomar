// Package catalog holds the game content: which crops exist and how they grow.
// Balancing changes should only need to touch this package.
package catalog

import (
	"time"

	"github.com/rhs96/cantinho-do-pomar/services/api/internal/domain/farm"
)

var starterCrops = []farm.Crop{
	{ID: "strawberry", Yield: 6, StageDurations: stages(10*time.Minute, 20*time.Minute, 30*time.Minute)},
	{ID: "tomato", Yield: 10, StageDurations: stages(30*time.Minute, time.Hour, 90*time.Minute)},
	{ID: "corn", Yield: 12, StageDurations: stages(time.Hour, 2*time.Hour, 3*time.Hour)},
	{ID: "pepper", Yield: 14, StageDurations: stages(time.Hour, 3*time.Hour, 4*time.Hour)},
	{ID: "pumpkin", Yield: 8, StageDurations: stages(2*time.Hour, 4*time.Hour, 6*time.Hour)},
	{ID: "watermelon", Yield: 6, StageDurations: stages(3*time.Hour, 5*time.Hour, 8*time.Hour)},
}

func stages(seed, sprout, growing time.Duration) [3]time.Duration {
	return [3]time.Duration{seed, sprout, growing}
}

type Catalog struct {
	crops []farm.Crop
	byID  map[farm.CropID]farm.Crop
}

func New(crops []farm.Crop) *Catalog {
	c := &Catalog{crops: crops, byID: make(map[farm.CropID]farm.Crop, len(crops))}
	for _, crop := range crops {
		c.byID[crop.ID] = crop
	}
	return c
}

func Default() *Catalog {
	return New(starterCrops)
}

func (c *Catalog) Crop(id farm.CropID) (farm.Crop, bool) {
	crop, ok := c.byID[id]
	return crop, ok
}

func (c *Catalog) Crops() []farm.Crop {
	return c.crops
}
