package calculator2

import (
	"math"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/data/poe2"
)

var (
	affixHaveRangeRe   = regexp.MustCompile(`\(-?[\d.]+--?[\d.]+\)`)
	affixSeriesNumRe   = regexp.MustCompile(`-?\d+\.?\d*`)
	affixMinMaxRe      = regexp.MustCompile(`\((\d[\d.]*)-(\d[\d.]*)\)`)
	affixTierNumRe     = regexp.MustCompile(`\d[\d.]*`)
	timeLostRadiusText = " Passive Skills in Radius also grant"
)

// AffixListEntry is one entry of an affix drop-down: a single mod (ModID set) or a series of tiers, lowest first.
type AffixListEntry struct {
	Label     string
	ModList   []string
	ModID     string
	HaveRange bool
	statOrder []float64
}

// AffixControl is one of the editor's affix drop-downs with its roll slider. List[0] is "None".
type AffixControl struct {
	OutputTable string
	OutputIndex int
	List        []AffixListEntry
	SelIndex    int
	SliderShown bool
	SliderVal   float64
	// DivCount is the slider's tier count; 0 is nil
	DivCount int
}

// AffixControls is UpdateAffixControls over the editor's item: one control per affix slot, prefixes first.
//
//pob2:src/Classes/ItemsTab.lua:2232-2246
func (tab *ItemsTab) AffixControls() []AffixControl {
	item := tab.DisplayItem
	if item == nil || !item.Crafted {
		return nil
	}
	prefixLimit := item.listLimit(false)
	controls := make([]AffixControl, 0, item.AffixLimit)
	for i := 1; i <= item.AffixLimit; i++ {
		if i <= prefixLimit {
			controls = append(controls, updateAffixControl(item, "Prefix", "prefixes", i))
		} else {
			controls = append(controls, updateAffixControl(item, "Suffix", "suffixes", i-prefixLimit))
		}
	}
	return controls
}

func affixAt(item *Item, outputTable string, outputIndex int) *ItemAffix {
	list := item.affixList(outputTable == "suffixes")
	for len(*list) < outputIndex {
		*list = append(*list, ItemAffix{ModID: "None"})
	}
	return &(*list)[outputIndex-1]
}

