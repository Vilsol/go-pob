package moddb

import (
	"strings"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/utils"
)

var _ ModStoreFuncs = (*ModDB)(nil)

type ModDB struct {
	*ModStore

	Mods map[string][]mod.Mod
}

func NewModDB() *ModDB {
	m := &ModDB{
		ModStore: NewModStore(nil),
		Mods:     make(map[string][]mod.Mod),
	}
	m.ModStore.Child = m
	return m
}

func (m *ModDB) Clone() ModStoreFuncs {
	if m == nil {
		return nil
	}

	out := NewModDB()
	out.AddDB(m)
	out.ModStore = m.ModStore.Clone()
	out.ModStore.Child = out
	return out
}

func (m *ModDB) AddMod(newMod mod.Mod) {
	if _, ok := m.Mods[newMod.Name()]; !ok {
		m.Mods[newMod.Name()] = make([]mod.Mod, 0)
	}
	m.Mods[newMod.Name()] = append(m.Mods[newMod.Name()], newMod)
}

func (m *ModDB) AddDB(db *ModDB) {
	if db == nil {
		return
	}
	for k, v := range db.Mods {
		m.Mods[k] = utils.CopySlice(v)
	}
}

func (m *ModDB) List(cfg *ListCfg, names ...string) []interface{} {
	result := make([]interface{}, 0)

	for _, name := range names {
		for _, mo := range m.Mods[name] {
			if mo.Type() == mod.TypeList &&
				(cfg == nil || cfg.Flags == nil || (*cfg.Flags)&mo.Flags() == mo.Flags()) &&
				(cfg == nil || cfg.KeywordFlags == nil || mod.MatchKeywordFlags(*cfg.KeywordFlags, mo.KeywordFlags())) &&
				(cfg == nil || cfg.Source == nil || *cfg.Source == mo.GetSource()) {

				value := m.evalMod(mo, cfg)
				if value != nil {
					result = append(result, value)
				}
			}
		}
	}

	if m.Parent != nil {
		result = append(result, m.Parent.List(cfg, names...)...)
	}

	return result
}

func (m *ModDB) Sum(modType mod.Type, cfg *ListCfg, names ...string) float64 {
	result := float64(0)

	for _, name := range names {
		for _, mo := range m.Mods[name] {
			if mo.Type() == modType &&
				(cfg == nil || cfg.Flags == nil || (*cfg.Flags)&mo.Flags() == mo.Flags()) &&
				(cfg == nil || cfg.KeywordFlags == nil || mod.MatchKeywordFlags(*cfg.KeywordFlags, mo.KeywordFlags())) &&
				(cfg == nil || cfg.Source == nil || *cfg.Source == mo.GetSource()) {

				value := m.evalMod(mo, cfg)
				if value != nil && value.Type() == mod.ModValueMultiTypeFloat {
					result += value.Float()
				}
			}
		}
	}

	if m.Parent != nil {
		result += m.Parent.Sum(modType, cfg, names...)
	}

	return result
}

func (m *ModDB) Flag(cfg *ListCfg, names ...string) bool {
	for _, name := range names {
		for _, mo := range m.Mods[name] {
			if mo.Type() == mod.TypeFlag &&
				(cfg == nil || cfg.Flags == nil || (*cfg.Flags)&mo.Flags() == mo.Flags()) &&
				(cfg == nil || cfg.KeywordFlags == nil || mod.MatchKeywordFlags(*cfg.KeywordFlags, mo.KeywordFlags())) &&
				(cfg == nil || cfg.Source == nil || *cfg.Source == mo.GetSource()) {

				value := m.evalMod(mo, cfg)
				if value != nil && value.Type() == mod.ModValueMultiTypeFlag {
					return true
				}
			}
		}
	}

	if m.Parent != nil {
		if m.Parent.Flag(cfg, names...) {
			return true
		}
	}

	return false
}

func (m *ModDB) More(cfg *ListCfg, names ...string) float64 {
	result := float64(1)

	for _, name := range names {
		for _, mo := range m.Mods[name] {
			if mo.Type() == mod.TypeMore &&
				(cfg == nil || cfg.Flags == nil || (*cfg.Flags)&mo.Flags() == mo.Flags()) &&
				(cfg == nil || cfg.KeywordFlags == nil || mod.MatchKeywordFlags(*cfg.KeywordFlags, mo.KeywordFlags())) &&
				(cfg == nil || cfg.Source == nil || *cfg.Source == mo.GetSource()) {

				value := m.evalMod(mo, cfg)
				if value != nil && value.Type() == mod.ModValueMultiTypeFloat {
					result = result * (1 + value.Float()/100)
				}
			}
		}
	}

	if m.Parent != nil {
		result *= m.Parent.More(cfg, names...)
	}

	return result
}

