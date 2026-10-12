package calculator

import (
	"maps"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/pob"
)

type ItemDBKind string

const (
	ItemDBUnique ItemDBKind = "UNIQUE"
	ItemDBRare   ItemDBKind = "RARE"
)

type itemDBList struct {
	items   []*Item
	leagues []string
	types   []string
}

var (
	itemDBOnce  [2]sync.Once
	itemDBLists [2]itemDBList
)

func itemDBIndex(kind ItemDBKind) int {
	if kind == ItemDBRare {
		return 1
	}
	return 0
}

//pob:block:src/Modules/Main.lua:155-192
func loadItemDB(kind ItemDBKind) itemDBList {
	byName := map[string]*Item{}
	if kind == ItemDBUnique {
		for _, e := range data.Uniques() {
			item := &Item{}
			item.parseRaw(data.SanitiseItemText(e.Raw), RarityUnique, true)
			if item.Base != nil {
				byName[item.Name] = item
			}
		}
	} else {
		for _, e := range data.Rares() {
			item := &Item{}
			item.parseRaw(data.SanitiseItemText(e.Raw), RarityRare, true)
			if item.Base == nil {
				continue
			}
			if item.Crafted {
				item = craftDBItem(item)
			}
			byName[item.Name] = item
		}
	}
	out := itemDBList{}
	for _, name := range slices.Sorted(maps.Keys(byName)) {
		out.items = append(out.items, byName[name])
	}
	out.leagues, out.types = loadLeaguesAndTypes(out.items)
	return out
}

//pob:block:src/Modules/Main.lua:173-183
func craftDBItem(item *Item) *Item {
	raw := item.Raw
	if !strings.HasPrefix(raw, "Rarity:") {
		raw = "Rarity: " + string(item.Rarity) + "\n" + raw
	}
	if implicit := data.CraftBaseInfo(item.BaseName).Implicit; implicit != "" && len(item.ImplicitModLines) == 0 {
		lines := strings.Split(implicit, "\n")
		raw = insertImplicits(raw, lines)
	}
	parsed := &Item{}
	parsed.parseRaw(raw, item.Rarity, true)
	st, ok := itemAffixState(raw, parsed)
	if !ok {
		return parsed
	}
	crafted := &Item{}
	crafted.parseRaw(craftText(raw, parsed, st), item.Rarity, true)
	return crafted
}

func insertImplicits(raw string, lines []string) string {
	rows := strings.Split(raw, "\n")
	at := len(rows)
	for k, r := range rows {
		if k >= 3 && !specLineRe.MatchString(r) {
			at = k
			break
		}
	}
	ins := append([]string{"Implicits: " + strconv.Itoa(len(lines))}, lines...)
	return strings.Join(slices.Insert(rows, at, ins...), "\n")
}

var specLineRe = regexp.MustCompile(`^[A-Za-z ]+: `)

func dbItemType(item *Item) string {
	if item.Type == string(data.ThrustingOneHandSword) {
		return string(data.OneHandSword)
	}
	return item.Type
}

func ItemDB(kind ItemDBKind) []*Item {
	k := itemDBIndex(kind)
	itemDBOnce[k].Do(func() { itemDBLists[k] = loadItemDB(kind) })
	return itemDBLists[k].items
}

func resetItemDB() {
	itemDBOnce = [2]sync.Once{}
	itemDBLists = [2]itemDBList{}
}

//pob:src/Classes/ItemDBControl.lua:61-79
func loadLeaguesAndTypes(items []*Item) ([]string, []string) {
	leagueFlag, typeFlag := map[string]bool{}, map[string]bool{}
	for _, item := range items {
		if item.League != "" {
			for _, m := range leagueNameRe.FindAllStringSubmatch(item.League, -1) {
				leagueFlag[m[1]] = true
			}
		}
		typeFlag[dbItemType(item)] = true
	}
	leagues := append([]string{"Any league", "No league"}, slices.Sorted(maps.Keys(leagueFlag))...)
	types := append([]string{"Any type", "Armour", "Jewellery", "One Handed Melee", "Two Handed Melee"}, slices.Sorted(maps.Keys(typeFlag))...)
	return leagues, types
}