//pob2:src/Classes/ItemsTab.lua:2368-2518
func updateAffixControl(item *Item, affixType, outputTable string, outputIndex int) AffixControl {
	extraTags := map[string]bool{}
	excludeGroups := map[string]bool{}
	for _, table := range []string{"prefixes", "suffixes"} {
		list := *item.affixList(table == "suffixes")
		for index := 1; index <= item.listLimit(table == "suffixes"); index++ {
			if index != outputIndex || table != outputTable {
				if index > len(list) {
					continue
				}
				if mod := item.affix(list[index-1].ModID); mod != nil {
					if mod.Group != "" {
						excludeGroups[mod.Group] = true
					}
					for _, tag := range mod.Tags {
						extraTags[tag] = true
					}
				}
			}
		}
	}
	var affixList []*poe2.ItemMod
	for n := range item.affixes {
		mod := &item.affixes[n]
		if mod.Type == affixType && !excludeGroups[mod.Group] && item.GetModSpawnWeight(mod, extraTags, nil) > 0 {
			affixList = append(affixList, mod)
		}
	}
	sort.SliceStable(affixList, func(a, b int) bool {
		modA, modB := affixList[a], affixList[b]
		for i := range max(len(modA.Lines), len(modB.Lines)) {
			switch {
			case i >= len(modA.Lines):
				return true
			case i >= len(modB.Lines):
				return false
			case modA.StatOrder[i] != modB.StatOrder[i]:
				return modA.StatOrder[i] < modB.StatOrder[i]
			}
		}
		return modA.Level > modB.Level
	})
	control := AffixControl{
		OutputTable: outputTable,
		OutputIndex: outputIndex,
		SelIndex:    1,
		List:        []AffixListEntry{{Label: "None"}},
		SliderVal:   defaultItemAffixQuality,
	}
	selAffix := affixAt(item, outputTable, outputIndex).ModID
	if item.Type == "Jewel" && item.Base.SubType != "Abyss" {
		for i, mod := range affixList {
			if selAffix == mod.ID {
				control.SelIndex = i + 2
			}
			modString := strings.Join(mod.Lines, "/")
			label := modString
			if item.Type == "Flask" {
				label = mod.Affix + "   ^8[" + modString + "]"
			}
			// shorten time-lost jewel affix labels to fit the dropdown
			if strings.Contains(item.BaseName, "Time-Lost") {
				label = strings.ReplaceAll(label, timeLostRadiusText, ":")
			}
			control.List = append(control.List, AffixListEntry{
				Label:     label,
				ModList:   []string{mod.ID},
				ModID:     mod.ID,
				HaveRange: affixHaveRangeRe.MatchString(modString),
			})
		}
	} else {
		var lastSeries *AffixListEntry
		for _, mod := range affixList {
			if lastSeries == nil || !slices.Equal(lastSeries.statOrder, mod.StatOrder) {
				modString := strings.Join(mod.Lines, "/")
				control.List = append(control.List, AffixListEntry{
					Label:     modString,
					ModList:   []string{},
					HaveRange: affixHaveRangeRe.MatchString(modString),
					statOrder: mod.StatOrder,
				})
				lastSeries = &control.List[len(control.List)-1]
			}
			if selAffix == mod.ID {
				control.SelIndex = len(control.List)
			}
			lastSeries.ModList = append([]string{mod.ID}, lastSeries.ModList...)
			if len(lastSeries.ModList) == 2 {
				lastSeries.Label = affixSeriesNumRe.ReplaceAllString(affixHaveRangeRe.ReplaceAllString(lastSeries.Label, "#"), "#")
				lastSeries.HaveRange = true
			}
		}
	}
	// the crafting sort's Default entry has no stat, so the power sort never runs
	if sel := control.List[control.SelIndex-1]; sel.HaveRange {
		divCount := len(sel.ModList)
		index := slices.Index(sel.ModList, selAffix) + 1
		affix := affixAt(item, outputTable, outputIndex)
		affixRange := defaultItemAffixQuality
		switch {
		case affix.Ranges != nil:
			affixRange = affix.Ranges[0]
		case affix.Range != nil:
			affixRange = *affix.Range
		}
		r := math.Min(1, math.Max(0, affixRange))
		// Avoid exact integer boundary that slider:GetDivVal's ceil would assign to the previous segment
		if r == 0 && index > 1 {
			r = 1e-4
		}
		control.SliderVal = (float64(index) - 1 + r) / float64(divCount)
		if divCount != 1 {
			control.DivCount = divCount
		}
		control.SliderShown = true
	}
	return control
}

// sliderDivVal is SliderControl:GetDivVal.
//
//pob2:src/Classes/SliderControl.lua:59-67
func sliderDivVal(val float64, divCount int) (int, float64) {
	if divCount > 1 {
		divIndex := max(int(math.Ceil(val*float64(divCount))), 1)
		return divIndex, float64(val*float64(divCount)) - float64(divIndex) + 1
	}
	return 1, val
}

// verifyRange flips range if it will form discontinuous values.
//
//pob2:block:src/Classes/ItemsTab.lua:892-931
func verifyRange(item *Item, r float64, index int, modList []string) float64 {
	var priorMod, nextMod *poe2.ItemMod
	if index-1 > 0 {
		priorMod = item.affix(modList[index-2])
	}
	if index+1 < len(modList) {
		nextMod = item.affix(modList[index])
	}
	// gets first valid range from a mod
	getMinMax := func(mod *poe2.ItemMod) (float64, float64, bool) {
		for _, line := range mod.Lines {
			if m := affixMinMaxRe.FindStringSubmatch(line); m != nil {
				lo, err1 := strconv.ParseFloat(m[1], 64)
				hi, err2 := strconv.ParseFloat(m[2], 64)
				if err1 == nil && err2 == nil {
					return lo, hi, true
				}
				return 0, 0, false
			}
		}
		return 0, 0, false
	}
	// assumes all pairs are ordered the same
	flipRange := func(modA, modB *poe2.ItemMod) bool {
		if modA == nil || modB == nil {
			return false
		}
		minA, maxA, okA := getMinMax(modA)
		minB, maxB, okB := getMinMax(modB)
		if !okA || !okB {
			return false
		}
		// if the mod goes in steps that aren't 1, then the code below this doesn't work
		if minA == math.Floor(minA) && maxA == math.Floor(maxA) && minB == math.Floor(minB) && maxB == math.Floor(maxB) {
			if minA < minB { // ascending
				return minA+1 == maxB
			}
			return minA-1 == maxB // descending
		}
		return false
	}
	if priorMod != nil {
		if flipRange(priorMod, item.affix(modList[index-1])) {
			r = 1 - r
		}
	} else if nextMod != nil {
		if flipRange(item.affix(modList[index-1]), nextMod) {
			r = 1 - r
		}
	}
	return r
}

