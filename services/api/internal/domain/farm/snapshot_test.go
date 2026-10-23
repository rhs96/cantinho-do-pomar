package farm

import (
	"testing"
	"time"
)

type lookup map[CropID]Crop

func (l lookup) Crop(id CropID) (Crop, bool) {
	c, ok := l[id]
	return c, ok
}

func TestSnapshotRoundTrip(t *testing.T) {
	crops := lookup{radish.ID: radish, tomato.ID: tomato}
	later := t0.Add(90 * time.Minute)

	grass := NewPlot(Position{Col: 2, Row: 0})

	watered := plantedPlot(t, tomato)
	_ = watered.Water(t0)

	withered := wateredRadish(t)
	_, _ = withered.Harvest(t0.Add(radish.TimeToRipe()))

	for _, original := range []*Plot{grass, watered, withered} {
		restored, err := RestorePlot(original.Snapshot(), crops)
		if err != nil {
			t.Fatalf("RestorePlot(%v) = %v", original.State(), err)
		}
		if restored.Snapshot() != original.Snapshot() {
			t.Errorf("round trip changed snapshot:\n got %+v\nwant %+v", restored.Snapshot(), original.Snapshot())
		}

		gotStage, gotErr := restored.Stage(later)
		wantStage, wantErr := original.Stage(later)
		if gotStage != wantStage || (gotErr == nil) != (wantErr == nil) {
			t.Errorf("%v: Stage() = %v, %v; want %v, %v", original.State(), gotStage, gotErr, wantStage, wantErr)
		}
	}
}

func TestRestorePlotRejectsBadData(t *testing.T) {
	tests := []struct {
		name string
		snap PlotSnapshot
	}{
		{"unknown crop", PlotSnapshot{State: PlotPlanted, CropID: "mandrake"}},
		{"unknown state", PlotSnapshot{State: "flooded"}},
	}

	for _, tt := range tests {
		if _, err := RestorePlot(tt.snap, lookup{}); err == nil {
			t.Errorf("%s: expected error", tt.name)
		}
	}
}
