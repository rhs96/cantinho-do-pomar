package farm

import (
	"errors"
	"testing"
	"time"
)

var t0 = time.Date(2026, 10, 22, 8, 0, 0, 0, time.UTC)

// radish ripens within a single watering, which keeps lifecycle tests short.
var radish = Crop{
	ID:             "radish",
	Yield:          4,
	StageDurations: [3]time.Duration{30 * time.Minute, time.Hour, 90 * time.Minute},
}

func plantedPlot(t *testing.T, c Crop) *Plot {
	t.Helper()
	p := NewPlot(Position{Col: 1, Row: 2})
	if err := p.Plow(); err != nil {
		t.Fatalf("Plow() = %v", err)
	}
	if err := p.Plant(c, t0); err != nil {
		t.Fatalf("Plant() = %v", err)
	}
	return p
}

func wateredRadish(t *testing.T) *Plot {
	t.Helper()
	p := plantedPlot(t, radish)
	if err := p.Water(t0); err != nil {
		t.Fatalf("Water() = %v", err)
	}
	return p
}

func TestNewPlotStartsAsGrass(t *testing.T) {
	p := NewPlot(Position{Col: 3, Row: 4})

	if p.State() != PlotGrass {
		t.Errorf("State() = %v, want grass", p.State())
	}
	if p.Position() != (Position{Col: 3, Row: 4}) {
		t.Errorf("Position() = %v", p.Position())
	}
	if _, ok := p.Crop(); ok {
		t.Error("new plot should have no crop")
	}
}

func TestPlow(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T) *Plot
		wantErr error
	}{
		{"grass", func(*testing.T) *Plot { return NewPlot(Position{}) }, nil},
		{"already plowed", func(*testing.T) *Plot {
			p := NewPlot(Position{})
			_ = p.Plow()
			return p
		}, ErrAlreadyPlowed},
		{"growing crop", wateredRadish, ErrPlotOccupied},
		{"withered crop", func(t *testing.T) *Plot {
			p := wateredRadish(t)
			if _, err := p.Harvest(t0.Add(radish.TimeToRipe())); err != nil {
				t.Fatalf("Harvest() = %v", err)
			}
			return p
		}, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := tt.setup(t)
			err := p.Plow()

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Plow() = %v, want %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if p.State() != PlotPlowed {
				t.Errorf("State() = %v, want plowed", p.State())
			}
			if _, ok := p.Crop(); ok {
				t.Error("plowed plot should have no crop")
			}
		})
	}
}

func TestPlant(t *testing.T) {
	t.Run("on grass", func(t *testing.T) {
		p := NewPlot(Position{})
		if err := p.Plant(radish, t0); !errors.Is(err, ErrNotPlowed) {
			t.Errorf("Plant() = %v, want ErrNotPlowed", err)
		}
	})

	t.Run("on occupied plot", func(t *testing.T) {
		p := plantedPlot(t, radish)
		if err := p.Plant(radish, t0); !errors.Is(err, ErrPlotOccupied) {
			t.Errorf("Plant() = %v, want ErrPlotOccupied", err)
		}
	})

	t.Run("on plowed plot", func(t *testing.T) {
		p := plantedPlot(t, radish)

		if p.State() != PlotPlanted {
			t.Errorf("State() = %v, want planted", p.State())
		}
		if c, ok := p.Crop(); !ok || c.ID != radish.ID {
			t.Errorf("Crop() = %v, %v", c.ID, ok)
		}
		if got, _ := p.Stage(t0); got != StageSeed {
			t.Errorf("Stage() = %v, want seed", got)
		}
		if p.IsWet(t0) {
			t.Error("freshly planted soil should be dry")
		}
	})
}

func TestStageProgression(t *testing.T) {
	p := wateredRadish(t)

	steps := []struct {
		after time.Duration
		want  Stage
	}{
		{10 * time.Minute, StageSeed},
		{30 * time.Minute, StageSprout},
		{2 * time.Hour, StageGrowing},
		{3 * time.Hour, StageRipe},
	}
	for _, s := range steps {
		if got, _ := p.Stage(t0.Add(s.after)); got != s.want {
			t.Errorf("Stage(+%v) = %v, want %v", s.after, got, s.want)
		}
	}
}

func TestNoGrowthWithoutWater(t *testing.T) {
	p := plantedPlot(t, radish)

	if got, _ := p.Stage(t0.Add(24 * time.Hour)); got != StageSeed {
		t.Errorf("Stage() after a dry day = %v, want seed", got)
	}
	if got := p.Remaining(t0.Add(24 * time.Hour)); got != radish.TimeToRipe() {
		t.Errorf("Remaining() = %v, want %v", got, radish.TimeToRipe())
	}
}