// SelectDisplayItemAffix is affix drop-down #control's selection of List[entry-1], with its slider at sliderVal.
//
//pob2:block:src/Classes/ItemsTab.lua:932-947
func (tab *ItemsTab) SelectDisplayItemAffix(control, entry int, sliderVal float64) error {
	controls := tab.AffixControls()
	if control < 1 || control > len(controls) {
		return errNoSlot
	}
	drop := controls[control-1]
	if entry < 1 || entry > len(drop.List) {
		return itemsEditError("no such affix")
	}
	item := tab.DisplayItem
	value := drop.List[entry-1]
	affix := ItemAffix{ModID: "None", Fractured: affixAt(item, drop.OutputTable, drop.OutputIndex).Fractured}
	if value.ModID != "" {
		affix.ModID = value.ModID
		affix.Range = floatp(sliderVal)
	} else if value.ModList != nil {
		index, r := sliderDivVal(sliderVal, len(value.ModList))
		affix.ModID = value.ModList[index-1]
		affix.Range = floatp(verifyRange(item, r, index, value.ModList))
	}
	*affixAt(item, drop.OutputTable, drop.OutputIndex) = affix
	item.Craft()
	return nil
}

// SetDisplayItemAffixRange is affix drop-down #control's roll slider moved to val.
//
//pob2:block:src/Classes/ItemsTab.lua:1067-1075
func (tab *ItemsTab) SetDisplayItemAffixRange(control int, val float64) error {
	controls := tab.AffixControls()
	if control < 1 || control > len(controls) {
		return errNoSlot
	}
	drop := controls[control-1]
	modList := drop.List[drop.SelIndex-1].ModList
	if !drop.SliderShown || len(modList) == 0 {
		return itemsEditError("the affix has no roll")
	}
	item := tab.DisplayItem
	affix := affixAt(item, drop.OutputTable, drop.OutputIndex)
	index, r := sliderDivVal(val, drop.DivCount)
	affix.ModID = modList[index-1]
	affix.Range = floatp(verifyRange(item, r, index, modList))
	affix.Ranges = nil
	item.Craft()
	return nil
}

// AffixRollTooltip is affix drop-down #control's roll slider tooltip at val: the rolled lines, "" for the separator,
// then the affix's tier, name, lines and level.
//
//pob2:block:src/Classes/ItemsTab.lua:1079-1104
func (tab *ItemsTab) AffixRollTooltip(control int, val float64) ([]string, error) {
	controls := tab.AffixControls()
	if control < 1 || control > len(controls) {
		return nil, errNoSlot
	}
	drop := controls[control-1]
	modList := drop.List[drop.SelIndex-1].ModList
	if len(modList) == 0 {
		return []string{}, nil
	}
	item := tab.DisplayItem
	index, r := sliderDivVal(val, drop.DivCount)
	r = verifyRange(item, r, index, modList)
	mod := item.affix(modList[index-1])
	if mod == nil {
		return []string{}, nil
	}
	var out []string
	for _, line := range mod.Lines {
		out = append(out, applyRange(line, r, nil, nil, nil, item.modSource))
	}
	out = append(out, "")
	if len(modList) > 1 {
		out = append(out, "^7Affix: Tier "+strconv.Itoa(len(modList)-index+1)+" ("+mod.Affix+")")
	} else {
		out = append(out, "^7Affix: "+mod.Affix)
	}
	out = append(out, mod.Lines...)
	if mod.Level > 1 {
		out = append(out, "Level: "+luaNum(mod.Level))
	}
	return out, nil
}

