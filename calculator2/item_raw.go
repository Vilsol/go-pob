package calculator2

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var buildRawRangeRe = regexp.MustCompile(`\(-?[\d.]+--?[\d.]+\)`)

// ItemAffix is one entry of item.prefixes or item.suffixes; Ranges holds a "{range:a,b}" list.
type ItemAffix struct {
	ModID     string
	Range     *float64
	Ranges    []float64
	Fractured bool
}

func sortedIDSpec(ids map[int]bool) string {
	keys := make([]int, 0, len(ids))
	for id := range ids {
		keys = append(keys, id)
	}
	sort.Ints(keys)
	parts := make([]string, len(keys))
	for n, id := range keys {
		parts[n] = strconv.Itoa(id)
	}
	return strings.Join(parts, ",")
}

func affixRaw(kind string, affix ItemAffix) string {
	rng := ""
	switch {
	case affix.Ranges != nil:
		parts := make([]string, len(affix.Ranges))
		for n, r := range affix.Ranges {
			parts[n] = luaNum(r)
		}
		rng = "{range:" + strings.Join(parts, ",") + "}"
	case affix.Range != nil:
		rng = "{range:" + luaNum(round(*affix.Range, 3)) + "}"
	}
	fractured := ""
	if affix.Fractured {
		fractured = "{fractured}"
	}
	return kind + ": " + fractured + rng + affix.ModID
}

//pob2:src/Classes/Item.lua:1923-2000
func (i *Item) writeModLine(rawLines []string, modLine *ModLine) []string {
	line := modLine.Line
	prependToAllLines := func(prefix string) {
		line = prefix + strings.ReplaceAll(line, "\n", "\n"+prefix)
	}
	if modLine.AugmentType != "" || modLine.Rune {
		if modLine.DisplayValueScalar != nil {
			valueScalar := 1.0
			if modLine.ValueScalar != nil {
				valueScalar = *modLine.ValueScalar
			}
			displayValueScalar := valueScalar * *modLine.DisplayValueScalar
			rangeVal := defaultItemAffixQuality
			if modLine.Range != nil {
				rangeVal = *modLine.Range
			}
			line = applyRange(modLine.Line, rangeVal, nil, &displayValueScalar, modLine.CorruptedRange, i.modSource)
		} else {
			line = modLine.Line
		}
	}
	if modLine.Range != nil && buildRawRangeRe.MatchString(line) {
		line = "{range:" + luaNum(round(*modLine.Range, 6)) + "}" + line
	}
	if modLine.CorruptedRange != nil {
		line = "{corruptedRange:" + luaNum(round(*modLine.CorruptedRange, 2)) + "}" + line
	}
	for _, f := range []struct {
		on  bool
		tag string
	}{
		{modLine.Rune, "{rune}"}, {modLine.Enchant, "{enchant}"}, {modLine.Custom, "{custom}"}, {modLine.Fractured, "{fractured}"},
		{modLine.Desecrated, "{desecrated}"}, {modLine.Mutated, "{mutated}"}, {modLine.Disabled, "{disabled}"},
		{modLine.Crafted, "{crafted}"}, {modLine.Prefix, "{prefix}"}, {modLine.Suffix, "{suffix}"}, {modLine.Unscalable, "{unscalable}"},
	} {
		if f.on {
			line = f.tag + line
		}
	}
	hasNewSelection := modLine.VersionList != nil || modLine.VariantGroupList != nil
	if hasNewSelection && len(modLine.ModTags) > 0 {
		line = "{tags:" + strings.Join(modLine.ModTags, ",") + "}" + line
	}
	if modLine.VariantGroupList != nil {
		prependToAllLines("{group:" + sortedIDSpec(modLine.VariantGroupList) + "}")
	}
	if modLine.VariantList != nil {
		prependToAllLines("{variant:" + sortedIDSpec(modLine.VariantList) + "}")
	}
	if modLine.VersionList != nil {
		prependToAllLines("{version:" + sortedIDSpec(modLine.VersionList) + "}")
	}
	if modLine.BaseVariantList != nil {
		prependToAllLines("{base:" + sortedIDSpec(modLine.BaseVariantList) + "}")
	}
	if !hasNewSelection && len(modLine.ModTags) > 0 {
		line = "{tags:" + strings.Join(modLine.ModTags, ",") + "}" + line
	}
	return append(rawLines, line)
}

