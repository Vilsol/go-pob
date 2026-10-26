package calculator2

import (
	"strconv"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// actor.weaponData<i> is the item's weaponData[i] here, as CalcSetup sets it for an equipped weapon.
//
//pob2:src/Modules/CalcPerform.lua:161-180
func addWeaponBaseStats(actor *Actor) {
	output := actor.Output
	dmgTypeList := []string{"Physical", "Lightning", "Cold", "Fire", "Chaos"}
	for i := 1; i <= 2; i++ {
		slot := "Weapon " + strconv.Itoa(i)
		// Section for martial weapons only for now
		item := actor.ItemList[slot]
		if item != nil && item.Base != nil && item.Base.Weapon != nil {
			weaponData := item.WeaponData[i]
			// Add base min and max damage
			for _, damageType := range dmgTypeList {
				if v, ok := weaponData[damageType+"Min"]; ok {
					output[damageType+"Min"+"OnWeapon "+strconv.Itoa(i)] = v
				}
				if v, ok := weaponData[damageType+"Max"]; ok {
					output[damageType+"Max"+"OnWeapon "+strconv.Itoa(i)] = v
				}
			}
			// Add total local accuracy
			baseModList := moddb.NewModList()
			for _, m := range item.BaseModList {
				baseModList.AddMod(m)
			}
			output["AccuracyOnWeapon "+strconv.Itoa(i)] = baseModList.Sum(mod.TypeBase, nil, "Accuracy")
		}
	}
}
