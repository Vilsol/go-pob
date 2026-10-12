package calculator

import (
	"math"
	"os"
	"testing"
)

// TestTinctureRangedWeapons pins CalcPerform.lua:1882-1892.
func TestTinctureRangedWeapons(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	for _, c := range []struct {
		build string
		want  map[string]float64
	}{
		{"1188-warden-void-shot.xml", map[string]float64{"CritChance": 94.8024, "TotalDPS": 1789348.5041803}},
		{"0566-warden-ice-shot-of-penetration.xml", map[string]float64{"CritChance": 87.489792, "TotalDPS": 2248239.8398684}},
	} {
		env := corpusEnvForLoadedData(t, c.build)
		if !env.ModDB.Flag(nil, "TinctureRangedWeapons") {
			t.Fatalf("%s: TinctureRangedWeapons absent — fixture no longer reaches :1882", c.build)
		}
		for key, want := range c.want {
			if got := env.Player.Output[key]; math.Abs(got-want) > want*1e-6 {
				t.Errorf("%s %s = %v, PoB has %v", c.build, key, got, want)
			}
		}
	}
}
