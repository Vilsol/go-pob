package calculator

import (
	"cmp"
	"maps"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/utils"
)

type affixSlot struct {
	modID     string
	rng       float64
	hasRange  bool
	fractured bool
}

type affixState struct {
	limit       int
	prefixLimit int
	suffixLimit int
	prefixes    []affixSlot
	suffixes    []affixSlot
	affixes     map[string]*data.AffixMod
}

var (
	affixSpecRe      = regexp.MustCompile(`^(Prefix|Suffix): (.*)$`)
	affixRangeSpecRe = regexp.MustCompile(`^\{range:([^}]+)\}(.+)$`)
	affixLimitRe     = regexp.MustCompile(`([+-])(\d+) (prefix|suffix) modifiers? allowed`)
	affixTierRangeRe = regexp.MustCompile(`\((\d[\d.]*)-(\d[\d.]*)\)`)
	affixAnyRangeRe  = regexp.MustCompile(`\(-?[\d.]+--?[\d.]+\)`)
	affixNumberRe    = regexp.MustCompile(`-?\d+\.?\d*`)
	craftNumberRe    = regexp.MustCompile(`\d+`)
)

//pob:block:src/Classes/Item.lua:776-797,src/Classes/Item.lua:1436-1480
func itemAffixState(raw string, item *Item) (affixState, bool) {
	if item.Base == nil || !item.Crafted {
		return affixState{}, false
	}
	st := affixState{affixes: map[string]*data.AffixMod{}}
	for _, m := range itemAffixes(item) {
		st.affixes[m.ID] = &m
	}
	for _, l := range splitRaw(raw) {
		m := affixSpecRe.FindStringSubmatch(strings.TrimSpace(l))
		if m == nil {
			continue
		}
		s := affixSlot{}
		val := m[2]
		if strings.HasPrefix(val, "{fractured}") {
			s.fractured, val = true, strings.TrimPrefix(val, "{fractured}")
		}
		s.modID = val
		if r := affixRangeSpecRe.FindStringSubmatch(val); r != nil {
			s.modID = r[2]
			if v, err := strconv.ParseFloat(strings.Split(r[1], ",")[0], 64); err == nil {
				s.rng, s.hasRange = v, true
			}
		}
		if !s.hasRange && s.modID != "None" {
			s.rng, s.hasRange = *defaultItemAffixQuality, true
		}
		if m[1] == "Prefix" {
			st.prefixes = append(st.prefixes, s)
		} else {
			st.suffixes = append(st.suffixes, s)
		}
	}
	var prefixLimit, suffixLimit *int
	for _, list := range [][]ModLine{item.EnchantModLines, item.ScourgeModLines, item.ImplicitModLines, item.ExplicitModLines} {
		for _, ml := range list {
			for _, m := range affixLimitRe.FindAllStringSubmatch(strings.ToLower(ml.Line), -1) {
				n, _ := strconv.Atoi(m[2])
				if m[1] == "-" {
					n = -n
				}
				p := &prefixLimit
				if m[3] == "suffix" {
					p = &suffixLimit
				}
				if *p == nil {
					*p = new(int)
				}
				**p += n
			}
		}
	}
	clamp := func(v, lo, hi int) int { return max(min(v, hi), lo) }
	orZero := func(p *int) int {
		if p == nil {
			return 0
		}
		return *p
	}
	switch item.Rarity {
	case RarityMagic:
		st.limit = 2
		if prefixLimit != nil || suffixLimit != nil {
			st.prefixLimit, st.suffixLimit = clamp(orZero(prefixLimit)+1, 0, 2), clamp(orZero(suffixLimit)+1, 0, 2)
			st.limit = st.prefixLimit + st.suffixLimit
		} else {
			st.prefixLimit, st.suffixLimit = 1, 1
		}
	case RarityRare:
		st.limit = 6
		if (item.Base.Type == "Jewel" && !(item.Base.SubType == "Abyss" && item.Corrupted)) || item.Base.Type == "Graft" {
			st.limit = 4
		}
		st.prefixLimit, st.suffixLimit = st.limit/2, st.limit/2
		if prefixLimit != nil || suffixLimit != nil {
			st.prefixLimit, st.suffixLimit = clamp(orZero(prefixLimit)+st.limit/2, 0, st.limit), clamp(orZero(suffixLimit)+st.limit/2, 0, st.limit)
			st.limit = st.prefixLimit + st.suffixLimit
		}
	default:
		return affixState{}, false
	}
	fix := func(list []affixSlot, limit int) []affixSlot {
		for len(list) < limit {
			list = append(list, affixSlot{modID: "None"})
		}
		for i := range limit {
			if list[i].modID == "None" || st.affixes[list[i].modID] != nil {
				continue
			}
			found := ""
			for id, mod := range st.affixes {
				if mod.Affix == list[i].modID && (found == "" || id < found) {
					found = id
				}
			}
			if found == "" {
				found = "None"
			}
			list[i].modID = found
		}
		return list
	}
	st.prefixes, st.suffixes = fix(st.prefixes, st.prefixLimit), fix(st.suffixes, st.suffixLimit)
	return st, true
}

