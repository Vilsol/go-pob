package calculator

import (
	"os"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestFrenzySpeedIsSplitByAttackAndCast covers CalcSetup.lua:512-513.
func TestFrenzySpeedIsSplitByAttackAndCast(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	env := corpusEnvForLoadedData(t, "0016-gladiator-dual-strike-of-ambidexterity.xml")

	var attack, cast, unflagged int
	for _, m := range env.Player.ModDB.Mods["Speed"] {
		if m.GetSource() != mod.SourceBase || m.Type() != mod.TypeIncrease {
			continue
		}
		var isFrenzy bool
		for _, tg := range m.Tags() {
			if mt, ok := tg.(*mod.MultiplierTag); ok {
				for _, v := range mt.VariableList {
					if v == "FrenzyCharge" {
						isFrenzy = true
					}
				}
			}
		}
		if !isFrenzy {
			continue
		}
		switch {
		case m.Flags()&mod.MFlagAttack != 0:
			attack++
		case m.Flags()&mod.MFlagCast != 0:
			cast++
		default:
			unflagged++
		}
	}

	if attack+cast+unflagged == 0 {
		t.Fatal("REFUSING: no Base INC Speed mod tagged with the FrenzyCharge multiplier; the " +
			"producer moved and this test measures nothing")
	}
	if unflagged != 0 {
		t.Errorf("%d Base per-frenzy-charge Speed mod(s) carry NO weapon flag; PoB splits this into "+
			"an Attack mod and a Cast mod so it cannot reach an ailment dotCfg "+
			"(CalcSetup.lua:512-513)", unflagged)
	}
	if attack != 1 || cast != 1 {
		t.Errorf("Base per-frenzy-charge Speed mods: %d Attack, %d Cast; want 1 and 1 "+
			"(CalcSetup.lua:512-513)", attack, cast)
	}
}
