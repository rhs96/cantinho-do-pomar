package farm

import (
	"errors"
	"time"
)

type PlotState string

const (
	PlotGrass    PlotState = "grass"
	PlotPlowed   PlotState = "plowed"
	PlotPlanted  PlotState = "planted"
	PlotWithered PlotState = "withered"
)

var (
	ErrAlreadyPlowed  = errors.New("plot is already plowed")
	ErrPlotOccupied   = errors.New("plot already has a crop")
	ErrNotPlowed      = errors.New("plot must be plowed first")
	ErrNothingPlanted = errors.New("nothing is planted here")
	ErrNotRipe        = errors.New("crop is not ripe yet")
	ErrAlreadyRipe    = errors.New("crop is ripe, harvest it instead")
	ErrAlreadyWet     = errors.New("soil is still wet")
)

// SoilMoisture is how long a watering keeps the plant growing.
const SoilMoisture = 4 * time.Hour

type Position struct {
	Col int
	Row int
}

type Harvest struct {
	Crop     CropID
	Quantity int
}

type Plot struct {
	pos   Position
	state PlotState
	crop  Crop

	// growth accumulated up to updatedAt. Anything after that is derived
	// from wetUntil, so reads never need to mutate the plot.
	growth    time.Duration
	updatedAt time.Time
	wetUntil  time.Time
}

func NewPlot(pos Position) *Plot {
	return &Plot{pos: pos, state: PlotGrass}
}

func (p *Plot) Position() Position { return p.pos }
func (p *Plot) State() PlotState   { return p.state }

func (p *Plot) Crop() (Crop, bool) {
	if p.state != PlotPlanted && p.state != PlotWithered {
		return Crop{}, false
	}
	return p.crop, true
}

func (p *Plot) Plow() error {
	switch p.state {
	case PlotPlowed:
		return ErrAlreadyPlowed
	case PlotPlanted:
		return ErrPlotOccupied
	}

	*p = Plot{pos: p.pos, state: PlotPlowed}
	return nil
}

func (p *Plot) Plant(c Crop, now time.Time) error {
	switch p.state {
	case PlotPlanted, PlotWithered:
		return ErrPlotOccupied
	case PlotGrass:
		return ErrNotPlowed
	}

	p.state = PlotPlanted
	p.crop = c
	p.growth = 0
	p.updatedAt = now
	p.wetUntil = time.Time{}
	return nil
}

func (p *Plot) Water(now time.Time) error {
	stage, err := p.Stage(now)
	if err != nil {
		return err
	}
	if stage == StageRipe {
		return ErrAlreadyRipe
	}
	if p.IsWet(now) {
		return ErrAlreadyWet
	}

	p.growth = p.growthAt(now)
	p.updatedAt = now
	p.wetUntil = now.Add(SoilMoisture)
	return nil
}

func (p *Plot) IsWet(now time.Time) bool {
	return p.state == PlotPlanted && now.Before(p.wetUntil)
}

func (p *Plot) Stage(now time.Time) (Stage, error) {
	if p.state != PlotPlanted {
		return 0, ErrNothingPlanted
	}
	return p.crop.StageAt(p.growthAt(now)), nil
}

// Remaining is the growth still needed to ripen, which only advances while the
// soil is wet.
func (p *Plot) Remaining(now time.Time) time.Duration {
	if p.state != PlotPlanted {
		return 0
	}
	return max(p.crop.TimeToRipe()-p.growthAt(now), 0)
}

// Harvest leaves the withered plant behind; the plot has to be plowed again.
func (p *Plot) Harvest(now time.Time) (Harvest, error) {
	stage, err := p.Stage(now)
	if err != nil {
		return Harvest{}, err
	}
	if stage != StageRipe {
		return Harvest{}, ErrNotRipe
	}

	p.state = PlotWithered
	return Harvest{Crop: p.crop.ID, Quantity: p.crop.Yield}, nil
}

func (p *Plot) growthAt(now time.Time) time.Duration {
	wetEnd := now
	if p.wetUntil.Before(wetEnd) {
		wetEnd = p.wetUntil
	}
	if !wetEnd.After(p.updatedAt) {
		return p.growth
	}
	return p.growth + wetEnd.Sub(p.updatedAt)
}
