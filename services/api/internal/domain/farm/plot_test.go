package farm

import (
	"errors"
	"testing"
	"time"
)

var t0 = time.Date(2026, 10, 22, 8, 0, 0, 0, time.UTC)

func plantedPlot(t *testing.T) *Plot {
	t.Helper()
	p := NewPlot(Position{Col: 1, Row: 2})
	if err := p.Plow(); err != nil {
		t.Fatalf("Plow() = %v", err)
	}
	if err := p.Plant(tomato, t0); err != nil {
		t.Fatalf("Plant() = %v", err)
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
		{"already plowed", func(t *testing.T) *Plot {
			p := NewPlot(Position{})
			_ = p.Plow()
			return p
		}, ErrAlreadyPlowed},
		{"growing crop", plantedPlot, ErrPlotOccupied},
		{"withered crop", func(t *testing.T) *Plot {
			p := plantedPlot(t)
			if _, err := p.Harvest(t0.Add(tomato.TimeToRipe())); err != nil {
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
			if err == nil && p.State() != PlotPlowed {
				t.Errorf("State() = %v, want plowed", p.State())
			}
		})
	}
}

func TestPlant(t *testing.T) {
	t.Run("on grass", func(t *testing.T) {
		p := NewPlot(Position{})
		if err := p.Plant(tomato, t0); !errors.Is(err, ErrNotPlowed) {
			t.Errorf("Plant() = %v, want ErrNotPlowed", err)
		}
	})

	t.Run("on occupied plot", func(t *testing.T) {
		p := plantedPlot(t)
		if err := p.Plant(tomato, t0); !errors.Is(err, ErrPlotOccupied) {
			t.Errorf("Plant() = %v, want ErrPlotOccupied", err)
		}
	})

	t.Run("on plowed plot", func(t *testing.T) {
		p := plantedPlot(t)

		if p.State() != PlotPlanted {
			t.Errorf("State() = %v, want planted", p.State())
		}
		if c, ok := p.Crop(); !ok || c.ID != tomato.ID {
			t.Errorf("Crop() = %v, %v", c.ID, ok)
		}
		if got, _ := p.Stage(t0); got != StageSeed {
			t.Errorf("Stage() = %v, want seed", got)
		}
	})
}

func TestStageProgression(t *testing.T) {
	p := plantedPlot(t)

	steps := []struct {
		after time.Duration
		want  Stage
	}{
		{30 * time.Minute, StageSeed},
		{time.Hour, StageSprout},
		{4 * time.Hour, StageGrowing},
		{6 * time.Hour, StageRipe},
	}
	for _, s := range steps {
		if got, _ := p.Stage(t0.Add(s.after)); got != s.want {
			t.Errorf("Stage(+%v) = %v, want %v", s.after, got, s.want)
		}
	}
}

func TestRemaining(t *testing.T) {
	p := plantedPlot(t)

	if got := p.Remaining(t0.Add(2 * time.Hour)); got != 4*time.Hour {
		t.Errorf("Remaining() = %v, want 4h", got)
	}
	if got := p.Remaining(t0.Add(10 * time.Hour)); got != 0 {
		t.Errorf("Remaining() after ripe = %v, want 0", got)
	}
	if got := NewPlot(Position{}).Remaining(t0); got != 0 {
		t.Errorf("Remaining() on empty plot = %v, want 0", got)
	}
}

func TestHarvest(t *testing.T) {
	t.Run("before ripe", func(t *testing.T) {
		p := plantedPlot(t)
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
		p := plantedPlot(t)

		h, err := p.Harvest(t0.Add(tomato.TimeToRipe()))
		if err != nil {
			t.Fatalf("Harvest() = %v", err)
		}
		if h != (Harvest{Crop: tomato.ID, Quantity: tomato.Yield}) {
			t.Errorf("Harvest() = %+v", h)
		}
		if p.State() != PlotWithered {
			t.Errorf("State() = %v, want withered", p.State())
		}
	})

	t.Run("twice", func(t *testing.T) {
		p := plantedPlot(t)
		ripe := t0.Add(tomato.TimeToRipe())
		_, _ = p.Harvest(ripe)

		if _, err := p.Harvest(ripe); !errors.Is(err, ErrNothingPlanted) {
			t.Errorf("second Harvest() = %v, want ErrNothingPlanted", err)
		}
	})
}
