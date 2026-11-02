package calculator2

import (
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/data/poe2"
)

var (
	affixValueParenRe = regexp.MustCompile(`-?\d+\.?\d*\(`)
	affixValueRe      = regexp.MustCompile(`-?\d+\.?\d*`)
	affixCombineNumRe = regexp.MustCompile(`\d+`)
)

type pendingAffix struct {
	modID  string
	suffix bool
}

// itemAffixesFor is self.affixes for a base.
//
//pob2:block:src/Classes/Item.lua:1101-1103
func itemAffixesFor(base *poe2.ItemBase) []poe2.ItemMod {
	if base.SubType != "" {
		if mods := poe2.ItemModsOf(base.Type + base.SubType); mods != nil {
			return mods
		}
	}
	if mods := poe2.ItemModsOf(base.Type); mods != nil {
		return mods
	}
	return poe2.ItemModsOf("Item")
}

// affix is self.affixes[modId].
func (i *Item) affix(modID string) *poe2.ItemMod {
	n := sort.Search(len(i.affixes), func(n int) bool { return i.affixes[n].ID >= modID })
	if n < len(i.affixes) && i.affixes[n].ID == modID {
		return &i.affixes[n]
	}
	return nil
}

// Affixes is self.affixes, sorted by mod id.
func (i *Item) Affixes() []poe2.ItemMod {
	return i.affixes
}

//pob2:src/Classes/Item.lua:1838-1855
func (i *Item) GetModSpawnWeight(mod *poe2.ItemMod, includeTags, excludeTags map[string]bool) float64 {
	weight := 0.0
	if i.Base != nil {
		for n, key := range mod.WeightKey {
			if hasTag(i.Base, key) || includeTags[key] && !excludeTags[key] {
				weight = mod.WeightVal[n]
				break
			}
		}
		for n, key := range mod.WeightMultiplierKey {
			if (hasTag(i.Base, key) || includeTags[key]) && !excludeTags[key] {
				weight = weight * mod.WeightMultiplierVal[n] / 100
				break
			}
		}
	}
	return weight
}

// findPendingAffixes is the advanced copy header's affix lookup by mod name.
//
//pob2:block:src/Classes/Item.lua:722-753
func (i *Item) findPendingAffixes(modName string) bool {
	i.pendingAffixList = nil
	var backupAffixList []pendingAffix
	// Lua walks self.affixes in pairs() order; this walks it by mod id
	for n := range i.affixes {
		modData := &i.affixes[n]
		// these can produce false positives, and only ever exist on the monk glove base
		if strings.HasPrefix(modData.ID, "HandWraps") && !strings.Contains(i.Name, "Fists of Stone") {
			continue
		}
		if modData.Affix != modName {
			continue
		}
		list := &backupAffixList
		if i.GetModSpawnWeight(modData, nil, nil) > 0 {
			list = &i.pendingAffixList
		}
		switch modData.Type {
		case "Prefix":
			*list = append(*list, pendingAffix{modID: modData.ID})
		case "Suffix":
			*list = append(*list, pendingAffix{modID: modData.ID, suffix: true})
		}
	}
	if len(i.pendingAffixList) == 0 && len(backupAffixList) > 0 {
		i.pendingAffixList = backupAffixList
	}
	return len(i.pendingAffixList) > 0
}

