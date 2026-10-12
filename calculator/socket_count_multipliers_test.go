package calculator

import (
	"os"
	"testing"
)

// TestSocketCountMultipliersMatchPoB pins CalcSetup.lua:1206-1252, the nine socket-count multipliers nothing in this tree wrote.
func TestSocketCountMultipliersMatchPoB(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	cases := map[string]map[string]float64{
		"0006-elementalist-storm-burst": {
			"SocketedGemsInBody Armour":     6,
			"SocketedRedGemsInBody Armour":  2,
			"SocketedBlueGemsInBody Armour": 4,
			"SocketedGemsInGloves":          3,
			"SocketedGreenGemsInGloves":     1,
			"EmptySocketInGloves":           1,
			"EmptyWhiteSocketsInAnySlot":    1,
			"EmptyRedSocketsInAnySlot":      0,
		},
		"0009-trickster-cobra-lash": {
			"SocketedGemsInHelmet":           3,
			"SocketedRedGemsInHelmet":        2,
			"SocketedGreenGemsInBody Armour": 3,
			"SocketedGemsInWeapon 1":         3,
			"EmptySocketInHelmet":            1,
			"EmptyRedSocketsInAnySlot":       1,
			"EmptyWhiteSocketsInAnySlot":     0,
			"SocketedBlueGemsInRing 2":       1,
		},
		"0012-occultist-unknown": {
			"SocketedGemsInBody Armour":  6,
			"SocketedGemsInGloves":       4,
			"SocketedBlueGemsInGloves":   3,
			"EmptySocketInBoots":         4,
			"EmptySocketInWeapon 1":      3,
			"EmptyRedSocketsInAnySlot":   5,
			"EmptyBlueSocketsInAnySlot":  1,
			"EmptyWhiteSocketsInAnySlot": 1,
		},
	}

	for build, want := range cases {
		t.Run(build, func(t *testing.T) {
			env := corpusEnvForLoadedData(t, build)
			if env.ItemModDB == nil {
				t.Fatalf("REFUSING: %s built no ItemModDB", build)
			}
			nonzero := 0
			for _, v := range want {
				if v != 0 {
					nonzero++
				}
			}
			if nonzero == 0 {
				t.Fatalf("REFUSING: every expectation for %s is zero, so this fixture cannot "+
					"separate a written producer from an absent one", build)
			}

			for key, w := range want {
				if got := env.ItemModDB.Multipliers[key]; got != w {
					t.Errorf("%s: %s = %v, want %v (PoB's own dump)", build, key, got, w)
				}
			}
		})
	}
}