// AffixEntryTooltip is an affix drop-down entry's hover tooltip: the mod, or the span of its tiers; nil for "None".
// No PoE2 affix adds a cluster notable, so the Lua's notable block after it never runs.
//
//pob2:block:src/Classes/ItemsTab.lua:951-995
func (item *Item) AffixEntryTooltip(e AffixListEntry) []string {
	modList := e.ModList
	if len(modList) == 0 {
		return nil
	}
	var out []string
	if e.ModID != "" || len(modList) == 1 {
		id := e.ModID
		if id == "" {
			id = modList[0]
		}
		mod := item.affix(id)
		if mod == nil {
			return nil
		}
		out = append(out, "^7Affix: "+mod.Affix)
		for _, line := range mod.Lines {
			out = append(out, "^7"+line)
		}
		if mod.Level > 1 {
			out = append(out, "Level: "+luaNum(mod.Level))
		}
		if len(mod.ModTags) > 0 {
			out = append(out, "Tags: "+strings.Join(mod.ModTags, ", "))
		}
		return out
	}
	out = append(out, "^7"+strconv.Itoa(len(modList))+" Tiers")
	minMod, maxMod := item.affix(modList[0]), item.affix(modList[len(modList)-1])
	if minMod == nil || maxMod == nil {
		return out
	}
	for l, line := range minMod.Lines {
		if l >= len(maxMod.Lines) {
			break
		}
		minLine := affixMinMaxRe.ReplaceAllString(line, "$1")
		maxLine := affixMinMaxRe.ReplaceAllString(maxMod.Lines[l], "$2")
		if maxLine == maxMod.Lines[l] {
			out = append(out, maxLine)
			continue
		}
		maxNums := affixTierNumRe.FindAllString(maxLine, -1)
		n := 0
		out = append(out, affixTierNumRe.ReplaceAllStringFunc(minLine, func(lo string) string {
			if n >= len(maxNums) {
				return lo
			}
			hi := maxNums[n]
			n++
			if lo == hi {
				return lo
			}
			return "(" + lo + "-" + hi + ")"
		}))
	}
	out = append(out, "Level: "+luaNum(minMod.Level)+" to "+luaNum(maxMod.Level))
	if len(maxMod.ModTags) > 0 {
		out = append(out, "Tags: "+strings.Join(maxMod.ModTags, ", "))
	}
	return out
}

// RangeLine is one of the editor's modifier range lines and its roll.
type RangeLine struct {
	Label string
	Range float64
}

// DisplayItemRangeLines is the editor's modifier range lines: the stacked sliders of a unique, else the range
// drop-down's list.
//
//pob2:block:src/Classes/ItemsTab.lua:2564-2577
func (tab *ItemsTab) DisplayItemRangeLines() []RangeLine {
	item := tab.DisplayItem
	if item == nil {
		return nil
	}
	// main.showAllItemAffixes is on by default: a unique shows each line whole beside its own slider
	stacked := item.Rarity == "UNIQUE" || item.Rarity == "RELIC"
	out := make([]RangeLine, 0, len(item.RangeLines))
	for _, ml := range item.RangeLines {
		label := ml.Line
		if !stacked {
			// primarily for Against the Darkness // a way to cut down on really long modLines, gsub could be updated for others
			label = strings.ReplaceAll(label, timeLostRadiusText, ":")
		}
		r := defaultItemAffixQuality
		if ml.Range != nil {
			r = *ml.Range
		}
		out = append(out, RangeLine{Label: label, Range: r})
	}
	return out
}

// SetDisplayItemRangeLine moves range line #line's (1-based) slider to val.
//
//pob2:block:src/Classes/ItemsTab.lua:1147-1166
func (tab *ItemsTab) SetDisplayItemRangeLine(line int, val float64) error {
	item := tab.DisplayItem
	if item == nil {
		return errNoDisplayItem
	}
	if line < 1 || line > len(item.RangeLines) {
		return errNoSlot
	}
	item.RangeLines[line-1].Range = floatp(val)
	item.BuildAndParseRaw()
	return nil
}

// DisplayItemQualityShown is whether the editor shows the item's quality.
//
//pob2:block:src/Classes/ItemsTab.lua:692-694
func (tab *ItemsTab) DisplayItemQualityShown() bool {
	item := tab.DisplayItem
	return item != nil && item.Quality != nil && item.Base != nil && item.Base.Quality != 0
}

// SetDisplayItemQuality is the editor's quality box: its digits, at most two.
//
//pob2:block:src/Classes/ItemsTab.lua:687-691
func (tab *ItemsTab) SetDisplayItemQuality(quality int) error {
	if !tab.DisplayItemQualityShown() {
		return itemsEditError("the item has no quality")
	}
	if quality < 0 || quality > 99 {
		return itemsEditError("the quality is two digits")
	}
	tab.DisplayItem.Quality = floatp(float64(quality))
	tab.DisplayItem.BuildAndParseRaw()
	return nil
}
