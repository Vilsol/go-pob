package calculator

import (
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/data"
	raw2 "github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/pob"
	"github.com/Vilsol/go-pob/utils"
)

var (
	implicitsSpecRe = regexp.MustCompile(`^Implicits: (\d+)`)
	rawRangeLineRe  = regexp.MustCompile(`\(-?[\d.]+--?[\d.]+\)`)
)

type implicitBlock struct {
	before, after                         []string
	enchant, scourge, classReq, implicits []string
}

func cutImplicitBlock(lines []string, multi map[string]bool) (implicitBlock, bool) {
	for k, l := range lines {
		m := implicitsSpecRe.FindStringSubmatch(strings.TrimSpace(l))
		if m == nil {
			continue
		}
		n, _ := strconv.Atoi(m[1])
		b := implicitBlock{before: slices.Clone(lines[:k])}
		var entries, keys []string
		end := k + 1
		for ; end < len(lines); end++ {
			t := strings.TrimSpace(lines[end])
			if last := len(entries) - 1; last >= 0 && multi[keys[last]+"\n"+rawModText(t)] {
				entries[last] += "\n" + t
				keys[last] += "\n" + rawModText(t)
				continue
			}
			if len(entries) == n {
				break
			}
			entries, keys = append(entries, t), append(keys, rawModText(t))
		}
		for _, line := range entries {
			crafted := strings.Contains(line, "{crafted}") || strings.Contains(line, " (crafted)")
			switch {
			case strings.Contains(line, "{enchant}") || strings.Contains(line, " (enchant)") || (crafted && len(b.enchant)+len(b.implicits) < n):
				b.enchant = append(b.enchant, line)
			case strings.Contains(line, "{scourge}") || strings.Contains(line, " (scourge)"):
				b.scourge = append(b.scourge, line)
			case strings.Contains(line, "Requires Class"):
				b.classReq = append(b.classReq, line)
			default:
				b.implicits = append(b.implicits, line)
			}
		}
		b.after = slices.Clone(lines[end:])
		return b, true
	}
	return implicitBlock{}, false
}

func (b implicitBlock) join() string {
	block := slices.Concat(b.enchant, b.scourge, b.classReq, b.implicits)
	out := slices.Concat(b.before, []string{"Implicits: " + strconv.Itoa(len(block))}, block, b.after)
	return strings.Join(out, "\n")
}

func craftBlock(raw string, item *Item) implicitBlock {
	multi := map[string]bool{}
	for _, list := range [][]ModLine{item.EnchantModLines, item.ScourgeModLines, item.ClassRequirementModLines, item.ImplicitModLines} {
		for _, ml := range list {
			if strings.Contains(ml.Line, "\n") {
				multi[ml.Line] = true
			}
		}
	}
	if b, ok := cutImplicitBlock(splitRaw(raw), multi); ok {
		return b
	}
	b, _ := cutImplicitBlock(splitRaw(buildRawText(item)), multi)
	return b
}

//pob:src/Classes/Item.lua:1842-1927
func writeRawModLine(ml ModLine) string {
	line := ml.Line
	if ml.Range != nil && rawRangeLineRe.MatchString(line) {
		line = "{range:" + strconv.FormatFloat(utils.RoundTo(*ml.Range, 6), 'f', -1, 64) + "}" + line
	}
	if ml.CorruptedRange != nil {
		line = "{corruptedRange:" + strconv.FormatFloat(utils.RoundTo(*ml.CorruptedRange, 2), 'f', -1, 64) + "}" + line
	}
	if ml.Disabled {
		line = "{disabled}" + line
	}
	if ml.Crafted {
		line = "{crafted}" + line
	}
	if ml.Custom {
		line = "{custom}" + line
	}
	if ml.Scourge {
		line = "{scourge}" + line
	}
	if ml.Crucible {
		line = "{crucible}" + line
	}
	if ml.Mutated {
		line = "{mutated}" + line
	}
	if ml.ModGroup != "" {
		line = "{modGroup:" + ml.ModGroup + "}" + strings.ReplaceAll(line, "\n", "\n{modGroup:"+ml.ModGroup+"}")
	}
	if ml.Fractured {
		line = "{fractured}" + line
	}
	if ml.Prefix {
		line = "{prefix}" + line
	}
	if ml.Suffix {
		line = "{suffix}" + line
	}
	if ml.Vestigial {
		line = "{vestigial}" + line
	}
	if len(ml.ModTags) > 0 {
		line = "{tags:" + strings.Join(ml.ModTags, ",") + "}" + line
	}
	if len(ml.VariantGroupList) > 0 {
		line = "{group:" + idSpec(ml.VariantGroupList) + "}" + strings.ReplaceAll(line, "\n", "\n{group:"+idSpec(ml.VariantGroupList)+"}")
	}
	if len(ml.VariantList) > 0 {
		ids := make([]int, 0, len(ml.VariantList))
		for id := range ml.VariantList {
			ids = append(ids, id)
		}
		slices.Sort(ids)
		parts := make([]string, len(ids))
		for k, id := range ids {
			parts[k] = strconv.Itoa(id)
		}
		line = "{variant:" + strings.Join(parts, ",") + "}" + strings.ReplaceAll(line, "\n", "\n{variant:"+strings.Join(parts, ",")+"}")
	}
	if len(ml.VersionList) > 0 {
		line = "{version:" + idSpec(ml.VersionList) + "}" + strings.ReplaceAll(line, "\n", "\n{version:"+idSpec(ml.VersionList)+"}")
	}
	return line
}

