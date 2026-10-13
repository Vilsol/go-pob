package moddb

import (
	"math"
	"strings"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/utils"
)

var _ ModStoreFuncs = (*ModDB)(nil)

//pob:block:src/Classes/ModDB.lua:308
func sourcePrefixMatch(source mod.Source, cfgSource mod.Source) bool {
	prefix, _, _ := strings.Cut(string(source), ":")
	return prefix == string(cfgSource)
}

type ModDB struct {
	*ModStore

	Mods map[string][]mod.Mod
}

//pob:src/Classes/ModDB.lua:20-23
func NewModDB() *ModDB {
	m := &ModDB{
		ModStore: NewModStore(nil),
		Mods:     make(map[string][]mod.Mod),
	}
	m.ModStore.Child = m
	return m
}

func NewModDBSized(n int) *ModDB {
	m := NewModDB()
	m.Mods = make(map[string][]mod.Mod, n)
	return m
}

func (m *ModDB) Clone() ModStoreFuncs {
	if m == nil {
		return nil
	}

	out := NewModDBSized(len(m.Mods))
	out.AddDB(m)
	out.ModStore = m.ModStore.Clone()
	out.ModStore.Child = out
	return out
}

//pob:src/Classes/ModDB.lua:25-31
func (m *ModDB) AddMod(newMod mod.Mod) {
	name := newMod.Name()
	m.Mods[name] = append(m.Mods[name], newMod)
}

//pob:src/Classes/ModStore.lua:250-259, src/Classes/ModDB.lua:302-321
func (m *ModDB) HasMod(modType mod.Type, cfg *ListCfg, names ...string) bool {
	for _, name := range names {
		for _, mo := range m.Mods[name] {
			if mo.Type() == modType &&
				cfgFlags(cfg)&mo.Flags() == mo.Flags() &&
				mod.MatchKeywordFlags(cfgKeywordFlags(cfg), mo.KeywordFlags()) &&
				(cfg == nil || cfg.Source == nil || sourcePrefixMatch(mo.GetSource(), *cfg.Source)) {
				return true
			}
		}
	}

	if m.Parent != nil {
		if parent, ok := m.Parent.(*ModDB); ok {
			return parent.HasMod(modType, cfg, names...)
		}
	}

	return false
}

//pob:src/Classes/ModStore.lua:114-119, src/Classes/ModDB.lua:37-65
func (m *ModDB) ReplaceMod(newMod mod.Mod) {
	if !m.replaceModInternal(newMod) {
		m.AddMod(newMod)
	}
}

//pob:src/Classes/ModDB.lua:37-65
func (m *ModDB) replaceModInternal(newMod mod.Mod) bool {
	for i, mo := range m.Mods[newMod.Name()] {
		if mo.Name() == newMod.Name() &&
			mo.Type() == newMod.Type() &&
			mo.Flags() == newMod.Flags() &&
			mo.KeywordFlags() == newMod.KeywordFlags() &&
			mo.GetSource() == newMod.GetSource() &&
			!mo.Replaced() {
			newMod.SetReplaced(true)
			m.Mods[newMod.Name()][i] = newMod
			return true
		}
	}

	if m.Parent != nil {
		if parent, ok := m.Parent.(*ModDB); ok {
			return parent.replaceModInternal(newMod)
		}
	}

	return false
}

//pob:src/Classes/ModStore.lua:127-132
func (m *ModDB) ConvertMod(oldName string, newMod mod.Mod) {
	if !m.convertModInternal(oldName, newMod) {
		m.AddMod(newMod)
	}
}

//pob:src/Classes/ModDB.lua:76-106
func (m *ModDB) convertModInternal(oldName string, newMod mod.Mod) bool {
	oldList, ok := m.Mods[oldName]
	if ok {
		for i, cur := range oldList {
			if cur.Name() == oldName &&
				cur.Type() == newMod.Type() &&
				cur.Flags() == newMod.Flags() &&
				cur.KeywordFlags() == newMod.KeywordFlags() &&
				cur.GetSource() == newMod.GetSource() {
				m.Mods[oldName] = append(oldList[:i:i], oldList[i+1:]...)
				m.Mods[newMod.Name()] = append(m.Mods[newMod.Name()], newMod)
				return true
			}
		}
	}
	switch parent := m.Parent.(type) {
	case *ModList:
		return parent.convertModInternal(oldName, newMod)
	case *ModDB:
		return parent.convertModInternal(oldName, newMod)
	}
	return false
}

