package calculator

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/pob"
)

const itemList0339 = "0339-reliquarian-shield-crush-of-the-chieftain-2.xml"

func itemName(t *testing.T, build *pob.PathOfBuilding, id int) string {
	t.Helper()
	text, ok := buildItemText(build, id)
	if !ok {
		t.Fatalf("REFUSING: no item %d", id)
	}
	return ParseItem(text).Name
}

// TestDeleteItemPrompt: ItemListControl.lua:331-369 on 0339 (set 2 "enmity" active, set 1 untitled, one tree): each confirmation PoB words.
func TestDeleteItemPrompt(t *testing.T) {
	initItemsTabData(t)
	build := loadCorpusBuild(t, itemList0339)
	const tail = ".\nAre you sure you want to delete it?"
	for _, tc := range []struct {
		name string
		id   int
		want string
	}{
		{"equipped in the active set", 61, " is currently equipped in Helmet" + tail},
		{"equipped only in another set", 16, " is currently equipped in Helmet in set 'Default'" + tail},
		{"socketed in the active tree", 22, " is currently equipped in Socket" + tail},
	} {
		if got := DeleteItemPrompt(build, tc.id); got != itemName(t, build, tc.id)+tc.want {
			t.Errorf("%s: %q", tc.name, got)
		}
	}
	unused := UnusedItemIDs(build)
	if len(unused) == 0 {
		t.Fatal("REFUSING: 0339 has no unused item")
	}
	if got := DeleteItemPrompt(build, unused[0]); got != "" {
		t.Errorf("an unused item asks %q", got)
	}
}

// TestUnusedItemIDs: ItemListControl.lua:23-29: Del Unused keeps what any set's active slot, an abyssal socket or an allocated tree socket holds.
func TestUnusedItemIDs(t *testing.T) {
	initItemsTabData(t)
	build := loadCorpusBuild(t, itemList0339)
	used := map[int]bool{}
	for _, set := range build.Items.ItemSets {
		for _, s := range set.Slots {
			used[s.ItemID] = true
		}
	}
	for _, spec := range build.Tree.Specs {
		alloc := specAllocated(spec)
		for _, s := range spec.Sockets {
			if alloc[s.NodeID] {
				used[s.ItemID] = true
			}
		}
	}
	var want []int
	for _, it := range build.Items.Items {
		if !used[it.ID] {
			want = append(want, it.ID)
		}
	}
	if got := UnusedItemIDs(build); !slices.Equal(got, want) {
		t.Fatalf("unused = %v, want %v", got, want)
	}
	if len(want) == 0 || len(want) == len(build.Items.Items) {
		t.Fatalf("REFUSING: vacuous (%d unused of %d)", len(want), len(build.Items.Items))
	}
	n := len(build.Items.Items)
	if !DeleteItems(build, UnusedItemIDs(build)) || len(build.Items.Items) != n-len(want) || len(UnusedItemIDs(build)) != 0 {
		t.Fatalf("Del Unused left %d items, want %d", len(build.Items.Items), n-len(want))
	}
}

// TestDeleteItemsSocketedJewel: ItemsTab.lua:1768-1815: deleting a jewel socketed in the active tree empties its socket and the build forgets the item.
func TestDeleteItemsSocketedJewel(t *testing.T) {
	initItemsTabData(t)
	for _, tc := range []struct {
		file   string
		jewel  int
		socket int64
	}{
		{itemList0339, 22, 10532},
		{"0028-occultist-forbidden-rite-of-soul-sacrifice.xml", 2, 26196},
	} {
		build := loadCorpusBuild(t, tc.file)
		if build.SocketedJewel(int(tc.socket)) != tc.jewel || !slices.Contains(build.Build.PassiveNodes, tc.socket) {
			t.Fatalf("REFUSING: %s does not socket jewel %d on an allocated %d", tc.file, tc.jewel, tc.socket)
		}
		before := slices.Clone(build.Build.PassiveNodes)
		if !DeleteItems(build, []int{tc.jewel}) {
			t.Fatalf("%s: nothing was deleted", tc.file)
		}
		if build.SocketedJewel(int(tc.socket)) != 0 {
			t.Fatalf("%s: the socket still holds the jewel", tc.file)
		}
		after := slices.Clone(build.Build.PassiveNodes)
		slices.Sort(before)
		slices.Sort(after)
		if !slices.Equal(before, after) {
			t.Fatalf("%s: the allocation changed: %d nodes, was %d", tc.file, len(after), len(before))
		}
		if _, ok := buildItemText(build, tc.jewel); ok {
			t.Fatalf("%s: the item is still in the build", tc.file)
		}
		if DeleteItems(build, []int{tc.jewel}) {
			t.Fatalf("%s: deleting a missing item reported a change", tc.file)
		}
	}
}

