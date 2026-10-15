package calculator

import (
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/utils"
)

type craftFlags struct {
	implicitsCannotBeChanged bool
	supportCrucibleTree      bool
	shieldCrucibleTree       bool
	twoHandSwordCrucibleTree bool
	restrictTag              bool
	restrictDamageType       bool
	no                       map[string]bool
	only                     map[string]bool
}

//pob:block:src/Classes/Item.lua:1212-1265
func itemCraftFlags(item *Item) craftFlags {
	f := craftFlags{no: map[string]bool{}, only: map[string]bool{}}
	for _, list := range [][]ModLine{item.EnchantModLines, item.ScourgeModLines, item.ClassRequirementModLines, item.ImplicitModLines, item.ExplicitModLines} {
		for _, ml := range list {
			if ml.Disabled {
				continue
			}
			switch l := strings.ToLower(ml.Line); {
			case l == "implicit modifiers cannot be changed":
				f.implicitsCannotBeChanged = true
			case l == "has a crucible passive skill tree with only support passive skills":
				f.supportCrucibleTree = true
			case l == "has a crucible passive skill tree":
				f.shieldCrucibleTree = true
			case l == "has a two handed sword crucible passive skill tree":
				f.twoHandSwordCrucibleTree = true
			case l == "cannot roll caster modifiers":
				f.restrictTag, f.no["caster"] = true, true
			case l == "cannot roll attack modifiers":
				f.restrictTag, f.no["attack"] = true, true
			case strings.HasPrefix(l, "cannot roll modifiers of non-") && strings.HasSuffix(l, " damage types"):
				if el := strings.TrimSuffix(strings.TrimPrefix(l, "cannot roll modifiers of non-"), " damage types"); slices.Contains(damageElements, el) {
					f.restrictDamageType, f.only[el] = true, true
				}
			}
		}
	}
	return f
}

var damageElements = []string{"fire", "cold", "lightning", "chaos", "physical"}

func influenceKeys(i ItemInfluence) []string {
	var out []string
	for _, k := range []struct {
		on  bool
		key string
	}{{i.Shaper, "shaper"}, {i.Elder, "elder"}, {i.Adjudicator, "adjudicator"}, {i.Basilisk, "basilisk"}, {i.Crusader, "crusader"}, {i.Eyrie, "eyrie"}, {i.Cleansing, "cleansing"}, {i.Tangle, "tangle"}} {
		if k.on {
			out = append(out, k.key)
		}
	}
	return out
}

func modSpawnWeight(item *Item, mod *data.AffixMod) float64 {
	return modSpawnWeightTags(item, mod, nil, nil, nil)
}

//pob:src/Classes/Item.lua:1693-1752
func modSpawnWeightTags(item *Item, mod *data.AffixMod, includeTags, excludeTags map[string]bool, baseTags []string) float64 {
	if item.Base == nil {
		return 0
	}
	base := data.CraftBaseInfo(item.Base.Name)
	if baseTags == nil {
		baseTags = base.Tags
	}
	hasInfluenceTag := func(key string) bool {
		for _, k := range influenceKeys(item.Influence) {
			if base.InfluenceTags[k] == key {
				return true
			}
		}
		return false
	}
	flags := itemCraftFlags(item)
	if flags.restrictTag {
		for _, key := range mod.ModTags {
			if flags.no[key] {
				return 0
			}
		}
	}
	if flags.restrictDamageType && len(flags.only) > 0 {
		for _, key := range mod.ModTags {
			if slices.Contains(damageElements, key) && !flags.only[key] {
				return 0
			}
		}
	}
	weight := 0.0
	for k, key := range mod.WeightKey {
		if (slices.Contains(baseTags, key) || includeTags[key] || hasInfluenceTag(key)) && !excludeTags[key] {
			weight = float64(mod.WeightVal[k])
			if hasInfluenceTag(key) && strings.Contains(mod.Affix, "Elevated") {
				weight = 1000
			}
			break
		}
	}
	for k, key := range mod.WeightMultiplierKey {
		if (slices.Contains(baseTags, key) || includeTags[key] || hasInfluenceTag(key)) && !excludeTags[key] {
			weight = weight * float64(mod.WeightMultiplierVal[k]) / 100
			break
		}
	}
	return weight
}