// takePendingAffix records the affix an advanced copy line belongs to and strips its roll ranges.
//
//pob2:block:src/Classes/Item.lua:1149-1197
func (i *Item) takePendingAffix(line string, ml *ModLine) string {
	if len(i.pendingAffixList) > 1 {
		// Probably a conqueror or Essence mod since the mod name is the same for all of them
		// Try to match the line against one of the mods there
		rangeLine := affixValueParenRe.ReplaceAllString(line, "(")
		valueStrippedLine := affixValueRe.ReplaceAllString(rangeLine, "#")
		var exactAffix, fallbackAffix *pendingAffix
		for n := range i.pendingAffixList {
			pending := &i.pendingAffixList[n]
			if modData := i.affix(pending.modID); modData != nil {
				for _, modDataLine := range modData.Lines {
					if line == modDataLine || rangeLine == modDataLine {
						exactAffix = pending
						break
					}
					if fallbackAffix == nil && valueStrippedLine == affixValueRe.ReplaceAllString(modDataLine, "#") {
						fallbackAffix = pending
					}
				}
			}
			if exactAffix != nil {
				break
			}
		}
		pick := i.pendingAffixList[0]
		if exactAffix != nil {
			pick = *exactAffix
		} else if fallbackAffix != nil {
			pick = *fallbackAffix
		}
		i.pendingAffixList = []pendingAffix{pick}
	}
	// Use rolling Delta/Range in case one range is 1-3 and another is 1-100 so we get the finest precision possible
	bestPrecisionDelta, bestPrecisionRange := -1.0, -1.0
	var rollRanges []float64
	pending := i.pendingAffixList[0]
	if affixMod := i.affix(pending.modID); affixMod != nil {
		ml.Order = floatp(affixMod.StatOrder[0])
	} else {
		ml.Order = nil
	}
	for _, g := range advRollRe.FindAllStringSubmatch(line, -1) {
		value, rng := g[1], g[2]
		mm := advRollSplitRe.FindStringSubmatch(rng)
		lo, _ := luaToNumber(mm[1])
		hi, _ := luaToNumber(mm[2])
		v, _ := luaToNumber(value)
		delta := hi - lo
		if delta > 0 {
			rollRanges = append(rollRanges, round((v-lo)/delta, 6))
		} else {
			rollRanges = append(rollRanges, 0.5)
		}
		if re, ok := luaPattern(luaPatternQuoteNumber(value) + "%(" + strings.ReplaceAll(rng, "-", "%-") + "%)"); ok {
			line = re.ReplaceAllLiteralString(line, value)
		}
		if delta > bestPrecisionDelta {
			bestPrecisionRange = round((v-lo)/delta, 3)
			bestPrecisionDelta = delta
		}
	}
	affix := ItemAffix{ModID: pending.modID, Fractured: ml.Fractured}
	switch {
	case len(rollRanges) > 1:
		affix.Ranges = rollRanges
	case bestPrecisionDelta > 0:
		affix.Range = floatp(bestPrecisionRange)
	default:
		affix.Range = floatp(0.5)
	}
	if pending.suffix {
		i.Suffixes = append(i.Suffixes, affix)
	} else {
		i.Prefixes = append(i.Prefixes, affix)
	}
	i.pendingAffixList = nil
	return line
}

// affixLimitFromLine is the "N prefix/suffix modifiers allowed" count.
//
//pob2:block:src/Classes/Item.lua:1263-1266
func affixLimitFromLine(limit *int, lineLower string, plus, minus *regexp.Regexp) *int {
	n := 0
	if limit != nil {
		n = *limit
	}
	if m := plus.FindStringSubmatch(lineLower); m != nil {
		v, _ := strconv.Atoi(m[1])
		n += v
	}
	if m := minus.FindStringSubmatch(lineLower); m != nil {
		v, _ := strconv.Atoi(m[1])
		n -= v
	}
	return &n
}

// listLimit is `list.limit or (self.affixLimit / 2)`.
func (i *Item) listLimit(suffix bool) int {
	if suffix {
		if i.SuffixLimit != nil {
			return *i.SuffixLimit
		}
	} else if i.PrefixLimit != nil {
		return *i.PrefixLimit
	}
	return i.AffixLimit / 2
}

func (i *Item) affixList(suffix bool) *[]ItemAffix {
	if suffix {
		return &i.Suffixes
	}
	return &i.Prefixes
}

func clampLimit(limit *int, add, hi int) *int {
	n := add
	if limit != nil {
		n += *limit
	}
	n = max(min(n, hi), 0)
	return &n
}

//pob2:block:src/Classes/Item.lua:1751-1792
func (i *Item) applyAffixLimit() {
	i.AffixLimit = 0
	if !i.Crafted {
		return
	}
	switch {
	case i.affixes == nil:
		i.Crafted = false
	case i.Rarity == "MAGIC":
		if i.PrefixLimit != nil || i.SuffixLimit != nil {
			i.PrefixLimit = clampLimit(i.PrefixLimit, 1, 2)
			i.SuffixLimit = clampLimit(i.SuffixLimit, 1, 2)
			i.AffixLimit = *i.PrefixLimit + *i.SuffixLimit
		} else {
			i.AffixLimit = 2
		}
	case i.Rarity == "RARE":
		i.AffixLimit = 6
		if i.Type == "Jewel" && !(i.Base.SubType == "Abyss" && i.Corrupted) {
			i.AffixLimit = 4
		}
		if i.PrefixLimit != nil || i.SuffixLimit != nil {
			i.PrefixLimit = clampLimit(i.PrefixLimit, i.AffixLimit/2, i.AffixLimit)
			i.SuffixLimit = clampLimit(i.SuffixLimit, i.AffixLimit/2, i.AffixLimit)
			i.AffixLimit = *i.PrefixLimit + *i.SuffixLimit
		}
	default:
		i.Crafted = false
	}
	if !i.Crafted {
		return
	}
	for _, suffix := range []bool{false, true} {
		list := i.affixList(suffix)
		for n := range i.listLimit(suffix) {
			if n >= len(*list) {
				*list = append(*list, ItemAffix{ModID: "None"})
			} else if (*list)[n].ModID != "None" && i.affix((*list)[n].ModID) == nil {
				for m := range i.affixes {
					if (*list)[n].ModID == i.affixes[m].Affix {
						(*list)[n].ModID = i.affixes[m].ID
						break
					}
				}
				if i.affix((*list)[n].ModID) == nil {
					(*list)[n].ModID = "None"
				}
			}
		}
	}
}

