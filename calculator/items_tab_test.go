package calculator

import (
	"slices"
	"testing"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/storage"
)

func initItemsTabData(t *testing.T) {
	t.Helper()
	if err := poe.InitializeAll(raw.DataContext(t.Context()), raw.LatestVersion, raw.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}
}

func donorItem(t *testing.T, file, slot string) string {
	t.Helper()
	_, text := activeSetSlot(loadCorpusBuild(t, file), slot)
	if text == "" {
		t.Fatalf("REFUSING: %s has no item in %s", file, slot)
	}
	return text
}

// TestIsItemValidForSlot: ItemsTab.lua:2457-2503 on the equipment slots, with 0280's wand in Weapon 1.
func TestIsItemValidForSlot(t *testing.T) {
	initItemsTabData(t)
	build := loadCorpusBuild(t, "0280-elementalist-fireball.xml")
	slots := []string{"Weapon 1", "Weapon 2", "Helmet", "Body Armour", "Gloves", "Boots", "Amulet", "Ring 1", "Ring 2", "Ring 3", "Belt", "Flask 1", "Flask 5"}
	for _, tc := range []struct {
		name, file, slot string
		want             []string
	}{
		{"helmet", "4788-ladder-GeordiMcSnuff_1894.GeordiFlameOne.xml", "Helmet", []string{"Helmet"}},
		{"ring", "0008-necromancer-vaal-summon-skeletons.xml", "Ring 1", []string{"Ring 1", "Ring 2", "Ring 3"}},
		{"staff beside a wand", "4788-ladder-GeordiMcSnuff_1894.GeordiFlameOne.xml", "Weapon 1 Swap", []string{"Weapon 1"}},
		{"shield beside a wand", "0339-reliquarian-shield-crush-of-the-chieftain-2.xml", "Weapon 2", []string{"Weapon 2"}},
		{"sword beside a wand", "0339-reliquarian-shield-crush-of-the-chieftain-2.xml", "Weapon 1", []string{"Weapon 1"}},
		{"wand beside a wand", "0280-elementalist-fireball.xml", "Weapon 1", []string{"Weapon 1", "Weapon 2"}},
		{"flask", "4788-ladder-GeordiMcSnuff_1894.GeordiFlameOne.xml", "Flask 5", []string{"Flask 1", "Flask 5"}},
		{"tincture", "13276-ladder-cuba_5717.cubamrc.xml", "Flask 1", []string{"Flask 1", "Flask 5"}},
	} {
		item := ParseItem(donorItem(t, tc.file, tc.slot))
		var got []string
		for _, s := range slots {
			if IsItemValidForSlot(build, item, s) {
				got = append(got, s)
			}
		}
		if !slices.Equal(got, tc.want) {
			t.Errorf("%s: fits %v, want %v", tc.name, got, tc.want)
		}
	}
}

// TestCompareItemSlots: ItemsTab.lua:5047-5158: every shown slot the item fits gets a what-if equal to Calc for that slot.
func TestCompareItemSlots(t *testing.T) {
	initItemsTabData(t)
	build := loadCorpusBuild(t, "0280-elementalist-fireball.xml")
	misc := GetMiscCalculator(build)

	ring := ParseItem(donorItem(t, "0008-necromancer-vaal-summon-skeletons.xml", "Ring 1"))
	got := misc.CompareItemSlots(ring)
	names := make([]string, 0, len(got))
	for _, c := range got {
		names = append(names, c.SlotName)
		want := misc.Calc(&CalcOverride{RepSlotName: c.SlotName, RepItem: ring})
		if where, ok := outputsAgree(c.Env.Player.Output, want.Player.Output); !ok {
			t.Errorf("%s: compare output differs from Calc: %s", c.SlotName, where)
		}
		if c.Replacing == nil {
			t.Errorf("%s: 0280 has a ring there, but no replaced item was reported", c.SlotName)
		}
	}
	slices.Sort(names)
	if !slices.Equal(names, []string{"Ring 1", "Ring 2"}) {
		t.Errorf("ring compares in %v, want Ring 1 and Ring 2 (Ring 3 is hidden without the Unseen Hand)", names)
	}

	if s := misc.CompareItemSlots(ParseItem(donorItem(t, "4788-ladder-GeordiMcSnuff_1894.GeordiFlameOne.xml", "Flask 5"))); s != nil {
		t.Errorf("a flask got %d equip comparisons; it compares by toggling", len(s))
	}

	misc = GetMiscCalculator(loadCorpusBuild(t, "0339-reliquarian-shield-crush-of-the-chieftain-2.xml"))
	unique := ParseItem(donorItem(t, "0339-reliquarian-shield-crush-of-the-chieftain-2.xml", "Ring 1"))
	one := 1
	unique.Limit = &one
	if only := misc.CompareItemSlots(unique); len(only) != 1 || only[0].SlotName != "Ring 1" {
		t.Errorf("a limited unique at its limit compares in %d slots, want only its own (Ring 1)", len(only))
	}
	unique.Limit = nil
	if n := len(misc.CompareItemSlots(unique)); n != 2 {
		t.Errorf("without a limit the unique compares in %d slots, want 2", n)
	}
}

