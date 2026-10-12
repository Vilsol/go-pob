package calculator

import (
	"strings"
	"testing"
)

// TestWitherAfterSelfCastCurse covers CalcPerform.lua:2704-2706 running INSIDE the per-skill buff loop.
func TestWitherAfterSelfCastCurse(t *testing.T) {
	guard := func(t *testing.T, xml string) {
		if !strings.Contains(xml, "on Hit if you&apos;ve cast Despair") {
			t.Fatalf("PREMISE BROKEN: build carries no `if you've cast <curse>` wither line")
		}
	}
	for _, name := range []string{
		"0596-assassin-viper-strike-of-the-mamba-6.xml",
		"1523-assassin-viper-strike-of-the-mamba-14.xml",
		"0937-assassin-viper-strike-of-the-mamba-13.xml",
	} {
		t.Run(name, func(t *testing.T) {
			assertBuildStats(t, name, []string{"PoisonDPS", "CombinedDPS"}, guard)
		})
	}
}

// TestCombustionEnemyFireResist covers CalcPerform.lua:2708-2720.
func TestCombustionEnemyFireResist(t *testing.T) {
	guard := func(t *testing.T, xml string) {
		if !strings.Contains(xml, "Combustion") {
			t.Fatalf("PREMISE BROKEN: build sockets no Combustion support")
		}
	}
	for _, name := range []string{
		"0196-slayer-whirling-blades.xml",
		"0863-elementalist-volatile-dead.xml",
	} {
		t.Run(name, func(t *testing.T) {
			assertBuildStats(t, name, []string{"IgniteDPS", "CombinedDPS"}, guard)
		})
	}
}
