package calculator2

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/mod"
)

// The item oracle's rows carry PoB-PoE2's own mod tables as JSON. These types rebuild them as Go
// mods the item code can work on, and print Go mods back in the same JSON shape.

type oracleKV struct {
	Key            string
	Value          any
	KeyOfScaledMod string
	Rest           map[string]any
}

type oracleNested struct {
	Mod  mod.Mod
	Rest map[string]any
}

type oracleList struct {
	JSON string
}

type oracleTag struct {
	TagType    mod.Type
	JSON       string
	Unscalable bool
}

func (t *oracleTag) Type() mod.Type { return t.TagType }

func (x oracleKV) Clone(bool) mod.Cloneable   { return x }
func (x oracleList) Clone(bool) mod.Cloneable { return x }

// oracleKVFloat is a { key, value, keyOfScaledMod = "value" } list whose value ScaleAddMod scales.
type oracleKVFloat struct {
	Key            string
	Value          float64
	KeyOfScaledMod string
	Rest           map[string]any
}

func (x oracleKVFloat) Clone(bool) mod.Cloneable { return x }
func (x oracleNested) Clone(bool) mod.Cloneable {
	return oracleNested{Mod: x.Mod.Clone(), Rest: x.Rest}
}

func decodeOracleMod(raw map[string]any) mod.Mod {
	name, _ := raw["name"].(string)
	typ, _ := raw["type"].(string)
	var m mod.Mod
	value := raw["value"]
	if str, ok := value.(string); ok {
		// PoB-PoE2's parser leaves some captured numbers as strings; Lua arithmetic coerces them.
		if f, ok := luaToNumber(str); ok {
			value = f
		}
	}
	switch v := value.(type) {
	case float64:
		m = mod.NewFloat(name, mod.Type(typ), v)
	case bool:
		m = mod.NewFlag(name, v)
		m.(*mod.BaseMod).ModType = mod.Type(typ)
	case map[string]any:
		m = mod.NewList(name, decodeOracleList(v))
		m.(*mod.BaseMod).ModType = mod.Type(typ)
	default:
		b, _ := json.Marshal(v)
		m = mod.NewList(name, oracleList{JSON: string(b)})
		m.(*mod.BaseMod).ModType = mod.Type(typ)
	}
	if f, ok := raw["flags"].(float64); ok {
		m.Flag(mod.MFlag(f))
	}
	if f, ok := raw["keywordFlags"].(float64); ok {
		m.KeywordFlag(mod.KeywordFlag(f))
	}
	if s, ok := raw["source"].(string); ok {
		m.Source(mod.Source(s))
	}
	tags, hasTags := raw["tags"].([]any)
	if !hasTags {
		for n := 1; raw[strconv.Itoa(n)] != nil; n++ {
			tags = append(tags, raw[strconv.Itoa(n)])
		}
	}
	for _, t := range tags {
		if tm, ok := t.(map[string]any); ok {
			m.Tag(decodeOracleTag(tm))
		} else {
			m.Tag(nil)
		}
	}
	return m
}

func decodeOracleList(v map[string]any) any {
	rest := map[string]any{}
	for k, val := range v {
		rest[k] = val
	}
	if inner, ok := v["mod"].(map[string]any); ok {
		delete(rest, "mod")
		return oracleNested{Mod: decodeOracleMod(inner), Rest: rest}
	}
	if k, ok := v["key"].(string); ok {
		if _, has := v["value"]; has {
			delete(rest, "key")
			delete(rest, "value")
			ks, _ := v["keyOfScaledMod"].(string)
			if f, ok := v["value"].(float64); ok && ks == "value" {
				delete(rest, "keyOfScaledMod")
				return oracleKVFloat{Key: k, Value: f, KeyOfScaledMod: ks, Rest: rest}
			}
			return oracleKV{Key: k, Value: v["value"], Rest: rest}
		}
	}
	b, _ := json.Marshal(v)
	return oracleList{JSON: string(b)}
}

