package calculator

import (
	"slices"
	"testing"
)

// TestSpectreAuraModsCarryTheSpectreSource: CalcPerform.lua:2724-2736 source a spectre's aura mods "Spectre:<aura> - <spectre>" (2814, 2829, 2839, 2872).
func TestSpectreAuraModsCarryTheSpectreSource(t *testing.T) {
	assertTablesMatch(t, "0031-guardian-raise-spectre.xml", func(c modSectionCase) bool {
		return slices.Equal(c.ModName, []string{"LightningMin"}) || slices.Equal(c.ModName, []string{"LightningMax"})
	})
}
