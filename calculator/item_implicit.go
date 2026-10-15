package calculator

import (
	"cmp"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/Vilsol/go-pob/data"
)

//pob:src/Classes/ItemsTab.lua:574-579
func addImplicitShown(item *Item) bool {
	t := item.Base.Type
	corruptible := t != "Flask"
	normalToRare := item.Rarity == RarityNormal || item.Rarity == RarityMagic || item.Rarity == RarityRare
	return t != "Tincture" && t != "Graft" && (corruptible || (t != "Flask" && t != "Jewel" && normalToRare)) &&
		!itemCraftFlags(item).implicitsCannotBeChanged
}

type ImplicitSource struct {
	ID    string
	Label string
}

//pob:block:src/Modules/Main.lua:163-174
var uniqueDBByTitle = sync.OnceValue(func() map[string]*Item {
	out := map[string]*Item{}
	for _, item := range ItemDB(ItemDBUnique) {
		out[luaLower(item.Title)] = item
	}
	return out
})

//pob:block:src/Modules/AddImplicitPopup.lua:161-181
func implicitSources(item *Item) []ImplicitSource {
	var out []ImplicitSource
	t := item.Base.Type
	titleLower := luaLower(item.Title)
	if u := uniqueDBByTitle()[titleLower]; item.Rarity == RarityUnique && u != nil && slices.Contains(data.VestigialUniqueBaseTypes(), t) &&
		(u.Source == "" || titleLower == "stasis prison") {
		out = append(out, ImplicitSource{"VESTIGIAL", "Vestigial"})
	}
	if item.Rarity != RarityUnique && item.Rarity != RarityRelic && (t == "Helmet" || t == "Body Armour" || t == "Gloves" || t == "Boots") {
		if item.Influence.Cleansing {
			out = append(out, ImplicitSource{"EXARCH", "Searing Exarch"})
		}
		if item.Influence.Tangle {
			out = append(out, ImplicitSource{"EATER", "Eater of Worlds"})
		}
	}
	return append(out, ImplicitSource{"CUSTOM", "Custom"})
}

type ImplicitMod struct {
	ID      string
	Label   string
	Lines   []string
	Type    string
	ModTags []string
}

type ImplicitGroup struct {
	Label string
	Mods  []ImplicitMod
}

type ImplicitChoices struct {
	Sources []ImplicitSource
	Groups  map[string][]ImplicitGroup
}

var (
	implicitGroupRangeRe = regexp.MustCompile(`\([\d.]+-[\d.]+\)`)
	implicitGroupNumRe   = regexp.MustCompile(`[\d.]+`)
	trailingDigitsRe     = regexp.MustCompile(`\d+$`)
)

//pob:block:src/Modules/AddImplicitPopup.lua:45-120
func buildImplicitMods(item *Item, sourceID string) []ImplicitGroup {
	type entry struct {
		mod  data.AffixMod
		kind string
	}
	type group struct {
		mod  data.AffixMod
		mods []entry
	}
	var groups []*group
	index := map[string]*group{}
	affixes := slices.Clone(itemAffixes(item))
	slices.SortFunc(affixes, func(a, b data.AffixMod) int { return strings.Compare(a.ID, b.ID) })
	add := func(key string, mod data.AffixMod, kind string) {
		g := index[key]
		if g == nil {
			g = &group{mod: mod}
			index[key] = g
			groups = append(groups, g)
		}
		g.mods = append(g.mods, entry{mod, kind})
	}
	statOrderLess := func(a, b data.AffixMod) (int, bool) {
		for i := 0; i < max(len(a.Lines), len(b.Lines)); i++ {
			switch {
			case i >= len(a.Lines):
				return -1, true
			case i >= len(b.Lines):
				return 1, true
			case a.StatOrder[i] != b.StatOrder[i]:
				return cmp.Compare(a.StatOrder[i], b.StatOrder[i]), true
			}
		}
		return 0, false
	}
	switch sourceID {
	//pob:block:src/Modules/AddImplicitPopup.lua:121-157
	case "VESTIGIAL":
		return vestigialImplicitMods(item)
	case "EXARCH", "EATER":
		for _, mod := range affixes {
			if modSpawnWeight(item, &mod) > 0 && strings.EqualFold(sourceID, mod.Type) {
				add(strings.ReplaceAll(strings.ReplaceAll(mod.Group, "PinnaclePresence", ""), "UniquePresence", ""), mod, strings.ToLower(sourceID))
			}
		}
		slices.SortStableFunc(groups, func(a, b *group) int {
			if c, ok := statOrderLess(a.mod, b.mod); ok {
				return c
			}
			return b.mod.Level - a.mod.Level
		})
		suffix := func(id string) int {
			n, _ := strconv.Atoi(trailingDigitsRe.FindString(id))
			return n
		}
		for _, g := range groups {
			slices.SortStableFunc(g.mods, func(x, y entry) int {
				a, b := x.mod, y.mod
				if a.Group != b.Group {
					switch {
					case strings.Contains(a.Group, "PinnaclePresence"):
						return 1
					case strings.Contains(b.Group, "PinnaclePresence"):
						return -1
					case strings.Contains(a.Group, "UniquePresence"):
						return 1
					}
					return -1
				}
				if len(a.Lines) > 0 && len(b.Lines) > 0 && a.StatOrder[0] == b.StatOrder[0] {
					return suffix(a.ID) - suffix(b.ID)
				}
				if c, ok := statOrderLess(a, b); ok {
					return c
				}
				return b.Level - a.Level
			})
		}
	}
	out := make([]ImplicitGroup, len(groups))
	for k, g := range groups {
		ig := ImplicitGroup{}
		for _, e := range g.mods {
			ig.Mods = append(ig.Mods, ImplicitMod{ID: e.mod.ID, Label: strings.Join(e.mod.Lines, "/"), Lines: e.mod.Lines, Type: e.kind, ModTags: e.mod.ModTags})
		}
		ig.Label = implicitGroupNumRe.ReplaceAllString(implicitGroupRangeRe.ReplaceAllString(ig.Mods[0].Label, "#"), "#")
		out[k] = ig
	}
	return out
}