func decodeOracleTag(t map[string]any) mod.Tag {
	typ, _ := t["type"].(string)
	onlyKeys := func(keys ...string) bool {
		for k := range t {
			found := false
			for _, want := range keys {
				found = found || k == want
			}
			if !found {
				return false
			}
		}
		return true
	}
	switch {
	case typ == "SlotNumber" && onlyKeys("type", "num"):
		n, _ := t["num"].(float64)
		return mod.SlotNumber(int(n))
	case typ == "InSlot" && onlyKeys("type", "num"):
		n, _ := t["num"].(float64)
		return mod.InSlot(int(n))
	case typ == "Condition" && onlyKeys("type", "var", "neg"):
		v, _ := t["var"].(string)
		neg, _ := t["neg"].(bool)
		return mod.Condition(v).Neg(neg)
	}
	b, _ := json.Marshal(t)
	unscalable, _ := t["unscalable"].(bool)
	return &oracleTag{TagType: mod.Type(typ), JSON: string(b), Unscalable: unscalable}
}

func encodeOracleTag(t mod.Tag) any {
	switch x := t.(type) {
	case nil:
		return nil
	case *mod.SlotNumberTag:
		return map[string]any{"type": "SlotNumber", "num": float64(x.N)}
	case *mod.InSlotTag:
		return map[string]any{"type": "InSlot", "num": float64(x.N)}
	case *mod.ConditionTag:
		out := map[string]any{"type": "Condition"}
		if len(x.VarList) == 1 {
			out["var"] = x.VarList[0]
		}
		if x.Negative {
			out["neg"] = true
		}
		return out
	case *oracleTag:
		var v any
		_ = json.Unmarshal([]byte(x.JSON), &v)
		return v
	}
	return fmt.Sprintf("<go tag %T>", t)
}

func encodeOracleList(v any) any {
	switch x := v.(type) {
	case oracleKV:
		out := map[string]any{"key": x.Key, "value": x.Value}
		for k, val := range x.Rest {
			out[k] = val
		}
		return out
	case oracleKVFloat:
		out := map[string]any{"key": x.Key, "value": x.Value, "keyOfScaledMod": x.KeyOfScaledMod}
		for k, val := range x.Rest {
			out[k] = val
		}
		return out
	case oracleNested:
		out := map[string]any{"mod": encodeOracleInnerMod(x.Mod)}
		for k, val := range x.Rest {
			out[k] = val
		}
		return out
	case oracleList:
		var out any
		_ = json.Unmarshal([]byte(x.JSON), &out)
		return out
	case mod.ArmourData:
		return map[string]any{"key": x.Key, "value": x.Value}
	case mod.MinionModifier:
		return map[string]any{"mod": encodeOracleInnerMod(x.Mod)}
	}
	return fmt.Sprintf("<go list %T>", v)
}

func encodeOracleMod(m mod.Mod) map[string]any {
	out := map[string]any{
		"name": m.Name(), "type": string(m.Type()), "flags": float64(m.Flags()),
		"keywordFlags": float64(m.KeywordFlags()),
	}
	if s := m.GetSource(); s != "" {
		out["source"] = string(s)
	}
	if v := m.Value(); v != nil {
		switch v.Type() {
		case mod.ModValueMultiTypeFloat:
			out["value"] = v.Float()
		case mod.ModValueMultiTypeFlag:
			out["value"] = v.Flag()
		case mod.ModValueMultiTypeList:
			out["value"] = encodeOracleList(v.ValueList)
		}
	}
	tags := make([]any, 0, len(m.Tags()))
	for _, t := range m.Tags() {
		tags = append(tags, encodeOracleTag(t))
	}
	out["tags"] = tags
	return out
}

