package calculator

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/utils"
)

// TestWeaponDataDropIsCounted: A weaponData key that resolves onto no SkillData field must be COUNTED, not skipped.
func TestWeaponDataDropIsCounted(t *testing.T) {
	const key = "noSuchSkillDataFieldExists"

	before := droppedSetCount(key)

	newWeaponSkillData(data.ItemClassName("Two Handed Axe"), map[string]float64{key: 1})

	testza.AssertTrue(t, droppedSetCount(key) > before,
		"newWeaponSkillData discarded weaponData[%q] without recording it. utils.DroppedKeys "+
			"is the only instrument that can see a lost calculation input, so a skip that "+
			"bypasses it is a blind spot in droppedKeyDistinctCeiling.", key)
}

// TestWeaponDataKnownKeyIsNotCounted: A key that DOES resolve must still be written, and must not be counted as dropped.
func TestWeaponDataKnownKeyIsNotCounted(t *testing.T) {
	const key = "countsAsDualWielding"

	before := droppedSetCount(key)

	sd := newWeaponSkillData(data.ItemClassName("Two Handed Axe"), map[string]float64{key: 1})

	testza.AssertTrue(t, sd.CountsAsDualWielding,
		"weaponData[%q] (ModParser.lua:2044) did not reach SkillData.CountsAsDualWielding, "+
			"which CalcSetup.lua:879-880 reads to synthesise the off-hand", key)
	testza.AssertEqual(t, before, droppedSetCount(key),
		"weaponData[%q] resolved onto a field and must not be counted as dropped", key)
}

func droppedSetCount(key string) int {
	_, counts := utils.DroppedKeys()
	for _, c := range counts {
		if c.Op == "set" && c.Key == key {
			return c.Count
		}
	}
	return 0
}
