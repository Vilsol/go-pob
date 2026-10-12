package calculator

import (
	"os"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestBonusesGainedFromEquippedSlotParses pins ModParser.lua:4588-4589.
func TestBonusesGainedFromEquippedSlotParses(t *testing.T) {
	for _, tc := range []struct {
		line string
		name string
		want float64
	}{
		{"20% increased bonuses gained from Equipped Gloves", "EffectOfBonusesFromGloves", 20},
		{"20% reduced bonuses gained from Equipped Boots", "EffectOfBonusesFromBoots", -20},
	} {
		t.Run(tc.line, func(t *testing.T) {
			e := ParseMod(tc.line, false)
			if e == nil || len(e.ModList) != 1 {
				t.Fatalf("ParseMod(%q) produced %v mods, want 1 — an unparsed remainder discards the WHOLE line", tc.line, e)
			}
			m := e.ModList[0]
			if m.Name() != tc.name || m.Type() != mod.TypeIncrease || m.Value().Float() != tc.want {
				t.Errorf("got %s %v %v, want %s INC %v", m.Name(), m.Type(), m.Value().Float(), tc.name, tc.want)
			}
		})
	}
}

// TestRunegraftOfTheBoundScalesGloveMods pins CalcSetup.lua:1142-1165 end to end.
func TestRunegraftOfTheBoundScalesGloveMods(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	for _, tc := range []struct {
		build string
		life  float64
		why   string
	}{
		{"1897-ladder-Stonesucer_1706.ProPewWinterBee.xml", 3244,
			"Runegraft of the Bound: gloves Life 95 -> 114, and 3204 -> PoB's 3244"},

		{"0230-warden-spectral-throw-of-materialising.xml", 4897,
			"Runegraft of the Bound, NEGATIVE arm: boots Life 26 -> 21"},

		{"0004-trickster-lightning-tendrils-of-eccentricity.xml", 3666,
			"CONTROL: gloves equipped, no runegraft; PoB also says 3666, so this is correct in BOTH states"},
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
