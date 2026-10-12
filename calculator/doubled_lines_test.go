package calculator

import (
	"math"
	"os"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestDoubledLinesEmitBothMods pins two more "is Doubled" lines, in the shape PoB's generic DOUBLED FORM produces (ModParser.lua:6791-6806).
func TestDoubledLinesEmitBothMods(t *testing.T) {
	for _, tc := range []struct {
		line string
		name string
	}{
		{"maximum recovery per energy shield leech is doubled", "MaxEnergyShieldLeechInstance"},
		{"maximum number of summoned phantasms is doubled", "ActivePhantasmLimit"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			entry := ParseMod(tc.line, false)
			if entry == nil || len(entry.ModList) != 2 {
				got := 0
				if entry != nil {
					got = len(entry.ModList)
				}
				t.Fatalf("%q: got %d mods, want 2 (%s MORE 100 + Multiplier:%sDoubled OVERRIDE 1)",
					tc.line, got, tc.name, tc.name)
			}
			first := entry.ModList[0]
			if first.Name() != tc.name || first.Type() != mod.TypeMore {
				t.Errorf("first mod is %s/%s, want %s/MORE", first.Name(), first.Type(), tc.name)
			}
			hasLimit := false
			for _, tag := range first.Tags() {
				mt, ok := tag.(*mod.MultiplierTag)
				if ok && mt.TagGlobalLimitKey != nil && *mt.TagGlobalLimitKey == tc.name+"DoubledLimit" {
					hasLimit = true
				}
			}
			if !hasLimit {
				t.Errorf("%q: the MORE mod carries no Multiplier tag with globalLimitKey %q",
					tc.line, tc.name+"DoubledLimit")
			}
			if second := entry.ModList[1]; second.Name() != "Multiplier:"+tc.name+"Doubled" ||
				second.Type() != mod.TypeOverride {
				t.Errorf("second mod is %s/%s, want Multiplier:%sDoubled/OVERRIDE",
					second.Name(), second.Type(), tc.name)
			}
		})
	}
}

// TestEnergyShieldLeechDoublingReachesTheBuild is the end-to-end half.
func TestEnergyShieldLeechDoublingReachesTheBuild(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	env := corpusEnvForLoadedData(t, "0161-assassin-viper-strike-of-the-mamba.xml")
	if got, want := env.Player.Output["MaxEnergyShieldLeechInstance"], 784.4; math.Abs(got-want) > want*0.005 {
		t.Errorf("MaxEnergyShieldLeechInstance = %v, want %v (PoB) — ours was exactly half", got, want)
	}

	ctrl := corpusEnvForLoadedData(t, "0004-trickster-lightning-tendrils-of-eccentricity.xml")
	if ctrl.Player.ModDB.Flag(nil, "Multiplier:MaxEnergyShieldLeechInstanceDoubled") {
		t.Errorf("CONTROL: the doubling multiplier is published on a build without the line")
	}
}
