package calculator

import (
	"os"
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/mod"
)

// TestBaseModConstantsMatchPoB covers CalcSetup.lua:517 and :535-536.
func TestBaseModConstantsMatchPoB(t *testing.T) {
	testza.AssertEqual(t, float64(30), float64(data.MaximumRage), "Misc.lua:118")
	testza.AssertEqual(t, float64(20), float64(data.InherentBlockWhileDualWieldingPercent), "Misc.lua:87")

	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true for the rest")
	}
	requireGameData(t, "3.29")

	env := corpusEnvForLoadedData(t, "0016-gladiator-dual-strike-of-ambidexterity.xml")

	var sawRage, sawBlock, sawDoubledBlock, sawDoubledSpeed bool
	for _, m := range env.Player.ModDB.Mods["MaximumRage"] {
		if m.GetSource() == mod.SourceBase {
			sawRage = true
			testza.AssertEqual(t, float64(data.MaximumRage), m.Value().Float(),
				"the Base MaximumRage mod (CalcSetup.lua:517)")
		}
	}
	for _, m := range env.Player.ModDB.Mods["BlockChance"] {
		if m.GetSource() != mod.SourceBase || m.Type() != mod.TypeBase {
			continue
		}
		switch m.Value().Float() {
		case float64(data.InherentBlockWhileDualWieldingPercent):
			sawBlock = true
		case float64(2 * data.InherentBlockWhileDualWieldingPercent):
			sawDoubledBlock = true
		}
	}
	for _, m := range env.Player.ModDB.Mods["Speed"] {
		if m.GetSource() == mod.SourceBase && m.Type() == mod.TypeMore &&
			m.Value().Float() == float64(2*data.DualWieldInherentAttackSpeedPlusPercentFinal) {
			sawDoubledSpeed = true
		}
	}

	if !sawRage {
		t.Error("REFUSING: no Base MaximumRage mod; the producer moved and this test measures nothing")
	}
	if !sawBlock {
		t.Errorf("no Base BlockChance mod at %v (CalcSetup.lua:535)", data.InherentBlockWhileDualWieldingPercent)
	}
	if !sawDoubledBlock {
		t.Errorf("no Base BlockChance mod at %v for DoubledInherentDualWieldingBlock (CalcSetup.lua:536)",
			2*data.InherentBlockWhileDualWieldingPercent)
	}
	if !sawDoubledSpeed {
		t.Errorf("no Base MORE Speed mod at %v for DoubledInherentDualWieldingSpeed (CalcSetup.lua:534)",
			2*data.DualWieldInherentAttackSpeedPlusPercentFinal)
	}
}