func (st *affixState) slot(index int) (*affixSlot, string, int) {
	if index < st.prefixLimit {
		return &st.prefixes[index], "Prefix", index
	}
	return &st.suffixes[index-st.prefixLimit], "Suffix", index - st.prefixLimit
}

//pob:src/Classes/Item.lua:2016-2079
func craftText(raw string, item *Item, st affixState) string {
	b := craftBlock(raw, item)
	type gen struct {
		line      string
		order     float64
		fractured bool
		prefix    bool
		tags      []string
		saved     bool
	}
	var lines []gen
	var kept []string
	for _, l := range b.after {
		t := strings.TrimSpace(l)
		switch {
		case slices.Contains(explicitTailLines, t) || strings.HasPrefix(t, "Foil Unique") || strings.Contains(t, "{crucible}"):
			kept = append(kept, l)
		case strings.Contains(t, "{crafted}") || strings.Contains(t, "{custom}") || strings.Contains(t, " (crafted)"):
			lines = append(lines, gen{line: l, saved: true})
		}
	}
	namePrefix, nameSuffix := "", ""
	level := item.Base.Req.Level
	byOrder := map[float64]int{}
	for _, list := range []struct {
		slots []affixSlot
		limit int
	}{{st.prefixes, st.prefixLimit}, {st.suffixes, st.suffixLimit}} {
		for i := 0; i < list.limit && i < len(list.slots); i++ {
			a := list.slots[i]
			mod := st.affixes[a.modID]
			if mod == nil {
				continue
			}
			switch mod.Type {
			case "Prefix":
				namePrefix = mod.Affix + " " + namePrefix
			case "Suffix":
				nameSuffix = nameSuffix + " " + mod.Affix
			}
			level = max(level, int(math.Floor(float64(mod.Level)*0.8)))
			r := 0.5
			if a.hasRange {
				r = a.rng
			}
			for k, line := range mod.Lines {
				line = data.ApplyRange(line, r, nil, nil)
				order := 0.0
				if k < len(mod.StatOrder) {
					order = mod.StatOrder[k]
				}
				if at, ok := byOrder[order]; ok {
					nums := craftNumberRe.FindAllString(line, -1)
					n := 0
					lines[at].line = craftNumberRe.ReplaceAllStringFunc(lines[at].line, func(s string) string {
						if n >= len(nums) {
							return s
						}
						x, _ := strconv.Atoi(s)
						y, _ := strconv.Atoi(nums[n])
						n++
						return strconv.Itoa(x + y)
					})
					continue
				}
				byOrder[order] = len(lines)
				lines = append(lines, gen{line: line, order: order, fractured: a.fractured, prefix: mod.Type == "Prefix", tags: mod.ModTags})
			}
		}
	}
	group := func(g gen) int {
		switch {
		case g.saved:
			return 3
		case g.fractured:
			return 1
		}
		return 2
	}
	if len(lines) > 1 {
		slices.SortStableFunc(lines, func(x, y gen) int {
			if gx, gy := group(x), group(y); gx != gy {
				return gx - gy
			} else if gx < 3 {
				return cmp.Compare(x.order, y.order)
			}
			return 0
		})
	}
	explicit := make([]string, 0, len(lines))
	for _, g := range lines {
		if g.saved {
			explicit = append(explicit, g.line)
			continue
		}
		line := g.line
		if g.fractured {
			line = "{fractured}" + line
		}
		if g.prefix {
			line = "{prefix}" + line
		} else {
			line = "{suffix}" + line
		}
		if len(g.tags) > 0 {
			line = "{tags:" + strings.Join(g.tags, ",") + "}" + line
		}
		explicit = append(explicit, line)
	}
	b.after = append(explicit, kept...)

	var before []string
	for _, l := range b.before {
		if t := strings.TrimSpace(l); affixSpecRe.MatchString(t) || strings.HasPrefix(t, "LevelReq: ") {
			continue
		}
		before = append(before, l)
	}
	at := len(before)
	for k, l := range before {
		if strings.TrimSpace(l) == "Crafted: true" {
			at = k + 1
		}
	}
	var specs []string
	write := func(name string, list []affixSlot, limit int) {
		for i := 0; i < limit && i < len(list); i++ {
			a := list[i]
			s := name + ": "
			if a.fractured {
				s += "{fractured}"
			}
			if a.hasRange && a.modID != "None" {
				s += "{range:" + strconv.FormatFloat(utils.RoundTo(a.rng, 3), 'f', -1, 64) + "}"
			}
			specs = append(specs, s+a.modID)
		}
	}
	write("Prefix", st.prefixes, st.prefixLimit)
	write("Suffix", st.suffixes, st.suffixLimit)
	before = slices.Insert(before, at, specs...)
	if item.Rarity == RarityMagic {
		for k, l := range before {
			if strings.HasPrefix(strings.TrimSpace(l), "Rarity: ") && k+1 < len(before) {
				before[k+1] = namePrefix + item.BaseName + nameSuffix
				break
			}
		}
	}
	b.before = append(before, "LevelReq: "+strconv.Itoa(level))
	return b.join()
}

