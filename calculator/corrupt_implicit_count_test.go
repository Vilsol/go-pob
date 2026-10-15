package calculator

import "testing"

// TestCorruptImplicitCount pins Item.lua:1640, :2281-2284 and ModParser.lua:5940-5941 (upstream #10206).
func TestCorruptImplicitCount(t *testing.T) {
	requireGameData(t, "3.29")

	item := &Item{}
	item.ParseRaw(`Rarity: UNIQUE
Kaom's Heart
Glorious Plate
Implicits: 2
Can be modified while Corrupted
Can have up to 5 Implicit Modifiers while Item has this Modifier
Has no Sockets
+500 to maximum Life
Corrupted`)
	if !item.IsUnique {
		t.Error("IsUnique = false for a unique")
	}
	if item.CorruptImplicitCount != 5 {
		t.Errorf("CorruptImplicitCount = %v, want 5", item.CorruptImplicitCount)
	}
	for _, ml := range item.ImplicitModLines {
		if ml.Extra != "" {
			t.Errorf("%q left unparsed: %q", ml.Line, ml.Extra)
		}
	}
	for _, m := range item.Mods {
		if m.Name() == "CorruptImplicitCount" {
			t.Error("CorruptImplicitCount stayed in the item's mod list; calcLocal consumes it")
		}
	}

	plain := &Item{}
	plain.ParseRaw("Rarity: RARE\nX\nGlorious Plate\n+50 to maximum Life")
	if plain.IsUnique || plain.CorruptImplicitCount != 0 {
		t.Errorf("rare: IsUnique %v CorruptImplicitCount %v, want false / 0", plain.IsUnique, plain.CorruptImplicitCount)
	}
}

// TestArmourDataCarriesBaseDefences pins Item.lua:2388-2394 (upstream #10210): the unscaled base roll beside each defence.
func TestArmourDataCarriesBaseDefences(t *testing.T) {
	requireGameData(t, "3.29")

	item := &Item{}
	item.ParseRaw("Rarity: RARE\nX\nGlorious Plate\nArmour: 1000\nArmourBasePercentile: 0.5\n100% increased Armour")
	if item.ArmourData == nil {
		t.Fatal("no armour data")
	}
	base := item.ArmourData["ArmourBase"]
	if base == 0 || base >= item.ArmourData["Armour"] {
		t.Errorf("ArmourBase = %v (Armour %v), want the unscaled base roll", base, item.ArmourData["Armour"])
	}
}
