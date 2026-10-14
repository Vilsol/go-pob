package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/pob"
	"github.com/Vilsol/go-pob/utils"
)

// TestMaximumCritChanceLineCapsCritChance: ModParser.lua:5481-5483 parses the line to CritChanceCap OVERRIDE; CalcOffence.lua:3021 caps CritChance at it.
func TestMaximumCritChanceLineCapsCritChance(t *testing.T) {
	t.Parallel()
	build := parseCorpusBuild(t, "0026-ascendant-fire-trap-of-blasting.xml")
	if got := GetMiscCalculator(build).Base().Player.Output["PreEffectiveCritChance"]; got <= 50 {
		t.Fatalf("REFUSING: crit chance is %v without the line, so a cap of 50 cannot show", got)
	}
	build.SetConfigOption(pob.Input{Name: "customMods", String: utils.Ptr("Maximum Critical Strike Chance is 50%")})
	if got := GetMiscCalculator(build).Base().Player.Output["PreEffectiveCritChance"]; got != 50 {
		t.Fatalf("PreEffectiveCritChance %v, want 50", got)
	}
}
