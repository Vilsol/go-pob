package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/utils"
)

// TestWeaponLocalCritChanceBase: Item.lua:2277 adds a LOCAL `CritChance BASE` to the weapon's base crit before the INC is applied.
//
//pob:block:src/Classes/Item.lua:2277
func TestWeaponLocalCritChanceBase(t *testing.T) {
	const claw = "Rarity: RARE\nTest Claw\nImperial Claw\nItem Level: 84\n"

	build := func(extra ...mod.Mod) (map[string]float64, []mod.Mod) {
		item := ParseItem(claw)
		if item.Base == nil || item.Base.Weapon == nil || item.Base.Weapon.CritChanceBase == 0 {
			t.Fatalf("REFUSING: Imperial Claw resolved no weapon base with a crit chance (%v)", item.Base)
		}
		item.WeaponData = map[int]map[string]float64{}
		global := item.buildModListForSlotNum(append([]mod.Mod{}, extra...), 1)
		return item.WeaponData[1], global
	}

	baseWD, _ := build()
	if baseWD["CritChance"] <= 0 {
		t.Fatalf("REFUSING: control CritChance = %v, want > 0", baseWD["CritChance"])
	}

	t.Run("a local CritChance BASE raises the weapon's crit", func(t *testing.T) {
		wd, _ := build(mod.NewFloat("CritChance", mod.TypeBase, 1))
		want := utils.RoundTo(baseWD["CritChance"]+1, 2)
		if wd["CritChance"] != want {
			t.Errorf("CritChance = %v, want %v (base %v + 1)", wd["CritChance"], want, baseWD["CritChance"])
		}
	})

	t.Run("BASE is added before INC multiplies", func(t *testing.T) {
		wd, _ := build(
			mod.NewFloat("CritChance", mod.TypeBase, 1),
			mod.NewFloat("CritChance", mod.TypeIncrease, 100),
		)
		want := utils.RoundTo((baseWD["CritChance"]+1)*2, 2)
		if wd["CritChance"] != want {
			t.Errorf("CritChance = %v, want %v — the BASE must be inside the INC's parentheses",
				wd["CritChance"], want)
		}
	})

	t.Run("the local mod does not escape as a global", func(t *testing.T) {
		_, global := build(mod.NewFloat("CritChance", mod.TypeBase, 1))
		if n := countModsNamed(global, "CritChance"); n != 0 {
			t.Errorf("CritChance survived into the global list %d times; calcLocal must drain "+
				"it, or a weapon affix raises crit for spells and for the other hand", n)
		}
	})
}