//pob:block:src/Classes/Item.lua:1772-2047
func buildRawText(item *Item) string {
	var out []string
	add := func(s ...string) { out = append(out, s...) }
	itoa := func(p *int) string { return strconv.Itoa(*p) }
	add("Rarity: " + string(item.Rarity))
	if item.Title != "" {
		add(item.Title, item.BaseName)
	} else {
		add(item.NamePrefix + item.BaseName + item.NameSuffix)
	}
	if item.Intangibility != nil {
		add("Intangibility: " + itoa(item.Intangibility) + "%")
	}
	if item.UniqueID != "" {
		add("Unique ID: " + item.UniqueID)
	}
	if item.League != "" {
		add("League: " + item.League)
	}
	if item.Unreleased {
		add("Unreleased: true")
	}
	add(itemInfluenceLines(item.Influence)...)
	if item.Crafted {
		add("Crafted: true")
		if st, ok := itemAffixState(item.Raw, item); ok {
			add(st.specLines()...)
		}
	}
	if item.Catalyst != nil && *item.Catalyst > 0 && *item.Catalyst <= len(catalystList) {
		add("Catalyst: " + catalystList[*item.Catalyst-1])
	}
	if item.CatalystQuality != nil {
		add("CatalystQuality: " + itoa(item.CatalystQuality))
	}
	if item.TalismanTier != nil {
		add("Talisman Tier: " + itoa(item.TalismanTier))
	}
	if item.ItemLevel != nil {
		add("Item Level: " + itoa(item.ItemLevel))
	}
	if item.MemoryStrands != nil {
		add("Memory Strands: " + itoa(item.MemoryStrands))
	}
	add(item.variantGroupRawLines()...)
	//pob:block:src/Classes/Item.lua:1936-1979
	if len(item.VariantList) > 0 {
		for _, name := range item.VariantList {
			add("Variant: " + name)
		}
		switch {
		case item.UsesVariantGroups:
			add(item.variantGroupSelectionLines()...)
		case item.Variant != nil:
			add("Selected Variant: " + itoa(item.Variant))
		}
		for _, alt := range []struct {
			on       bool
			v        *int
			has, sel string
		}{{item.HasAltVariant, item.VariantAlt, "Has Alt Variant", "Selected Alt Variant"},
			{item.HasAltVariant2, item.VariantAlt2, "Has Alt Variant Two", "Selected Alt Variant Two"},
			{item.HasAltVariant3, item.VariantAlt3, "Has Alt Variant Three", "Selected Alt Variant Three"},
			{item.HasAltVariant4, item.VariantAlt4, "Has Alt Variant Four", "Selected Alt Variant Four"},
			{item.HasAltVariant5, item.VariantAlt5, "Has Alt Variant Five", "Selected Alt Variant Five"}} {
			if alt.on && alt.v != nil && !item.UsesVariantGroups {
				add(alt.has+": true", alt.sel+": "+itoa(alt.v))
			}
		}
		if item.AllowDuplicateVariants {
			add("Allow Duplicate Variants: true")
		}
	}
	if item.Quality != nil {
		add("Quality: " + itoa(item.Quality))
	}
	if len(item.Sockets) > 0 {
		add("Sockets: " + socketsText(item.Sockets))
	}
	if lvl, ok := item.Requirements["level"]; ok {
		add("LevelReq: " + strconv.Itoa(int(lvl)))
	}
	if item.JewelRadiusLabel != "" {
		add("Radius: " + item.JewelRadiusLabel)
	}
	if item.Limit != nil {
		add("Limited to: " + itoa(item.Limit))
	}
	if item.ClassRestriction != "" {
		add("Requires Class " + item.ClassRestriction)
	}
	add("Implicits: " + strconv.Itoa(len(item.EnchantModLines)+len(item.ImplicitModLines)+len(item.ScourgeModLines)))
	for _, list := range [][]ModLine{item.EnchantModLines, item.ScourgeModLines, item.ClassRequirementModLines, item.ImplicitModLines, item.ExplicitModLines, item.CrucibleModLines} {
		for _, ml := range list {
			add(writeRawModLine(ml))
		}
	}
	if item.Split {
		add("Split")
	}
	if item.Mirrored {
		add("Mirrored")
	}
	if item.Fractured {
		add("Fractured Item")
	}
	if item.Corrupted || item.Scourge {
		add("Corrupted")
	}
	if item.FoilType != "" {
		add("Foil Unique (" + item.FoilType + ")")
	}
	return strings.Join(out, "\n")
}