type AffixSeries struct {
	Label     string
	ModList   []string
	HaveRange bool
	Tooltip   []string
	Mid       []string
}

type AffixControl struct {
	Type        string
	Series      []AffixSeries
	Sel         int
	SliderShown bool
	SliderVal   float64
	DivCount    int
}

type AffixControls struct {
	Sorting  bool
	Controls []AffixControl
}

func affixTooltip(st affixState, modList []string, single string) []string {
	var out []string
	if single != "" || len(modList) == 1 {
		id := single
		if id == "" {
			id = modList[0]
		}
		mod := st.affixes[id]
		out = append(out, "^7Affix: "+mod.Affix)
		for _, l := range mod.Lines {
			out = append(out, "^7"+l)
		}
		if mod.Level > 1 {
			out = append(out, "Level: "+strconv.Itoa(mod.Level))
		}
		if len(mod.ModTags) > 0 {
			out = append(out, "Tags: "+strings.Join(mod.ModTags, ", "))
		}
		return append(out, clusterNotableTooltip(mod)...)
	}
	out = append(out, "^7"+strconv.Itoa(len(modList))+" Tiers")
	minMod, maxMod := st.affixes[modList[0]], st.affixes[modList[len(modList)-1]]
	for l, line := range minMod.Lines {
		if l >= len(maxMod.Lines) {
			break
		}
		minLine := affixTierRangeRe.ReplaceAllString(line, "$1")
		maxLine := affixTierRangeRe.ReplaceAllString(maxMod.Lines[l], "$2")
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
	out = append(out, "Level: "+strconv.Itoa(minMod.Level)+" to "+strconv.Itoa(maxMod.Level))
	if len(maxMod.ModTags) > 0 {
		out = append(out, "Tags: "+strings.Join(maxMod.ModTags, ", "))
	}
	return out
}

var affixTierNumRe = regexp.MustCompile(`\d[\d.]*`)

var clusterNotableRe = regexp.MustCompile(`^1 Added Passive Skill is (.*)$`)

//pob:block:src/Classes/ItemsTab.lua:860-914,src/Modules/Data.lua:849-885
func clusterNotableTooltip(mod *data.AffixMod) []string {
	if mod == nil || len(mod.Lines) == 0 {
		return nil
	}
	m := clusterNotableRe.FindStringSubmatch(mod.Lines[0])
	if m == nil {
		return nil
	}
	name := m[1]
	node, ok := clusterNodeMap(data.LatestTreeVersion)[name]
	if !ok {
		return nil
	}
	out := []string{"--", "^xF8E6CA" + name}
	if len(node.Stats) > 0 {
		out = append(out, "")
		for _, line := range node.Stats {
			if entry := ParseMod(line, false); line != " " && (entry.ModList == nil || modExtra(entry) != "") {
				out = append(out, colorUnsupported+line+notSupportedTooltipText)
			} else {
				out = append(out, colorMagic+line)
			}
		}
	}
	if len(node.ReminderText) > 0 {
		out = append(out, "--")
		for _, line := range node.ReminderText {
			out = append(out, "^xA0A080"+line)
		}
	}
	sizes, types := clusterInfoForNotable(name)
	if len(sizes) == 0 {
		return out
	}
	out = append(out, "--", "^7"+name+" can appear on:")
	for k, size := range sizes {
		out = append(out, colorMagic+size+":")
		skills := data.ClusterJewels.Jewels[size].Skills
		for _, t := range types {
			if skill, ok := skills[t]; ok {
				out = append(out, "^7    "+skill.Name)
			}
		}
		if k > 0 {
			out = append(out, "")
		}
	}
	return out
}

func clusterInfoForNotable(name string) ([]string, []string) {
	mods, _ := data.AffixTable("JewelCluster")
	skillHas := map[string]bool{}
	for _, mod := range mods {
		if len(mod.Lines) == 0 {
			continue
		}
		if m := clusterNotableRe.FindStringSubmatch(mod.Lines[0]); m == nil || m[1] != name {
			continue
		}
		for k, key := range mod.WeightKey {
			if k < len(mod.WeightVal) && mod.WeightVal[k] > 0 {
				skillHas[key] = true
			}
		}
	}
	var sizes, types []string
	for _, size := range slices.Sorted(maps.Keys(data.ClusterJewels.Jewels)) {
		found := false
		for _, skill := range slices.Sorted(maps.Keys(data.ClusterJewels.Jewels[size].Skills)) {
			if skillHas[skill] {
				found = true
				if !slices.Contains(types, skill) {
					types = append(types, skill)
				}
			}
		}
		if found {
			sizes = append(sizes, size)
		}
	}
	slices.Sort(types)
	return sizes, types
}

func midTier(modList []string) string {
	if len(modList) == 1 {
		return modList[0]
	}
	return modList[int(math.Floor(float64(float64(len(modList)-1)*(*defaultItemAffixQuality))+0.5))]
}

//pob:block:src/Classes/ItemsTab.lua:717-722,src/Classes/ItemsTab.lua:2108-2332
func ItemAffixControls(raw string) (AffixControls, bool) {
	item := ParseItem(raw)
	st, ok := itemAffixState(raw, item)
	if !ok {
		return AffixControls{}, false
	}
	_, cluster := data.ClusterJewels.Jewels[item.BaseName]
	out := AffixControls{Sorting: !cluster}
	//pob:block:src/Classes/ItemsTab.lua:2150-2155
	extraTags := map[string]bool{}
	if cj, ok := data.ClusterJewels.Jewels[item.BaseName]; ok {
		if skill, ok := cj.Skills[clusterJewelSkillOf(raw)]; ok {
			extraTags[skill.Tag] = true
		}
	}
	for i := range st.limit {
		self, affixType, _ := st.slot(i)
		excludeGroups := map[string]bool{}
		for j := range st.limit {
			if j == i {
				continue
			}
			if other, _, _ := st.slot(j); st.affixes[other.modID] != nil && st.affixes[other.modID].Group != "" {
				excludeGroups[st.affixes[other.modID].Group] = true
			}
		}
		var affixList []string
		for id, mod := range st.affixes {
			if mod.Type != affixType || excludeGroups[mod.Group] || mod.Affix == "Subterranean" || mod.Affix == "of the Underground" {
				continue
			}
			if modSpawnWeightTags(item, mod, extraTags, nil) > 0 || id == self.modID {
				affixList = append(affixList, id)
			}
		}
		slices.SortFunc(affixList, func(x, y string) int {
			a, b := st.affixes[x], st.affixes[y]
			for k := 0; k < max(len(a.Lines), len(b.Lines)); k++ {
				switch {
				case k >= len(a.Lines):
					return -1
				case k >= len(b.Lines):
					return 1
				case a.StatOrder[k] != b.StatOrder[k]:
					return cmp.Compare(a.StatOrder[k], b.StatOrder[k])
				}
			}
			if a.Level != b.Level {
				return a.Level - b.Level
			}
			return strings.Compare(x, y)
		})
		c := AffixControl{Type: affixType, Series: []AffixSeries{{Label: "None"}}, SliderVal: *defaultItemAffixQuality}
		var last *AffixSeries
		var lastOrder []float64
		for _, id := range affixList {
			mod := st.affixes[id]
			if last == nil || !slices.Equal(lastOrder, mod.StatOrder) {
				label := strings.Join(mod.Lines, "/")
				c.Series = append(c.Series, AffixSeries{Label: label, HaveRange: affixAnyRangeRe.MatchString(label)})
				last, lastOrder = &c.Series[len(c.Series)-1], mod.StatOrder
			}
			last.ModList = append(last.ModList, id)
			if len(last.ModList) == 2 {
				last.Label = affixNumberRe.ReplaceAllString(affixAnyRangeRe.ReplaceAllString(last.Label, "#"), "#")
				last.HaveRange = true
			}
		}
		for k := 1; k < len(c.Series); k++ {
			s := &c.Series[k]
			s.Tooltip = affixTooltip(st, s.ModList, "")
			s.Mid = st.affixes[midTier(s.ModList)].Lines
			if slices.Contains(s.ModList, self.modID) {
				c.Sel = k
			}
		}
		if c.Sel > 0 && c.Series[c.Sel].HaveRange {
			s := c.Series[c.Sel]
			c.DivCount = len(s.ModList)
			index := slices.Index(s.ModList, self.modID) + 1
			r := 0.5
			if self.hasRange {
				r = min(1, max(0, self.rng))
			}
			if r == 0 && index > 1 {
				r = 1e-4
			}
			c.SliderVal = (float64(index) - 1 + r) / float64(c.DivCount)
			if c.DivCount == 1 {
				c.DivCount = 0
			}
			c.SliderShown = true
		}
		out.Controls = append(out.Controls, c)
	}
	return out, true
}

//pob:src/Classes/SliderControl.lua:55-63
func divVal(val float64, divCount int) (int, float64) {
	if divCount > 1 {
		div := max(int(math.Ceil(val*float64(divCount))), 1)
		return div, float64(val*float64(divCount)) - float64(div) + 1
	}
	return 1, val
}

//pob:src/Classes/ItemsTab.lua:756-795
func verifyRange(st affixState, modList []string, rng float64, index int) float64 {
	minMax := func(mod *data.AffixMod) (float64, float64, bool) {
		for _, line := range mod.Lines {
			if m := affixTierRangeRe.FindStringSubmatch(line); m != nil {
				lo, _ := strconv.ParseFloat(m[1], 64)
				hi, _ := strconv.ParseFloat(m[2], 64)
				return lo, hi, true
			}
		}
		return 0, 0, false
	}
	flip := func(a, b *data.AffixMod) bool {
		minA, maxA, okA := minMax(a)
		minB, maxB, okB := minMax(b)
		if !okA || !okB || minA != math.Floor(minA) || maxA != math.Floor(maxA) || minB != math.Floor(minB) || maxB != math.Floor(maxB) {
			return false
		}
		if minA < minB {
			return minA+1 == maxB
		}
		return minA-1 == maxB
	}
	cur := st.affixes[modList[index-1]]
	switch {
	case index-1 > 0:
		if flip(st.affixes[modList[index-2]], cur) {
			rng = 1 - rng
		}
	case index+1 < len(modList):
		if flip(cur, st.affixes[modList[index]]) {
			rng = 1 - rng
		}
	}
	return rng
}

func AffixEditText(raw string, slot int, series int, slider float64) (string, bool) {
	item := ParseItem(raw)
	st, ok := itemAffixState(raw, item)
	controls, ok2 := ItemAffixControls(raw)
	if !ok || !ok2 || slot < 0 || slot >= st.limit || series < 0 || series >= len(controls.Controls[slot].Series) {
		return raw, false
	}
	a, _, _ := st.slot(slot)
	s := controls.Controls[slot].Series[series]
	if series == 0 {
		*a = affixSlot{modID: "None", fractured: a.fractured}
	} else {
		div := len(s.ModList)
		index, rng := divVal(slider, div)
		index = min(index, div)
		*a = affixSlot{modID: s.ModList[index-1], rng: verifyRange(st, s.ModList, rng, index), hasRange: true, fractured: a.fractured}
	}
	return craftText(raw, item, st), true
}

//pob:src/Classes/ItemsTab.lua:943-968
func AffixRollTooltip(raw string, slot int, slider float64) []string {
	item := ParseItem(raw)
	st, ok := itemAffixState(raw, item)
	controls, ok2 := ItemAffixControls(raw)
	if !ok || !ok2 || slot < 0 || slot >= len(controls.Controls) || controls.Controls[slot].Sel == 0 {
		return nil
	}
	modList := controls.Controls[slot].Series[controls.Controls[slot].Sel].ModList
	index, rng := divVal(slider, len(modList))
	index = min(index, len(modList))
	rng = verifyRange(st, modList, rng, index)
	mod := st.affixes[modList[index-1]]
	var out []string
	for _, line := range mod.Lines {
		out = append(out, data.ApplyRange(line, rng, nil, nil))
	}
	out = append(out, "")
	if len(modList) > 1 {
		out = append(out, "^7Affix: Tier "+strconv.Itoa(len(modList)-index+1)+" ("+mod.Affix+")")
	} else {
		out = append(out, "^7Affix: "+mod.Affix)
	}
	out = append(out, mod.Lines...)
	if mod.Level > 1 {
		out = append(out, "Level: "+strconv.Itoa(mod.Level))
	}
	return out
}

//pob:src/Classes/ItemsTab.lua:2219-2297
func (m *MiscCalculator) AffixPowers(raw string, slot int, stat string) []float64 {
	ps, ok := powerStatByName(stat)
	item := ParseItem(raw)
	st, ok1 := itemAffixState(raw, item)
	controls, ok2 := ItemAffixControls(raw)
	if !ok || !ok1 || !ok2 || slot < 0 || slot >= len(controls.Controls) {
		return nil
	}
	c := controls.Controls[slot]
	base := raw
	if a, _, _ := st.slot(slot); a.modID != "None" {
		*a = affixSlot{modID: "None", fractured: a.fractured}
		base = craftText(raw, item, st)
	}
	baseItem := ParseItem(base)
	texts, found := make([]string, len(c.Series)), make([]bool, len(c.Series))
	for k := 1; k < len(c.Series); k++ {
		mod := st.affixes[midTier(c.Series[k].ModList)]
		b := craftBlock(base, baseItem)
		var lines []string
		for _, l := range mod.Lines {
			if len(mod.ModTags) > 0 {
				l = "{tags:" + strings.Join(mod.ModTags, ",") + "}" + l
			}
			lines = append(lines, l)
		}
		appendExplicit(&b, lines)
		texts[k], found[k] = b.join(), true
	}
	return m.sortValues(item.getPrimarySlot(), texts, found, ps)
}