func TestGrowthPausesWhenSoilDriesOut(t *testing.T) {
	p := plantedPlot(t, tomato)
	if err := p.Water(t0); err != nil {
		t.Fatalf("Water() = %v", err)
	}

	// Soil dries at +4h and the player only comes back at +10h.
	dry := t0.Add(10 * time.Hour)
	if got := p.Remaining(dry); got != 2*time.Hour {
		t.Fatalf("Remaining() while dry = %v, want 2h", got)
	}
	if got, _ := p.Stage(dry); got != StageGrowing {
		t.Fatalf("Stage() while dry = %v, want growing", got)
	}

	if err := p.Water(dry); err != nil {
		t.Fatalf("second Water() = %v", err)
	}
	if got, _ := p.Stage(dry.Add(2*time.Hour - time.Second)); got != StageGrowing {
		t.Errorf("Stage() just before ripe = %v, want growing", got)
	}
	if got, _ := p.Stage(dry.Add(2 * time.Hour)); got != StageRipe {
		t.Errorf("Stage() after second watering = %v, want ripe", got)
	}
}

func TestWater(t *testing.T) {
	t.Run("nothing planted", func(t *testing.T) {
		p := NewPlot(Position{})
		_ = p.Plow()
		if err := p.Water(t0); !errors.Is(err, ErrNothingPlanted) {
			t.Errorf("Water() = %v, want ErrNothingPlanted", err)
		}
	})

	t.Run("still wet", func(t *testing.T) {
		p := wateredRadish(t)
		if err := p.Water(t0.Add(time.Hour)); !errors.Is(err, ErrAlreadyWet) {
			t.Errorf("Water() = %v, want ErrAlreadyWet", err)
		}
	})

	t.Run("ripe", func(t *testing.T) {
		p := wateredRadish(t)
		if err := p.Water(t0.Add(SoilMoisture)); !errors.Is(err, ErrAlreadyRipe) {
			t.Errorf("Water() = %v, want ErrAlreadyRipe", err)
		}
	})

	t.Run("wet window", func(t *testing.T) {
		p := wateredRadish(t)
		if !p.IsWet(t0.Add(SoilMoisture - time.Second)) {
			t.Error("soil should still be wet just before SoilMoisture")
		}
		if p.IsWet(t0.Add(SoilMoisture)) {
			t.Error("soil should be dry after SoilMoisture")
		}
	})
}

func TestRemaining(t *testing.T) {
	p := wateredRadish(t)

	if got := p.Remaining(t0.Add(time.Hour)); got != 2*time.Hour {
		t.Errorf("Remaining() = %v, want 2h", got)
	}
	if got := p.Remaining(t0.Add(10 * time.Hour)); got != 0 {
		t.Errorf("Remaining() after ripe = %v, want 0", got)
	}
	if got := NewPlot(Position{}).Remaining(t0); got != 0 {
		t.Errorf("Remaining() on empty plot = %v, want 0", got)
	}
}

func TestHarvest(t *testing.T) {
	ripe := t0.Add(radish.TimeToRipe())

	t.Run("before ripe", func(t *testing.T) {
		p := wateredRadish(t)
		if _, err := p.Harvest(t0.Add(time.Hour)); !errors.Is(err, ErrNotRipe) {
			t.Fatalf("Harvest() = %v, want ErrNotRipe", err)
		}
		if p.State() != PlotPlanted {
			t.Errorf("failed harvest changed state to %v", p.State())
		}
	})

	t.Run("nothing planted", func(t *testing.T) {
		if _, err := NewPlot(Position{}).Harvest(t0); !errors.Is(err, ErrNothingPlanted) {
			t.Errorf("Harvest() = %v, want ErrNothingPlanted", err)
		}
	})

	t.Run("ripe", func(t *testing.T) {
		p := wateredRadish(t)

		h, err := p.Harvest(ripe)
		if err != nil {
			t.Fatalf("Harvest() = %v", err)
		}
		if h != (Harvest{Crop: radish.ID, Quantity: radish.Yield}) {
			t.Errorf("Harvest() = %+v", h)
		}
		if p.State() != PlotWithered {
			t.Errorf("State() = %v, want withered", p.State())
		}
	})

	t.Run("twice", func(t *testing.T) {
		p := wateredRadish(t)
		_, _ = p.Harvest(ripe)

		if _, err := p.Harvest(ripe); !errors.Is(err, ErrNothingPlanted) {
			t.Errorf("second Harvest() = %v, want ErrNothingPlanted", err)
		}
	})
}
