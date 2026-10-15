package moddb

import (
	"math"
	"reflect"
	"slices"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/utils"
)

var _ ModStoreFuncs = (*ModList)(nil)

type ModList struct {
	*ModStore

	mods []mod.Mod

	names listNameFilter
}

func (m *ModList) modsForAny(names []string) []mod.Mod {
	for _, name := range names {
		if m.names.mayHave(name) {
			return m.mods
		}
	}
	return nil
}

func (m *ModList) modsFor(name string) []mod.Mod {
	if !m.names.mayHave(name) {
		return nil
	}
	return m.mods
}

//pob:src/Classes/ModList.lua:20-25
func NewModList() *ModList {
	m := &ModList{
		ModStore: &ModStore{},
		mods:     make([]mod.Mod, 0),
	}
	m.ModStore.Child = m
	return m
}

func (m *ModList) Clone() ModStoreFuncs {
	if m == nil {
		return nil
	}

	out := NewModList()
	out.AddDB(m)
	out.ModStore = m.ModStore.Clone()
	out.ModStore.Child = out
	return out
}

//pob:src/Classes/ModList.lua:27-29
func (m *ModList) AddMod(newMod mod.Mod) {
	m.names.add(newMod.Name())
	m.mods = append(m.mods, newMod)
}

//pob:src/Classes/ModStore.lua:125-130, src/Classes/ModList.lua:36-50
func (m *ModList) ReplaceMod(newMod mod.Mod) {
	if !m.replaceModInternal(newMod) {
		m.AddMod(newMod)
	}
}

//pob:src/Classes/ModList.lua:36-50
func (m *ModList) replaceModInternal(newMod mod.Mod) bool {
	for i, mo := range m.mods {
		if mo.Name() == newMod.Name() &&
			mo.Type() == newMod.Type() &&
			mo.Flags() == newMod.Flags() &&
			mo.KeywordFlags() == newMod.KeywordFlags() &&
			mo.GetSource() == newMod.GetSource() {
			m.mods[i] = newMod
			return true
		}
	}

	switch parent := m.Parent.(type) {
	case *ModList:
		return parent.replaceModInternal(newMod)
	case *ModDB:
		return parent.replaceModInternal(newMod)
	}

	return false
}

//pob:src/Classes/ModStore.lua:138-143
func (m *ModList) ConvertMod(oldName string, newMod mod.Mod) {
	if !m.convertModInternal(oldName, newMod) {
		m.AddMod(newMod)
	}
}

