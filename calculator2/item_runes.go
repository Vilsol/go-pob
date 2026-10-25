package calculator2

import (
	"math"
	"sort"
	"strings"

	"github.com/Vilsol/go-pob/data/poe2"
	"github.com/Vilsol/go-pob/mod"
)

//pob2:block:src/Classes/Item.lua:1397-1407
func getRuneLineParts(modLine string) (string, []float64) {
	var values []float64
	stripped := gsubFunc(runeDigitsRe, modLine, -1, func(g []string) string {
		v, _ := luaToNumber(g[1])
		values = append(values, v)
		return "#"
	})
	if len(values) == 0 {
		values = append(values, 1)
	}
	return stripped, values
}

func valueAt(a []float64, i int) float64 {
	if i < len(a) {
		return a[i]
	}
	return 0
}

//pob2:block:src/Classes/Item.lua:1427-1436
func compareRuneValueSets(a, b []float64) bool {
	for i := 0; i < max(len(a), len(b)); i++ {
		if av, bv := valueAt(a, i), valueAt(b, i); av != bv {
			return av > bv
		}
	}
	return false
}

//pob2:block:src/Classes/Item.lua:1438-1445
func runeValueSetsEqual(a, b []float64) bool {
	for i := 0; i < max(len(a), len(b)); i++ {
		if math.Abs(valueAt(a, i)-valueAt(b, i)) > 1e-9 {
			return false
		}
	}
	return true
}

//pob2:block:src/Classes/Item.lua:1447-1453
func addRuneValueSets(a, b []float64) []float64 {
	out := make([]float64, max(len(a), len(b)))
	for i := range out {
		out[i] = valueAt(a, i) + valueAt(b, i)
	}
	return out
}

//pob2:block:src/Classes/Item.lua:1455-1462
func runeValueSetExceeds(valueSet, target []float64) bool {
	for i := 0; i < max(len(valueSet), len(target)); i++ {
		if valueAt(valueSet, i) > valueAt(target, i)+1e-9 {
			return true
		}
	}
	return false
}

type groupedRune struct {
	name          string
	typ           string
	values        []float64
	effectApplied bool
}

//pob2:block:src/Classes/Item.lua:1464-1497
func findRuneCombination(groupedRunes []*groupedRune, targetValues []float64, maxRunes int, maxRuneCounts map[string]int) (map[int]int, int) {
	var bestCounts map[int]int
	bestCount := -1
	counts := map[int]int{}
	var search func(startIndex, count int, sum []float64)
	search = func(startIndex, count int, sum []float64) {
		if runeValueSetsEqual(sum, targetValues) {
			if bestCount < 0 || count < bestCount {
				bestCount = count
				bestCounts = map[int]int{}
				for k, v := range counts {
					bestCounts[k] = v
				}
			}
			return
		}
		if count >= maxRunes || (bestCount >= 0 && count >= bestCount) {
			return
		}
		for index := startIndex; index < len(groupedRunes); index++ {
			if maxRuneCounts == nil || counts[index] < maxRuneCounts[groupedRunes[index].name] {
				nextSum := addRuneValueSets(sum, groupedRunes[index].values)
				if !runeValueSetExceeds(nextSum, targetValues) {
					counts[index]++
					search(index, count+1, nextSum)
					counts[index]--
				}
			}
		}
	}
	search(0, 0, nil)
	return bestCounts, bestCount
}

