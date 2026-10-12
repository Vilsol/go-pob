package calculator

import (
	"fmt"
	"testing"

	"github.com/MarvinJWendt/testza"
)

func TestGainNoDefenceFromSlotFlagNames(t *testing.T) {
	for _, c := range []struct{ line, want string }{
		{"Gain no Armour from Equipped Body Armour", "GainNoArmourFromBody Armour"},
		{"Gain no Energy Shield from Equipped Body Armour", "GainNoEnergyShieldFromBody Armour"},
		{"Gain no Evasion from Equipped Shield", "GainNoEvasionFromWeapon 2"},
		{"Gain no Ward from Equipped Helmet", "GainNoWardFromHelmet"},
	} {
		t.Run(c.line, func(t *testing.T) {
			e := ParseMod(c.line, false)
			if e.Extra != "" {
				t.Fatalf("%q left Extra=%q, so the line would be DROPPED by processModLine", c.line, e.Extra)
			}
			testza.AssertEqual(t, 1, len(e.ModList), "expected exactly one flag mod")
			if len(e.ModList) == 1 {
				testza.AssertEqual(t, c.want, e.ModList[0].Name())
				testza.AssertTrue(t, e.ModList[0].Value().Flag(), "flag must be true")
			}
		})
	}

	e := ParseMod("Gain no inherent bonuses from strength", false)
	testza.AssertEqual(t, 1, len(e.ModList))
	if len(e.ModList) == 1 {
		testza.AssertEqual(t, "NoStrengthAttributeBonuses", e.ModList[0].Name())
	}
}

var gainNoCorpusBuilds = []string{
	"0031-guardian-raise-spectre.xml",
	"0365-luminary-vaal-cold-snap.xml",
	"0548-guardian-raise-spectre-2.xml",
	"0549-guardian-tempest-shield.xml",
	"0583-guardian-summon-sentinel-of-radiance.xml",
	"0904-reliquarian-vaal-arc-7.xml",
	"0908-guardian-summon-holy-relic.xml",
	"0920-inquisitor-dominating-blow-of-inspiring.xml",
	"1064-guardian-dominating-blow-of-inspiring-4.xml",
	"1411-guardian-dominating-blow-of-inspiring-6.xml",
	"1929-ladder-arminyou_4843.Dominating_Blow_Job__.xml",
}

// TestGainNoArmourFromBodyArmourIsInertOnThisCorpus pins the census that explains the measured zero.
func TestGainNoArmourFromBodyArmourIsInertOnThisCorpus(t *testing.T) {
	requireGameData(t, "3.29")

	flagged, binding := 0, []string{}
	for _, b := range gainNoCorpusBuilds {
		env := corpusEnvForLoadedData(t, b)
		if !env.Player.ModDB.Flag(nil, "GainNoArmourFromBody Armour") {
			continue
		}
		flagged++
		if it := env.Player.ItemList["Body Armour"]; it != nil && it.ArmourData != nil && it.ArmourData.Armour > 0 {
			binding = append(binding, fmt.Sprintf("%s (raw Armour %v)", b, it.ArmourData.Armour))
		}
		testza.AssertEqual(t, float64(0), env.Player.Output["ArmourOnBody Armour"],
			"%s sets GainNoArmourFromBody Armour, so the slot must publish no armour", b)
	}

	if flagged == 0 {
		t.Fatalf("REFUSING: not one of the %d builds carrying the mod set the flag. "+
			"The ModParser pattern (ModParser.lua:2370) has regressed and the CalcDefence.lua:849-852 "+
			"guards are unreachable again", len(gainNoCorpusBuilds))
	}
	if len(binding) > 0 {
		t.Fatalf("the guard is no longer inert -- %d build(s) now equip a body armour with nonzero "+
			"raw Armour while the flag is set: %v.\nThe MEASURED ZERO recorded for this change no "+
			"longer holds; assert these against the oracle instead of this census", len(binding), binding)
	}
	t.Logf("census: %d of %d builds set the flag, 0 binding (every one equips a body armour with 0 raw Armour)",
		flagged, len(gainNoCorpusBuilds))
}
