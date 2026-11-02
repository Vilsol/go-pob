package calculator2

import (
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/Vilsol/go-pob/data/poe2"
)

// The Items page's unique database (Main.lua's uniqueDB) and its list control (ItemDBControl). The rare database
// needs data.itemMods to craft its templates, which data/poe2 does not carry yet.

func addRuneforgingBaseMappings(item *Item, bases []runeforgingBase) {
	//pob2:block:src/Modules/ItemTools.lua:14-42
	currentBaseLine := item.baseLines[item.BaseName]
	if item.VariantList != nil && !item.UsesVersionedOrGroupedVariants() && currentBaseLine.variantList != nil {
		for variantID := 1; variantID <= len(item.VariantList); variantID++ {
			hasBase := false
			for _, baseLine := range item.baseLines {
				if baseLine.variantList == nil || baseLine.variantList[variantID] {
					hasBase = true
					break
				}
			}
			if !hasBase {
				currentBaseLine.variantList[variantID] = true
			}
		}
	}
	for name, baseLine := range item.baseLines {
		baseLine.baseVariantList = map[int]bool{1: true}
		item.baseLines[name] = baseLine
	}
	item.BaseList = []string{}
	for baseID, base := range bases {
		item.BaseList = append(item.BaseList, base.variantName)
		if baseID > 0 {
			item.baseLines[base.name] = baseLine{line: base.name, baseVariantList: map[int]bool{baseID + 1: true}}
		}
	}
}