// TestDeleteItemsEmptiesEverySet: ItemsTab.lua:1769-1782: every set's slot holding the item is emptied.
func TestDeleteItemsEmptiesEverySet(t *testing.T) {
	initItemsTabData(t)
	build := loadCorpusBuild(t, itemList0339)
	const amulet = 9
	DeleteItems(build, []int{amulet})
	for _, set := range build.Items.ItemSets {
		for _, s := range set.Slots {
			if s.ItemID == amulet {
				t.Fatalf("set %s still holds the item in %s", set.ID, s.Name)
			}
		}
	}
}

// TestDeleteAllItems: ItemListControl.lua:47-63: Del All empties the active set's slots and every tree's sockets; another set keeps its ids.
func TestDeleteAllItems(t *testing.T) {
	initItemsTabData(t)
	build := loadCorpusBuild(t, itemList0339)
	DeleteAllItems(build)
	if len(build.Items.Items) != 0 || len(build.Tree.Specs[0].Sockets) != 0 {
		t.Fatalf("%d items, %d sockets left", len(build.Items.Items), len(build.Tree.Specs[0].Sockets))
	}
	for _, set := range build.Items.ItemSets {
		held := slices.ContainsFunc(set.Slots, func(s pob.Slot) bool { return s.ItemID != 0 })
		if active := set.ID == "2"; active == held {
			t.Errorf("set %s holds items: %v", set.ID, held)
		}
	}
}

// TestSortItems: ItemsTab.lua:1731-1765 on 0339.
func TestSortItems(t *testing.T) {
	initItemsTabData(t)
	build := loadCorpusBuild(t, itemList0339)
	n := len(build.Items.Items)
	SortItems(build)
	if len(build.Items.Items) != n {
		t.Fatalf("%d items after sorting, want %d", len(build.Items.Items), n)
	}
	ids := make([]int, n)
	for k, it := range build.Items.Items {
		ids[k] = it.ID
	}
	set, _ := setItems(build)
	if ids[0] != setSlotItemID(set, "Weapon 1") {
		t.Fatalf("first = %d, want the active set's Weapon 1 item %d (order %v)", ids[0], setSlotItemID(set, "Weapon 1"), ids)
	}
	if slices.Index(ids, 61) > slices.Index(ids, 16) {
		t.Fatalf("set 1's helmet sorts before the active set's: %v", ids)
	}
	slots := orderedListSlots(build)
	var loose []string
	firstJewel := -1
	for k, it := range build.Items.Items {
		p := ParseItem(it.Text)
		if p.Type != "Jewel" {
			if firstJewel >= 0 {
				t.Fatalf("%s (%d) sorts after a jewel: %v", p.Name, it.ID, ids)
			}
			continue
		}
		if firstJewel < 0 {
			firstJewel = k
		}
		if slot, _ := equippedSlotForItem(build, slots, it.ID); slot == nil {
			loose = append(loose, p.Name)
		} else if len(loose) > 0 {
			t.Fatalf("socketed %s sorts after a loose jewel: %v", p.Name, ids)
		}
	}
	if !slices.IsSorted(loose) || len(loose) < 2 {
		t.Fatalf("loose jewels not by name: %q", loose)
	}
}

