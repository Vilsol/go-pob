package calculator

import (
	"regexp"
	"slices"
	"strconv"
	"strings"

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

//pob:block:src/Classes/Item.lua:1188-1241
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
	return modSpawnWeightTags(item, mod, nil, nil)
}

//pob:src/Classes/Item.lua:1654-1713
func modSpawnWeightTags(item *Item, mod *data.AffixMod, includeTags, excludeTags map[string]bool) float64 {
	if item.Base == nil {
		return 0
	}
	base := data.CraftBaseInfo(item.Base.Name)
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
		if (slices.Contains(base.Tags, key) || includeTags[key] || hasInfluenceTag(key)) && !excludeTags[key] {
			weight = float64(mod.WeightVal[k])
			if hasInfluenceTag(key) && strings.Contains(mod.Affix, "Elevated") {
				weight = 1000
			}
			break
		}
	}
	for k, key := range mod.WeightMultiplierKey {
		if (slices.Contains(base.Tags, key) || includeTags[key] || hasInfluenceTag(key)) && !excludeTags[key] {
			weight = weight * float64(mod.WeightMultiplierVal[k]) / 100
			break
		}
	}
	return weight
}

//pob:block:src/Modules/Data.lua:1322-1348
var rareLikeUniques = map[string]bool{
	"subsume the source":      true,
	"the crimson storm":       true,
	"dread captain's cutlass": true,
}

//pob:block:src/Classes/Item.lua:1008-1010
func itemAffixes(item *Item) []data.AffixMod {
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

//pob:src/Classes/ItemsTab.lua:3027-3049
func buildImplicitList(item *Item, modType string) []CorruptMod {
	var mods []data.AffixMod
	for _, mod := range itemAffixes(item) {
		if mod.Type == modType && modSpawnWeight(item, &mod) > 0 {
			mods = append(mods, mod)
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

//pob:block:src/Classes/ItemsTab.lua:3176-3212,src/Classes/Item.lua:1026
func ItemCorruptChoices(raw string) (CorruptChoices, bool) {
	item := ParseItem(raw)
	if item.Base == nil || item.Base.Type == "Flask" {
		return CorruptChoices{}, false
	}
	out := CorruptChoices{
		Unique:                   item.Rarity == RarityUnique || item.Rarity == RarityRelic,
		ImplicitsCannotBeChanged: itemCraftFlags(item).implicitsCannotBeChanged,
		Corrupted:                buildImplicitList(item, "Corrupted"),
		ScourgeUpside:            buildImplicitList(item, "ScourgeUpside"),
		ScourgeDownside:          buildImplicitList(item, "ScourgeDownside"),
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

//pob:src/Classes/ItemsTab.lua:3182-3186
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
	for _, id := range ids {
		for _, mod := range itemAffixes(item) {
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

//pob:src/Classes/ItemsTab.lua:3136-3175
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
			for _, mod := range corruptModsByID(item, "Corrupted", ids) {
				lines = append(lines, corruptLines(mod, false)...)
			}
		}
		if len(lines) > 0 {
			if scourge {
				b.scourge = lines
			} else {
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

//pob:src/Classes/ItemsTab.lua:3064-3088
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
		if modType == "Corrupted" {
			b.implicits = corruptLines(mod, source == "Scourge")
		} else {
			b.scourge = corruptLines(mod, source == "Scourge")
		}
		texts[k], found[k] = b.join(), true
	}
	return m.sortValues(item.getPrimarySlot(), texts, found, ps)
}