type rareLikeUnique struct {
	affixes              func() []data.AffixMod
	validBases           func() [][]string
	prefixLimit          int
	suffixLimit          int
	ignoreModType        bool
	allowDuplicateGroups bool
}

//pob:block:src/Modules/Data.lua:1330-1384
var (
	subsumeTheSourceMods = sync.OnceValue(func() []data.AffixMod {
		t, _ := data.AffixTable("JewelAbyss")
		var out []data.AffixMod
		for _, mod := range t {
			if mod.Type != "Corrupted" {
				out = append(out, mod)
			}
		}
		return out
	})
	veiledAffixes = func(affixes ...string) func() []data.AffixMod {
		return sync.OnceValue(func() []data.AffixMod {
			var out []data.AffixMod
			for _, mod := range data.VeiledMods() {
				if slices.Contains(affixes, mod.Affix) {
					out = append(out, mod)
				}
			}
			return out
		})
	}
	veiledMods        = veiledAffixes("Chosen", "of the Order")
	veiledSuffixes    = veiledAffixes("of the Order")
	caneOfKulemakMods = veiledAffixes("Chosen", "Catarina's", "of the Order")
	queensHungerMods  = caneOfKulemakMods
	explicitMods      = func() []data.AffixMod {
		t, _ := data.AffixTable("Explicit")
		return t
	}
	thatWhichWasTaken = sync.OnceValue(func() []data.AffixMod {
		t, _ := data.AffixTable("JewelCharm")
		var out []data.AffixMod
		for _, mod := range t {
			if !strings.HasSuffix(mod.ID, "1") {
				out = append(out, mod)
			}
		}
		return out
	})
	veiledMasterSpawnTags = sync.OnceValue(func() []string {
		var out []string
		for _, mod := range data.VeiledMods() {
			for _, tag := range mod.WeightKey {
				if veiledPrefixTagRe.MatchString(tag) || veiledSuffixTagRe.MatchString(tag) {
					out = append(out, tag)
				}
			}
		}
		return out
	})
	veiledPrefixTagRe = regexp.MustCompile(`^[A-Za-z_]+_veiled_prefix`)
	veiledSuffixTagRe = regexp.MustCompile(`^[A-Za-z_]+_veiled_suffix`)
)

func baseListTags(baseType, subType string) func() [][]string {
	return sync.OnceValue(func() [][]string {
		var out [][]string
		bases := data.CraftBases()
		for _, name := range slices.Sorted(maps.Keys(bases)) {
			if b := bases[name]; b.Type == baseType && b.SubType == subType && !b.Hidden {
				out = append(out, b.Tags)
			}
		}
		return out
	})
}

func baseWithTags(name string, tags func() []string) func() [][]string {
	return sync.OnceValue(func() [][]string {
		return [][]string{append(slices.Clone(data.CraftBaseInfo(name).Tags), tags()...)}
	})
}

