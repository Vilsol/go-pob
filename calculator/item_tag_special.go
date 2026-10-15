package calculator

import (
	"sort"
	"strings"
)

//pob:block:src/Modules/Data.lua:731-755
var itemTagSpecial = map[string]map[string][]string{
	"defence": {},
	"evasion": {
		"ring": {"chance to Evade", "Cannot Evade"},
	},
	"life": {
		"body armour": {"Blood Magic", "Eternal Youth", "Ghost Reaver", "Mind Over Matter", "The Agnostic", "Vaal Pact", "Zealot's Oath", "^Cannot Leech$"},
	},
}

//pob:block:src/Modules/Data.lua:757-853
var itemTagSpecialExclusionPattern = map[string]map[string][]string{
	"defence": {},
	"evasion": {
		"ring": {},
	},
	"life": {
		"amulet":      {"lower Life on Hit", "Spectres' Life", "when on Full Life", "when on Low Life", "^Allocates"},
		"belt":        {"Life as Extra Maximum Energy Shield", "Life Recovery from Flasks is applied to nearby Allies", "Life Flasks gain", "when on Full Life", "when on Low Life", "^Allocates"},
		"body armour": {"Life as Physical Damage", "Life as Extra Maximum Energy Shield", "maximum Life as Fire Damage", "while on Full Life", "while you are on Full Life", "when on Full Life", "when on Low Life", "Gain Maximum Life instead of Maximum Energy Shield", "^Socketed Gems are Supported by Level", "^Allocates", "Void Spawns' Life"},
		"boots":       {"Enemy's Life", "^Enemies Cannot Leech Life", "their Life as Chaos Damage", "when on Full Life", "when on Low Life", "^Allocates"},
		"gloves":      {"maximum Life as Physical Damage", "Traps Cost Life", "when on Full Life", "when on Low Life", "^Allocates"},
		"helmet":      {"Recouped as Life", "Life when you Suppress", "Leech when on Low Life", "while no Life is Reserved", "^Socketed Gems are Supported by Level", "when on Full Life", "when on Low Life", "^Allocates"},
		"ring 1":      {"Energy Shield instead of Life", "increased Damage while Leeching Life", "when on Full Life", "when on Low Life", "^Allocates"},
		"ring 2":      {"Energy Shield instead of Life", "increased Damage while Leeching Life", "when on Full Life", "when on Low Life", "^Allocates"},
		"weapon 1":    {"^Socketed Gems are Supported by Level", "maximum Life as Chaos Damage", "total Maximum Life and Energy Shield as Fire Damage", "Life as Physical Damage", "maximum Life as Fire Damage", "when on Full Life", "when on Low Life", "^Allocates"},
		"weapon 2":    {"maximum Life as Chaos Damage", "^Socketed Gems Cost and Reserve Life", "increased Damage while Leeching Life", "Life as Physical Damage", "maximum Life as Fire Damage", "when on Full Life", "when on Low Life", "^Allocates"},
	},
}

//pob:block:src/Classes/Item.lua:289-333
func findModifierSubstring(explicitLines, otherLines []string, substring, itemSlotName string) bool {
	lines := explicitLines
	if trimmed := strings.ReplaceAll(substring, "explicit ", ""); trimmed != substring {
		substring = trimmed
	} else {
		lines = make([]string, 0, len(explicitLines)+len(otherLines))
		lines = append(lines, explicitLines...)
		lines = append(lines, otherLines...)
	}

	exclusions := itemTagSpecialExclusionPattern[substring][itemSlotName]
	specials := itemTagSpecial[substring][itemSlotName]

	for _, line := range lines {
		if strings.Contains(line, substring) && !strings.Contains(line, substring+" modifier") {
			excluded := false
			for _, pattern := range exclusions {
				if luaFindPattern(line, pattern) {
					excluded = true
					break
				}
			}
			if !excluded {
				return true
			}
		}
		for _, pattern := range specials {
			if luaFindPattern(line, pattern) {
				return true
			}
		}
	}
	return false
}

func luaFindPattern(line, pattern string) bool {
	p := strings.ToLower(pattern)
	anchoredStart := strings.HasPrefix(p, "^")
	anchoredEnd := strings.HasSuffix(p, "$")
	p = strings.TrimSuffix(strings.TrimPrefix(p, "^"), "$")
	switch {
	case anchoredStart && anchoredEnd:
		return line == p
	case anchoredStart:
		return strings.HasPrefix(line, p)
	case anchoredEnd:
		return strings.HasSuffix(line, p)
	default:
		return strings.Contains(line, p)
	}
}

//pob:block:src/Classes/ModStore.lua:691-745

func (a *Actor) EquippedSlots() []string {
	slots := make([]string, 0, len(a.ItemList))
	for slot, item := range a.ItemList {
		if item == nil || item.Type == "Jewel" || item.Type == "Graft" {
			continue
		}
		slots = append(slots, slot)
	}
	sort.Strings(slots)
	return slots
}

func (a *Actor) ItemInSlot(slot string) bool {
	return a.ItemList[slot] != nil
}

func (a *Actor) ItemModifierMatches(slot, searchCond string) bool {
	item := a.ItemList[slot]
	if item == nil {
		return false
	}
	return findModifierSubstring(item.ExplicitModLines, item.OtherModLines, searchCond, strings.ToLower(slot))
}

func (a *Actor) ItemRarityIs(slot, rarity string) bool {
	item := a.ItemList[slot]
	return item != nil && strings.EqualFold(item.Rarity, rarity)
}

func (a *Actor) ItemCorruptedIs(slot string, corrupted bool) bool {
	item := a.ItemList[slot]
	return item != nil && item.Corrupted == corrupted
}

func (a *Actor) ItemNameIs(slot, name string) bool {
	item := a.ItemList[slot]
	return item != nil && strings.EqualFold(item.Name, name)
}

//pob:block:src/Classes/ModStore.lua:707-745

func (i *Item) CfgNameIs(name string) bool {
	return strings.EqualFold(i.Name, name)
}

func (i *Item) CfgRarityIs(rarity string) bool {
	return strings.EqualFold(string(i.Rarity), rarity)
}

func (i *Item) CfgCorruptedIs(corrupted bool) bool {
	return i.Corrupted == corrupted
}

func (i *Item) CfgModifierMatches(searchCond, slot string) bool {
	lower := func(dst []string, lines []ModLine) []string {
		for _, ml := range lines {
			if !i.checkModLineVariant(ml) {
				continue
			}
			dst = append(dst, strings.ToLower(ml.Line))
		}
		return dst
	}
	explicit := lower(nil, i.ExplicitModLines)
	other := lower(nil, i.EnchantModLines)
	other = lower(other, i.ScourgeModLines)
	other = lower(other, i.ImplicitModLines)
	return findModifierSubstring(explicit, other, searchCond, slot)
}