func implicitLines(implicit string) []string {
	var out []string
	for _, line := range strings.Split(implicit, "\n") {
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func addRuneforgingImplicits(item *Item, bases []runeforgingBase) {
	//pob2:block:src/Modules/ItemTools.lua:44-82
	originalBaseImplicits := map[string]bool{}
	for _, line := range implicitLines(item.Base.Implicit) {
		originalBaseImplicits[line] = true
	}
	existingItemImplicits := map[string]bool{}
	for _, modLine := range item.ImplicitModLines {
		existingItemImplicits[modLine.Line] = true
	}
	originalImplicitCount := len(item.ImplicitModLines)
	for baseID := 2; baseID <= len(bases); baseID++ {
		baseData, _ := poe2.BaseByName(bases[baseID-1].name)
		craftImplicits := map[string]bool{}
		for implicitIndex, line := range implicitLines(baseData.Implicit) {
			craftImplicits[line] = true
			if !originalBaseImplicits[line] && !existingItemImplicits[line] {
				var modTags []string
				if implicitIndex < len(baseData.ImplicitModTypes) {
					modTags = baseData.ImplicitModTypes[implicitIndex]
				}
				item.ImplicitModLines = append(item.ImplicitModLines, &ModLine{Line: line, Range: floatp(1), ModTags: modTags,
					BaseVariantList: map[int]bool{baseID: true}})
			}
		}
		for modIndex := 0; modIndex < originalImplicitCount; modIndex++ {
			modLine := item.ImplicitModLines[modIndex]
			if originalBaseImplicits[modLine.Line] {
				if modLine.BaseVariantList == nil {
					modLine.BaseVariantList = map[int]bool{1: true}
				}
				if craftImplicits[modLine.Line] {
					modLine.BaseVariantList[baseID] = true
				} else {
					delete(modLine.BaseVariantList, baseID)
				}
			}
		}
	}
}

type runeforgingBase struct{ name, variantName string }

// AddRuneforgingBaseVariants adds a unique's runeforged and runemastered bases as base variants.
//
//pob2:src/Modules/ItemTools.lua:85-109
func AddRuneforgingBaseVariants(item *Item) {
	if item.Rarity != "UNIQUE" || len(item.BaseList) > 0 {
		return
	}
	bases := []runeforgingBase{{name: item.BaseName, variantName: "Regular Base"}}
	seenBases := map[string]bool{item.BaseName: true}
	for _, craft := range poe2.RuneforgingCraftsOf(item.BaseName) {
		if _, ok := poe2.BaseByName(craft.Name); !seenBases[craft.Name] && ok {
			seenBases[craft.Name] = true
			variantName := "Runemastered"
			if strings.Contains(craft.Name, "Runeforged") {
				variantName = "Runeforged"
			}
			bases = append(bases, runeforgingBase{name: craft.Name, variantName: variantName})
		}
	}
	if len(bases) == 1 {
		return
	}
	addRuneforgingBaseMappings(item, bases)
	addRuneforgingImplicits(item, bases)
	item.SelectedBase = intp(1)
	item.BuildAndParseRaw()
}

// ItemDBKind is an item database: "UNIQUE" or "RARE".
type ItemDBKind string

const (
	ItemDBUnique ItemDBKind = "UNIQUE"
	ItemDBRare   ItemDBKind = "RARE"
)

// loadUniqueDB is Main.lua's uniqueDB, by item name.
//
//pob2:block:src/Modules/Main.lua:155-169
func loadUniqueDB() map[string]*Item {
	list := map[string]*Item{}
	modSource := ItemModSource(WithParser(ParseMod))
	for _, kind := range poe2.UniqueDB {
		for _, raw := range kind.Raws {
			newItem := &Item{modSource: modSource, ID: -1}
			newItem.ParseRaw(sanitiseText(raw), "UNIQUE", true)
			if newItem.Base != nil {
				AddRuneforgingBaseVariants(newItem)
				list[newItem.Name] = newItem
			}
		}
	}
	return list
}

var uniqueDB = sync.OnceValue(loadUniqueDB)

// ItemDB is the database's items by name; the rare database is empty until data/poe2 carries data.itemMods.
func ItemDB(kind ItemDBKind) map[string]*Item {
	if kind == ItemDBUnique {
		return uniqueDB()
	}
	return map[string]*Item{}
}

// ItemDBSlots is the Slot dropdown; ItemDBRequirements and ItemDBObtainable the unique database's dropdowns.
//
//pob2:block:src/Classes/ItemDBControl.lua:40-41
var ItemDBSlots = []string{"Any slot", "Weapon 1", "Weapon 2", "Helmet", "Body Armour", "Gloves", "Boots", "Amulet", "Ring", "Belt", "Jewel"}

//pob2:block:src/Classes/ItemDBControl.lua:56-61
var (
	ItemDBRequirements = []string{"Any requirements", "Current level", "Current attributes", "Current useable"}
	ItemDBObtainable   = []string{"Obtainable", "Any source", "Unobtainable", "Vendor Recipe", "Upgraded", "Boss Item", "Corruption"}
)

//pob2:block:src/Classes/ItemDBControl.lua:66-68
var ItemDBSearchModes = []string{"Anywhere", "Names", "Modifiers"}

var leagueNameRe = regexp.MustCompile(` ?([0-9A-Za-z ]+),?`)

// ItemDBLists is the database's League and Type dropdowns, as LoadLeaguesAndTypes fills them.
//
//pob2:src/Classes/ItemDBControl.lua:74-92
func ItemDBLists(kind ItemDBKind) ([]string, []string) {
	leagueFlag := map[string]bool{}
	typeFlag := map[string]bool{}
	for _, item := range ItemDB(kind) {
		if item.League != "" {
			for _, m := range leagueNameRe.FindAllStringSubmatch(item.League, -1) {
				leagueFlag[m[1]] = true
			}
		}
		typeFlag[item.Type] = true
	}
	leagues := append([]string{"Any league", "No league"}, sortedKeysOf(leagueFlag)...)
	types := append([]string{"Any type", "Armour", "Jewellery", "One Handed Melee", "Two Handed Melee"}, sortedKeysOf(typeFlag)...)
	return leagues, types
}

func sortedKeysOf(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// ItemDBFilter is the database's dropdowns, 1-based as their selIndex, and its search box.
type ItemDBFilter struct {
	Slot        int
	Type        int
	League      int
	Requirement int
	Obtainable  int
	Search      string
	SearchMode  int
}

// ItemDBContext is what the requirement filter compares against: the character level and the main output's
// attributes.
type ItemDBContext struct {
	CharacterLevel int
	Str, Dex, Int  float64
}

var (
	itemDBSearchEscapeRe = regexp.MustCompile(`[-.+\[\]$^%?*]`)
	itemDBSlotNumRe      = regexp.MustCompile(` \d`)
)

// asciiLower is string.lower, which lowers ASCII only.
func asciiLower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 32
		}
	}
	return string(b)
}