//pob:block:src/Modules/Data.lua:1398-1455
var rareLikeUniques = map[string]rareLikeUnique{
	"subsume the source": {
		validBases:           baseListTags("Jewel", "Abyss"),
		affixes:              subsumeTheSourceMods,
		prefixLimit:          4,
		suffixLimit:          0,
		ignoreModType:        true,
		allowDuplicateGroups: true,
	},
	"the crimson storm": {
		affixes:     veiledSuffixes,
		prefixLimit: 0,
		suffixLimit: 1,
	},
	"dread captain's cutlass": {
		validBases:  baseWithTags("Ghostflame Blade", func() []string { return []string{"deepwater_sword"} }),
		affixes:     explicitMods,
		prefixLimit: 3,
		suffixLimit: 3,
	},
	"paradoxica": {
		affixes:     veiledMods,
		prefixLimit: 1,
		suffixLimit: 1,
	},
	"cane of kulemak": {
		validBases:  baseWithTags("Serpentine Staff", func() []string { return []string{"catarina_veiled_prefix"} }),
		affixes:     caneOfKulemakMods,
		prefixLimit: 2,
		suffixLimit: 2,
	},
	"replica paradoxica": {
		validBases: baseWithTags("Vaal Rapier", func() []string {
			var out []string
			for _, tag := range veiledMasterSpawnTags() {
				if strings.Contains(tag, "prefix") {
					out = append(out, tag)
				}
			}
			return out
		}),
		affixes:     func() []data.AffixMod { return data.VeiledMods() },
		prefixLimit: 3,
		suffixLimit: 3,
	},
	"the queen's hunger": {
		validBases:  baseWithTags("Vaal Regalia", func() []string { return []string{"catarina_veiled_prefix"} }),
		affixes:     queensHungerMods,
		prefixLimit: 1,
		suffixLimit: 1,
	},
	"that which was taken": {
		validBases:    baseListTags("Jewel", "Charm"),
		affixes:       thatWhichWasTaken,
		prefixLimit:   4,
		suffixLimit:   0,
		ignoreModType: true,
	},
}

//pob:block:src/Classes/Item.lua:1032-1040
func itemAffixes(item *Item) []data.AffixMod {
	if rl, ok := item.rareLike(); ok {
		return rl.affixes()
	}
	baseType, subType := item.Base.Type, item.Base.SubType
	if pb, ok := data.CraftBaseKnown(item.Base.Name); ok {
		baseType, subType = pb.Type, pb.SubType
	}
	if subType != "" {
		if t, ok := data.AffixTable(baseType + subType); ok {
			return t
		}
	}
	if t, ok := data.AffixTable(baseType); ok {
		return t
	}
	t, _ := data.AffixTable("Item")
	return t
}

type CorruptMod struct {
	ID      string
	Label   string
	Lines   []string
	Group   string
	ModTags []string
}

type CorruptRoll struct {
	Index int
	Line  string
	Value float64
}

type CorruptChoices struct {
	Unique                   bool
	ImplicitsCannotBeChanged bool
	MaxImplicits             int
	Glimpse                  []CorruptMod
	Corrupted                []CorruptMod
	ScourgeUpside            []CorruptMod
	ScourgeDownside          []CorruptMod
	Rolls                    []CorruptRoll
}

var (
	corruptSortParenRe = regexp.MustCompile(`\(.*?\)`)
	corruptSortSignRe  = regexp.MustCompile(`[+\-%]`)
	corruptSortDigitRe = regexp.MustCompile(`\d+`)
)

func corruptSortKey(m data.AffixMod) string {
	if len(m.Lines) == 0 {
		return ""
	}
	s := corruptSortParenRe.ReplaceAllString(strings.ToLower(m.Lines[0]), "$")
	return corruptSortDigitRe.ReplaceAllString(corruptSortSignRe.ReplaceAllString(s, ""), "$")
}

//pob:src/Classes/ItemsTab.lua:3145-3175
func buildImplicitList(item *Item, modType string) []CorruptMod {
	var mods []data.AffixMod
	if modType == corruptGlimpse {
		mods = slices.Clone(data.SpecialCorruptionMods())
	} else {
		for _, mod := range itemAffixes(item) {
			if mod.Type == modType && modSpawnWeight(item, &mod) > 0 {
				mods = append(mods, mod)
			}
		}
	}
	slices.SortStableFunc(mods, func(a, b data.AffixMod) int {
		if c := strings.Compare(corruptSortKey(a), corruptSortKey(b)); c != 0 {
			return c
		}
		return a.Level - b.Level
	})
	out := make([]CorruptMod, len(mods))
	for k, m := range mods {
		out[k] = CorruptMod{ID: m.ID, Label: strings.Join(m.Lines, "/"), Lines: m.Lines, Group: m.Group, ModTags: m.ModTags}
	}
	return out
}

