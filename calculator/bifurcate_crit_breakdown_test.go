package calculator

import (
	"slices"
	"testing"
)

// TestBifurcatedCritDamageBonusBreakdownMod: CalcOffence.lua:3168-3169 add a descriptive CritMultiplier MORE mod for the bifurcated crit bonus.
func TestBifurcatedCritDamageBonusBreakdownMod(t *testing.T) {
	assertTablesMatch(t, "0060-assassin-icicle-mine.xml", func(c modSectionCase) bool {
		return slices.Equal(c.ModName, []string{"CritMultiplier"})
	})
}
