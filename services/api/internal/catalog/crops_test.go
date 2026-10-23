package catalog

import "testing"

func TestDefaultCatalogIsConsistent(t *testing.T) {
	c := Default()

	if len(c.Crops()) == 0 {
		t.Fatal("catalog is empty")
	}

	seen := map[string]bool{}
	for _, crop := range c.Crops() {
		if seen[string(crop.ID)] {
			t.Errorf("duplicate crop %q", crop.ID)
		}
		seen[string(crop.ID)] = true

		if crop.Yield <= 0 {
			t.Errorf("%s: yield must be positive", crop.ID)
		}
		for i, d := range crop.StageDurations {
			if d <= 0 {
				t.Errorf("%s: stage %d has non-positive duration", crop.ID, i)
			}
		}

		got, ok := c.Crop(crop.ID)
		if !ok || got.ID != crop.ID {
			t.Errorf("Crop(%q) lookup failed", crop.ID)
		}
	}

	if _, ok := c.Crop("mandrake"); ok {
		t.Error("unknown crop should not be found")
	}
}