// parseRunes rebuilds or infers the socketed runes from the rune lines.
//
//pob2:src/Classes/Item.lua:1386-1627
//nolint:gocyclo // mirrors the Lua block
func (i *Item) parseRunes(mode string) {
	if !(i.Base.Weapon != nil || i.Base.Armour != nil || hasTag(i.Base, "wand") || hasTag(i.Base, "staff") || hasTag(i.Base, "sceptre") || i.ItemSocketCount > 0) {
		i.Sockets = nil
		i.ItemSocketCount = 0
		i.Runes = nil
		return
	}
	shouldFixRunesOnItem := len(i.Runes) == 0
	canRebuildRunes := len(i.Runes) > 0
	for _, r := range i.Runes {
		if r != "None" && len(poe2.RuneModsByName(r)) == 0 {
			canRebuildRunes = false
			break
		}
	}
	if canRebuildRunes {
		disabledRuneLines := map[string]int{}
		for _, ml := range i.RuneModLines {
			if ml.Disabled {
				s, _ := getRuneLineParts(ml.Line)
				disabledRuneLines[s]++
			}
		}
		i.UpdateRunes()
		for _, ml := range i.RuneModLines {
			s, _ := getRuneLineParts(ml.Line)
			if disabledRuneLines[s] > 0 {
				ml.Disabled = true
				disabledRuneLines[s]--
			}
		}
	}

	effects := map[string]float64{"SocketedAugmentItemEffect": 0, "SocketedRuneEffect": 0, "SocketedSoulCoreEffect": 0}
	if mode == "GAME" && shouldFixRunesOnItem {
		for _, lines := range [][]*ModLine{i.EnchantModLines, i.ImplicitModLines, i.ExplicitModLines} {
			for _, eml := range lines {
				effectModList := eml.ModList
				for _, m := range effectModList {
					if _, ok := effects[m.Name()]; ok && m.Type() == mod.TypeIncrease {
						if ranged := getRangedModList(i, eml); ranged != nil {
							effectModList = ranged
						}
						break
					}
				}
				for _, m := range effectModList {
					if _, ok := effects[m.Name()]; ok && m.Type() == mod.TypeIncrease {
						effects[m.Name()] += m.Value().Float() / 100
					}
				}
			}
		}
	}

	statGroupedRunes := map[string][]*groupedRune{}
	broadItemType, specificItemType := i.GetSocketedAugmentTypes()
	// data.itemMods.Runes is walked with pairs(); this walks it sorted by rune and slot.
	for idx := range poe2.RuneMods {
		slotMod := &poe2.RuneMods[idx]
		slotType := slotMod.Slot
		if !(slotType == broadItemType || slotType == specificItemType || (slotMod.Type == "SoulCore" && i.SocketedSoulCoreTypes[slotType])) {
			continue
		}
		effectModifier := effects["SocketedAugmentItemEffect"] + effects["Socketed"+slotMod.Type+"Effect"]
		var valueScalar *float64
		if effectModifier != 0 {
			valueScalar = floatp(1 + effectModifier)
		}
		add := func(line string) {
			if valueScalar != nil {
				bondedPrefix := ""
				if strings.HasPrefix(line, runeBondedPrefix) {
					bondedPrefix = runeBondedPrefix
				}
				line = bondedPrefix + applyRange(strings.TrimPrefix(line, runeBondedPrefix), 1, nil, valueScalar, nil, i.modSource)
			}
			stripped, values := getRuneLineParts(line)
			group := statGroupedRunes[stripped]
			for _, existing := range group {
				if existing.name == slotMod.Rune {
					existing.values = addRuneValueSets(existing.values, values)
					return
				}
			}
			statGroupedRunes[stripped] = append(group, &groupedRune{name: slotMod.Rune, typ: slotMod.Type, values: values, effectApplied: effectModifier != 0})
		}
		for _, line := range slotMod.Lines {
			add(line)
		}
		for _, line := range slotMod.Bonded {
			add(runeBondedPrefix + line)
		}
	}
	for _, runes := range statGroupedRunes {
		sort.SliceStable(runes, func(a, b int) bool { return compareRuneValueSets(runes[a].values, runes[b].values) })
	}

	remainingRunes := i.ItemSocketCount
	inferredRuneCounts := map[string]int{}
	for _, ml := range i.RuneModLines {
		stripped, targetValues := getRuneLineParts(ml.Line)
		groupedRunes := statGroupedRunes[stripped]
		if groupedRunes == nil || ml.Bonded {
			continue
		}
		result, numRunes := findRuneCombination(groupedRunes, targetValues, i.ItemSocketCount, nil)
		if result == nil {
			continue
		}
		addedRuneCount := 0
		for index, r := range groupedRunes {
			addedRuneCount += max(result[index]-inferredRuneCounts[r.name], 0)
		}
		if addedRuneCount > remainingRunes {
			continue
		}
		remainingRunes -= addedRuneCount
		ml.RuneCount = numRunes
		effectApplied := false
		for index, r := range groupedRunes {
			runeCount := result[index]
			if runeCount > 0 {
				ml.AugmentType = r.typ
				effectApplied = effectApplied || r.effectApplied
				if shouldFixRunesOnItem {
					for n := inferredRuneCounts[r.name] + 1; n <= runeCount; n++ {
						i.Runes = append(i.Runes, r.name)
					}
				}
				inferredRuneCounts[r.name] = max(inferredRuneCounts[r.name], runeCount)
			}
		}
		ml.SocketedRuneEffectAlreadyApplied = effectApplied
	}
	for _, ml := range i.RuneModLines {
		if !ml.Bonded {
			continue
		}
		stripped, targetValues := getRuneLineParts(ml.Line)
		groupedRunes := statGroupedRunes[stripped]
		if groupedRunes == nil {
			continue
		}
		result, _ := findRuneCombination(groupedRunes, targetValues, i.ItemSocketCount, inferredRuneCounts)
		if result == nil {
			continue
		}
		for index, r := range groupedRunes {
			if result[index] > 0 {
				ml.AugmentType = r.typ
				ml.SocketedRuneEffectAlreadyApplied = r.effectApplied || ml.SocketedRuneEffectAlreadyApplied
			}
		}
	}
	if shouldFixRunesOnItem && len(i.Runes) > 0 {
		i.UpdateRunes()
	}
}