func deref(v *float64) float64 {
	if v == nil {
		return 0
	}
	return *v
}

//pob2:src/Classes/ItemDBControl.lua:94-208
//nolint:gocyclo // mirrors the Lua branch chain
func doesItemMatchFilters(kind ItemDBKind, item *Item, f ItemDBFilter, ctx ItemDBContext, leagues, types []string) bool {
	if f.Slot > 1 && f.Slot <= len(ItemDBSlots) {
		primarySlot := item.GetPrimarySlot()
		want := ItemDBSlots[f.Slot-1]
		if primarySlot != want && itemDBSlotNumRe.ReplaceAllString(primarySlot, "") != want {
			return false
		}
	}
	if f.Type > 1 {
		switch {
		case f.Type == 2:
			if item.Base.Armour == nil {
				return false
			}
		case f.Type == 3:
			if !(item.Type == "Amulet" || item.Type == "Ring" || item.Type == "Belt") {
				return false
			}
		case f.Type == 4 || f.Type == 5:
			weaponInfo, ok := weaponTypeInfo[item.Type]
			if !(ok && weaponInfo.melee && ((f.Type == 4 && weaponInfo.oneHand) || (f.Type == 5 && !weaponInfo.oneHand))) {
				return false
			}
			if item.Type == "Staff" && item.Base.SubType != "Warstaff" {
				return false
			}
		case f.Type > len(types) || item.Type != types[f.Type-1]:
			return false
		}
	}
	if kind == ItemDBUnique && f.League > 1 {
		if (f.League == 2 && item.League != "") || (f.League > 2 && (item.League == "" || f.League > len(leagues) || !strings.Contains(item.League, leagues[f.League-1]))) {
			return false
		}
	}
	if kind == ItemDBUnique && f.Obtainable != 2 {
		source := item.Source
		obtainable := !(source == "No longer obtainable" || item.League == "Race Events")
		switch {
		case (f.Obtainable <= 1 && !obtainable) || (f.Obtainable == 3 && obtainable):
			return false
		case f.Obtainable == 4 && source != "Vendor Recipe":
			return false
		case f.Obtainable == 5 && !strings.Contains(source, "Upgraded from"):
			return false
		case f.Obtainable == 6 && !strings.Contains(source, "Drops from unique"):
			return false
		case f.Obtainable == 7 && !strings.Contains(source, "Vaal Orb"):
			return false
		}
	}
	if kind == ItemDBUnique && f.Requirement > 1 {
		if (f.Requirement == 2 || f.Requirement == 4) && item.Requirements.Level != nil && *item.Requirements.Level > float64(ctx.CharacterLevel) {
			return false
		}
		if f.Requirement > 2 && (deref(item.Requirements.StrMod) > ctx.Str || deref(item.Requirements.DexMod) > ctx.Dex || deref(item.Requirements.IntMod) > ctx.Int) {
			return false
		}
	}
	searchStr := itemDBSearchEscapeRe.ReplaceAllString(asciiLower(f.Search), "%$0")
	if strings.TrimSpace(searchStr) == "" {
		return true
	}
	found := false
	mode := max(f.SearchMode, 1)
	if mode == 1 || mode == 2 {
		if matchOrPattern(asciiLower(item.Name), searchStr) {
			found = true
		}
	}
	if mode == 1 || mode == 3 {
		for _, list := range [][]*ModLine{item.EnchantModLines, item.RuneModLines, item.ImplicitModLines, item.ExplicitModLines} {
			for _, ml := range list {
				if matchOrPattern(asciiLower(ml.Line), searchStr) {
					found = true
					break
				}
			}
		}
		if !found {
			searchStr = strings.ReplaceAll(searchStr, " ", "")
			for _, m := range item.BaseModList {
				if matchOrPattern(asciiLower(m.Name()), searchStr) {
					found = true
					break
				}
			}
		}
	}
	return found
}

//pob2:src/Modules/Common.lua:1009-1050
func matchOrPattern(s, pattern string) bool {
	for p := range generateOrPatterns(pattern) {
		lit, ok := escapedLiteral(p)
		if ok && strings.Contains(s, lit) {
			return true
		}
	}
	return false
}