// Craft rebuilds the explicit modifiers from the item's affixes.
//
//pob2:src/Classes/Item.lua:2257-2321
func (i *Item) Craft() {
	// Save off any custom mods so they can be re-added at the end
	var savedMods []*ModLine
	for _, ml := range i.ExplicitModLines {
		if ml.Custom {
			savedMods = append(savedMods, ml)
		}
	}
	i.ExplicitModLines = []*ModLine{}
	i.NamePrefix = ""
	i.NameSuffix = ""
	i.Requirements.Level = floatp(math.Max(i.Base.Req.Level, i.Requirements.RuneLevel))
	statOrder := map[float64]*ModLine{}
	for _, suffix := range []bool{false, true} {
		list := i.affixList(suffix)
		for n := range i.listLimit(suffix) {
			if n >= len(*list) {
				*list = append(*list, ItemAffix{ModID: "None"})
			}
			affix := (*list)[n]
			mod := i.affix(affix.ModID)
			if mod == nil {
				continue
			}
			switch mod.Type {
			case "Prefix":
				i.NamePrefix = mod.Affix + " " + i.NamePrefix
			case "Suffix":
				i.NameSuffix = i.NameSuffix + " " + mod.Affix
			}
			i.Requirements.Level = floatp(math.Max(*i.Requirements.Level, math.Floor(mod.Level*0.8)))
			for k, line := range mod.Lines {
				line = applyAffixRange(line, affix, i.modSource)
				order := mod.StatOrder[k]
				if prev := statOrder[order]; prev != nil {
					// Combine stats
					start := 0
					prev.Line = affixCombineNumRe.ReplaceAllStringFunc(prev.Line, func(num string) string {
						loc := affixCombineNumRe.FindStringIndex(line[start:])
						if loc == nil {
							return num
						}
						other := line[start+loc[0] : start+loc[1]]
						start += loc[1]
						a, _ := strconv.ParseFloat(num, 64)
						b, _ := strconv.ParseFloat(other, 64)
						return luaNum(a + b)
					})
				} else {
					ml := &ModLine{Line: line, Order: floatp(order), ModTags: mod.ModTags, Unscalable: mod.Unscalable, Fractured: affix.Fractured}
					if ml.ModTags == nil {
						ml.ModTags = []string{}
					}
					ml.setFlag(strings.ToLower(mod.Type))
					at := len(i.ExplicitModLines)
					for l, existing := range i.ExplicitModLines {
						if existing.Order != nil && *existing.Order > order {
							at = l
							break
						}
					}
					i.ExplicitModLines = append(i.ExplicitModLines, nil)
					copy(i.ExplicitModLines[at+1:], i.ExplicitModLines[at:])
					i.ExplicitModLines[at] = ml
					statOrder[order] = ml
				}
			}
		}
	}

	// Restore the custom mods
	i.ExplicitModLines = append(i.ExplicitModLines, savedMods...)
	if len(i.ExplicitModLines) > 1 {
		sortCraftedModLines(i.ExplicitModLines)
	}

	i.BuildAndParseRaw()
}

// applyAffixRange is itemLib.applyRange(line, affix.range or 0.5).
func applyAffixRange(line string, affix ItemAffix, modSource ItemModSource) string {
	if affix.Ranges != nil {
		return applyRange(line, 0, affix.Ranges, nil, nil, modSource)
	}
	r := defaultItemAffixQuality
	if affix.Range != nil {
		r = *affix.Range
	}
	return applyRange(line, r, nil, nil, nil, modSource)
}