var leagueNameRe = regexp.MustCompile(` ?([A-Za-z0-9 ]+),?`)

//pob:src/Classes/ItemDBControl.lua:29
var ItemDBSlots = []string{"Any slot", "Weapon 1", "Weapon 2", "Helmet", "Body Armour", "Gloves", "Boots", "Amulet", "Ring", "Belt", "Jewel", "Flask", "Graft 1", "Graft 2"}

//pob:block:src/Classes/ItemDBControl.lua:44-54
var (
	ItemDBRequirements = []string{"Any requirements", "Current level", "Current attributes", "Current useable"}
	ItemDBObtainable   = []string{"Obtainable", "Any source", "Unobtainable", "Vendor Recipe", "Upgraded", "Boss Item", "Corruption"}
	ItemDBSearchModes  = []string{"Anywhere", "Names", "Modifiers"}
)

func ItemDBLists(kind ItemDBKind) ([]string, []string) {
	ItemDB(kind)
	l := itemDBLists[itemDBIndex(kind)]
	return l.leagues, l.types
}

type ItemDBFilter struct {
	Slot        int
	Type        int
	League      int
	Requirement int
	Obtainable  int
	Search      string
	SearchMode  int
}

type ItemDBContext struct {
	CharacterLevel int
	Str            float64
	Dex            float64
	Int            float64
}

var (
	searchEscapeRe = regexp.MustCompile(`[-.+\[\]$^%?*]`)
	slotNumSuffix  = regexp.MustCompile(` \d`)
)

