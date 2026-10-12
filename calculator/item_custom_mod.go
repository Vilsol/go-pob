package calculator

import (
	"cmp"
	"slices"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/data"
)

//pob:block:src/Modules/Data.lua:1342-1346
var rareLikeUniqueCustomSources = map[string]map[string]bool{
	"dread captain's cutlass": {"ESSENCE": true, "VEILED": true, "CUSTOM": true},
}

func supportsCustomModifiers(item *Item) map[string]bool {
	if !item.RareLikeUnique {
		return nil
	}
	return rareLikeUniqueCustomSources[luaLower(item.Title)]
}

//pob:src/Classes/ItemsTab.lua:991-993
func addCustomShown(item *Item) bool {
	return item.Rarity == RarityMagic || item.Rarity == RarityRare || supportsCustomModifiers(item) != nil
}

//pob:block:src/Classes/ItemsTab.lua:3557-3589
func customModSources(item *Item) []ImplicitSource {
	var out []ImplicitSource
	t := item.Base.Type
	if t != "Tincture" && t != "Graft" {
		if t != "Jewel" {
			out = append(out, ImplicitSource{"MASTER", "Crafting Bench"})
		}
		if t != "Jewel" && t != "Flask" && t != "Graft" {
			out = append(out, ImplicitSource{"ESSENCE", "Essence"}, ImplicitSource{"VEILED", "Veiled"}, ImplicitSource{"BEASTCRAFT", "Beastcraft"})
		}
		if t == "Helmet" || t == "Body Armour" || t == "Gloves" || t == "Boots" {
			out = append(out, ImplicitSource{"NECROPOLIS", "Necropolis"})
		}
		if _, cluster := data.ClusterJewels.Jewels[item.BaseName]; !cluster && t != "Flask" && t != "Graft" {
			out = append(out, ImplicitSource{"DELVE", "Delve"})
		}
		if !item.Crafted {
			out = append(out, ImplicitSource{"PREFIX", "Prefix"}, ImplicitSource{"SUFFIX", "Suffix"})
		}
	}
	if applicable := supportsCustomModifiers(item); applicable != nil {
		out = slices.DeleteFunc(out, func(s ImplicitSource) bool { return !applicable[s.ID] })
	}
	return append(out, ImplicitSource{"CUSTOM", "Custom"})
}

type CustomMod struct {
	ID      string
	Label   string
	Lines   []string
	Type    string
	ModTags []string
}

type CustomModChoices struct {
	Sources []ImplicitSource
	Mods    map[string][]CustomMod
}

//pob:src/Classes/Item.lua:1715-1726
func necropolisModSpawnWeight(item *Item, mod *data.AffixMod) int {
	tags := data.CraftBaseInfo(item.Base.Name).Tags
	for k, key := range mod.WeightKey {
		if slices.Contains(tags, strings.Replace(key, "necropolis_", "", 1)) {
			return mod.WeightVal[k]
		}
	}
	return 0
}

func prefixFirst(a, b data.AffixMod, ai, bi string) int {
	if a.Type != b.Type {
		switch {
		case a.Type == "Prefix" && b.Type == "Suffix":
			return -1
		case b.Type == "Prefix" && a.Type == "Suffix":
			return 1
		}
		return 0
	}
	return strings.Compare(ai, bi)
}