// encodeOracleInnerMod prints a value.mod as the oracle's generic encoder does: tags as "1".."n".
func encodeOracleInnerMod(m mod.Mod) map[string]any {
	out := encodeOracleMod(m)
	tags, _ := out["tags"].([]any)
	delete(out, "tags")
	for n, t := range tags {
		out[strconv.Itoa(n+1)] = t
	}
	return out
}

func encodeOracleModList(list []mod.Mod) []any {
	out := make([]any, 0, len(list))
	for _, m := range list {
		out = append(out, encodeOracleMod(m))
	}
	return out
}

// oracleEqual compares two decoded JSON values, with a relative tolerance on numbers and the
// oracle's empty table ({}) equal to an empty list.
func oracleEqual(a, b any) bool {
	if isEmptyJSON(a) && isEmptyJSON(b) {
		return true
	}
	if f, ok := numericString(a); ok {
		a = f
	}
	if f, ok := numericString(b); ok {
		b = f
	}
	switch x := a.(type) {
	case float64:
		y, ok := b.(float64)
		return ok && (x == y || math.Abs(x-y) <= 1e-9*math.Max(math.Abs(x), math.Abs(y)))
	case []any:
		y, ok := b.([]any)
		if !ok || len(x) != len(y) {
			return false
		}
		for i := range x {
			if !oracleEqual(x[i], y[i]) {
				return false
			}
		}
		return true
	case map[string]any:
		y, ok := b.(map[string]any)
		if !ok {
			return false
		}
		for k, v := range x {
			if !oracleEqual(v, y[k]) {
				return false
			}
		}
		for k, v := range y {
			if _, seen := x[k]; !seen && !oracleEqual(nil, v) {
				return false
			}
		}
		return true
	}
	return reflect.DeepEqual(a, b)
}

func numericString(v any) (float64, bool) {
	if s, ok := v.(string); ok {
		return luaToNumber(s)
	}
	return 0, false
}

func isEmptyJSON(v any) bool {
	switch x := v.(type) {
	case nil:
		return true
	case []any:
		return len(x) == 0
	case map[string]any:
		return len(x) == 0
	}
	return false
}

func oracleJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// luaArray is a JSON array the oracle prints as {} when empty.
type luaArray[T any] []T

func (a *luaArray[T]) UnmarshalJSON(b []byte) error {
	if string(b) == "{}" || string(b) == "null" {
		*a = nil
		return nil
	}
	return json.Unmarshal(b, (*[]T)(a)) //nolint:wrapcheck // a test decoder
}

type oracleParse struct {
	ModList []map[string]any `json:"modList"`
	Extra   *string          `json:"extra"`
	nilList bool
}