// sortedBaseLines is pairs(self.baseLines) in the order of the base names; Lua's order is its hash order.
func (i *Item) sortedBaseLines() []*ModLine {
	names := make([]string, 0, len(i.baseLines))
	for name := range i.baseLines {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]*ModLine, len(names))
	for n, name := range names {
		b := i.baseLines[name]
		out[n] = &ModLine{Line: b.line, VariantList: b.variantList, BaseVariantList: b.baseVariantList, VersionList: b.versionList, VariantGroupList: b.variantGroupList}
	}
	return out
}

func joinSockets(n int, s string) string {
	parts := make([]string, n)
	for k := range parts {
		parts[k] = s
	}
	return strings.Join(parts, " ")
}

// BuildRaw is the item's text as PoB-PoE2 writes it into a build file.
//
//pob2:src/Classes/Item.lua:1857-2132
func (i *Item) BuildRaw() string {
	var rawLines []string
	usesVersionedOrGroupedVariants := i.UsesVersionedOrGroupedVariants()
	if len(i.RuneModLines) > 0 {
		i.ApplySocketedRuneDisplayScalars()
	}
	rawLines = append(rawLines, "Rarity: "+i.Rarity)
	if i.Title != "" {
		rawLines = append(rawLines, i.Title, i.BaseName)
	} else {
		rawLines = append(rawLines, i.NamePrefix+i.BaseName+i.NameSuffix)
	}
	if i.CharmLimit != nil {
		rawLines = append(rawLines, "Charm Slots: "+luaNum(*i.CharmLimit))
	}
	if i.SpiritValue != nil {
		rawLines = append(rawLines, "Spirit: "+luaNum(*i.SpiritValue))
	}
	if i.ArmourData != nil {
		for _, t := range [][2]string{{"Armour", "Armour"}, {"Evasion", "Evasion"}, {"EnergyShield", "Energy Shield"}, {"Ward", "Runic Ward"}} {
			if v, ok := i.ArmourData[t[0]]; ok && v > 0 {
				rawLines = append(rawLines, t[1]+": "+luaNum(v))
			}
		}
	}
	if i.UniqueID != "" {
		rawLines = append(rawLines, "Unique ID: "+i.UniqueID)
	}
	if i.League != "" {
		rawLines = append(rawLines, "League: "+i.League)
	}
	if i.Unreleased {
		rawLines = append(rawLines, "Unreleased: true")
	}
	if i.Crafted {
		rawLines = append(rawLines, "Crafted: true")
		for _, affix := range i.Prefixes {
			rawLines = append(rawLines, affixRaw("Prefix", affix))
		}
		for _, affix := range i.Suffixes {
			rawLines = append(rawLines, affixRaw("Suffix", affix))
		}
	}
	if i.Catalyst > 0 {
		rawLines = append(rawLines, "Catalyst: "+catalystList[i.Catalyst-1])
	}
	if i.CatalystQuality != nil {
		rawLines = append(rawLines, "CatalystQuality: "+luaNum(*i.CatalystQuality))
	}
	// No PoE2 item is a cluster jewel, so the cluster jewel lines are never written.
	if i.TalismanTier != nil {
		rawLines = append(rawLines, "Talisman Tier: "+luaNum(*i.TalismanTier))
	}
	if i.ItemLevel != nil {
		rawLines = append(rawLines, "Item Level: "+luaNum(*i.ItemLevel))
	}
	if i.VersionList != nil {
		for _, versionName := range i.VersionList {
			rawLines = append(rawLines, "Version: "+versionName)
		}
		if i.SelectedVersion != nil {
			rawLines = append(rawLines, "Selected Version: "+strconv.Itoa(*i.SelectedVersion))
		}
	}
	if i.BaseList != nil {
		for _, baseName := range i.BaseList {
			rawLines = append(rawLines, "Base Variant: "+baseName)
		}
		if i.SelectedBase != nil {
			rawLines = append(rawLines, "Selected Base Variant: "+strconv.Itoa(*i.SelectedBase))
		}
	}
	variant := func(v *int) string {
		if v == nil {
			return "nil"
		}
		return strconv.Itoa(*v)
	}
	if i.VariantList != nil {
		for _, variantName := range i.VariantList {
			rawLines = append(rawLines, "Variant: "+variantName)
		}
		switch {
		case i.HasIndependentVariants():
			rawLines = append(rawLines, "Selected Variant: "+variant(i.Variant))
		case usesVersionedOrGroupedVariants:
			groupIDs := make([]int, 0, len(i.VariantGroups))
			for id := range i.VariantGroups {
				groupIDs = append(groupIDs, id)
			}
			sort.Ints(groupIDs)
			for _, groupID := range groupIDs {
				if variantID, ok := i.VariantGroupSelections[groupID]; ok {
					rawLines = append(rawLines, "Selected Variant Group: "+strconv.Itoa(groupID)+"="+strconv.Itoa(variantID))
				}
			}
		default:
			rawLines = append(rawLines, "Selected Variant: "+variant(i.Variant))
		}
		for _, bl := range i.sortedBaseLines() {
			if bl.VariantList != nil || bl.VersionList != nil || bl.VariantGroupList != nil || bl.BaseVariantList != nil {
				rawLines = i.writeModLine(rawLines, bl)
			}
		}
		for _, alt := range []struct {
			has   bool
			v     *int
			label string
		}{
			{i.HasAltVariant, i.VariantAlt, ""}, {i.HasAltVariant2, i.VariantAlt2, " Two"}, {i.HasAltVariant3, i.VariantAlt3, " Three"},
			{i.HasAltVariant4, i.VariantAlt4, " Four"}, {i.HasAltVariant5, i.VariantAlt5, " Five"},
		} {
			if !usesVersionedOrGroupedVariants && alt.has {
				rawLines = append(rawLines, "Has Alt Variant"+alt.label+": true", "Selected Alt Variant"+alt.label+": "+variant(alt.v))
			}
		}
		if i.AllowDuplicateVariants {
			rawLines = append(rawLines, "Allow Duplicate Variants: true")
		}
	}
	if i.VariantList == nil {
		for _, bl := range i.sortedBaseLines() {
			if bl.VersionList != nil || bl.VariantGroupList != nil || bl.BaseVariantList != nil {
				rawLines = i.writeModLine(rawLines, bl)
			}
		}
	}
	if i.Quality != nil {
		rawLines = append(rawLines, "Quality: "+luaNum(*i.Quality))
	}
	if i.ItemSocketCount > 0 {
		rawLines = append(rawLines, "Sockets: "+joinSockets(i.ItemSocketCount, "S"))
		for n := 0; n < i.ItemSocketCount; n++ {
			name := "None"
			if n < len(i.Runes) {
				name = i.Runes[n]
			}
			rawLines = append(rawLines, "Rune: "+name)
		}
	}
	if i.JewelSocketCount > 0 {
		rawLines = append(rawLines, "Sockets: "+joinSockets(i.JewelSocketCount, "J"))
	}
	if i.Requirements.Level != nil {
		rawLines = append(rawLines, "LevelReq: "+luaNum(*i.Requirements.Level))
	}
	if i.JewelRadiusLabel != "" {
		rawLines = append(rawLines, "Radius: "+i.JewelRadiusLabel)
	}
	if i.Limit != nil {
		rawLines = append(rawLines, "Limited to: "+luaNum(*i.Limit))
	}
	if i.ClassRestriction != "" {
		rawLines = append(rawLines, "Requires Class "+i.ClassRestriction)
	}
	rawLines = append(rawLines, "Implicits: "+strconv.Itoa(len(i.RuneModLines)+len(i.EnchantModLines)+len(i.ImplicitModLines)))
	for _, list := range [][]*ModLine{i.RuneModLines, i.EnchantModLines, i.ClassRequirementModLines, i.ImplicitModLines, i.ExplicitModLines} {
		for _, modLine := range list {
			rawLines = i.writeModLine(rawLines, modLine)
		}
	}
	if i.Mirrored {
		rawLines = append(rawLines, "Mirrored")
	}
	if i.Sanctified {
		rawLines = append(rawLines, "Sanctified")
	}
	if i.DoubleCorrupted {
		rawLines = append(rawLines, "Twice Corrupted")
	} else if i.Corrupted {
		rawLines = append(rawLines, "Corrupted")
	}
	return strings.Join(rawLines, "\n")
}

//pob2:src/Classes/Item.lua:2134-2137
func (i *Item) BuildAndParseRaw() {
	i.ParseRaw(i.BuildRaw(), "", false)
}