func itemInfluenceLines(i ItemInfluence) []string {
	var out []string
	for _, f := range influenceInfoAll {
		if f.on(i) {
			out = append(out, f.display+" Item")
		}
	}
	return out
}

//pob:block:src/Classes/Item.lua:1041-1049
func ItemEnchantments(item *Item) *data.EnchantmentTable {
	if item == nil || item.Base == nil {
		return nil
	}
	if item.Base.Flask != nil {
		if item.Base.SubType == "Utility" {
			return data.Enchantments("UtilityFlask")
		}
		return data.Enchantments("Flask")
	}
	return data.Enchantments(item.Base.Type)
}

//pob:src/Classes/ItemsTab.lua:70-74
func isAnointable(item *Item) bool {
	return item != nil && item.Base != nil && !data.CraftBaseInfo(item.Base.Name).CannotBeAnointed && !isTalismanBase(item.Base) &&
		(item.CanBeAnointed || item.Base.Type == "Amulet")
}

//pob:src/Classes/ItemsTab.lua:523-531
func enchantButtons(item *Item) int {
	switch {
	case ItemEnchantments(item) == nil:
		return 0
	case item.CanHaveTwoEnchants && len(item.EnchantModLines) > 0:
		return 2
	}
	return 1
}

//pob:src/Classes/ItemsTab.lua:535-564
func anointButtons(item *Item) int {
	if !isAnointable(item) {
		return 0
	}
	n := 1
	if item.CanHaveTwoEnchants && len(item.EnchantModLines) > 0 {
		n = 2
		if item.CanHaveThreeEnchants && len(item.EnchantModLines) > 1 {
			n = 3
			if item.CanHaveFourEnchants && len(item.EnchantModLines) > 2 {
				n = 4
			}
		}
	}
	return n
}

//pob:src/Classes/ItemsTab.lua:2832-2856
func enchantText(raw string, slot int, line string, remove bool) (string, bool) {
	item := ParseItem(raw)
	if item.Base == nil || slot < 1 {
		return raw, false
	}
	b := craftBlock(raw, item)
	if remove {
		if slot <= len(b.enchant) {
			b.enchant = slices.Delete(b.enchant, slot-1, slot)
		}
		return b.join(), true
	}
	if first, rest, ok := strings.Cut(line, "/"); ok && first != "" && strings.Split(rest, "/")[0] != "" {
		second := strings.Split(rest, "/")[0]
		b.enchant = []string{"{crafted}" + first, "{crafted}" + second}
		return b.join(), true
	}
	if !item.CanHaveTwoEnchants && len(b.enchant) > 1 {
		b.enchant = b.enchant[:1]
	}
	if len(b.enchant) >= slot {
		b.enchant = slices.Delete(b.enchant, slot-1, slot)
	}
	b.enchant = slices.Insert(b.enchant, min(slot-1, len(b.enchant)), "{crafted}"+line)
	return b.join(), true
}

