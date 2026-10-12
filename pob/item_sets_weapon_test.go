package pob

import "testing"

func weaponSetBuild(main int, slots ...string) *PathOfBuilding {
	b := &PathOfBuilding{Items: Items{ActiveItemSet: 1, ItemSets: []ItemSet{{ID: "1"}}}}
	b.Build.MainSocketGroup = main
	groups := make([]Skill, len(slots))
	for i, s := range slots {
		groups[i] = Skill{Slot: s}
	}
	b.Skills = Skills{ActiveSkillSet: 1, SkillSets: []SkillSet{{ID: 1, Skills: groups}}}
	return b
}

func second(b *PathOfBuilding) bool {
	s := b.Items.ItemSets[0].UseSecondWeaponSet
	return s != nil && *s
}

// TestSetUseSecondWeaponSetMovesTheMainGroup: ItemsTab.lua:244-258: II on a main group socketed in Weapon 1 moves it to the first group in a set-2 weapon slot.
func TestSetUseSecondWeaponSetMovesTheMainGroup(t *testing.T) {
	b := weaponSetBuild(2, "Helmet", "Weapon 1", "Body Armour", "Weapon 2 Swap Abyssal Socket 1", "Weapon 1 Swap")
	if !b.SetUseSecondWeaponSet(true) || !second(b) {
		t.Fatal("II did not switch the set to its second weapon set")
	}
	if b.Build.MainSocketGroup != 4 {
		t.Fatalf("mainSocketGroup = %d, want 4 (the first group whose slot has weaponSet 2)", b.Build.MainSocketGroup)
	}
	if b.Items.UseSecondWeaponSet == nil || !*b.Items.UseSecondWeaponSet {
		t.Fatal("<Items useSecondWeaponSet> does not follow the active set (ItemsTab.lua:1325)")
	}
	if !b.SetUseSecondWeaponSet(false) || second(b) || b.Build.MainSocketGroup != 2 {
		t.Fatalf("I: second=%v main=%d, want false / 2", second(b), b.Build.MainSocketGroup)
	}
}

// TestSetUseSecondWeaponSetKeepsANonWeaponMainGroup: The main group moves only when its own slot is in the weapon set being left.
func TestSetUseSecondWeaponSetKeepsANonWeaponMainGroup(t *testing.T) {
	b := weaponSetBuild(1, "Helmet", "Weapon 1 Swap")
	b.SetUseSecondWeaponSet(true)
	if b.Build.MainSocketGroup != 1 {
		t.Fatalf("mainSocketGroup = %d, want 1 (a Helmet group has no weaponSet)", b.Build.MainSocketGroup)
	}
	b = weaponSetBuild(1, "Weapon 1", "Helmet")
	b.SetUseSecondWeaponSet(true)
	if b.Build.MainSocketGroup != 1 || !second(b) {
		t.Fatalf("mainSocketGroup = %d, want 1 (no group in a set-2 slot to move to)", b.Build.MainSocketGroup)
	}
}

// TestSetUseSecondWeaponSetSameSetIsNoChange: ItemsTab.lua:225, 245: the button of the set in use does nothing.
func TestSetUseSecondWeaponSetSameSetIsNoChange(t *testing.T) {
	b := weaponSetBuild(1, "Weapon 1", "Weapon 1 Swap")
	if b.SetUseSecondWeaponSet(false) || b.Items.ItemSets[0].UseSecondWeaponSet != nil {
		t.Fatal("I on a set already using weapon set I changed it")
	}
	b.SetUseSecondWeaponSet(true)
	b.Build.MainSocketGroup = 1
	if b.SetUseSecondWeaponSet(true) || b.Build.MainSocketGroup != 1 {
		t.Fatal("II on a set already using weapon set II changed it")
	}
}