// TestToggleEquipItem: ItemListControl.lua:290-314: Ctrl+click equips in the primary slot or unequips.
func TestToggleEquipItem(t *testing.T) {
	initItemsTabData(t)
	build := loadCorpusBuild(t, itemList0339)
	active := func(slot string) int {
		set, _ := setItems(build)
		return setSlotItemID(set, slot)
	}
	if !ToggleEquipItem(build, 16, false) || active("Helmet") != 16 {
		t.Fatalf("Helmet = %d, want 16", active("Helmet"))
	}
	if !ToggleEquipItem(build, 16, false) || active("Helmet") != 0 {
		t.Fatalf("a second Ctrl+click left Helmet = %d", active("Helmet"))
	}
	if !ToggleEquipItem(build, 20, true) || active("Ring 2") != 20 {
		t.Fatalf("Shift: Ring 2 = %d, want 20", active("Ring 2"))
	}
	build.SetUseSecondWeaponSet(true)
	if !ToggleEquipItem(build, 18, false) || active("Weapon 1 Swap") != 18 {
		t.Fatalf("Weapon 1 Swap = %d, want 18", active("Weapon 1 Swap"))
	}
	if ToggleEquipItem(build, 22, false) {
		t.Fatal("a jewel has a primary slot")
	}
}

// TestActiveTreeSockets: ItemsTab.lua:1652-1670: the socket slots are the allocated main-tree sockets by id; 0339's cluster sub-sockets are not.
func TestActiveTreeSockets(t *testing.T) {
	initItemsTabData(t)
	build := loadCorpusBuild(t, itemList0339)
	got := ActiveTreeSockets(build)
	alloc := specAllocated(build.Tree.Specs[build.ActiveSpecIndex()])
	if len(got) == 0 || !slices.IsSorted(got) {
		t.Fatalf("sockets = %v", got)
	}
	for _, id := range got {
		if !alloc[id] || id >= 65536 {
			t.Fatalf("socket %d is not an allocated main-tree socket", id)
		}
	}
	if !slices.Contains(got, 10532) {
		t.Fatalf("10532 (jewel 22) missing from %v", got)
	}
}

// TestShownAbyssalSlots: ItemSlotControl.lua:98-111: an abyssal slot shows only under a parent item with that many abyssal sockets.
func TestShownAbyssalSlots(t *testing.T) {
	initItemsTabData(t)
	build := loadCorpusBuild(t, itemList0339)
	if got := ShownAbyssalSlots(build); len(got) != 0 {
		t.Fatalf("0339 wears no abyssal item, shown %v", got)
	}
	const belt = "Rarity: RARE\nAbyss Clasp\nStygian Vise\nItem Level: 84\nImplicits: 1\nHas 1 Abyssal Socket\n+40 to maximum Life"
	EquipItem(build, belt, "Belt")
	if got := ShownAbyssalSlots(build); !slices.Equal(got, []string{"Belt Abyssal Socket 1"}) {
		t.Fatalf("shown %v, want [Belt Abyssal Socket 1]", got)
	}
}

// TestEffectiveTinctureStats: ItemsTab.lua:5011-5038 on 13276's tincture.
func TestEffectiveTinctureStats(t *testing.T) {
	initItemsTabData(t)
	build := loadCorpusBuild(t, "13276-ladder-cuba_5717.cubamrc.xml")
	var tincture *Item
	for _, it := range build.Items.Items {
		if p := ParseItem(it.Text); p.Base != nil && p.Base.Tincture != nil {
			tincture = p
		}
	}
	if tincture == nil {
		t.Fatal("REFUSING: 13276 has no tincture")
	}
	got := GetMiscCalculator(build).EffectiveTinctureStats(tincture)
	if len(got) < 3 || got[0] != "^7Effective tincture stats:" {
		t.Fatalf("stats = %q", got)
	}
	last := got[len(got)-1]
	if !strings.HasPrefix(last, "^8Tincture Cooldown when deactivated: ^7") || !strings.HasSuffix(last, "^8 seconds") {
		t.Fatalf("last line = %q", last)
	}
	if !slices.ContainsFunc(got, func(l string) bool { return strings.HasPrefix(l, "^8Mana Burn Inflicted Every Second: ^7") }) {
		t.Fatalf("no mana burn line: %q", got)
	}
	if GetMiscCalculator(build).EffectiveFlaskStats(tincture) != nil {
		t.Fatal("a tincture drew flask stats")
	}
}