//pob:src/Classes/ItemsTab.lua:3011-3023
func anointText(raw string, slot int, name string) (string, bool) {
	item := ParseItem(raw)
	if item.Base == nil || slot < 1 {
		return raw, false
	}
	b := craftBlock(raw, item)
	if len(b.enchant) >= slot {
		b.enchant = slices.Delete(b.enchant, slot-1, slot)
	}
	if name != "" {
		b.enchant = slices.Insert(b.enchant, min(slot-1, len(b.enchant)), "{crafted}Allocates "+name)
	}
	return b.join(), true
}

//pob:src/Classes/ItemsTab.lua:2978-2993
func ItemAnoints(item *Item) []string {
	var out []string
	for _, list := range [][]ModLine{item.EnchantModLines, item.ScourgeModLines, item.ImplicitModLines, item.ExplicitModLines} {
		for _, ml := range list {
			if k := strings.Index(ml.Line, "Allocates "); k >= 0 && anointNameRe.MatchString(ml.Line[k+len("Allocates "):]) {
				out = append(out, ml.Line[k+len("Allocates "):])
			}
		}
	}
	return out
}

var anointNameRe = regexp.MustCompile(`^[a-zA-Z ]`)

type EnchantSource struct {
	Name  string
	Label string
	Lines []string
}

type EnchantSkill struct {
	Skill   string
	Sources []EnchantSource
}

type EnchantChoices struct {
	HaveSkills bool
	Skills     []EnchantSkill
	Used       []string
}

//pob:block:src/Classes/ItemsTab.lua:2768-2809
func ItemEnchantChoices(build *pob.PathOfBuilding, raw string) (EnchantChoices, bool) {
	item := ParseItem(raw)
	table := ItemEnchantments(item)
	if table == nil {
		return EnchantChoices{}, false
	}
	sourcesOf := func(lists map[string][]string) []EnchantSource {
		var out []EnchantSource
		for _, s := range data.EnchantmentSources() {
			if lines, ok := lists[s.Name]; ok {
				out = append(out, EnchantSource{Name: s.Name, Label: s.Label, Lines: lines})
			}
		}
		return out
	}
	if len(table.Skills) == 0 {
		return EnchantChoices{Skills: []EnchantSkill{{Sources: sourcesOf(table.Sources)}}}, true
	}
	out := EnchantChoices{HaveSkills: true}
	for _, name := range slices.Sorted(maps.Keys(table.Skills)) {
		out.Skills = append(out.Skills, EnchantSkill{Skill: name, Sources: sourcesOf(table.Skills[name])})
	}
	if build != nil && len(build.Skills.SkillSets) > 0 {
		used := map[string]bool{}
		for _, group := range build.Skills.SkillSets[build.ActiveSkillSetIndex()].Skills {
			for _, gem := range group.Gems {
				gemData, _ := resolveGemInstance(gem)
				if gemData == nil {
					continue
				}
				for _, ge := range raw2.GrantedEffectsForGem(gemData) {
					name := (&GrantedEffect{Raw: ge}).Name()
					if !ge.IsSupport && table.Skills[name] != nil {
						used[name] = true
					}
				}
			}
		}
		out.Used = slices.Sorted(maps.Keys(used))
	}
	return out, true
}

func powerStatByName(stat string) (PowerStat, bool) {
	for _, s := range PowerStats() {
		if s.Stat == stat {
			return s, true
		}
	}
	return PowerStat{}, false
}

//pob:src/Classes/ItemsTab.lua:2857-2882
func (m *MiscCalculator) EnchantSortValues(raw string, slot int, lines []string, stat string) []float64 {
	ps, ok := powerStatByName(stat)
	item := ParseItem(raw)
	out := make([]float64, len(lines))
	if !ok || item.Base == nil {
		return out
	}
	texts, found := make([]string, len(lines)), make([]bool, len(lines))
	for k, line := range lines {
		texts[k], found[k] = enchantText(raw, slot, line, false)
	}
	return m.sortValues(item.getPrimarySlot(), texts, found, ps)
}

type AnointNotable struct {
	ID           int64
	Name         string
	Stats        []string
	ReminderText []string
	Recipe       []string
	Unsupported  []bool
}

