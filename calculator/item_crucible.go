package calculator

import (
	"slices"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/data"
)

//pob:src/Classes/ItemsTab.lua:1002-1004
func addCrucibleShown(item *Item) bool {
	return item.getPrimarySlot() == "Weapon 1" || item.Base.Type == "Shield" || itemCraftFlags(item).shieldCrucibleTree
}

//pob:src/Classes/Item.lua:2654-2684
func canHaveMod(item *Item, mod *data.AffixMod) bool {
	include := map[string]bool{}
	caster, minion := data.CrucibleTagUniques()
	if slices.Contains(caster, item.Title) {
		include["caster_unique_weapon"] = true
	}
	if slices.Contains(minion, item.Title) {
		include["minion_unique_weapon"] = true
	}
	flags := itemCraftFlags(item)
	switch {
	case flags.supportCrucibleTree:
		k := slices.Index(mod.WeightKey, "crucible_unique_staff")
		return k >= 0 && mod.WeightVal[k] != 0
	case flags.shieldCrucibleTree:
		return modSpawnWeightTags(item, mod, map[string]bool{"crucible_unique_helmet": true, "shield": true}, nil) > 0
	case flags.twoHandSwordCrucibleTree:
		return modSpawnWeightTags(item, mod, map[string]bool{"two_hand_weapon": true}, map[string]bool{"one_hand_weapon": true}) > 0
	}
	return modSpawnWeightTags(item, mod, include, nil) > 0
}

type CrucibleChoice struct {
	ID    string
	Label string
	Lines []string
	Type  string
}

type CrucibleChoices struct {
	Nodes    [][]CrucibleChoice
	Selected []string
}

func crucibleLines(raw string) []string {
	var out []string
	for _, l := range splitRaw(raw) {
		if strings.Contains(l, "{crucible}") {
			out = append(out, rawModText(l))
		}
	}
	return out
}

//pob:block:src/Classes/ItemsTab.lua:3664-3709,src/Classes/ItemsTab.lua:3731-3758
func ItemCrucibleChoices(raw string) (CrucibleChoices, bool) {
	item := ParseItem(raw)
	if item.Base == nil || !addCrucibleShown(item) {
		return CrucibleChoices{}, false
	}
	nodes := data.TreeVersions[data.LatestTreeVersion].Tree().Nodes
	itemModMap := map[string]bool{}
	for _, l := range crucibleLines(raw) {
		itemModMap[l] = true
	}
	type entry struct {
		CrucibleChoice
		order string
	}
	lists := make([][]entry, 5)
	selections := map[int]string{}
	for _, mod := range data.CrucibleMods() {
		if !canHaveMod(item, &mod.AffixMod) {
			continue
		}
		lines := make([]string, len(mod.Lines))
		for k, l := range mod.Lines {
			lines[k] = checkLineForAllocates(l, nodes)
		}
		if len(lines) > 0 && itemModMap[lines[0]] && (len(lines) < 2 || itemModMap[lines[1]]) && len(mod.NodeLocation) > 0 {
			if _, taken := selections[mod.NodeLocation[0]]; taken && len(mod.NodeLocation) > 1 {
				selections[mod.NodeLocation[1]] = mod.ID
			} else {
				selections[mod.NodeLocation[0]] = mod.ID
			}
		}
		for _, loc := range mod.NodeLocation {
			if loc >= 1 && loc <= 5 {
				lists[loc-1] = append(lists[loc-1], entry{CrucibleChoice{ID: mod.ID, Label: strings.Join(lines, "/") + " - Tier: " + strconv.Itoa(mod.Tier), Lines: lines, Type: mod.Type}, mod.ID})
			}
		}
	}
	out := CrucibleChoices{Nodes: make([][]CrucibleChoice, 5), Selected: make([]string, 5)}
	for k, list := range lists {
		slices.SortStableFunc(list, func(a, b entry) int {
			if a.Type != b.Type {
				switch {
				case a.Type == "Spawn" && b.Type == "MergeOnly":
					return -1
				case b.Type == "Spawn" && a.Type == "MergeOnly":
					return 1
				}
				return 0
			}
			return strings.Compare(a.order, b.order)
		})
		for _, e := range list {
			out.Nodes[k] = append(out.Nodes[k], e.CrucibleChoice)
		}
		out.Selected[k] = selections[k+1]
	}
	return out, true
}

//pob:src/Classes/ItemsTab.lua:3710-3730
func CrucibleText(raw string, ids []string) (string, bool) {
	item := ParseItem(raw)
	c, ok := ItemCrucibleChoices(raw)
	if !ok {
		return raw, false
	}
	b := craftBlock(raw, item)
	b.after = slices.DeleteFunc(b.after, func(l string) bool { return strings.Contains(l, "{crucible}") })
	var lines []string
	for k, id := range ids {
		if k >= len(c.Nodes) || id == "" {
			continue
		}
		for _, e := range c.Nodes[k] {
			if e.ID != id {
				continue
			}
			var tags []string
			for _, mod := range data.CrucibleMods() {
				if mod.ID == id {
					tags = mod.ModTags
				}
			}
			for _, l := range e.Lines {
				l = "{crucible}" + l
				if len(tags) > 0 {
					l = "{tags:" + strings.Join(tags, ",") + "}" + l
				}
				lines = append(lines, l)
			}
			break
		}
	}
	appendExplicit(&b, lines)
	return b.join(), true
}
