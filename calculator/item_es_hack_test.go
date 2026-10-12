package calculator

import "testing"

// TestItemEnergyShieldHackZeroesIntRequirement: Item.lua:2561-2565 zeroes BOTH the energy shield and the base int requirement on four named uniques.
//
//pob:block:src/Classes/Item.lua:2561-2565
func TestItemEnergyShieldHackZeroesIntRequirement(t *testing.T) {
	cases := []struct {
		name string
		raw  string
	}{
		{"The Apostate", "Rarity: UNIQUE\nThe Apostate\nCabalist Regalia\nItem Level: 84\n"},
		{"Tabula Rasa", "Rarity: UNIQUE\nTabula Rasa\nSimple Robe\nItem Level: 84\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			item := ParseItem(c.raw)
			if item.Base == nil {
				t.Fatalf("REFUSING: %q resolved no base", c.name)
			}
			if item.Requirements["int"] != 0 {
				t.Errorf("requirements.int = %v, want 0", item.Requirements["int"])
			}
			if item.Requirements["intMod"] != 0 {
				t.Errorf("requirements.intMod = %v, want 0", item.Requirements["intMod"])
			}
			if item.ArmourData["EnergyShield"] != 0 {
				t.Errorf("armourData.EnergyShield = %v, want 0", item.ArmourData["EnergyShield"])
			}
		})
	}

	t.Run("an unlisted Cabalist Regalia keeps its int requirement", func(t *testing.T) {
		item := ParseItem("Rarity: RARE\nPlain Vestment\nCabalist Regalia\nItem Level: 84\n")
		if item.Requirements["int"] == 0 {
			t.Fatalf("REFUSING: the control base has no int requirement, so the cases above " +
				"prove nothing about the hack")
		}
	})
}