// TestSocketJewelTooltipAttributesInRadius: ItemsTab.lua:4580-4592: a single-func threshold jewel socketed on the tree shows the attributes its radius holds.
func TestSocketJewelTooltipAttributesInRadius(t *testing.T) {
	initItemsTabData(t)
	attrLine := func(tip ItemTooltip) string {
		for _, l := range tip.Lines {
			if strings.HasPrefix(l.Text, "^x7F7F7FAttributes in Radius: ") {
				return l.Text
			}
		}
		return ""
	}
	build := loadCorpusBuild(t, "0243-necromancer-raise-spectre.xml")
	if tip, ok := GetMiscCalculator(build).SocketJewelTooltip(36634, ItemTooltipContext{}); !ok || attrLine(tip) != "" {
		t.Fatalf("Unending Hunger (multi-func threshold) draws %q", attrLine(tip))
	}
	const jewel = "Rarity: UNIQUE\nIcy Test\nViridian Jewel\nRadius: Large\nImplicits: 0\nWith at least 40 Dexterity in Radius, Ice Shot has 25% increased Area of Effect"
	if !build.SetItemText(3, jewel) {
		t.Fatal("REFUSING: 0243 has no item 3")
	}
	misc := GetMiscCalculator(build)
	data := misc.base.JewelRadiusData["36634"]
	tip, ok := misc.SocketJewelTooltip(36634, ItemTooltipContext{})
	if !ok || data["Dex"] == 0 {
		t.Fatalf("REFUSING: no Dex in the radius of 36634 (%v)", data)
	}
	if want := fmt.Sprintf("^x7F7F7FAttributes in Radius: ^x70FF70%d Dex^7", int(data["Dex"])); data["Str"] == 0 && data["Int"] == 0 && attrLine(tip) != want {
		t.Fatalf("line = %q, want %q", attrLine(tip), want)
	} else if attrLine(tip) == "" {
		t.Fatalf("no Attributes in Radius line for %v", data)
	}
	if line := attrLine(AddItemTooltip(ParseItem(jewel), ItemTooltipContext{})); line != "" {
		t.Fatalf("an unsocketed jewel draws %q", line)
	}
}

// TestCompareAddedMod: ItemsTab.lua:2408-2422: the display item with the mod added, against it as it is, in its primary slot.
func TestCompareAddedMod(t *testing.T) {
	initItemsTabData(t)
	build := loadCorpusBuild(t, "0280-elementalist-fireball.xml")
	misc := GetMiscCalculator(build)
	_, text := activeSetSlot(build, "Helmet")
	base, added := misc.CompareAddedMod(text, []string{"+50 to maximum Life"}, "crafted")
	if base == nil || added == nil {
		t.Fatal("no comparison")
	}
	if got := added.Player.Output["Life"] - base.Player.Output["Life"]; got <= 0 {
		t.Fatalf("Life +%v, want more", got)
	}
	if base.Player.Output["Life"] != misc.Base().Player.Output["Life"] {
		t.Fatalf("the base (the helmet as it is, in Helmet) differs from the build: %v vs %v", base.Player.Output["Life"], misc.Base().Player.Output["Life"])
	}
	if b, a := misc.CompareAddedMod("not an item", nil, ""); b != nil || a != nil {
		t.Fatal("an item with no base compared")
	}
}

