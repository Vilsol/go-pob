package calculator

import (
	"testing"
)

// TestSeismicTrapOfSwellsOverlap: the transfigured Seismic Trap carries its own wave-overlap preDamageFunc (act_dex.lua:13619-13668).
func TestSeismicTrapOfSwellsOverlap(t *testing.T) {
	requireGameData(t, "3.29")
	env := corpusEnvForLoadedData(t, "1634-pathfinder-seismic-trap-of-swells.xml")
	for k, want := range map[string]float64{"OverlapChance": 37.345679012346, "SkillDPSMultiplier": 2.6141975308642} {
		if got := env.Player.Output[k]; !oracleClose(got, want, oracleDumpTolerance) {
			t.Errorf("%s = %v, want %v (PoB)", k, got, want)
		}
	}
}

// TestEquippedFlaskSetsHaveCondition: every equipped flask sets Have<BaseName> on the item modDB (CalcSetup.lua:927), which gates Tree:6038's crit chance.
func TestEquippedFlaskSetsHaveCondition(t *testing.T) {
	requireGameData(t, "3.29")
	env := corpusEnvForLoadedData(t, "1634-pathfinder-seismic-trap-of-swells.xml")
	for k, want := range map[string]float64{"CritChance": 68.5251, "TotalDPS": 10354067.932831} {
		if got := env.Player.Output[k]; !oracleClose(got, want, oracleDumpTolerance) {
			t.Errorf("%s = %v, want %v (PoB)", k, got, want)
		}
	}
}