// UpdateRunes rebuilds the rune mod lines from the item's runes.
//
//pob2:src/Classes/Item.lua:2140-2212
func (i *Item) UpdateRunes() {
	if i.Requirements.NaturalLevel != nil {
		i.Requirements.Level = i.Requirements.NaturalLevel
	}
	i.RuneModLines = nil
	statOrder := map[string]*ModLine{}
	addModLine := func(slotMod *poe2.RuneMod, line string, order float64, bonded bool) {
		displayLine := line
		orderKey := slotMod.Type + ":"
		if bonded {
			displayLine = runeBondedPrefix + line
			orderKey += "Bonded:"
		}
		orderKey += luaNum(order)
		if existing, ok := statOrder[orderKey]; ok {
			start := 0
			existing.Line = gsubFunc(runeDigitsRe, existing.Line, -1, func(g []string) string {
				loc := runeDigitsRe.FindStringSubmatchIndex(displayLine[start:])
				other := displayLine[start+loc[2] : start+loc[3]]
				start += loc[3]
				a, _ := luaToNumber(g[1])
				b, _ := luaToNumber(other)
				return luaNum(a + b)
			})
			parseLine := bondedStripRe.ReplaceAllString(existing.Line, "")
			modList, extra := i.modSource(parseLine)
			if modList == nil {
				modList = []mod.Mod{}
			}
			existing.ModList, existing.Extra = modList, extra
			return
		}
		modList, extra := i.modSource(line)
		if modList == nil {
			modList = []mod.Mod{}
		}
		ml := &ModLine{Line: displayLine, Order: floatp(order), ModList: modList, Extra: extra, Rune: true, Enchant: true, AugmentType: slotMod.Type, Bonded: bonded}
		pos := len(i.RuneModLines)
		for idx, existing := range i.RuneModLines {
			if *existing.Order > order {
				pos = idx
				break
			}
		}
		i.RuneModLines = append(i.RuneModLines, nil)
		copy(i.RuneModLines[pos+1:], i.RuneModLines[pos:])
		i.RuneModLines[pos] = ml
		statOrder[orderKey] = ml
	}
	baseType, specificType := i.GetSocketedAugmentTypes()
	soulCoreTypes := make([]string, 0, len(i.SocketedSoulCoreTypes))
	for t := range i.SocketedSoulCoreTypes {
		soulCoreTypes = append(soulCoreTypes, t)
	}
	sort.Strings(soulCoreTypes)
	for n := 0; n < i.ItemSocketCount; n++ {
		if n >= len(i.Runes) || i.Runes[n] == "None" {
			continue
		}
		name := i.Runes[n]
		var gathered []*poe2.RuneMod
		if r, ok := poe2.RuneModForSlot(name, baseType); ok {
			gathered = append(gathered, r)
		}
		if r, ok := poe2.RuneModForSlot(name, specificType); ok {
			gathered = append(gathered, r)
		}
		for _, sct := range soulCoreTypes {
			if r, ok := poe2.RuneModForSlot(name, sct); ok && r.Type == "SoulCore" {
				gathered = append(gathered, r)
			}
		}
		for _, slotMod := range gathered {
			for idx, line := range slotMod.Lines {
				addModLine(slotMod, line, valueAt(slotMod.StatOrder, idx), false)
			}
			for idx, line := range slotMod.Bonded {
				addModLine(slotMod, line, valueAt(slotMod.BondedStatOrder, idx), true)
			}
		}
	}
}

//pob2:src/Classes/Item.lua:2214-2228
func (i *Item) ApplySocketedRuneDisplayScalars() {
	for _, ml := range i.RuneModLines {
		effectModifier := i.SocketedAugmentItemEffectModifier
		switch ml.AugmentType {
		case "SoulCore":
			effectModifier += i.SocketedSoulCoreEffectModifier
		case "Rune":
			effectModifier += i.SocketedRuneEffectModifier
		}
		if effectModifier != 0 && !ml.SocketedRuneEffectAlreadyApplied {
			ml.DisplayValueScalar = floatp(1 + effectModifier)
		} else {
			ml.DisplayValueScalar = nil
		}
	}
}
