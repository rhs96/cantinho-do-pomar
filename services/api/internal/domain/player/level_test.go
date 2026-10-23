package player

import "testing"

func TestXPForLevel(t *testing.T) {
	want := map[int]int64{0: 0, 1: 0, 2: 50, 3: 150, 4: 300, 5: 500, 10: 2250}
	for level, xp := range want {
		if got := XPForLevel(level); got != xp {
			t.Errorf("XPForLevel(%d) = %d, want %d", level, got, xp)
		}
	}
}

func TestLevelFor(t *testing.T) {
	tests := []struct {
		xp   int64
		want int
	}{
		{0, 1}, {49, 1}, {50, 2}, {149, 2}, {150, 3}, {499, 4}, {500, 5}, {2250, 10},
	}
	for _, tt := range tests {
		if got := LevelFor(tt.xp); got != tt.want {
			t.Errorf("LevelFor(%d) = %d, want %d", tt.xp, got, tt.want)
		}
	}
}

func TestGainXP(t *testing.T) {
	p := Player{XP: 40}

	if p.GainXP(5) {
		t.Error("45 XP should not level up")
	}
	if !p.GainXP(5) {
		t.Error("50 XP should reach level 2")
	}
	if p.Level() != 2 || p.XP != 50 {
		t.Errorf("level, xp = %d, %d", p.Level(), p.XP)
	}
	if p.GainXP(0) || p.GainXP(-10) || p.XP != 50 {
		t.Error("non-positive xp must be ignored")
	}
}
