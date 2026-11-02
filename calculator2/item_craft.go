package calculator2

import (
	"sort"
	"strings"
	"sync"

	"github.com/Vilsol/go-pob/data/poe2"
)

// ItemBaseListEntry is one row of data.itemBaseLists.
type ItemBaseListEntry struct {
	Label string
	Name  string
	Base  *poe2.ItemBase
}

var itemBaseListsOnce = sync.OnceValues(buildItemBaseLists)

//pob2:src/Modules/Data.lua:1115-1146
func buildItemBaseLists() (map[string][]ItemBaseListEntry, []string) {
	lists := map[string][]ItemBaseListEntry{}
	for k := range poe2.ItemBases {
		base := &poe2.ItemBases[k]
		if !base.Hidden {
			typ := base.Type
			if base.SubType != "" {
				typ = typ + ": " + base.SubType
			}
			lists[typ] = append(lists[typ], ItemBaseListEntry{Label: baseLabel(base.Name), Name: base.Name, Base: base})
		}
	}
	typeList := make([]string, 0, len(lists))
	for typ, list := range lists {
		typeList = append(typeList, typ)
		// every base carries a req table, so Lua's req presence branches never run
		sort.SliceStable(list, func(a, b int) bool {
			la, lb := list[a].Base.Req.Level, list[b].Base.Req.Level
			if la == lb {
				return list[a].Name < list[b].Name
			}
			return max(la, 1) > max(lb, 1)
		})
	}
	sort.Strings(typeList)
	return lists, typeList
}

// baseLabel is name:gsub(" %(.+%)","").
func baseLabel(name string) string {
	if i := strings.Index(name, " ("); i >= 0 {
		if j := strings.LastIndex(name, ")"); j > i+2 {
			return name[:i] + name[j+1:]
		}
	}
	return name
}

// ItemBaseLists is data.itemBaseLists: the shown bases by "type" or "type: subType".
func ItemBaseLists() map[string][]ItemBaseListEntry {
	lists, _ := itemBaseListsOnce()
	return lists
}

// ItemBaseTypeList is data.itemBaseTypeList, sorted.
func ItemBaseTypeList() []string {
	_, types := itemBaseListsOnce()
	return types
}

// CraftRarities is the Craft Item popup's rarity list.
//
//pob2:block:src/Classes/ItemsTab.lua:20-26
var CraftRarities = []string{"NORMAL", "MAGIC", "RARE", "UNIQUE", "RELIC"}

// CraftItem is the Craft Item popup's item from the base: raritySel is its 1-based rarity choice, title the Name
// field, shown for rare and up; a nil modSource is modLib.parseMod.
//
//pob2:block:src/Classes/ItemsTab.lua:2764-2830
func CraftItem(raritySel int, title, baseName string, modSource ItemModSource) (*Item, bool) {
	base, ok := poe2.BaseByName(baseName)
	if !ok || base.Hidden || raritySel < 1 || raritySel > len(CraftRarities) {
		return nil, false
	}
	if modSource == nil {
		modSource = ItemModSource(WithParser(ParseMod))
	}
	// item.charmLimit = base.charmLimit and item.spiritValue = base.spiritValue read the base list's row, which has
	// neither, so both stay nil
	item := &Item{ID: -1, Name: base.Name, Base: base, BaseName: base.Name, modSource: modSource}
	item.BuffModLines = []*ModLine{}
	item.EnchantModLines = []*ModLine{}
	item.RuneModLines = []*ModLine{}
	item.ClassRequirementModLines = []*ModLine{}
	item.ImplicitModLines = []*ModLine{}
	item.ExplicitModLines = []*ModLine{}
	item.Sockets = []ItemSocket{}
	item.Runes = []string{}
	if base.Quality != 0 {
		item.Quality = floatp(0)
	}
	if base.SocketLimit != 0 && (base.Weapon != nil || base.Armour != nil || hasTag(base, "wand") || hasTag(base, "staff") || hasTag(base, "sceptre")) {
		for range int(base.SocketLimit) {
			item.Sockets = append(item.Sockets, ItemSocket{Group: 0})
		}
		item.ItemSocketCount = len(item.Sockets)
	}
	if base.Flask != nil || (base.Type == "Jewel" && base.SubType == "Charm") || base.Type == "Charm" {
		if raritySel == 3 {
			raritySel = 2
		}
	}
	if base.Type == "Transcendent Limb" {
		raritySel = 1
	}
	if raritySel == 2 || raritySel == 3 {
		item.Crafted = true
	}
	item.Rarity = CraftRarities[raritySel-1]
	if raritySel >= 3 {
		if strings.TrimSpace(title) != "" {
			item.Title = title
		} else {
			item.Title = "New Item"
		}
	}
	if base.Implicit != "" {
		implicitIndex := 0
		for _, line := range strings.Split(base.Implicit, "\n") {
			if line == "" {
				continue
			}
			ml := &ModLine{Line: line, ModTags: []string{}}
			if implicitIndex < len(base.ImplicitModTypes) {
				ml.ModTags = base.ImplicitModTypes[implicitIndex]
			}
			item.ImplicitModLines = append(item.ImplicitModLines, ml)
			implicitIndex++
		}
	}
	if base.VariantList != nil {
		item.VariantList = append([]string(nil), base.VariantList...)
		item.Variant = intp(1)
		item.baseLines = map[string]baseLine{}
	}
	if base.Type == "Jewel" && base.SubType == "Radius" {
		item.JewelRadiusLabel = "Small"
	}
	item.NormaliseQuality()
	item.BuildAndParseRaw()
	return item, true
}

// CraftDisplayItem is the Craft Item popup's Create: the item opens in the editor.
//
//pob2:block:src/Classes/ItemsTab.lua:2853-2863
func (tab *ItemsTab) CraftDisplayItem(raritySel int, title, baseName string) error {
	item, ok := CraftItem(raritySel, title, baseName, tab.modSource)
	if !ok {
		return itemsEditError("no such base")
	}
	tab.SetDisplayItem(item)
	return nil
}