func (p *oracleParse) UnmarshalJSON(b []byte) error {
	var raw struct {
		ModList json.RawMessage `json:"modList"`
		Extra   *string         `json:"extra"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err //nolint:wrapcheck // a test decoder
	}
	p.Extra = raw.Extra
	if string(raw.ModList) == "null" || len(raw.ModList) == 0 {
		p.nilList = true
		return nil
	}
	if string(raw.ModList) == "{}" {
		p.ModList = []map[string]any{}
		return nil
	}
	return json.Unmarshal(raw.ModList, &p.ModList) //nolint:wrapcheck // a test decoder
}

// oracleModSource answers each line from the parse log PoB-PoE2 recorded for the same item, and
// records every line PoB never asked for.
type oracleModSource struct {
	log      map[string]oracleParse
	unlogged []string
}

func (s *oracleModSource) parse(line string) ([]mod.Mod, string) {
	p, ok := s.log[line]
	if !ok {
		s.unlogged = append(s.unlogged, line)
		return nil, line
	}
	extra := ""
	if p.Extra != nil {
		extra = *p.Extra
	}
	if p.nilList {
		return nil, extra
	}
	out := make([]mod.Mod, 0, len(p.ModList))
	for _, m := range p.ModList {
		out = append(out, decodeOracleMod(m))
	}
	return out, extra
}

type oracleModLine struct {
	Line        string           `json:"line"`
	Extra       *string          `json:"extra"`
	Range       *float64         `json:"range"`
	ValueScalar *float64         `json:"valueScalar"`
	Flags       any              `json:"flags"`
	AugmentType *string          `json:"augmentType"`
	Ranged      *string          `json:"ranged"`
	ModList     []map[string]any `json:"modList"`
}

type oracleItem struct {
	Build           string                       `json:"build"`
	ID              int                          `json:"id"`
	Raws            []string                     `json:"raws"`
	Attrib          map[string]float64           `json:"attrib"`
	Ranges          luaArray[map[string]float64] `json:"ranges"`
	Name            *string                      `json:"name"`
	BaseName        *string                      `json:"baseName"`
	Rarity          *string                      `json:"rarity"`
	Title           *string                      `json:"title"`
	Type            *string                      `json:"type"`
	Quality         *float64                     `json:"quality"`
	ItemLevel       *float64                     `json:"itemLevel"`
	Corrupted       bool                         `json:"corrupted"`
	ItemSocketCount int                          `json:"itemSocketCount"`
	Runes           any                          `json:"runes"`
	Requirements    map[string]float64           `json:"requirements"`
	ArmourData      map[string]any               `json:"armourData"`
	WeaponData      any                          `json:"weaponData"`
	SpiritValue     *float64                     `json:"spiritValue"`
	CharmLimit      *float64                     `json:"charmLimit"`
	BuffModLines    []oracleModLine              `json:"buffModLines"`
	EnchantModLines []oracleModLine              `json:"enchantModLines"`
	RuneModLines    []oracleModLine              `json:"runeModLines"`
	ImplicitLines   []oracleModLine              `json:"implicitModLines"`
	ExplicitLines   []oracleModLine              `json:"explicitModLines"`
	SlotMods        [][]map[string]any           `json:"slotMods"`
	ParseLog        map[string]oracleParse       `json:"parseLog"`
}

func optInt(v float64, ok bool) *int {
	if !ok {
		return nil
	}
	return intp(int(v))
}

// loadOracleItem runs the Go item code on the oracle row's own input and mod source.
func loadOracleItem(row *oracleItem) (*Item, *oracleModSource) {
	src := &oracleModSource{log: row.ParseLog}
	get := func(k string) *int { v, ok := row.Attrib[k]; return optInt(v, ok) }
	variants := ItemVariants{Variant: get("variant"), VariantAlt: get("variantAlt"), VariantAlt2: get("variantAlt2"),
		VariantAlt3: get("variantAlt3"), VariantAlt4: get("variantAlt4"), VariantAlt5: get("variantAlt5")}
	ranges := make([]ItemModRange, 0, len(row.Ranges))
	for _, r := range row.Ranges {
		ranges = append(ranges, ItemModRange{ID: int(r["id"]), Range: r["range"]})
	}
	raw := ""
	if len(row.Raws) > 0 {
		raw = row.Raws[len(row.Raws)-1]
	}
	return LoadItem(row.ID, raw, variants, ranges, src.parse), src
}

func modLineFlags(ml *ModLine) []string {
	var out []string
	for _, f := range []struct {
		on   bool
		name string
	}{{ml.Crafted, "crafted"}, {ml.Custom, "custom"}, {ml.Disabled, "disabled"}, {ml.Enchant, "enchant"},
		{ml.Fractured, "fractured"}, {ml.Implicit, "implicit"}, {ml.Desecrated, "desecrated"}, {ml.Mutated, "mutated"},
		{ml.Rune, "rune"}, {ml.Unscalable, "unscalable"}, {ml.Prefix, "prefix"}, {ml.Suffix, "suffix"},
		{ml.Bonded, "bonded"}, {ml.SocketedRuneEffectAlreadyApplied, "socketedRuneEffectAlreadyApplied"}} {
		if f.on {
			out = append(out, f.name)
		}
	}
	return out
}

func fmtFloatPtr(p *float64) string {
	if p == nil {
		return "nil"
	}
	return strconv.FormatFloat(*p, 'g', -1, 64)
}

func strOr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func floatEq(a *float64, b *float64) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return oracleEqual(*a, *b)
}

// compareOracleItem returns the first disagreement per field class between our item and PoB's.
func compareOracleItem(row *oracleItem, item *Item, src *oracleModSource) map[string]string {
	bad := map[string]string{}
	note := func(class, format string, args ...any) {
		if _, seen := bad[class]; !seen {
			bad[class] = fmt.Sprintf(format, args...)
		}
	}
	if len(src.unlogged) > 0 {
		note("parse-request", "we asked for %q, which PoB never parsed", src.unlogged[0])
	}
	if item.BaseName != strOr(row.BaseName) {
		note("base", "base %q, PoB %q", item.BaseName, strOr(row.BaseName))
	}
	if item.Rarity != strOr(row.Rarity) {
		note("rarity", "rarity %q, PoB %q", item.Rarity, strOr(row.Rarity))
	}
	if item.Name != strOr(row.Name) || item.Title != strOr(row.Title) || item.Type != strOr(row.Type) {
		note("name", "name/title/type %q/%q/%q, PoB %q/%q/%q", item.Name, item.Title, item.Type, strOr(row.Name), strOr(row.Title), strOr(row.Type))
	}
	if !floatEq(item.Quality, row.Quality) {
		note("quality", "quality %s, PoB %s", fmtFloatPtr(item.Quality), fmtFloatPtr(row.Quality))
	}
	var pobRunes []string
	if rs, ok := row.Runes.([]any); ok {
		for _, r := range rs {
			s, _ := r.(string)
			pobRunes = append(pobRunes, s)
		}
	}
	if item.ItemSocketCount != row.ItemSocketCount || strings.Join(item.Runes, "|") != strings.Join(pobRunes, "|") {
		note("sockets", "sockets %d runes %v, PoB %d %v", item.ItemSocketCount, item.Runes, row.ItemSocketCount, pobRunes)
	}
	req := item.Requirements
	for k, ours := range map[string]*float64{"level": req.Level, "str": &req.Str, "dex": &req.Dex, "int": &req.Int,
		"strMod": req.StrMod, "dexMod": req.DexMod, "intMod": req.IntMod} {
		pob, ok := row.Requirements[k]
		if ours == nil && !ok {
			continue
		}
		if ours == nil || !ok || !oracleEqual(*ours, pob) {
			note("requirements", "requirements.%s %s, PoB %v (ok=%v)", k, fmtFloatPtr(ours), pob, ok)
		}
	}
	lists := []struct {
		name string
		ours []*ModLine
		pob  []oracleModLine
	}{{"buff", item.BuffModLines, row.BuffModLines}, {"enchant", item.EnchantModLines, row.EnchantModLines},
		{"rune", item.RuneModLines, row.RuneModLines}, {"implicit", item.ImplicitModLines, row.ImplicitLines},
		{"explicit", item.ExplicitModLines, row.ExplicitLines}}
	for _, l := range lists {
		if len(l.ours) != len(l.pob) {
			var ol, pl []string
			for _, m := range l.ours {
				ol = append(ol, m.Line)
			}
			for _, m := range l.pob {
				pl = append(pl, m.Line)
			}
			note("modline-count", "%s lines %q, PoB %q", l.name, ol, pl)
			continue
		}
		for n, ours := range l.ours {
			pob := l.pob[n]
			if ours.Line != pob.Line || ours.Extra != strOr(pob.Extra) {
				note("modline-text", "%s[%d] %q extra %q, PoB %q extra %q", l.name, n, ours.Line, ours.Extra, pob.Line, strOr(pob.Extra))
			}
			if !floatEq(ours.Range, pob.Range) || !floatEq(ours.ValueScalar, pob.ValueScalar) {
				note("modline-range", "%s[%d] %q range %s scalar %s, PoB %s %s", l.name, n, ours.Line, fmtFloatPtr(ours.Range), fmtFloatPtr(ours.ValueScalar), fmtFloatPtr(pob.Range), fmtFloatPtr(pob.ValueScalar))
			}
			var pf []string
			if fs, ok := pob.Flags.([]any); ok {
				for _, f := range fs {
					s, _ := f.(string)
					pf = append(pf, s)
				}
			}
			of := modLineFlags(ours)
			sort.Strings(pf)
			sort.Strings(of)
			if strings.Join(of, ",") != strings.Join(pf, ",") {
				note("modline-flags", "%s[%d] %q flags %v, PoB %v", l.name, n, ours.Line, of, pf)
			}
			if pob.Ranged != nil && item.RangedLine(ours) != *pob.Ranged {
				note("modline-ranged", "%s[%d] %q ranged %q, PoB %q", l.name, n, ours.Line, item.RangedLine(ours), *pob.Ranged)
			}
			if ours.AugmentType != strOr(pob.AugmentType) {
				note("modline-augment", "%s[%d] %q augment %q, PoB %q", l.name, n, ours.Line, ours.AugmentType, strOr(pob.AugmentType))
			}
			pm := make([]any, len(pob.ModList))
			for k, m := range pob.ModList {
				pm[k] = m
			}
			om := encodeOracleModList(ours.ModList)
			if !oracleEqual(om, pm) {
				note("modline-mods", "%s[%d] %q mods %s, PoB %s", l.name, n, ours.Line, oracleJSON(om), oracleJSON(pm))
			}
		}
	}
	for _, k := range []string{"Armour", "Evasion", "EnergyShield", "Ward", "BlockChance"} {
		var pob any
		if row.ArmourData != nil {
			pob = row.ArmourData[k]
		}
		var ours any
		if v, ok := item.ArmourData[k]; ok {
			ours = v
		}
		if !oracleEqual(ours, pob) {
			note("defence", "armourData.%s %v, PoB %v", k, ours, pob)
		}
	}
	if item.Base != nil {
		var ourSlots [][]mod.Mod
		if item.SlotModList != nil {
			ourSlots = item.SlotModList
		} else {
			ourSlots = [][]mod.Mod{item.ModList}
		}
		if len(ourSlots) != len(row.SlotMods) {
			note("slot-mods", "%d slot lists, PoB %d", len(ourSlots), len(row.SlotMods))
		} else {
			for s := range ourSlots {
				pm := make([]any, len(row.SlotMods[s]))
				for k, m := range row.SlotMods[s] {
					pm[k] = m
				}
				om := encodeOracleModList(ourSlots[s])
				if !oracleEqual(om, pm) {
					k := 0
					for k < len(om) && k < len(pm) && oracleEqual(om[k], pm[k]) {
						k++
					}
					var o, p any
					if k < len(om) {
						o = om[k]
					}
					if k < len(pm) {
						p = pm[k]
					}
					note("slot-mods", "slot %d: %d mods, PoB %d; first difference at %d: %s, PoB %s", s+1, len(om), len(pm), k, oracleJSON(o), oracleJSON(p))
					break
				}
			}
		}
	} else if len(row.SlotMods) > 0 {
		note("slot-mods", "no base, PoB has %d slot lists", len(row.SlotMods))
	}
	return bad
}

// knownItemCause names the cause of an item's disagreement when it is one already diagnosed outside
// the item code, or "" when it is not.
func knownItemCause(row *oracleItem, item *Item, src *oracleModSource, bad map[string]string) string {
	if _, ok := bad["modline-augment"]; ok && len(bad) == 1 {
		return "order: equal rune rolls of two augment types, picked in pairs() order"
	}
	return ""
}
