package calculator

import (
	"testing"
)

func itemModSummary(item *Item) map[string]float64 {
	out := map[string]float64{}
	for _, m := range item.Mods {
		if v := m.Value(); v != nil {
			out[m.Name()] += v.Float()
		}
	}
	return out
}

// Expected values come from Item.lua's BuildModList on the same raw text.
func TestItemBuildModListStepsAgainstPoB(t *testing.T) {
	requireGameData(t, "3.30")
	t.Run("zero-value ranged line is dropped", func(t *testing.T) {
		item := ParseItem("Rarity: RARE\nTest\nGold Ring\nImplicits: 0\n{range:0}+(0-10)% to Fire Resistance\n+5 to Strength")
		for _, m := range item.Mods {
			if m.Name() == "FireResist" {
				t.Errorf("FireResist mod kept, PoB drops the zero-value line")
			}
		}
	})
	t.Run("duplicate variants add the mod per selection", func(t *testing.T) {
		item := ParseItem("Rarity: UNIQUE\nTest\nHeavy Belt\nVariant: A\nVariant: B\nSelected Variant: 1\nHas Alt Variant: true\nSelected Alt Variant: 1\nAllow Duplicate Variants: true\nImplicits: 0\n{variant:1}+10 to Strength\n{variant:2}+7 to Dexterity")
		if got := itemModSummary(item); got["Str"] != 20 || got["Dex"] != 0 {
			t.Errorf("Str %v Dex %v, want 20 and 0", got["Str"], got["Dex"])
		}
	})
	t.Run("socket colour flag without sockets", func(t *testing.T) {
		item := ParseItem("Rarity: RARE\nTest\nGold Ring\nImplicits: 0\nGems Socketed always have the quality bonus from Socket Colour")
		if !item.SocketColourAlwaysMatches || len(item.Sockets) != 0 {
			t.Errorf("colourAlwaysMatches %v sockets %d, want true and 0", item.SocketColourAlwaysMatches, len(item.Sockets))
		}
		for _, m := range item.Mods {
			if m.Name() == "SocketAlwaysMatches" {
				t.Errorf("SocketAlwaysMatches left in the mod list")
			}
		}
	})
	t.Run("local requirement mods leave the slot list", func(t *testing.T) {
		item := ParseItem("Rarity: RARE\nTest\nIron Hat\nImplicits: 0\n25% reduced Attribute Requirements\n+5 to Strength")
		for _, m := range item.Mods {
			switch m.Name() {
			case "StrRequirement", "DexRequirement", "IntRequirement":
				t.Errorf("%s left in the mod list", m.Name())
			}
		}
		if item.Requirements["strMod"] != 6 {
			t.Errorf("strMod %v, want 6", item.Requirements["strMod"])
		}
	})
	t.Run("granted skill keeps its influence requirement", func(t *testing.T) {
		item := ParseItem("Rarity: RARE\nTest\nPaua Amulet\nImplicits: 0\nCurse Enemies with Vulnerability on Hit if 1 Hunter Items are Equipped")
		if len(item.GrantedSkills) != 1 || item.GrantedSkills[0].SkillID != "Vulnerability" {
			t.Fatalf("granted %+v", item.GrantedSkills)
		}
		r := item.GrantedSkills[0].ItemRequirement
		if r == nil || r.Multiplier != "HunterItem" || r.Count != 1 {
			t.Errorf("itemRequirement %+v, want HunterItem 1", r)
		}
	})
	t.Run("implicit mod tags set hasModTags", func(t *testing.T) {
		item := ParseItem("Rarity: RARE\nTest\nPaua Amulet\nImplicits: 1\n{tags:attribute}+20 to Strength\n+5 to Dexterity")
		if !item.HasModTags {
			t.Errorf("hasModTags false, PoB true")
		}
	})
}

// Item.lua:2406-2409 divide by a zero variance too; LuaJIT clamps the result to 0 or 1.
func TestArmourBasePercentileZeroVariance(t *testing.T) {
	requireGameData(t, "3.30")
	for armour, want := range map[string]float64{"50": 0, "100": 1, "150": 1} {
		item := ParseItem("Rarity: RARE\nTest\nSilk Gloves\nArmour: " + armour + "\nImplicits: 0\n+100 to Armour")
		if got := item.ArmourData["ArmourBasePercentile"]; got != want || item.ArmourData["Armour"] != 100 {
			t.Errorf("Armour: %s: percentile %v armour %v, want %v and 100", armour, got, item.ArmourData["Armour"], want)
		}
	}
}

// Item.lua:666 takes the whole remainder after ": " as the spec value.
func TestItemSpecValueIsWholeRemainder(t *testing.T) {
	requireGameData(t, "3.30")
	item := ParseItem("Rarity: RARE\nTest\nGold Ring\nUnique ID: 5 abc\nNote: 5 stuff\nLeague: 3 Standard\nQuality: +20% (augmented)\nImplicits: 0\n+5 to Strength")
	if item.UniqueID != "5 abc" || item.Note != "5 stuff" || item.League != "3 Standard" {
		t.Errorf("unique id %q note %q league %q", item.UniqueID, item.Note, item.League)
	}
	if item.Quality == nil || *item.Quality != 20 {
		t.Errorf("quality %v, want 20", item.Quality)
	}
}