func (m *ModDB) Override(cfg *ListCfg, names ...string) *mod.ModValueMulti {
	mappedNames := make(map[string]bool, 0)
	for _, name := range names {
		mappedNames[name] = true
	}

	for _, name := range names {
		for _, mo := range m.Mods[name] {
			if mo.Type() == mod.TypeOverride &&
				(cfg == nil || cfg.Flags == nil || (*cfg.Flags)&mo.Flags() == mo.Flags()) &&
				(cfg == nil || cfg.KeywordFlags == nil || mod.MatchKeywordFlags(*cfg.KeywordFlags, mo.KeywordFlags())) &&
				(cfg == nil || cfg.Source == nil || *cfg.Source == mo.GetSource()) {

				value := m.evalMod(mo, cfg)
				if value != nil {
					return value
				}
			}
		}
	}

	if m.Parent != nil {
		p := m.Parent.Override(cfg, names...)
		if p != nil {
			return p
		}
	}

	return nil
}

func (m *ModDB) AddList(list *ModList) {
	for _, newMod := range list.mods {
		m.AddMod(newMod)
	}
}

/*
function ModDBClass:TabulateInternal(context, result, modType, cfg, flags, keywordFlags, source, ...)
	local globalLimits = { }
	for i = 1, select('#', ...) do
		local modName = select(i, ...)
		local modList = self.mods[modName]
		if modList then
			for i = 1, #modList do
				local mod = modList[i]
				if (mod.type == modType or not modType) and band(flags, mod.flags) == mod.flags and MatchKeywordFlags(keywordFlags, mod.keywordFlags) and (not source or mod.source:match("[^:]+") == source) then
					local value
					if mod[1] then
						value = context:EvalMod(mod, cfg) or 0
						if mod[1].globalLimit and mod[1].globalLimitKey then
							globalLimits[mod[1].globalLimitKey] = globalLimits[mod[1].globalLimitKey] or 0
							if globalLimits[mod[1].globalLimitKey] + value > mod[1].globalLimit then
								value = mod[1].globalLimit - globalLimits[mod[1].globalLimitKey]
							end
							globalLimits[mod[1].globalLimitKey] = globalLimits[mod[1].globalLimitKey] + value
						end
					else
						value = mod.value
					end
					if value and (value ~= 0 or mod.type == "OVERRIDE") then
						t_insert(result, { value = value, mod = mod })
					end
				end
			end
		end
	end
	if self.parent then
		self.parent:TabulateInternal(context, result, modType, cfg, flags, keywordFlags, source, ...)
	end
end

*/

type ModResult struct {
	Value float64
	Mod   mod.Mod
}

func (m *ModDB) TabulateInternal(context ModStoreFuncs, result *[]ModResult, modType mod.Type,
	cfg *ListCfg, flags mod.MFlag, keywordFlags mod.KeywordFlag, source mod.Source, modNames ...string) {
	// TODO Table
	//globalLimits := make(map[string]float64)

	for _, modName := range modNames {
		modList, exists := m.Mods[modName]
		if !exists {
			continue
		}

		for _, mMod := range modList {
			if (modType == "" || mMod.Type() == modType) &&
				(flags&mMod.Flags()) == mMod.Flags() &&
				mod.MatchKeywordFlags(keywordFlags, mMod.KeywordFlags()) &&
				(source == "" || strings.Split(string(mMod.GetSource()), ":")[0] == string(source)) {

				var value float64

				if mMod.Value().Type() == mod.ModValueMultiTypeFloat {
					value = mMod.Value().Float()
				} else {
					/*
						TODO Table
						value = context.evalMod(mMod, cfg).Float()

						if mMod.Values[0].GlobalLimit > 0 && mMod.Values[0].GlobalLimitKey != "" {
							key := mMod.Values[0].GlobalLimitKey
							currentLimit := globalLimits[key]

							if currentLimit+value > mMod.Values[0].GlobalLimit {
								value = mMod.Values[0].GlobalLimit - currentLimit
							}
							globalLimits[key] += value
						}
					*/
				}

				if value != 0 || mMod.Type() == mod.TypeOverride {
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

func (m *ModDB) Max(cfg *ListCfg, modNames ...string) float64 {
	globalMax := float64(0)
	for _, value := range m.Tabulate(mod.TypeMAX, cfg, modNames...) {
		val := m.evalMod(value.Mod, cfg).Float()
		if val > globalMax {
			globalMax = val
		}
	}
	return globalMax
}

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