//pob:block:src/Classes/PassiveTree.lua:525-597
func unsupportedStatLines(stats []string) []bool {
	out := make([]bool, len(stats))
	for i := 0; i < len(stats); i++ {
		entry := ParseMod(stats[i], false)
		if entry.ModList != nil && strings.TrimSpace(entry.Extra) == "" {
			continue
		}
		combined := false
		for end := i + 1; end < len(stats) && !combined; end++ {
			comb := strings.Join(stats[i:end+1], " ")
			if c := ParseMod(comb, true); c.ModList != nil && strings.TrimSpace(c.Extra) == "" {
				combined, i = true, end
			}
		}
		out[i] = !combined
	}
	return out
}

//pob:block:src/Classes/NotableDBControl.lua:17-19,src/Classes/NotableDBControl.lua:123-129
func AnointNotables() []AnointNotable {
	var out []AnointNotable
	for _, n := range data.TreeVersions[data.LatestTreeVersion].Tree().Nodes {
		if len(n.Recipe) == 0 || n.Skill == nil || n.Name == nil {
			continue
		}
		var stats []string
		for _, l := range fixupTreeStatText(n.Stats) {
			for _, part := range strings.Split(l, "\n") {
				if part != "" {
					stats = append(stats, part)
				}
			}
		}
		e := AnointNotable{ID: *n.Skill, Name: *n.Name, Stats: stats, ReminderText: n.ReminderText, Unsupported: unsupportedStatLines(stats)}
		for _, o := range n.Recipe {
			e.Recipe = append(e.Recipe, string(o))
		}
		out = append(out, e)
	}
	slices.SortFunc(out, func(a, b AnointNotable) int {
		if c := strings.Compare(a.Name, b.Name); c != 0 {
			return c
		}
		return int(a.ID - b.ID)
	})
	return out
}

func anointNode(id int64) (data.Node, bool) {
	n, ok := data.TreeVersions[data.LatestTreeVersion].Tree().Nodes[strconv.FormatInt(id, 10)]
	return n, ok && n.Name != nil
}

//pob:block:src/Classes/NotableDBControl.lua:117-121,src/Classes/NotableDBControl.lua:131-149
func (m *MiscCalculator) AnointPowers(raw string, slot int, ids []int64, stat string) []float64 {
	ps, ok := powerStatByName(stat)
	item := ParseItem(raw)
	out := make([]float64, len(ids))
	if !ok || item.Base == nil {
		return out
	}
	texts, found := make([]string, len(ids)+1), make([]bool, len(ids)+1)
	texts[0], _ = anointText(raw, slot, "")
	found[0] = true
	for k, id := range ids {
		if n, ok := anointNode(id); ok && len(n.Stats) > 0 {
			texts[k+1], _ = anointText(raw, slot, *n.Name)
			found[k+1] = true
		}
	}
	values := m.sortValues(item.Base.Type, texts, found, ps)
	for k := range ids {
		if found[k+1] {
			out[k] = values[k+1] - values[0]
		}
	}
	return out
}

type AnointCompare struct {
	Note   string
	Header string
	Base   *Environment
	New    *Environment
}

//pob:src/Classes/ItemsTab.lua:3028-3066
func (m *MiscCalculator) CompareAnoint(raw string, slot int, id int64, actionText string) (AnointCompare, bool) {
	item := ParseItem(raw)
	if item.Base == nil {
		return AnointCompare{}, false
	}
	if actionText == "" {
		actionText = "Anointing"
	}
	name := ""
	header := "^7" + actionText + " nothing will give you: "
	if id != 0 {
		n, ok := anointNode(id)
		if !ok {
			return AnointCompare{}, false
		}
		name = *n.Name
		if slices.Contains(m.build.Build.PassiveNodes, id) {
			return AnointCompare{Note: "^7" + actionText + " " + name + " changes nothing because this node is already allocated on the tree."}, true
		}
		if slices.Contains(ItemAnoints(item), name) {
			return AnointCompare{Note: "^7" + actionText + " " + name + " changes nothing because this node is already anointed."}, true
		}
		header = "^7" + actionText + " " + name + " will give you: "
	}
	repSlot := item.Base.Type
	text, _ := anointText(raw, slot, name)
	return AnointCompare{
		Header: header,
		Base:   m.Calc(&CalcOverride{RepSlotName: repSlot, RepItem: item}),
		New:    m.Calc(&CalcOverride{RepSlotName: repSlot, RepItem: ParseItem(text)}),
	}, true
}