const corruptGlimpse = "Glimpse of Chaos"

func specialCorruptLine(item *Item) *ModLine {
	var out *ModLine
	for k := range item.ImplicitModLines {
		for _, m := range item.ImplicitModLines[k].ModList {
			if m.Name() == "CorruptImplicitCount" {
				out = &item.ImplicitModLines[k]
			}
		}
	}
	return out
}

//pob:block:src/Classes/ItemsTab.lua:3129-3142,src/Classes/ItemsTab.lua:3344-3378,src/Classes/Item.lua:1050
func ItemCorruptChoices(raw string) (CorruptChoices, bool) {
	item := ParseItem(raw)
	if item.Base == nil || item.Base.Type == "Flask" {
		return CorruptChoices{}, false
	}
	maxImplicits := 2
	if item.CorruptImplicitCount != 0 {
		maxImplicits = int(item.CorruptImplicitCount)
	}
	for _, ml := range item.ImplicitModLines {
		for _, m := range ml.ModList {
			if m.Name() == "CorruptImplicitCount" {
				maxImplicits--
			}
		}
	}
	out := CorruptChoices{
		Unique:                   item.Rarity == RarityUnique || item.Rarity == RarityRelic,
		ImplicitsCannotBeChanged: itemCraftFlags(item).implicitsCannotBeChanged,
		MaxImplicits:             maxImplicits,
		Corrupted:                buildImplicitList(item, "Corrupted"),
		ScourgeUpside:            buildImplicitList(item, "ScourgeUpside"),
		ScourgeDownside:          buildImplicitList(item, "ScourgeDownside"),
	}
	if strings.Contains(luaLower(item.Title), "glimpse of chaos") && item.IsUnique {
		out.Glimpse = buildImplicitList(item, corruptGlimpse)
	}
	if out.Unique {
		for k, ml := range item.ExplicitModLines {
			r := *defaultItemAffixQuality
			if ml.Range != nil {
				r = *ml.Range
			}
			scalar := ml.ValueScalar
			if scalar == 0 {
				scalar = 1
			}
			one, two := 1.0, 2.0
			if data.ApplyRange(ml.Line, r, &scalar, &one) == data.ApplyRange(ml.Line, r, &scalar, &two) || !item.ItemModLineVisible(ml) {
				continue
			}
			v := 1.0
			if ml.CorruptedRange != nil {
				v = *ml.CorruptedRange
			}
			out.Rolls = append(out.Rolls, CorruptRoll{Index: k, Line: data.ApplyRange(ml.Line, r, &scalar, &v), Value: v})
		}
	}
	return out, true
}

//pob:src/Classes/ItemsTab.lua:3350-3354
func CorruptedRangeLine(raw string, index int, value float64) string {
	item := ParseItem(raw)
	if index < 0 || index >= len(item.ExplicitModLines) {
		return ""
	}
	ml := item.ExplicitModLines[index]
	r := *defaultItemAffixQuality
	if ml.Range != nil {
		r = *ml.Range
	}
	scalar := ml.ValueScalar
	if scalar == 0 {
		scalar = 1
	}
	return data.ApplyRange(ml.Line, r, &scalar, &value)
}

func corruptModsByID(item *Item, modType string, ids []string) []data.AffixMod {
	var out []data.AffixMod
	pool := itemAffixes(item)
	if modType == corruptGlimpse {
		pool = data.SpecialCorruptionMods()
	}
	for _, id := range ids {
		for _, mod := range pool {
			if modType == corruptGlimpse && mod.ID == id {
				out = append(out, mod)
				break
			}
			if mod.ID == id && mod.Type == modType {
				out = append(out, mod)
				break
			}
		}
	}
	return out
}