//pob:src/Classes/ModDB.lua:119-130
func (m *ModDB) AddDB(db *ModDB) {
	if db == nil {
		return
	}
	for k, v := range db.Mods {
		m.Mods[k] = append(m.Mods[k], v...)
	}
}

//pob:src/Classes/ModStore.lua:194-205
func (m *ModDB) List(cfg *ListCfg, names ...string) []interface{} {
	result := make([]interface{}, 0)
	m.ListInternal(m, &result, cfg, names...)
	return result
}

//pob:src/Classes/ModDB.lua:243-266
func (m *ModDB) ListInternal(context ModStoreFuncs, result *[]interface{}, cfg *ListCfg, names ...string) {
	for _, name := range names {
		for _, mo := range m.Mods[name] {
			if mo.Type() == mod.TypeList &&
				cfgFlags(cfg)&mo.Flags() == mo.Flags() &&
				mod.MatchKeywordFlags(cfgKeywordFlags(cfg), mo.KeywordFlags()) &&
				(cfg == nil || cfg.Source == nil || sourcePrefixMatch(mo.GetSource(), *cfg.Source)) {

				value := context.evalMod(mo, cfg)
				if value != nil {
					*result = append(*result, value.ValueList)
				}
			}
		}
	}

	if m.Parent != nil {
		m.Parent.ListInternal(context, result, cfg, names...)
	}
}

//pob:src/Classes/ModStore.lua:150-159
func (m *ModDB) Sum(modType mod.Type, cfg *ListCfg, names ...string) float64 {
	return m.SumInternal(m, modType, cfg, names...)
}

//pob:src/Classes/ModDB.lua:91-121
func (m *ModDB) SumInternal(context ModStoreFuncs, modType mod.Type, cfg *ListCfg, names ...string) float64 {
	result := float64(0)

	globalLimits := make(map[string]float64)

	for _, name := range names {
		for _, mo := range m.Mods[name] {
			if mo.Type() == modType &&
				cfgFlags(cfg)&mo.Flags() == mo.Flags() &&
				mod.MatchKeywordFlags(cfgKeywordFlags(cfg), mo.KeywordFlags()) &&
				(cfg == nil || cfg.Source == nil || sourcePrefixMatch(mo.GetSource(), *cfg.Source)) {

				var value float64
				if evaluated := context.evalMod(mo, cfg); evaluated != nil &&
					evaluated.Type() == mod.ModValueMultiTypeFloat {
					value = evaluated.Float()
				}

				value = applyGlobalLimits(globalLimits, tagGlobalLimits(mo), value)

				result += value
			}
		}
	}

	if m.Parent != nil {
		result += m.Parent.SumInternal(context, modType, cfg, names...)
	}

	return result
}

//pob:src/Classes/ModStore.lua:172-181
func (m *ModDB) Flag(cfg *ListCfg, names ...string) bool {
	return m.FlagInternal(m, cfg, names...)
}

//pob:src/Classes/ModDB.lua:196-217
func (m *ModDB) FlagInternal(context ModStoreFuncs, cfg *ListCfg, names ...string) bool {
	for _, name := range names {
		for _, mo := range m.Mods[name] {
			if mo.Type() == mod.TypeFlag &&
				cfgFlags(cfg)&mo.Flags() == mo.Flags() &&
				mod.MatchKeywordFlags(cfgKeywordFlags(cfg), mo.KeywordFlags()) &&
				(cfg == nil || cfg.Source == nil || sourcePrefixMatch(mo.GetSource(), *cfg.Source)) {

				value := context.evalMod(mo, cfg)
				if value != nil && value.Type() == mod.ModValueMultiTypeFlag && value.Flag() {
					return true
				}
			}
		}
	}

	if m.Parent != nil {
		if m.Parent.FlagInternal(context, cfg, names...) {
			return true
		}
	}

	return false
}