// TestCompareReplacedImplicits: ItemsTab.lua:2503-2525 with replaceImplicits.
func TestCompareReplacedImplicits(t *testing.T) {
	initItemsTabData(t)
	build := loadCorpusBuild(t, "0280-elementalist-fireball.xml")
	misc := GetMiscCalculator(build)
	_, text := activeSetSlot(build, "Helmet")
	_, none := misc.CompareReplacedImplicits(text, nil)
	_, life := misc.CompareReplacedImplicits(text, []string{"+50 to maximum Life"})
	base, added := misc.CompareAddedMod(text, []string{"+50 to maximum Life"}, "")
	if none == nil || life == nil || len(ParseItem(text).ImplicitModLines) == 0 {
		t.Fatal("REFUSING: no comparison, or a helmet with no implicit to replace")
	}
	want := added.Player.Output["Life"] - base.Player.Output["Life"]
	if got := life.Player.Output["Life"] - none.Player.Output["Life"]; want <= 0 || math.Abs(got-want) > 1e-9 {
		t.Fatalf("Life +%v over no implicits, adding gives +%v", got, want)
	}
	big, _ := ImplicitText(text, "CUSTOM", "", "+1000 to maximum Life")
	if with, without := misc.CompareReplacedImplicits(big, nil); with.Player.Output["Life"]-without.Player.Output["Life"] < 1000 {
		t.Fatalf("an implicit of +1000 Life kept: %v vs %v", with.Player.Output["Life"], without.Player.Output["Life"])
	}
}

// TestItemListLoadoutFilter: ItemListControl.lua:77-183: 0339's loadout filter lists the fixed entries.
func TestItemListLoadoutFilter(t *testing.T) {
	initItemsTabData(t)
	build := loadCorpusBuild(t, itemList0339)
	filters := ItemListLoadoutFilters(build)
	if len(filters) < 4 || !slices.Equal(filters[:3], []string{"Any Loadout", "Current Loadout", "Unused Items"}) {
		t.Fatalf("filters %q", filters)
	}
	if got := ItemListFiltered(build, "Any Loadout"); len(got) != len(build.Items.Items) {
		t.Fatalf("Any Loadout lists %d of %d", len(got), len(build.Items.Items))
	}
	if got := ItemListFiltered(build, "Unused Items"); !slices.Equal(got, UnusedItemIDs(build)) {
		t.Fatalf("Unused Items %v, Del Unused %v", got, UnusedItemIDs(build))
	}
	current := ItemListFiltered(build, "Current Loadout")
	set, _ := setItems(build)
	for _, s := range set.Slots {
		if s.ItemID != 0 && !slices.Contains(current, s.ItemID) {
			t.Fatalf("Current Loadout misses %s's %d", s.Name, s.ItemID)
		}
	}
	for _, s := range build.Items.ItemSets {
		title := "Default"
		if s.Title != nil {
			title = *s.Title
		}
		if !slices.Contains(filters, title) {
			t.Fatalf("set %q is not a filter", title)
		}
		got := ItemListFiltered(build, title)
		for _, slot := range s.Slots {
			if slot.ItemID != 0 && !slices.Contains(got, slot.ItemID) {
				t.Fatalf("filter %q misses %d", title, slot.ItemID)
			}
		}
	}
}

// TestItemListLoadoutFilterAgainstPoB grades the filter's entries and the list under each against the pin's ItemListControl on six builds.
func TestItemListLoadoutFilterAgainstPoB(t *testing.T) {
	initItemsTabData(t)
	raw, err := os.ReadFile("../testdata/items-ui/item_list_filter.json")
	if err != nil {
		t.Fatalf("REFUSING: testdata/items-ui/item_list_filter.json: %v", err)
	}
	var want map[string]struct {
		Filters []string
		Lists   []struct {
			Filter string
			Items  []int
		}
	}
	if err := json.Unmarshal(raw, &want); err != nil || len(want) < 5 {
		t.Fatalf("REFUSING: the fixture does not decode: %v", err)
	}
	for file, w := range want {
		build := loadCorpusBuild(t, file)
		if got := ItemListLoadoutFilters(build); !slices.Equal(got, w.Filters) {
			t.Errorf("%s filters:\n ours %q\n PoB  %q", file, got, w.Filters)
		}
		for _, l := range w.Lists {
			if got := ItemListFiltered(build, l.Filter); !slices.Equal(got, l.Items) {
				t.Errorf("%s %q: ours %v, PoB %v", file, l.Filter, got, l.Items)
			}
		}
	}
}
