package calculator

import (
	"slices"
	"testing"
)

// TestStormshroudBreakdownMod: CalcDefence.lua:1572-1578 add Stormshroud's shock avoidance as AvoidShockAppliesToElementalAilments for the breakdown.
func TestStormshroudBreakdownMod(t *testing.T) {
	assertTablesMatch(t, "0116-reliquarian-blade-blast.xml", func(c modSectionCase) bool {
		return slices.Contains(c.ModName, "AvoidShockAppliesToElementalAilments")
	})
}
