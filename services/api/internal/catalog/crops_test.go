package catalog

import "testing"

func TestDefaultCatalogIsConsistent(t *testing.T) {
	c := Default()

	if len(c.Entries()) == 0 {
		t.Fatal("catalog is empty")
	}

	seen := map[string]bool{}
	lastLevel := 0
	for _, e := range c.Entries() {
		if seen[string(e.ID)] {
			t.Errorf("duplicate crop %q", e.ID)
		}
		seen[string(e.ID)] = true

		if e.Yield <= 0 || e.SeedPrice <= 0 || e.SellPrice <= 0 || e.HarvestXP <= 0 {
			t.Errorf("%s: yield, prices and xp must be positive", e.ID)
		}
		for i, d := range e.StageDurations {
			if d <= 0 {
				t.Errorf("%s: stage %d has non-positive duration", e.ID, i)
			}
		}
		if e.MinLevel < lastLevel {
			t.Errorf("%s: entries must be ordered by MinLevel", e.ID)
		}
		lastLevel = e.MinLevel

		if crop, ok := c.Crop(e.ID); !ok || crop.ID != e.ID {
			t.Errorf("Crop(%q) lookup failed", e.ID)
		}
	}

	if _, ok := c.Crop("mandrake"); ok {
		t.Error("unknown crop should not be found")
	}
	if c.Entries()[0].MinLevel != 1 {
		t.Error("at least one crop must be available at level 1")
	}
}

// Every crop should pay off, and slower crops should never earn less per
// harvest than faster ones, otherwise nobody would plant them.
func TestEconomyIsSane(t *testing.T) {
	var prev Entry
	for i, e := range Default().Entries() {
		if e.Profit() <= 0 {
			t.Errorf("%s: profit %d, want positive", e.ID, e.Profit())
		}
		if i > 0 && e.TimeToRipe() > prev.TimeToRipe() && e.Profit() < prev.Profit() {
			t.Errorf("%s takes longer than %s but earns less (%d < %d)", e.ID, prev.ID, e.Profit(), prev.Profit())
		}
		prev = e
	}
}
