package farm

import "time"

type CropID string

type Stage int

const (
	StageSeed Stage = iota
	StageSprout
	StageGrowing
	StageRipe
)

func (s Stage) String() string {
	switch s {
	case StageSeed:
		return "seed"
	case StageSprout:
		return "sprout"
	case StageGrowing:
		return "growing"
	case StageRipe:
		return "ripe"
	default:
		return "unknown"
	}
}

// Crop describes how a plant grows. Durations are counted only while the
// soil is moist, so they measure effective growth rather than wall time.
type Crop struct {
	ID    CropID
	Yield int

	// How long the plant stays in each stage before ripening:
	// seed, sprout and growing, in that order.
	StageDurations [3]time.Duration
}

func (c Crop) TimeToRipe() time.Duration {
	var total time.Duration
	for _, d := range c.StageDurations {
		total += d
	}
	return total
}

func (c Crop) StageAt(growth time.Duration) Stage {
	var elapsed time.Duration
	for i, d := range c.StageDurations {
		elapsed += d
		if growth < elapsed {
			return Stage(i)
		}
	}
	return StageRipe
}