func generateOrPatterns(pattern string) map[string]bool {
	type group struct {
		open, close int
		patterns    map[string]bool
	}
	var groups []group
	for k := 0; k < len(pattern); k++ {
		if pattern[k] != '(' {
			continue
		}
		depth := 0
		for e := k; e < len(pattern); e++ {
			switch pattern[e] {
			case '(':
				depth++
			case ')':
				depth--
			}
			if depth == 0 {
				groups = append(groups, group{open: k, close: e, patterns: generateOrPatterns(pattern[k+1 : e])})
				k = e
				break
			}
		}
	}
	first := len(pattern)
	if len(groups) > 0 {
		first = groups[0].open
	}
	generated := []string{pattern[:first]}
	for i, g := range groups {
		next := len(pattern)
		if i+1 < len(groups) {
			next = groups[i+1].open
		}
		regular := pattern[g.close+1 : next]
		var temp []string
		for _, sub := range generated {
			for gp := range g.patterns {
				temp = append(temp, sub+gp+regular)
			}
		}
		generated = temp
	}
	out := map[string]bool{}
	for _, g := range generated {
		for _, p := range strings.Split(g, "|") {
			if p != "" {
				out[p] = true
			}
		}
	}
	return out
}

// escapedLiteral is a search pattern's text; the search box escapes every magic character but the parentheses
// and bars generateOrPatterns splits on.
func escapedLiteral(p string) (string, bool) {
	var b strings.Builder
	for k := 0; k < len(p); k++ {
		switch c := p[k]; c {
		case '%':
			if k+1 >= len(p) {
				return "", false
			}
			k++
			b.WriteByte(p[k])
		case '(', ')':
			return "", false
		default:
			b.WriteByte(c)
		}
	}
	return b.String(), true
}

// FilterItemDB is ListBuilder's list for the name sort: the items the filters keep, by name without a leading "The ".
//
//pob2:block:src/Classes/ItemDBControl.lua:242-248
//pob2:block:src/Classes/ItemDBControl.lua:274-294
func FilterItemDB(kind ItemDBKind, f ItemDBFilter, ctx ItemDBContext) []*Item {
	leagues, types := ItemDBLists(kind)
	var list []*Item
	for _, item := range ItemDB(kind) {
		if doesItemMatchFilters(kind, item, f, ctx, leagues, types) {
			list = append(list, item)
		}
	}
	sortName := func(name string) string { return strings.TrimPrefix(name, "The ") }
	sort.SliceStable(list, func(a, b int) bool { return sortName(list[a].Name) < sortName(list[b].Name) })
	return list
}

// AddDBItem is Ctrl+click on a database row: the item is added, and equipped in its primary slot (the weapon swap
// set's when it is active; with second, the slot's second when the item fits it). Without equip it is only added,
// which the page's Add button offers and PoB-PoE2 does not.
//
//pob2:block:src/Classes/ItemDBControl.lua:348-373
func (tab *ItemsTab) AddDBItem(kind ItemDBKind, name string, equip, second, additionalRingSlot bool) (*Item, error) {
	item := ItemDB(kind)[name]
	if item == nil {
		return nil, itemsEditError("no such item")
	}
	newItem := ParseItem(item.Raw, tab.modSource)
	newItem.NormaliseQuality()
	tab.AddItem(newItem, true, additionalRingSlot)
	slotName := newItem.GetPrimarySlot()
	if slot := tab.Slots[slotName]; slot != nil && equip {
		if slot.WeaponSet == 1 && tab.activeItemSet().UseSecondWeaponSet {
			slotName += " Swap"
		}
		if second {
			altSlot := strings.ReplaceAll(slotName, "1", "2")
			if tab.IsItemValidForSlot(newItem, altSlot, nil) {
				slotName = altSlot
			}
		}
		tab.SetSelItemID(tab.Slots[slotName], newItem.ID)
	}
	tab.PopulateSlots()
	return newItem, nil
}

// OpenDBItem is a double click on a database row: the item opens in the editor.
//
//pob2:block:src/Classes/ItemDBControl.lua:374-381
func (tab *ItemsTab) OpenDBItem(kind ItemDBKind, name string) error {
	item := ItemDB(kind)[name]
	if item == nil {
		return itemsEditError("no such item")
	}
	return tab.CreateDisplayItemFromRaw(item.Raw, true)
}
