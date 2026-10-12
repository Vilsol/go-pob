package calculator

import (
	"math"
	"testing"
)

// TestUtilityFlaskDurationCompoundsQuality covers Item.lua:2395.
func TestUtilityFlaskDurationCompoundsQuality(t *testing.T) {
	requireGameData(t, "3.29")

	raw := "Rarity: MAGIC\nExperimenter's Quicksilver Flask of Adrenaline\n" +
		"Quality: 20\nLevelReq: 4\nImplicits: 0\n" +
		"30% increased Duration\n"
	item := ParseItem(raw)
	if item.Base == nil || item.Base.Flask == nil {
		t.Fatalf("REFUSING: fixture resolved no flask base — the assertion below would read a " +
			"zero-valued map and pass for the wrong reason")
	}
	if item.FlaskData == nil {
		t.Fatalf("REFUSING: the fixture produced no flaskData at all")
	}

	base := item.Base.Flask.Duration
	if base <= 0 {
		t.Fatalf("REFUSING: the base flask duration is %v; the fixture has no work", base)
	}

	want := math.Round(base*1.30*1.20*10) / 10
	wrong := math.Round(base*(1+(30+20)/100.0)*10) / 10
	if want == wrong {
		t.Fatalf("REFUSING: the two formulas agree on this fixture (%v), so it cannot fail", want)
	}

	if got := item.FlaskData["duration"]; got != want {
		t.Errorf("utility flask duration = %v, want %v (PoB compounds the two factors; the "+
			"summed form gives %v)", got, want, wrong)
	}
}
