package calculator

import "testing"

func TestPinnedLife(t *testing.T) {
	assertDivergesFromOracle(t, "b0001.xml", "Life", 4321.5, "life is low")
}

func TestNotPinned(t *testing.T) {
	assertMatchesOracle(t, "b0004.xml", "Mana")
}

func TestPinnedOthers(t *testing.T) {
	assertDivergesFromOracleDump(t, "b0003.xml", "EnergyShield", -7.5e+02, "es is high")
	assertDivergesFromOracle(t, "b0002.xml", "Armour", 99.5, "armour")
}

// ⚠️ FOUR PINNED FUNCTIONS, NOT TWO, BECAUSE THE -run REGEX ORDER IS A MAP-ORDER CONTROL AND A
// MAP-ORDER CONTROL IS FLAKY. With two names the two implementations coincide half the time, so a
// mutation dropping the sort passed as often as it failed; with four the chance of agreement by
// luck is one in twenty-four. the rule for these: run repeatedly and print the count.
func TestPinnedEvasion(t *testing.T) {
	assertDivergesFromOracle(t, "b0005.xml", "Evasion", 12.5, "evasion")
}

func TestPinnedWard(t *testing.T) {
	assertDivergesFromOracle(t, "b0006.xml", "Ward", 7, "ward")
}
