package data

import (
	"testing"

	"github.com/MarvinJWendt/testza"
)

var keyTagMismatch = map[string]string{
	"affliction_chance_to_block_attack_damage": "affliction_chance_to_block",
	"affliction_chance_to_block_spell_damage":  "affliction_chance_to_block",
	"affliction_aura_effect":                   "old_do_not_use_affliction_aura_effect",
	"affliction_curse_effect":                  "old_do_not_use_affliction_curse_effect",
}

// TestClusterJewelData: Expectations are read off PoB's src/Data/ClusterJewels.lua directly.
func TestClusterJewelData(t *testing.T) {
	testza.AssertLen(t, ClusterJewels.Jewels, 3)
	testza.AssertLen(t, ClusterJewels.NotableSortOrder, 300)
	testza.AssertLen(t, ClusterJewels.Keystones, 8)

	expected := map[string]struct {
		size            string
		sizeIndex       int
		minNodes        int
		maxNodes        int
		smallIndicies   []int
		notableIndicies []int
		socketIndicies  []int
		totalIndicies   int
		skills          int
	}{
		"Small Cluster Jewel":  {"Small", 0, 2, 3, []int{0, 4, 2}, []int{4}, []int{4}, 6, 17},
		"Medium Cluster Jewel": {"Medium", 1, 4, 6, []int{0, 6, 8, 4, 10, 2}, []int{6, 10, 2, 0}, []int{6}, 12, 21},
		"Large Cluster Jewel":  {"Large", 2, 8, 12, []int{0, 4, 6, 8, 10, 2, 7, 5, 9, 3, 11, 1}, []int{6, 4, 8, 10, 2}, []int{4, 8, 6}, 12, 17},
	}

	total := 0
	for name, e := range expected {
		jewel, ok := ClusterJewels.Jewels[name]
		testza.AssertTrue(t, ok, name)
		testza.AssertEqual(t, e.size, jewel.Size, name)
		testza.AssertEqual(t, e.sizeIndex, jewel.SizeIndex, name)
		testza.AssertEqual(t, e.minNodes, jewel.MinNodes, name)
		testza.AssertEqual(t, e.maxNodes, jewel.MaxNodes, name)
		testza.AssertEqual(t, e.smallIndicies, jewel.SmallIndicies, name)
		testza.AssertEqual(t, e.notableIndicies, jewel.NotableIndicies, name)
		testza.AssertEqual(t, e.socketIndicies, jewel.SocketIndicies, name)
		testza.AssertEqual(t, e.totalIndicies, jewel.TotalIndicies, name)
		testza.AssertLen(t, jewel.Skills, e.skills, name)
		total += len(jewel.Skills)

		for tag, skill := range jewel.Skills {
			if want, ok := keyTagMismatch[tag]; ok {
				testza.AssertEqual(t, want, skill.Tag, tag)
			} else {
				testza.AssertEqual(t, tag, skill.Tag, name)
			}
			testza.AssertNotEqual(t, "", skill.Name, tag)
			testza.AssertNotEqual(t, "", skill.Icon, tag)
			testza.AssertNotZero(t, len(skill.Stats), tag)
			testza.AssertNotZero(t, len(skill.Enchant), tag)
		}
	}
	testza.AssertEqual(t, 55, total)

	for tag, skill := range ClusterJewels.Jewels["Small Cluster Jewel"].Skills {
		testza.AssertEqual(t, "", skill.MasteryIcon, tag)
	}
	for _, name := range []string{"Medium Cluster Jewel", "Large Cluster Jewel"} {
		for tag, skill := range ClusterJewels.Jewels[name].Skills {
			testza.AssertNotEqual(t, "", skill.MasteryIcon, tag)
		}
	}

	testza.AssertEqual(t, ClusterJewelSkill{
		Name:    "Life",
		Icon:    "Art/2DArt/SkillIcons/passives/IncreasedMaximumLifeNode.png",
		Tag:     "affliction_maximum_life",
		Stats:   []string{"4% increased maximum Life"},
		Enchant: []string{"Added Small Passive Skills grant: 4% increased maximum Life"},
	}, ClusterJewels.Jewels["Small Cluster Jewel"].Skills["affliction_maximum_life"])

	testza.AssertEqual(t, ClusterJewelSkill{
		Name:        "Minion Life",
		Icon:        "Art/2DArt/SkillIcons/passives/IncreaseMinionLifeNode.png",
		MasteryIcon: "Art/2DArt/SkillIcons/passives/AltMasteryGroupMinions.png",
		Tag:         "affliction_minion_life",
		Stats:       []string{"Minions have 12% increased maximum Life"},
		Enchant:     []string{"Added Small Passive Skills grant: Minions have 12% increased maximum Life"},
	}, ClusterJewels.Jewels["Medium Cluster Jewel"].Skills["affliction_minion_life"])

	testza.AssertEqual(t, ClusterJewelSkill{
		Name:        "Trap and Mine Damage",
		Icon:        "Art/2DArt/SkillIcons/passives/TrapAndMineDmgNode.png",
		MasteryIcon: "Art/2DArt/SkillIcons/passives/AltMasteryTraps.png",
		Tag:         "affliction_trap_and_mine_damage",
		Stats: []string{
			"12% increased Trap Damage",
			"12% increased Mine Damage",
		},
		Enchant: []string{
			"Added Small Passive Skills grant: 12% increased Trap Damage",
			"Added Small Passive Skills grant: 12% increased Mine Damage",
		},
	}, ClusterJewels.Jewels["Medium Cluster Jewel"].Skills["affliction_trap_and_mine_damage"])

	testza.AssertEqual(t, ClusterJewelSkill{
		Name:        "Mace and Staff Damage",
		Icon:        "Art/2DArt/SkillIcons/passives/NodeMaceandStaffDamage.png",
		MasteryIcon: "Art/2DArt/SkillIcons/passives/AltMasteryGroupMaceAndStaff.png",
		Tag:         "affliction_mace_and_staff_damage",
		Stats: []string{
			"Staff Attacks deal 12% increased Damage with Hits and Ailments",
			"Mace or Sceptre Attacks deal 12% increased Damage with Hits and Ailments",
		},
		Enchant: []string{
			"Added Small Passive Skills grant: Staff Attacks deal 12% increased Damage with Hits and Ailments",
			"Added Small Passive Skills grant: Mace or Sceptre Attacks deal 12% increased Damage with Hits and Ailments",
		},
	}, ClusterJewels.Jewels["Large Cluster Jewel"].Skills["affliction_mace_and_staff_damage"])

	testza.AssertEqual(t, 11552, ClusterJewels.NotableSortOrder["Wizardry"])
	testza.AssertEqual(t, 15607, ClusterJewels.NotableSortOrder["Aggressive Defence"])
	testza.AssertEqual(t, "Disciple of Kitava", ClusterJewels.Keystones[0])
	testza.AssertEqual(t, "Pitfighter", ClusterJewels.Keystones[7])
}