//pob:src/Classes/ModStore.lua:161-170
func (m *ModDB) More(cfg *ListCfg, names ...string) float64 {
	return m.MoreInternal(m, cfg, names...)
}

func moreHighPrecisionFor(name string) (int, bool) {
	byType, ok := highPrecisionMods[name]
	if !ok {
		return 0, false
	}
	p, ok := byType[mod.TypeMore]
	return p, ok
}

func luaRound(val float64, dec int) float64 {
	p := math.Pow(10, float64(dec))
	return math.Floor(float64(val*p)+0.5) / p
}

//pob:src/Classes/ModDB.lua:157-194
func (m *ModDB) MoreInternal(context ModStoreFuncs, cfg *ListCfg, names ...string) float64 {
	result := float64(1)
	modPrecision := 0
	havePrecision := false

	globalLimits := make(map[string]float64)

	for _, name := range names {
		modResult := float64(1)
		for _, mo := range m.Mods[name] {
			if mo.Type() == mod.TypeMore &&
				cfgFlags(cfg)&mo.Flags() == mo.Flags() &&
				mod.MatchKeywordFlags(cfgKeywordFlags(cfg), mo.KeywordFlags()) &&
				(cfg == nil || cfg.Source == nil || sourcePrefixMatch(mo.GetSource(), *cfg.Source)) {

				var value float64
				if evaluated := context.evalMod(mo, cfg); evaluated != nil &&
					evaluated.Type() == mod.ModValueMultiTypeFloat {
					value = evaluated.Float()
				}

				value = applyGlobalLimits(globalLimits, tagGlobalLimits(mo), value)

				modResult = modResult * (1 + value/100)

				if p, ok := moreHighPrecisionFor(mo.Name()); ok {
					if havePrecision {
						modPrecision = max(modPrecision, p)
					} else {
						modPrecision, havePrecision = p, true
					}
				}
			}
		}
		if havePrecision {
			power := math.Pow(10, float64(modPrecision))
			result = math.Floor(result*modResult*power) / power
		} else {
			result = result * luaRound(modResult, 2)
		}
	}

	if m.Parent != nil {
		result *= m.Parent.MoreInternal(context, cfg, names...)
	}

	return result
}

//pob:src/Classes/ModStore.lua:183-192
func (m *ModDB) Override(cfg *ListCfg, names ...string) *mod.ModValueMulti {
	return m.OverrideInternal(m, cfg, names...)
}

//pob:src/Classes/ModDB.lua:219-241
func (m *ModDB) OverrideInternal(context ModStoreFuncs, cfg *ListCfg, names ...string) *mod.ModValueMulti {
	for _, name := range names {
		for _, mo := range m.Mods[name] {
			if mo.Type() == mod.TypeOverride &&
				cfgFlags(cfg)&mo.Flags() == mo.Flags() &&
				mod.MatchKeywordFlags(cfgKeywordFlags(cfg), mo.KeywordFlags()) &&
				(cfg == nil || cfg.Source == nil || sourcePrefixMatch(mo.GetSource(), *cfg.Source)) {

				value := context.evalMod(mo, cfg)
				if value != nil {
					return value
				}
			}
		}
	}

	if m.Parent != nil {
		p := m.Parent.OverrideInternal(context, cfg, names...)
		if p != nil {
			return p
		}
	}

	return nil
}

//pob:src/Classes/ModDB.lua:108-117
func (m *ModDB) AddList(list *ModList) {
	for _, newMod := range list.mods {
		m.AddMod(newMod)
	}
}

//pob:block:src/Classes/ModStore.lua:57-71
func (m *ModDB) ScaleAddList(list *ModList, scale float64) {
	if list == nil {
		return
	}

	if scale == 1 {
		m.AddList(list)
		return
	}

	for _, newMod := range list.mods {
		if modIsUnscalable(newMod) {
			m.AddMod(newMod)
			continue
		}
		m.AddMod(scaleModClone(newMod, scale))
	}
}