//pob:src/Classes/ModList.lua:58-71
func (m *ModList) convertModInternal(oldName string, newMod mod.Mod) bool {
	for i, cur := range m.mods {
		if cur.Name() == oldName &&
			cur.Type() == newMod.Type() &&
			cur.Flags() == newMod.Flags() &&
			cur.KeywordFlags() == newMod.KeywordFlags() &&
			cur.GetSource() == newMod.GetSource() {
			m.names.add(newMod.Name())
			m.mods[i] = newMod
			return true
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

func (m *ModList) Mods() []mod.Mod {
	return m.mods
}

func (m *ModList) SetMods(mods []mod.Mod) {
	m.mods = mods
	m.names = listNameFilter{}
	for _, mo := range mods {
		m.names.add(mo.Name())
	}
}

//pob:src/Classes/ModList.lua:88-94
func (m *ModList) AddDB(db *ModList) {
	if db == nil {
		return
	}

	for i, b := range db.names {
		m.names[i] |= b
	}
	m.mods = append(m.mods, db.mods...)
}

//pob:src/Classes/ModList.lua:88-94
func (m *ModList) AddList(other *ModList) {
	m.AddDB(other)
}

//pob:src/Classes/ModStore.lua:109-111
func (m *ModList) NewMod(name string, modType mod.Type, value float64, source mod.Source, tags ...mod.Tag) {
	newMod := mod.NewFloat(name, modType, value).Source(source)
	if len(tags) > 0 {
		newMod.Tag(tags...)
	}
	m.AddMod(newMod)
}

//pob:src/Modules/Common.lua:709-715
func modScaleRound(v float64) float64 {
	return math.Floor(float64(v*100)+0.5) / 100
}

//pob:block:src/Classes/ModStore.lua:68-83
func (m *ModList) ScaleAddList(src *ModList, scale float64) {
	if src == nil {
		return
	}

	if scale == 1 {
		m.AddList(src)
		return
	}

	for _, sm := range src.mods {
		if modIsUnscalable(sm) {
			m.AddMod(sm)
			continue
		}
		m.AddMod(scaleModClone(sm, scale))
	}
}

//pob:block:src/Classes/ModStore.lua:57-63
func modIsUnscalable(m mod.Mod) bool {
	for _, raw := range m.Tags() {
		if tag, ok := raw.(*mod.GlobalEffectTag); ok && tag.UnscalableTag {
			return true
		}
	}
	return false
}

//pob:block:src/Classes/ModStore.lua:59-72
func scaleModClone(sm mod.Mod, scale float64) mod.Mod {
	scaled := sm.Clone()
	value := scaled.Value()
	if value == nil {
		return scaled
	}
	switch value.Type() {
	case mod.ModValueMultiTypeFloat:
		value.SetFloat(scaleValuePrec(scaled.Name(), scaled.Type(), value.Float(), scale))
	case mod.ModValueMultiTypeList:
		if inner, ok := value.ValueList.(mod.ScalableNestedMod); ok {
			scaled := inner.ScaleNestedMod(func(m mod.Mod, v float64) float64 { return scaleValuePrec(m.Name(), m.Type(), v, scale) })

			if reflect.TypeOf(value.ValueList).Kind() == reflect.Pointer {
				scaled = scaled.Clone(true)
			}

			value.ValueList = scaled
		}
	}
	return scaled
}

//pob:src/Classes/ModStore.lua:218-229
func (m *ModList) List(cfg *ListCfg, names ...string) []interface{} {
	result := make([]interface{}, 0)
	m.ListInternal(m, &result, cfg, names...)
	return result
}

//pob:src/Classes/ModList.lua:199-220
func (m *ModList) ListInternal(context ModStoreFuncs, result *[]interface{}, cfg *ListCfg, names ...string) {
	for _, name := range names {
		for _, mo := range m.modsFor(name) {
			if modName(mo) != name {
				continue
			}
			if modMatches(mo, mod.TypeList, cfg) {

				value := context.evalMod(mo, cfg)
				if value != nil {
					*result = append(*result, value.ValueList)
				}
			}
		}
	}

	if m.Parent != nil {
		listIn(m.Parent, context, result, cfg, names...)
	}
}

//pob:src/Classes/ModStore.lua:165-174
func (m *ModList) Sum(modType mod.Type, cfg *ListCfg, names ...string) float64 {
	return m.SumInternal(m, modType, cfg, names...)
}

//pob:src/Classes/ModList.lua:101-120
func (m *ModList) SumInternal(context ModStoreFuncs, modType mod.Type, cfg *ListCfg, names ...string) float64 {
	result := float64(0)

	for _, mo := range m.modsForAny(names) {
		if !slices.Contains(names, modName(mo)) {
			continue
		}

		if modMatches(mo, modType, cfg) {

			value := context.evalMod(mo, cfg)
			if value != nil && value.Type() == mod.ModValueMultiTypeFloat {
				result += value.Float()
			}
		}
	}

	if m.Parent != nil {
		result += sumIn(m.Parent, context, modType, cfg, names...)
	}

	return result
}

//pob:src/Classes/ModStore.lua:179-188
func (m *ModList) More(cfg *ListCfg, names ...string) float64 {
	return m.MoreInternal(m, cfg, names...)
}

//pob:src/Classes/ModList.lua:122-154
func (m *ModList) MoreInternal(context ModStoreFuncs, cfg *ListCfg, names ...string) float64 {
	result := float64(1)
	modPrecision := 0
	havePrecision := false

	for _, name := range names {
		modResult := float64(1)
		for _, mo := range m.modsFor(name) {
			if modName(mo) != name {
				continue
			}

			if modMatches(mo, mod.TypeMore, cfg) {

				value := context.evalMod(mo, cfg)
				if value != nil && value.Type() == mod.ModValueMultiTypeFloat {
					modResult = modResult * (1 + value.Float()/100)
				}
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
			power := pow10Of(modPrecision)
			result = math.Floor(result*modResult*power) / power
		} else {
			result = result * luaRound(modResult, 2)
		}
	}

	if m.Parent != nil {
		result *= moreIn(m.Parent, context, cfg, names...)
	}

	return result
}

//pob:src/Classes/ModStore.lua:190-199
func (m *ModList) Flag(cfg *ListCfg, names ...string) bool {
	return m.FlagInternal(m, cfg, names...)
}

//pob:src/Classes/ModList.lua:156-175
func (m *ModList) FlagInternal(context ModStoreFuncs, cfg *ListCfg, names ...string) bool {
	for _, mo := range m.modsForAny(names) {
		if !slices.Contains(names, modName(mo)) {
			continue
		}

		if modMatches(mo, mod.TypeFlag, cfg) {

			value := context.evalMod(mo, cfg)
			if value != nil && value.Type() == mod.ModValueMultiTypeFlag && value.Flag() {
				return true
			}
		}
	}

	if m.Parent != nil {
		if flagIn(m.Parent, context, cfg, names...) {
			return true
		}
	}

	return false
}

//pob:src/Classes/ModStore.lua:204-213
func (m *ModList) Override(cfg *ListCfg, names ...string) *mod.ModValueMulti {
	return m.OverrideInternal(m, cfg, names...)
}

//pob:src/Classes/ModList.lua:177-197
func (m *ModList) OverrideInternal(context ModStoreFuncs, cfg *ListCfg, names ...string) *mod.ModValueMulti {
	for _, name := range names {
		for _, mo := range m.modsFor(name) {
			if modName(mo) != name {
				continue
			}

			if modMatches(mo, mod.TypeOverride, cfg) {

				value := context.evalMod(mo, cfg)
				if value != nil {
					return value
				}
			}
		}
	}

	if m.Parent != nil {
		p := overrideIn(m.Parent, context, cfg, names...)
		if p != nil {
			return p
		}
	}

	return nil
}

//pob:src/Classes/ModList.lua:222-243
func (m *ModList) TabulateInternal(context ModStoreFuncs, result *[]ModResult, modType mod.Type,
	cfg *ListCfg, flags mod.MFlag, keywordFlags mod.KeywordFlag, source mod.Source, modNames ...string) {

	for _, mMod := range m.modsForAny(modNames) {
		if !slices.Contains(modNames, modName(mMod)) {
			continue
		}

		if (modType == "" || mMod.Type() == modType) &&
			(flags&mMod.Flags()) == mMod.Flags() &&
			mod.MatchKeywordFlags(keywordFlags, mMod.KeywordFlags()) &&
			(source == "" || sourcePrefixMatch(mMod.GetSource(), source)) {

			var value float64

			gated := false
			nonNumeric := false
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

	if m.Parent != nil {
		tabulateIn(m.Parent, context, result, modType, cfg, flags, keywordFlags, source, modNames...)
	}
}

//pob:src/Classes/ModStore.lua:235-246
func (m *ModList) Tabulate(modType mod.Type, cfg *ListCfg, modNames ...string) []ModResult {
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

//pob:src/Classes/ModStore.lua:248-257
func (m *ModList) Max(cfg *ListCfg, modNames ...string) float64 {
	globalMax := float64(0)
	for _, value := range m.Tabulate(mod.TypeMAX, cfg, modNames...) {
		multi := m.evalMod(value.Mod, cfg)
		if multi != nil {
			val := multi.Float()
			if val > globalMax {
				globalMax = val
			}
		}
	}
	return globalMax
}

//pob:src/Classes/ModStore.lua:259-268
func (m *ModList) Min(cfg *ListCfg, modNames ...string) (float64, bool) {
	var globalMin float64
	found := false
	for _, value := range m.Tabulate(mod.TypeMIN, cfg, modNames...) {
		multi := m.evalMod(value.Mod, cfg)
		if multi == nil {
			continue
		}
		if val := multi.Float(); !found || val < globalMin {
			globalMin, found = val, true
		}
	}
	return globalMin, found
}

//pob:src/Classes/ModStore.lua:145-159
func (m *ModList) Combine(modType mod.Type, cfg *ListCfg, modNames ...string) *mod.ModValueMulti {
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

//pob:block:src/Modules/Data.lua:436-554
var highPrecisionMods = map[string]map[mod.Type]int{
	"CritChance": {mod.TypeBase: 2}, "SelfCritChance": {mod.TypeBase: 2},
	"LifeRegenPercent": {mod.TypeBase: 2}, "ManaRegenPercent": {mod.TypeBase: 2},
	"EnergyShieldRegenPercent": {mod.TypeBase: 2},
	"LifeRegen":                {mod.TypeBase: 1}, "ManaRegen": {mod.TypeBase: 1},
	"EnergyShieldRegen": {mod.TypeBase: 1}, "RageRegen": {mod.TypeBase: 1},
	"LifeDegenPercent": {mod.TypeBase: 2}, "LifeDegenPercentTincture": {mod.TypeBase: 2},
	"ManaDegenPercent": {mod.TypeBase: 2}, "ManaDegenPercentTincture": {mod.TypeBase: 2},
	"EnergyShieldDegenPercent": {mod.TypeBase: 2},
	"LifeDegen":                {mod.TypeBase: 1}, "ManaDegen": {mod.TypeBase: 1},
	"EnergyShieldDegen": {mod.TypeBase: 1},
	"DamageLifeLeech":   {mod.TypeBase: 2}, "PhysicalDamageLifeLeech": {mod.TypeBase: 2},
	"ElementalDamageLifeLeech": {mod.TypeBase: 2}, "FireDamageLifeLeech": {mod.TypeBase: 2},
	"ColdDamageLifeLeech": {mod.TypeBase: 2}, "LightningDamageLifeLeech": {mod.TypeBase: 2},
	"ChaosDamageLifeLeech": {mod.TypeBase: 2},
	"DamageManaLeech":      {mod.TypeBase: 2}, "PhysicalDamageManaLeech": {mod.TypeBase: 2},
	"ElementalDamageManaLeech": {mod.TypeBase: 2}, "FireDamageManaLeech": {mod.TypeBase: 2},
	"ColdDamageManaLeech": {mod.TypeBase: 2}, "LightningDamageManaLeech": {mod.TypeBase: 2},
	"ChaosDamageManaLeech":    {mod.TypeBase: 2},
	"DamageEnergyShieldLeech": {mod.TypeBase: 2}, "PhysicalDamageEnergyShieldLeech": {mod.TypeBase: 2},
	"ElementalDamageEnergyShieldLeech": {mod.TypeBase: 2}, "FireDamageEnergyShieldLeech": {mod.TypeBase: 2},
	"ColdDamageEnergyShieldLeech": {mod.TypeBase: 2}, "LightningDamageEnergyShieldLeech": {mod.TypeBase: 2},
	"ChaosDamageEnergyShieldLeech": {mod.TypeBase: 2},
	"SupportManaMultiplier":        {mod.TypeMore: 4},
}

const defaultHighPrecision = 1

func scaleValuePrec(name string, modType mod.Type, orig, scale float64) float64 {
	precision, ok := 0, false
	if byType, hit := highPrecisionMods[name]; hit {
		precision, ok = byType[modType]
		if precision == 0 {
			ok = false
		}
	}
	if !ok && math.Floor(orig) != orig {
		precision, ok = defaultHighPrecision, true
	}
	if ok {
		power := math.Pow(10, float64(precision))
		return math.Floor(orig*scale*power) / power
	}
	return math.Trunc(modScaleRound(orig * scale))
}