func vestigialImplicitMods(item *Item) []ImplicitGroup {
	table, _ := data.AffixTable("Vestigial")
	mappings := data.VestigialModMappings()
	var out []ImplicitGroup
	for _, mod := range table {
		uniqueTitle, ok := mappings[mod.ID]
		if !ok {
			continue
		}
		titleLower := luaLower(uniqueTitle)
		unique := uniqueDBByTitle()[titleLower]
		if unique == nil || unique.Base.Type != item.Base.Type || luaLower(item.Title) == titleLower {
			continue
		}
		label := colorVestigial + strings.Join(mod.Lines, "/")
		out = append(out, ImplicitGroup{Label: label, Mods: []ImplicitMod{{ID: mod.ID, Label: label, Lines: mod.Lines, Type: "vestigial", ModTags: mod.ModTags}}})
	}
	slices.SortStableFunc(out, func(a, b ImplicitGroup) int { return strings.Compare(a.Mods[0].ID, b.Mods[0].ID) })
	return out
}

func ItemImplicitChoices(raw string) (ImplicitChoices, bool) {
	item := ParseItem(raw)
	if item.Base == nil {
		return ImplicitChoices{}, false
	}
	out := ImplicitChoices{Sources: implicitSources(item), Groups: map[string][]ImplicitGroup{}}
	for _, s := range out.Sources {
		if s.ID != "CUSTOM" {
			out.Groups[s.ID] = buildImplicitMods(item, s.ID)
		}
	}
	return out, true
}

//pob:block:src/Modules/AddImplicitPopup.lua:186-211,src/Modules/AddImplicitPopup.lua:296-309
func ImplicitText(raw string, sourceID string, modID string, custom string) (string, bool) {
	item := ParseItem(raw)
	if item.Base == nil {
		return raw, false
	}
	if sourceID == "CUSTOM" {
		b := craftBlock(raw, item)
		if strings.TrimSpace(custom) != "" {
			b.implicits = append(b.implicits, "{custom}"+custom)
		}
		return b.join(), true
	}
	return implicitText(raw, item, buildImplicitMods(item, sourceID), modID)
}

func implicitText(raw string, item *Item, groups []ImplicitGroup, modID string) (string, bool) {
	b := craftBlock(raw, item)
	for _, g := range groups {
		for _, m := range g.Mods {
			if m.ID != modID {
				continue
			}
			tags := m.ModTags
			lines := make([]string, len(m.Lines))
			for k, l := range m.Lines {
				lines[k] = "{" + m.Type + "}" + l
				if len(tags) > 0 {
					lines[k] = "{tags:" + strings.Join(tags, ",") + "}" + lines[k]
				}
			}
			//pob:block:src/Modules/AddImplicitPopup.lua:186-211
			if m.Type == "vestigial" {
				b.implicits = nil
			}
			if m.Type == "exarch" || m.Type == "eater" {
				for k, l := range b.implicits {
					if strings.Contains(l, "{"+m.Type+"}") {
						for i, line := range lines {
							if k+i < len(b.implicits) {
								b.implicits[k+i] = line
							} else {
								b.implicits = append(b.implicits, line)
							}
						}
						return b.join(), true
					}
				}
			}
			b.implicits = append(b.implicits, lines...)
			return b.join(), true
		}
	}
	return raw, false
}

//pob:src/Modules/AddImplicitPopup.lua:212-225
func (m *MiscCalculator) ImplicitSortValues(raw string, sourceID string, ids []string, stat string) []float64 {
	ps, ok := powerStatByName(stat)
	item := ParseItem(raw)
	out := make([]float64, len(ids))
	if !ok || item.Base == nil {
		return out
	}
	texts, found := implicitSortTexts(raw, item, sourceID, ids)
	return m.sortValues(item.getPrimarySlot(), texts, found, ps)
}

func implicitSortTexts(raw string, item *Item, sourceID string, ids []string) ([]string, []bool) {
	texts, found := make([]string, len(ids)), make([]bool, len(ids))
	if sourceID == "CUSTOM" {
		for k, id := range ids {
			texts[k], found[k] = ImplicitText(raw, sourceID, id, "")
		}
		return texts, found
	}
	groups := buildImplicitMods(item, sourceID)
	for k, id := range ids {
		texts[k], found[k] = implicitText(raw, ParseItem(raw), groups, id)
	}
	return texts, found
}