type ModResult struct {
	Value float64
	Mod   mod.Mod
}

//pob:src/Classes/ModDB.lua:231-263
func (m *ModDB) TabulateInternal(context ModStoreFuncs, result *[]ModResult, modType mod.Type,
	cfg *ListCfg, flags mod.MFlag, keywordFlags mod.KeywordFlag, source mod.Source, modNames ...string) {
	globalLimits := make(map[string]float64)

	for _, modName := range modNames {
		modList, exists := m.Mods[modName]
		if !exists {
			continue
		}

		for _, mMod := range modList {
			if (modType == "" || mMod.Type() == modType) &&
				(flags&mMod.Flags()) == mMod.Flags() &&
				mod.MatchKeywordFlags(keywordFlags, mMod.KeywordFlags()) &&
				(source == "" || sourcePrefixMatch(mMod.GetSource(), source)) {

				var value float64

				nonNumeric := false

				gated := false

				if len(mMod.Tags()) > 0 {
					evaluated := context.evalMod(mMod, cfg)
					switch {
					case evaluated == nil:
						gated = true
					case evaluated.Type() == mod.ModValueMultiTypeFloat:
						value = evaluated.Float()
					default:
						nonNumeric = true
					}

					value = applyGlobalLimits(globalLimits, tagGlobalLimits(mMod), value)
				} else if v := mMod.Value(); v != nil {
					if v.Type() == mod.ModValueMultiTypeFloat {
						value = v.Float()
					} else {
						nonNumeric = true
					}
				}

				if !gated && (nonNumeric || value != 0 || mMod.Type() == mod.TypeOverride) {
					*result = append(*result, ModResult{
						Value: value,
						Mod:   mMod,
					})
				}
			}
		}
	}

	if m.Parent != nil {
		m.Parent.TabulateInternal(context, result, modType, cfg, flags, keywordFlags, source, modNames...)
	}
}

//pob:src/Classes/ModStore.lua:207-218
func (m *ModDB) Tabulate(modType mod.Type, cfg *ListCfg, modNames ...string) []ModResult {
	var flags *mod.MFlag
	var keywordFlags *mod.KeywordFlag
	var source *mod.Source

	if cfg != nil {
		flags = cfg.Flags
		keywordFlags = cfg.KeywordFlags
		source = cfg.Source
	}

	if flags == nil {
		flags = (*mod.MFlag)(utils.Ptr(0))
	}

	if keywordFlags == nil {
		keywordFlags = (*mod.KeywordFlag)(utils.Ptr(0))
	}

	if source == nil {
		source = (*mod.Source)(utils.Ptr(""))
	}

	results := make([]ModResult, 0)
	m.TabulateInternal(m, &results, modType, cfg, *flags, *keywordFlags, *source, modNames...)
	return results
}

//pob:src/Classes/ModStore.lua:220-229
func (m *ModDB) Max(cfg *ListCfg, modNames ...string) float64 {
	globalMax := float64(0)
	for _, value := range m.Tabulate(mod.TypeMAX, cfg, modNames...) {
		multi := m.evalMod(value.Mod, cfg)
		if multi == nil {
			continue
		}
		if val := multi.Float(); val > globalMax {
			globalMax = val
		}
	}
	return globalMax
}

//pob:src/Classes/ModStore.lua:134-148
func (m *ModDB) Combine(modType mod.Type, cfg *ListCfg, modNames ...string) *mod.ModValueMulti {
	switch modType {
	case mod.TypeMore:
		return mod.NewModValueFloat(m.More(cfg, modNames...))
	case mod.TypeFlag:
		return mod.NewModValueFlag(m.Flag(cfg, modNames...))
	case mod.TypeOverride:
		return m.Override(cfg, modNames...)
	case mod.TypeList:
		return mod.NewModValueList(m.List(cfg, modNames...))
	case mod.TypeMAX:
		return mod.NewModValueFloat(m.Max(cfg, modNames...))
	default:
		return mod.NewModValueFloat(m.Sum(modType, cfg, modNames...))
	}
}
