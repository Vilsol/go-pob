package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestWeaponLocalElementalDamageScalesAddedRanges: Item.lua:2262-2265 scales a weapon's LOCAL added elemental ranges by `Local<type>Damage INC` plus a once-hoisted `LocalElementalDamage INC`.
//
//pob:block:src/Classes/Item.lua:2298
func TestWeaponLocalElementalDamageScalesAddedRanges(t *testing.T) {
	const claw = "Rarity: RARE\nTest Claw\nImperial Claw\nItem Level: 84\n"

	build := func(extra ...mod.Mod) map[string]float64 {
		item := ParseItem(claw)
		if item.Base == nil || item.Base.Weapon == nil {
			t.Fatalf("REFUSING: Imperial Claw resolved no weapon base (%v)", item.Base)
		}
		baseList := make([]mod.Mod, 0, 2+len(extra))
		baseList = append(baseList,
			mod.NewFloat("FireMin", mod.TypeBase, 10),
			mod.NewFloat("FireMax", mod.TypeBase, 20))
		baseList = append(baseList, extra...)
		item.WeaponData = map[int]map[string]float64{}
		item.buildModListForSlotNum(baseList, 1)
		return item.WeaponData[1]
	}

	t.Run("control: unscaled local fire damage", func(t *testing.T) {
		wd := build()
		if wd["FireMin"] != 10 || wd["FireMax"] != 20 {
			t.Fatalf("REFUSING: control FireMin/FireMax = %v/%v, want 10/20 — the fixture is "+
				"not delivering local added fire damage at all", wd["FireMin"], wd["FireMax"])
		}
	})

	t.Run("LocalElementalDamage scales it", func(t *testing.T) {
		wd := build(mod.NewFloat("LocalElementalDamage", mod.TypeIncrease, 50))
		if wd["FireMin"] != 15 || wd["FireMax"] != 30 {
			t.Errorf("FireMin/FireMax = %v/%v, want 15/30", wd["FireMin"], wd["FireMax"])
		}
	})

	t.Run("LocalFireDamage scales it and adds to the elemental term", func(t *testing.T) {
		wd := build(
			mod.NewFloat("LocalFireDamage", mod.TypeIncrease, 50),
			mod.NewFloat("LocalElementalDamage", mod.TypeIncrease, 50),
		)
		if wd["FireMin"] != 20 || wd["FireMax"] != 40 {
			t.Errorf("FireMin/FireMax = %v/%v, want 20/40 (the two INCs are ADDITIVE)",
				wd["FireMin"], wd["FireMax"])
		}
	})

	t.Run("one LocalElementalDamage scales EVERY elemental type", func(t *testing.T) {
		item := ParseItem(claw)
		item.WeaponData = map[int]map[string]float64{}
		item.buildModListForSlotNum([]mod.Mod{
			mod.NewFloat("LightningMin", mod.TypeBase, 10), mod.NewFloat("LightningMax", mod.TypeBase, 10),
			mod.NewFloat("ColdMin", mod.TypeBase, 10), mod.NewFloat("ColdMax", mod.TypeBase, 10),
			mod.NewFloat("FireMin", mod.TypeBase, 10), mod.NewFloat("FireMax", mod.TypeBase, 10),
			mod.NewFloat("LocalElementalDamage", mod.TypeIncrease, 100),
		}, 1)
		wd := item.WeaponData[1]
		for _, name := range []string{"LightningMin", "ColdMin", "FireMin"} {
			if wd[name] != 20 {
				t.Errorf("%s = %v, want 20 — LocalElementalDamage was drained before this type",
					name, wd[name])
			}
		}
	})

	t.Run("neither name escapes the item as a global", func(t *testing.T) {
		item := ParseItem(claw)
		item.WeaponData = map[int]map[string]float64{}
		global := item.buildModListForSlotNum([]mod.Mod{
			mod.NewFloat("FireMin", mod.TypeBase, 10),
			mod.NewFloat("FireMax", mod.TypeBase, 20),
			mod.NewFloat("LocalElementalDamage", mod.TypeIncrease, 50),
			mod.NewFloat("LocalFireDamage", mod.TypeIncrease, 50),
		}, 1)
		for _, name := range []string{"LocalElementalDamage", "LocalFireDamage"} {
			if n := countModsNamed(global, name); n != 0 {
				t.Errorf("%s survived into the global list %d times; calcLocal must drain it", name, n)
			}
		}
	})

	t.Run("the parser produces it (ModParser.lua:3150)", func(t *testing.T) {
		entry := ParseMod("Has 25% increased Elemental Damage", false)
		if entry == nil || entry.Extra != "" || len(entry.ModList) != 1 ||
			entry.ModList[0].Name() != "LocalElementalDamage" || entry.ModList[0].Value().Float() != 25 {
			t.Fatalf("ParseMod(%q) = %+v, want one LocalElementalDamage INC 25", "Has 25% increased Elemental Damage", entry)
		}
	})
}