// TestCompareItemSlotsEmptyFirst: ItemsTab.lua:5135-5152 sorts empty slots first.
func TestCompareItemSlotsEmptyFirst(t *testing.T) {
	initItemsTabData(t)
	build := loadCorpusBuild(t, "0280-elementalist-fireball.xml")
	build.SetSlotItem("Ring 2", 0)
	got := GetMiscCalculator(build).CompareItemSlots(ParseItem(donorItem(t, "0008-necromancer-vaal-summon-skeletons.xml", "Ring 1")))
	if len(got) != 2 || got[0].SlotName != "Ring 2" || got[0].Replacing != nil {
		t.Fatalf("want the empty Ring 2 first, got %d slots, first %q", len(got), got[0].SlotName)
	}
}

// TestEquipAndAddItem: Equipping a two-handed weapon empties Weapon 2.
func TestEquipAndAddItem(t *testing.T) {
	initItemsTabData(t)
	staff := donorItem(t, "4788-ladder-GeordiMcSnuff_1894.GeordiFlameOne.xml", "Weapon 1 Swap")

	build := loadCorpusBuild(t, "0280-elementalist-fireball.xml")
	id := EquipItem(build, staff, "Weapon 1")
	set, _ := setItems(build)
	if setSlotItemID(set, "Weapon 1") != id || setSlotItemID(set, "Weapon 2") != 0 {
		t.Errorf("after equipping a staff: Weapon 1 = %d (want %d), Weapon 2 = %d (want 0)", setSlotItemID(set, "Weapon 1"), id, setSlotItemID(set, "Weapon 2"))
	}

	build = loadCorpusBuild(t, "0280-elementalist-fireball.xml")
	build.SetSlotItem("Ring 2", 0)
	ringID, slot := GetMiscCalculator(build).AddItem(donorItem(t, "0008-necromancer-vaal-summon-skeletons.xml", "Ring 1"))
	set, _ = setItems(build)
	if slot != "Ring 2" || setSlotItemID(set, "Ring 2") != ringID {
		t.Errorf("Save equipped the ring in %q, want the free Ring 2", slot)
	}

	build = loadCorpusBuild(t, "0280-elementalist-fireball.xml")
	if _, slot := GetMiscCalculator(build).AddItem(donorItem(t, "0008-necromancer-vaal-summon-skeletons.xml", "Ring 1")); slot != "" {
		t.Errorf("Save equipped the ring in %q with no free ring slot", slot)
	}
}

// TestReplaceItemKeepsID: Saving an edited item keeps its id.
func TestReplaceItemKeepsID(t *testing.T) {
	initItemsTabData(t)
	ring := donorItem(t, "0008-necromancer-vaal-summon-skeletons.xml", "Ring 1")
	build := loadCorpusBuild(t, "0280-elementalist-fireball.xml")
	build.SetSlotItem("Ring 2", 0)
	slot, _ := activeSetSlot(build, "Ring 1")
	if slot.ItemID == 0 {
		t.Fatal("REFUSING: the build has no Ring 1")
	}
	count := len(build.Items.Items)
	if !GetMiscCalculator(build).ReplaceItem(slot.ItemID, ring) {
		t.Fatalf("ReplaceItem(%d) found no item", slot.ItemID)
	}
	after, text := activeSetSlot(build, "Ring 1")
	if after.ItemID != slot.ItemID || ParseItem(text).Name != ParseItem(ring).Name {
		t.Errorf("Ring 1 holds item %d %q, want item %d %q", after.ItemID, ParseItem(text).Name, slot.ItemID, ParseItem(ring).Name)
	}
	if len(build.Items.Items) != count {
		t.Errorf("the item list grew from %d to %d", count, len(build.Items.Items))
	}
	if free, _ := activeSetSlot(build, "Ring 2"); free.ItemID != 0 {
		t.Errorf("the edit equipped item %d in the free Ring 2", free.ItemID)
	}
	if GetMiscCalculator(build).ReplaceItem(9999, ring) {
		t.Error("ReplaceItem(9999) replaced an item the build does not have")
	}
}

// TestFormatModLine: ItemTools.lua:357-378.
func TestFormatModLine(t *testing.T) {
	half := 0.5
	for _, tc := range []struct {
		ml   ModLine
		line string
		kind ModLineKind
		show bool
	}{
		{ModLine{Line: "+45 to maximum Life"}, "+45 to maximum Life", ModLineMagic, true},
		{ModLine{Line: "+0% to Fire Resistance"}, "", "", false},
		{ModLine{Line: "Adds 0 to 0 Fire Damage"}, "", "", false},
		{ModLine{Line: "Adds 0 to 12 Fire Damage"}, "Adds 0 to 12 Fire Damage", ModLineMagic, true},
		{ModLine{Line: "+(10-20) to maximum Life", Range: &half}, "+15 to maximum Life", ModLineMagic, true},
		{ModLine{Line: "+45 to maximum Life", Crafted: true}, "+45 to maximum Life", ModLineCrafted, true},
		{ModLine{Line: "+45 to maximum Life", Crafted: true, Fractured: true}, "+45 to maximum Life", ModLineFractured, true},
		{ModLine{Line: "Nonsense", Extra: "Nonsense"}, "Nonsense", ModLineUnsupported, true},
	} {
		line, kind, show := FormatModLine(tc.ml)
		if line != tc.line || kind != tc.kind || show != tc.show {
			t.Errorf("%q: got (%q, %q, %v), want (%q, %q, %v)", tc.ml.Line, line, kind, show, tc.line, tc.kind, tc.show)
		}
	}
}
