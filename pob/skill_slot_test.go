package pob

import "testing"

// TestSetSocketGroupSlotKeepsTheImbuedSupport: SkillsTab.lua:168-184: a group's imbued support moves with it to a slot that has none.
func TestSetSocketGroupSlotKeepsTheImbuedSupport(t *testing.T) {
	str := func(s string) *string { return &s }
	b := &PathOfBuilding{Skills: Skills{SkillSets: []SkillSet{{Skills: []Skill{
		{Slot: "Body Armour", ImbuedSupport: str("Added Fire Damage")},
		{Slot: "Helmet", ImbuedSupport: str("Faster Attacks")},
		{Slot: "Gloves"},
	}}}}}
	groups := b.Skills.SkillSets[0].Skills

	b.SetSocketGroupSlot(0, 0, "Boots")
	if groups[0].Slot != "Boots" || groups[0].ImbuedSupport == nil {
		t.Fatalf("to a free slot: %q %v", groups[0].Slot, groups[0].ImbuedSupport)
	}
	b.SetSocketGroupSlot(0, 0, "Helmet")
	if groups[0].Slot != "Helmet" || groups[0].ImbuedSupport != nil {
		t.Fatalf("to a slot with an imbued support: %q %v", groups[0].Slot, groups[0].ImbuedSupport)
	}
	b.SetSocketGroupSlot(0, 1, "")
	if groups[1].Slot != "" || groups[1].ImbuedSupport != nil {
		t.Fatalf("to None: %q %v", groups[1].Slot, groups[1].ImbuedSupport)
	}
	b.SetSocketGroupSlot(0, 2, "Ring 1")
	if groups[2].Slot != "Ring 1" || groups[2].ImbuedSupport != nil {
		t.Fatalf("a group without one: %q %v", groups[2].Slot, groups[2].ImbuedSupport)
	}
}

// TestSetSocketGroupImbuedSupport: SkillsTab.lua:296-324: the selector sets the group's imbued support by gem name and "" clears it.
func TestSetSocketGroupImbuedSupport(t *testing.T) {
	b := &PathOfBuilding{Skills: Skills{SkillSets: []SkillSet{{Skills: []Skill{{Slot: "Body Armour"}, {}}}}}}
	groups := b.Skills.SkillSets[0].Skills
	b.SetSocketGroupImbuedSupport(0, 0, "Added Fire Damage")
	if groups[0].ImbuedSupport == nil || *groups[0].ImbuedSupport != "Added Fire Damage" {
		t.Fatalf("set: %v", groups[0].ImbuedSupport)
	}
	b.SetSocketGroupImbuedSupport(0, 0, "")
	if groups[0].ImbuedSupport != nil {
		t.Fatalf("cleared: %v", *groups[0].ImbuedSupport)
	}
	b.SetSocketGroupImbuedSupport(0, 1, "Added Fire Damage")
	if groups[1].ImbuedSupport != nil {
		t.Fatalf("no slot: %v", *groups[1].ImbuedSupport)
	}
}
