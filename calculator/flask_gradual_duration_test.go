package calculator

import (
	"math"
	"os"
	"testing"
)

// TestFlaskGradualRecoveryIgnoresDuration pins Item.lua:2381 and :2389.
func TestFlaskGradualRecoveryIgnoresDuration(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	item := ParseItem(`Rarity: MAGIC
Experimenter's Divine Life Flask of Allaying
Item Level: 83
Quality: 20
LevelReq: 60
Implicits: 0
38% increased Duration
25% of Recovery applied Instantly`)
	if item == nil || item.FlaskData == nil {
		t.Fatal("fixture is useless: the flask did not parse")
	}

	base := item.FlaskData["lifeBase"]
	if base == 0 {
		t.Fatal("fixture is useless: lifeBase is 0, so no recovery split happened")
	}
	if item.FlaskData["duration"] == 0 {
		t.Fatal("fixture is useless: no duration")
	}

	instant, gradual := item.FlaskData["lifeInstant"], item.FlaskData["lifeGradual"]
	if math.Abs((instant+gradual)-base) > base*1e-9 {
		t.Errorf("lifeInstant %v + lifeGradual %v = %v, want lifeBase %v — Item.lua:2380/2381 split "+
			"the SAME pool, so the two halves must sum to it; a duration factor on the gradual half "+
			"inflates the total by %.1f%%",
			instant, gradual, instant+gradual, base, 100*((instant+gradual)/base-1))
	}
}
