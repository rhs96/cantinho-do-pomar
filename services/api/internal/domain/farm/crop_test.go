package farm

import (
	"testing"
	"time"
)

var tomato = Crop{
	ID:             "tomato",
	Yield:          10,
	StageDurations: [3]time.Duration{time.Hour, 2 * time.Hour, 3 * time.Hour},
}

func TestCropTimeToRipe(t *testing.T) {
	if got := tomato.TimeToRipe(); got != 6*time.Hour {
		t.Errorf("TimeToRipe() = %v, want 6h", got)
	}
}

func TestCropStageAt(t *testing.T) {
	tests := []struct {
		growth time.Duration
		want   Stage
	}{
		{0, StageSeed},
		{59 * time.Minute, StageSeed},
		{time.Hour, StageSprout},
		{3*time.Hour - time.Second, StageSprout},
		{3 * time.Hour, StageGrowing},
		{6*time.Hour - time.Second, StageGrowing},
		{6 * time.Hour, StageRipe},
		{48 * time.Hour, StageRipe},
	}

	for _, tt := range tests {
		if got := tomato.StageAt(tt.growth); got != tt.want {
			t.Errorf("StageAt(%v) = %v, want %v", tt.growth, got, tt.want)
		}
	}
}
