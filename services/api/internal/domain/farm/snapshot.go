package farm

import (
	"fmt"
	"time"
)

// PlotSnapshot is the persisted form of a Plot.
type PlotSnapshot struct {
	Pos      Position
	State    PlotState
	CropID   CropID
	Growth   time.Duration
	GrowthAt time.Time
	WetUntil time.Time
}

type CropLookup interface {
	Crop(id CropID) (Crop, bool)
}

func (p *Plot) Snapshot() PlotSnapshot {
	return PlotSnapshot{
		Pos:      p.pos,
		State:    p.state,
		CropID:   p.crop.ID,
		Growth:   p.growth,
		GrowthAt: p.updatedAt,
		WetUntil: p.wetUntil,
	}
}

func RestorePlot(s PlotSnapshot, crops CropLookup) (*Plot, error) {
	p := &Plot{
		pos:       s.Pos,
		state:     s.State,
		growth:    s.Growth,
		updatedAt: s.GrowthAt,
		wetUntil:  s.WetUntil,
	}

	switch s.State {
	case PlotGrass, PlotPlowed:
		return p, nil
	case PlotPlanted, PlotWithered:
		c, ok := crops.Crop(s.CropID)
		if !ok {
			return nil, fmt.Errorf("plot %v: unknown crop %q", s.Pos, s.CropID)
		}
		p.crop = c
		return p, nil
	default:
		return nil, fmt.Errorf("plot %v: unknown state %q", s.Pos, s.State)
	}
}