//pob:src/Classes/ItemDBControl.lua:81-197
//nolint:gocyclo // Mirrors the Lua branch chain.
func doesItemMatchFilters(kind ItemDBKind, item *Item, f ItemDBFilter, ctx ItemDBContext) bool {
	leagues, types := ItemDBLists(kind)
	if f.Slot > 1 && f.Slot <= len(ItemDBSlots) {
		primarySlot := item.getPrimarySlot()
		want := ItemDBSlots[f.Slot-1]
		if primarySlot != want && slotNumSuffix.ReplaceAllString(primarySlot, "") != want {
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
			if t := dbItemType(item); !(t == "Amulet" || t == "Ring" || t == "Belt") {
				return false
			}
		case f.Type == 4 || f.Type == 5:
			info := data.WeaponTypes[data.ItemClassName(dbItemType(item))]
			if !(info != nil && info.Melee && ((f.Type == 4 && info.OneHand) || (f.Type == 5 && !info.OneHand))) {
				return false
			}
		case f.Type > len(types) || dbItemType(item) != types[f.Type-1]:
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
		if (f.Requirement == 2 || f.Requirement == 4) && item.Requirements["level"] > float64(ctx.CharacterLevel) {
			return false
		}
		if f.Requirement > 2 && (item.Requirements["strMod"] > ctx.Str || item.Requirements["dexMod"] > ctx.Dex || item.Requirements["intMod"] > ctx.Int) {
			return false
		}
	}
	searchStr := searchEscapeRe.ReplaceAllString(luaLower(f.Search), "%$0")
	if strings.TrimSpace(searchStr) == "" {
		return true
	}
	found := false
	mode := max(f.SearchMode, 1)
	if mode == 1 || mode == 2 {
		searchName := luaLower(item.Name)
		for _, ml := range item.ExplicitModLines {
			if ml.NewModID != "" {
				searchName = "foulborn " + searchName
				break
			}
		}
		if matchOrPattern(searchName, searchStr) {
			found = true
		}
	}
	if mode == 1 || mode == 3 {
		for _, list := range [][]ModLine{item.EnchantModLines, item.ImplicitModLines} {
			for _, ml := range list {
				if matchOrPattern(luaLower(ml.Line), searchStr) {
					found = true
					break
				}
			}
		}
		for _, ml := range item.ExplicitModLines {
			searchLine := luaLower(ml.Line)
			if ml.NewModID != "" {
				newMod, ok := data.FoulbornModLines(ml.NewModID, false)
				if !ok {
					newMod, ok = data.FoulbornModLines(ml.NewModID, true)
				}
				if ok {
					searchLine += luaLower(strings.Join(newMod, " "))
				}
			}
			if matchOrPattern(searchLine, searchStr) {
				found = true
				break
			}
		}
		if !found {
			searchStr = strings.ReplaceAll(searchStr, " ", "")
			for _, m := range item.BaseModList {
				if matchOrPattern(luaLower(m.Name()), searchStr) {
					found = true
					break
				}
			}
		}
	}
	return found
}

//pob:src/Modules/Common.lua:976-1017
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

//pob:src/Classes/ItemDBControl.lua:205-229
func ItemDBSortStats() []PowerStat {
	return PowerStats()
}

//pob:src/Classes/ItemDBControl.lua:20
func sortName(name string) string {
	return strings.TrimPrefix(name, "The ")
}

//pob:block:src/Classes/ItemDBControl.lua:231-237,src/Classes/ItemDBControl.lua:261-283
func FilterItemDB(kind ItemDBKind, f ItemDBFilter, ctx ItemDBContext) []*Item {
	var list []*Item
	for _, item := range ItemDB(kind) {
		if doesItemMatchFilters(kind, item, f, ctx) {
			list = append(list, item)
		}
	}
	slices.SortStableFunc(list, func(a, b *Item) int {
		return strings.Compare(sortName(a.Name), sortName(b.Name))
	})
	return list
}

//pob:block:src/Classes/ItemDBControl.lua:239-259
func (m *MiscCalculator) ItemDBPower(item *Item, stat PowerStat) float64 {
	power := math.Inf(-1)
	ring := m.additionalRingSlot()
	for _, s := range orderedItemSlots {
		if !IsItemValidForSlot(m.build, item, s.name) || !slotShown(m.build, s, ring) {
			continue
		}
		override := &CalcOverride{RepSlotName: s.name, RepItem: item}
		switch {
		case item.Base.Flask != nil:
			override = &CalcOverride{ToggleFlask: item}
		case item.Base.Tincture != nil:
			override = &CalcOverride{ToggleTincture: item}
		}
		power = math.Max(power, powerStatValue(m.Calc(override), stat))
	}
	return power
}

func ItemDBItem(kind ItemDBKind, name string) *Item {
	items := ItemDB(kind)
	if k := slices.IndexFunc(items, func(it *Item) bool { return it.Name == name }); k >= 0 {
		return items[k]
	}
	return nil
}

//pob:block:src/Classes/ItemDBControl.lua:337-338
func ItemDBText(kind ItemDBKind, name string) (string, bool) {
	item := ItemDBItem(kind, name)
	if item == nil {
		return "", false
	}
	newItem := ParseItem(item.Raw)
	newItem.NormaliseQuality()
	return buildRawText(newItem), true
}

//pob:block:src/Classes/ItemDBControl.lua:335-360,src/Classes/ItemListControl.lua:275-284
func AddDBItem(build *pob.PathOfBuilding, kind ItemDBKind, name string, equip, second bool) (int, bool) {
	text, ok := ItemDBText(kind, name)
	if !ok {
		return 0, false
	}
	id := build.AddItem(text)
	if equip {
		item := ParseItem(text)
		slotName := item.getPrimarySlot()
		if i := slices.IndexFunc(orderedItemSlots, func(s itemSlot) bool { return s.name == slotName }); i >= 0 {
			set, _ := setItems(build)
			if orderedItemSlots[i].weaponSet == 1 && useSecondWeaponSet(set) {
				slotName += " Swap"
			}
			if second {
				if alt := strings.ReplaceAll(slotName, "1", "2"); IsItemValidForSlot(build, item, alt) {
					slotName = alt
				}
			}
			build.SetSlotItem(slotName, id)
		}
	}
	PopulateSlots(build)
	return id, true
}

//pob:block:src/Classes/ItemDBControl.lua:239-281
func (m *MiscCalculator) SortItemDB(list []*Item, stat PowerStat) []float64 {
	power := make(map[*Item]float64, len(list))
	for _, item := range list {
		power[item] = m.ItemDBPower(item, stat)
	}
	slices.SortStableFunc(list, func(a, b *Item) int {
		if power[a] != power[b] {
			if power[a] > power[b] {
				return -1
			}
			return 1
		}
		return strings.Compare(sortName(a.Name), sortName(b.Name))
	})
	out := make([]float64, len(list))
	for k, item := range list {
		out[k] = power[item]
	}
	return out
}

func (i *Item) PrimarySlot() string {
	return i.getPrimarySlot()
}

func PowerStatByName(stat string) (PowerStat, bool) {
	return powerStatByName(stat)
}

func dropText(build *pob.PathOfBuilding, id int, kind ItemDBKind, name string) (string, bool, bool) {
	if id > 0 {
		text, ok := buildItemText(build, id)
		return text, true, ok
	}
	text, ok := ItemDBText(kind, name)
	return text, false, ok
}

func DropItemPrimarySlot(build *pob.PathOfBuilding, id int, kind ItemDBKind, name string) string {
	text, _, ok := dropText(build, id, kind, name)
	item := ParseItem(text)
	if !ok || item.Base == nil {
		return ""
	}
	return item.getPrimarySlot()
}

//pob:block:src/Classes/ItemSlotControl.lua:114-130
func DropItemOnSlot(build *pob.PathOfBuilding, slotName string, id int, kind ItemDBKind, name string) bool {
	text, inBuild, ok := dropText(build, id, kind, name)
	if !ok || !IsItemValidForSlot(build, ParseItem(text), slotName) {
		return false
	}
	if !inBuild {
		id = build.AddItem(text)
	}
	build.SetSlotItem(slotName, id)
	PopulateSlots(build)
	return true
}

//pob:src/Classes/ItemsTab.lua:1613-1642
func EquipItemInSet(build *pob.PathOfBuilding, setID int, id int, kind ItemDBKind, name string, second bool) bool {
	text, inBuild, ok := dropText(build, id, kind, name)
	item := ParseItem(text)
	if !ok || item.Base == nil {
		return false
	}
	slotName := item.getPrimarySlot()
	i := slices.IndexFunc(orderedItemSlots, func(s itemSlot) bool { return s.name == slotName })
	if i < 0 {
		return false
	}
	var set *pob.ItemSet
	for k := range build.Items.ItemSets {
		if n, err := strconv.Atoi(build.Items.ItemSets[k].ID); err == nil && n == setID {
			set = &build.Items.ItemSets[k]
		}
	}
	if set == nil {
		return false
	}
	if orderedItemSlots[i].weaponSet == 1 && useSecondWeaponSet(*set) {
		slotName += " Swap"
	}
	if !inBuild {
		id = build.AddItem(text)
	}
	altSlot := strings.ReplaceAll(slotName, "1", "2")
	if second && IsItemValidForSlot(build, item, altSlot) {
		slotName = altSlot
	}
	if setID == build.Items.ActiveItemSet {
		build.SetSlotItem(slotName, id)
	} else {
		build.SetSetSlotItem(setID, slotName, id)
		if alt := build.SetSlotItemID(setID, altSlot); alt != 0 {
			if altText, ok := buildItemText(build, alt); ok && !IsItemValidForSlot(build, ParseItem(altText), altSlot) {
				build.SetSetSlotItem(setID, altSlot, 0)
			}
		}
	}
	PopulateSlots(build)
	return true
}
