package calculator

import (
	"os"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestInfluenceItemThresholdParses pins ModParser.lua:1372.
func TestInfluenceItemThresholdParses(t *testing.T) {
	for _, tc := range []struct {
		line      string
		name      string
		modType   mod.Type
		value     float64
		variable  string
		threshold float64
	}{
		{"15% increased maximum Life if 2 Elder Items are Equipped", "Life", mod.TypeIncrease, 15, "ElderItem", 2},
		{"13% increased Strength if 2 Warlord Items are Equipped", "Str", mod.TypeIncrease, 13, "WarlordItem", 2},
	} {
		t.Run(tc.line, func(t *testing.T) {
			e := ParseMod(tc.line, false)
			if e == nil || len(e.ModList) != 1 {
				t.Fatalf("ParseMod(%q) produced %v mods, want 1", tc.line, e)
			}
			m := e.ModList[0]
			if m.Name() != tc.name || m.Type() != tc.modType || m.Value().Float() != tc.value {
				t.Fatalf("got %s %v %v, want %s %v %v", m.Name(), m.Type(), m.Value().Float(), tc.name, tc.modType, tc.value)
			}
			tags := m.Tags()
			if len(tags) != 1 {
				t.Fatalf("got %d tags, want 1 (the MultiplierThreshold)", len(tags))
			}
			mt, ok := tags[0].(*mod.MultiplierThresholdTag)
			if !ok {
				t.Fatalf("tag is %T, want *mod.MultiplierThresholdTag", tags[0])
			}
			if mt.Variable != tc.variable {
				t.Errorf("threshold variable = %q, want %q", mt.Variable, tc.variable)
			}
			if mt.TagThreshold == nil || *mt.TagThreshold != tc.threshold {
				t.Errorf("threshold = %v, want %v", mt.TagThreshold, tc.threshold)
			}
		})
	}
}

// TestInfluenceItemCountsAreCounted pins CalcSetup.lua:1190-1205.
func TestInfluenceItemCountsAreCounted(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	for _, tc := range []struct {
		build string
		life  float64
		why   string
	}{
		{"2027-ladder-ninorly_5573.Lumininy.xml", 18921,
			"Bound By Destiny's '15% increased maximum Life if 2 Elder Items are Equipped' was dropped whole"},

		{"0004-trickster-lightning-tendrils-of-eccentricity.xml", 3666,
			"CONTROL: no influence-threshold mod; must not move"},
	} {
		t.Run(tc.build, func(t *testing.T) {
			env := corpusEnvForLoadedData(t, tc.build)
			got, ok := env.Player.Output["Life"]
			if !ok {
				t.Fatal("Life ABSENT")
			}
			if got != tc.life {
				t.Errorf("Life = %v, want %v — %s", got, tc.life, tc.why)
			}
		})
	}
}
