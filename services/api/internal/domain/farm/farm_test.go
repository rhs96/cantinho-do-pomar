package farm

import (
	"errors"
	"testing"
)

func TestNewFarm(t *testing.T) {
	f, err := New("p1", 3, 2)
	if err != nil {
		t.Fatalf("New() = %v", err)
	}

	if got := len(f.Plots()); got != 6 {
		t.Fatalf("len(Plots()) = %d, want 6", got)
	}
	for i, p := range f.Plots() {
		want := Position{Col: i % 3, Row: i / 3}
		if p.Position() != want {
			t.Errorf("Plots()[%d] at %v, want %v", i, p.Position(), want)
		}
		if p.State() != PlotGrass {
			t.Errorf("Plots()[%d] state = %v, want grass", i, p.State())
		}
	}
}

func TestNewFarmRejectsInvalidSize(t *testing.T) {
	for _, size := range [][2]int{{0, 6}, {6, 0}, {MaxSize + 1, 1}, {-1, -1}} {
		if _, err := New("p1", size[0], size[1]); !errors.Is(err, ErrInvalidSize) {
			t.Errorf("New(%v) = %v, want ErrInvalidSize", size, err)
		}
	}
}

func TestFarmPlot(t *testing.T) {
	f, _ := New("p1", 4, 3)

	p, err := f.Plot(Position{Col: 3, Row: 2})
	if err != nil {
		t.Fatalf("Plot() = %v", err)
	}
	if p.Position() != (Position{Col: 3, Row: 2}) {
		t.Errorf("Plot() returned %v", p.Position())
	}

	for _, pos := range []Position{{-1, 0}, {0, -1}, {4, 0}, {0, 3}} {
		if _, err := f.Plot(pos); !errors.Is(err, ErrOutOfBounds) {
			t.Errorf("Plot(%v) = %v, want ErrOutOfBounds", pos, err)
		}
	}
}

func TestRestoreFarm(t *testing.T) {
	planted := plantedPlot(t, radish) // at {1, 2}

	f, err := Restore("f1", "p1", 3, 3, []*Plot{planted})
	if err != nil {
		t.Fatalf("Restore() = %v", err)
	}
	if f.ID != "f1" {
		t.Errorf("ID = %q", f.ID)
	}

	got, _ := f.Plot(Position{Col: 1, Row: 2})
	if got != planted {
		t.Error("restored plot not placed at its position")
	}
	other, _ := f.Plot(Position{Col: 0, Row: 0})
	if other.State() != PlotGrass {
		t.Errorf("missing plot state = %v, want grass", other.State())
	}

	if _, err := Restore("f1", "p1", 1, 1, []*Plot{planted}); !errors.Is(err, ErrOutOfBounds) {
		t.Errorf("Restore() with plot outside grid = %v, want ErrOutOfBounds", err)
	}
}
