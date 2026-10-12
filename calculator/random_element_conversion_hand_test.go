package calculator

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/mod"
)

const randomElementHandFloor = 25

// TestRandomElementConversionKeepsItsHand pins a BOTH-ENDS-ABSENT pair that neither half could demonstrate alone.
func TestRandomElementConversionKeepsItsHand(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	corpusDir, names := corpusNamesForLoadedData(t)

	var carriers, withHandedConversion int
	for _, name := range names {
		raw, err := os.ReadFile(corpusDir + "/" + name)
		if err != nil {
			continue
		}
		if !strings.Contains(string(raw), "is Converted to a random Element") {
			continue
		}
		carriers++

		env := corpusEnvForLoadedData(t, name)
		if env == nil || env.Player == nil || env.Player.MainSkill == nil || env.Player.MainSkill.SkillModList == nil {
			continue
		}
		if handConditioned(env.Player.MainSkill.SkillModList.Mods()) {
			withHandedConversion++
		}
	}

	if carriers == 0 {
		t.Fatalf("REFUSING: no corpus build carries the weapon-local random-element conversion")
	}
	fmt.Printf("=== RANDOM-ELEMENT CONVERSION === %d of %d carriers keep a hand-conditioned derived mod\n",
		withHandedConversion, carriers)

	if withHandedConversion < randomElementHandFloor {
		t.Errorf("hand-conditioned derived conversions: %d of %d carriers, floor %d. Either "+
			"ModParser.lua:3482's {Hand}Attack condition has been dropped from the parser row, or "+
			"CalcOffence.lua:947's weapon1Cfg/weapon2Cfg passes have been dropped from the "+
			"tabulate loop — the condition is only ever true under those two cfgs",
			withHandedConversion, carriers, randomElementHandFloor)
	}
}

func handConditioned(mods []mod.Mod) bool {
	for _, m := range mods {
		switch m.Name() {
		case "PhysicalDamageConvertToFire", "PhysicalDamageConvertToCold", "PhysicalDamageConvertToLightning":
		default:
			continue
		}
		for _, tag := range m.Tags() {
			cond, ok := tag.(*mod.ConditionTag)
			if !ok {
				continue
			}
			for _, v := range cond.VarList {
				if v == "MainHandAttack" || v == "OffHandAttack" {
					return true
				}
			}
		}
	}
	return false
}

// TestRandomElementConversionParsesWithItsHand is the PARSER half of the pair pinned above.
func TestRandomElementConversionParsesWithItsHand(t *testing.T) {
	t.Run("weapon-local random-element conversion is gated to that hand", func(t *testing.T) {
		mods, extra := parseMod("100% of Physical Damage from Hits with this Weapon is Converted to a random Element", 1)
		testza.AssertEqual(t, "", extra)
		if len(mods) != 1 {
			t.Fatalf("want 1 mod, got %d", len(mods))
		}
		testza.AssertEqual(t, "PhysicalDamageConvertToRandom", mods[0].Name())
		tags := mods[0].Tags()
		if len(tags) != 1 {
			t.Fatalf("want exactly the {Hand}Attack condition, got %d tags: %+v", len(tags), tags)
		}
		cond, ok := tags[0].(*mod.ConditionTag)
		if !ok {
			t.Fatalf("want a ConditionTag, got %T", tags[0])
		}
		testza.AssertTrue(t, len(cond.VarList) == 1 && cond.VarList[0] == "{Hand}Attack", "condition must be {Hand}Attack, got %+v", cond.VarList)

		global, extra := parseMod("40% of Physical Damage Converted to a random Element", 1)
		testza.AssertEqual(t, "", extra)
		if len(global) != 1 {
			t.Fatalf("control: want 1 mod, got %d", len(global))
		}
		testza.AssertEqual(t, 0, len(global[0].Tags()), "the GLOBAL form must stay untagged")
	})
}