//pob:src/Classes/ItemsTab.lua:3409-3556
func buildCustomMods(raw string, item *Item, sourceID string) []CustomMod {
	type entry struct {
		id    string
		label string
		mod   data.AffixMod
		kind  string
		order string
	}
	var list []entry
	lines := func(m data.AffixMod) string { return strings.Join(m.Lines, "/") }
	switch sourceID {
	case "MASTER":
		excludeGroups := map[string]bool{}
		if st, ok := itemAffixState(raw, item); ok {
			for _, list := range []struct {
				slots []affixSlot
				limit int
			}{{st.prefixes, st.prefixLimit}, {st.suffixes, st.suffixLimit}} {
				for i := 0; i < list.limit && i < len(list.slots); i++ {
					if mod := st.affixes[list.slots[i].modID]; mod != nil {
						excludeGroups[mod.Group] = true
					}
				}
			}
		}
		for i, craft := range data.MasterMods() {
			if slices.Contains(craft.Types, item.Base.Type) && !excludeGroups[craft.Group] {
				list = append(list, entry{"master:" + strconv.Itoa(i), lines(craft.AffixMod) + " ^8(" + craft.Type + ")", craft.AffixMod, "crafted", strconv.Itoa(10000 + i)})
			}
		}
		slices.SortStableFunc(list, func(a, b entry) int { return prefixFirst(a.mod, b.mod, a.order, b.order) })
	case "ESSENCE":
		type essenceEntry struct {
			entry
			kind, tier int
		}
		var es []essenceEntry
		affixes := itemAffixes(item)
		for _, e := range data.Essences() {
			modID := e.Mods[item.Base.Type]
			for _, mod := range affixes {
				if mod.ID == modID {
					es = append(es, essenceEntry{entry{e.ID, e.Name + "   ^8[" + lines(mod) + "]" + " (" + mod.Type + ")", mod, "custom", ""}, e.Type, e.Tier})
					break
				}
			}
		}
		slices.SortStableFunc(es, func(a, b essenceEntry) int {
			if a.kind != b.kind {
				return b.kind - a.kind
			}
			return b.tier - a.tier
		})
		for _, e := range es {
			list = append(list, e.entry)
		}
	case "PREFIX", "SUFFIX":
		for _, mod := range itemAffixes(item) {
			if strings.EqualFold(sourceID, mod.Type) && modSpawnWeight(item, &mod) > 0 {
				list = append(list, entry{mod.ID, mod.Affix + "   ^8[" + lines(mod) + "]", mod, "custom", mod.ID})
			}
		}
		slices.SortStableFunc(list, func(x, y entry) int {
			a, b := x.mod, y.mod
			for i := 0; i < max(len(a.Lines), len(b.Lines)); i++ {
				switch {
				case i >= len(a.Lines):
					return -1
				case i >= len(b.Lines):
					return 1
				case a.StatOrder[i] != b.StatOrder[i]:
					return cmp.Compare(a.StatOrder[i], b.StatOrder[i])
				}
			}
			return b.Level - a.Level
		})
	case "VEILED", "NECROPOLIS", "BEASTCRAFT", "DELVE":
		var mods []data.AffixMod
		switch sourceID {
		case "VEILED":
			mods = data.VeiledMods()
		case "NECROPOLIS":
			mods = data.NecropolisMods()
		case "BEASTCRAFT":
			mods = data.BeastCraftMods()
		case "DELVE":
			mods = itemAffixes(item)
		}
		for _, mod := range mods {
			ok := true
			switch sourceID {
			case "VEILED":
				ok = modSpawnWeight(item, &mod) > 0
			case "NECROPOLIS":
				ok = necropolisModSpawnWeight(item, &mod) > 0
			case "DELVE":
				//pob:block:src/Classes/Item.lua:1728-1730
				ok = (mod.Affix == "Subterranean" || mod.Affix == "of the Underground") && modSpawnWeight(item, &mod) > 0
			}
			if ok {
				list = append(list, entry{mod.ID, lines(mod) + " (" + mod.Type + ")", mod, "custom", mod.ID})
			}
		}
		slices.SortStableFunc(list, func(a, b entry) int { return prefixFirst(a.mod, b.mod, a.order, b.order) })
	}
	out := make([]CustomMod, len(list))
	for k, e := range list {
		out[k] = CustomMod{ID: e.id, Label: e.label, Lines: e.mod.Lines, Type: e.kind, ModTags: e.mod.ModTags}
	}
	return out
}

func ItemCustomModChoices(raw string) (CustomModChoices, bool) {
	item := ParseItem(raw)
	if item.Base == nil {
		return CustomModChoices{}, false
	}
	out := CustomModChoices{Sources: customModSources(item), Mods: map[string][]CustomMod{}}
	for _, s := range out.Sources {
		if s.ID != "CUSTOM" {
			out.Mods[s.ID] = buildCustomMods(raw, item, s.ID)
		}
	}
	return out, true
}

var explicitTailLines = []string{"Split", "Mirrored", "Fractured Item", "Corrupted"}

func appendExplicit(b *implicitBlock, lines []string) {
	at := len(b.after)
	for k, l := range b.after {
		l = strings.TrimSpace(l)
		if slices.Contains(explicitTailLines, l) || strings.HasPrefix(l, "Foil Unique") || strings.Contains(l, "{crucible}") {
			at = k
			break
		}
	}
	b.after = slices.Insert(b.after, at, lines...)
}

func customModLines(m CustomMod, nodes map[string]data.Node) []string {
	out := make([]string, len(m.Lines))
	for k, l := range m.Lines {
		out[k] = "{" + m.Type + "}" + checkLineForAllocates(l, nodes)
		if len(m.ModTags) > 0 {
			out[k] = "{tags:" + strings.Join(m.ModTags, ",") + "}" + out[k]
		}
	}
	return out
}

//pob:src/Classes/ItemsTab.lua:3591-3609
func CustomModText(raw string, sourceID string, modID string, custom string) (string, bool) {
	item := ParseItem(raw)
	if item.Base == nil {
		return raw, false
	}
	b := craftBlock(raw, item)
	if sourceID == "CUSTOM" {
		if strings.TrimSpace(custom) != "" {
			appendExplicit(&b, []string{"{custom}" + custom})
		}
		return b.join(), true
	}
	for _, m := range buildCustomMods(raw, item, sourceID) {
		if m.ID == modID {
			appendExplicit(&b, customModLines(m, nil))
			return b.join(), true
		}
	}
	return raw, false
}

//pob:src/Classes/ItemsTab.lua:3356-3371
func (m *MiscCalculator) CustomModSortValues(raw string, sourceID string, ids []string, stat string) []float64 {
	ps, ok := powerStatByName(stat)
	item := ParseItem(raw)
	out := make([]float64, len(ids))
	if !ok || item.Base == nil {
		return out
	}
	nodes := data.TreeVersions[data.LatestTreeVersion].Tree().Nodes
	mods := buildCustomMods(raw, item, sourceID)
	texts, found := make([]string, len(ids)), make([]bool, len(ids))
	for k, id := range ids {
		for _, cm := range mods {
			if cm.ID != id {
				continue
			}
			b := craftBlock(raw, item)
			appendExplicit(&b, customModLines(cm, nodes))
			texts[k], found[k] = b.join(), true
			break
		}
	}
	return m.sortValues(item.getPrimarySlot(), texts, found, ps)
}