func corruptLines(mod data.AffixMod, scourge bool) []string {
	out := make([]string, 0, len(mod.Lines))
	for _, line := range mod.Lines {
		if scourge {
			line = "{scourge}" + line
		}
		if len(mod.ModTags) > 0 {
			line = "{tags:" + strings.Join(mod.ModTags, ",") + "}" + line
		}
		out = append(out, line)
	}
	return out
}

func markCorrupted(b *implicitBlock) {
	if !slices.ContainsFunc(b.after, func(l string) bool { return strings.TrimSpace(l) == "Corrupted" }) {
		b.after = append(b.after, "Corrupted")
	}
}

//pob:src/Classes/ItemsTab.lua:3290-3343
func CorruptItemText(raw string, source string, ids []string, downside []string, rolls map[int]float64) (string, bool) {
	item := ParseItem(raw)
	if item.Base == nil {
		return raw, false
	}
	b := craftBlock(raw, item)
	markCorrupted(&b)
	if rolls == nil {
		var lines []string
		scourge := source == "Scourge"
		if scourge {
			for _, mod := range slices.Concat(corruptModsByID(item, "ScourgeUpside", ids), corruptModsByID(item, "ScourgeDownside", downside)) {
				lines = append(lines, corruptLines(mod, true)...)
			}
		} else {
			modType := "Corrupted"
			if source == corruptGlimpse {
				modType = corruptGlimpse
			}
			for _, mod := range corruptModsByID(item, modType, ids) {
				lines = append(lines, corruptLines(mod, false)...)
			}
		}
		if len(lines) > 0 {
			if scourge {
				b.scourge = lines
			} else {
				//pob:block:src/Classes/ItemsTab.lua:3294-3303,src/Classes/ItemsTab.lua:3325-3327
				if special := specialCorruptLine(item); special != nil {
					lines = append([]string{writeRawModLine(*special)}, lines...)
				}
				b.implicits = lines
			}
		}
		return b.join(), true
	}
	text := b.join()
	lines := splitRaw(text)
	parsed := ParseItem(text)
	next := 0
	for k, ml := range parsed.ExplicitModLines {
		for j := next; j < len(lines); j++ {
			if rawModText(lines[j]) != ml.Line {
				continue
			}
			next = j + 1
			if v, ok := rolls[k]; ok {
				line := corruptedRangeRe.ReplaceAllString(strings.TrimSpace(lines[j]), "")
				if v != 1 {
					line = "{corruptedRange:" + strconv.FormatFloat(utils.RoundTo(v, 2), 'f', -1, 64) + "}" + line
				}
				lines[j] = line
			}
			break
		}
	}
	return strings.Join(lines, "\n"), true
}

//pob:src/Classes/ItemsTab.lua:3219-3243
func (m *MiscCalculator) CorruptSortValues(raw string, source string, modType string, ids []string, stat string) []float64 {
	ps, ok := powerStatByName(stat)
	item := ParseItem(raw)
	out := make([]float64, len(ids))
	if !ok || item.Base == nil {
		return out
	}
	texts, found := make([]string, len(ids)), make([]bool, len(ids))
	for k, id := range ids {
		mods := corruptModsByID(item, modType, []string{id})
		if len(mods) == 0 {
			continue
		}
		mod := mods[0]
		b := craftBlock(raw, item)
		markCorrupted(&b)
		if modType == "ScourgeUpside" || modType == "ScourgeDownside" {
			b.scourge = corruptLines(mod, source == "Scourge")
		} else {
			b.implicits = corruptLines(mod, source == "Scourge")
		}
		texts[k], found[k] = b.join(), true
	}
	return m.sortValues(item.getPrimarySlot(), texts, found, ps)
}

func (i *Item) rareLike() (rareLikeUnique, bool) {
	if !i.RareLikeUnique {
		return rareLikeUnique{}, false
	}
	rl, ok := rareLikeUniques[luaLower(i.Title)]
	return rl, ok
}
