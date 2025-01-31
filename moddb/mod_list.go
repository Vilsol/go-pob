package moddb

import (
	"strings"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/utils"
)

var _ ModStoreFuncs = (*ModList)(nil)

type ModList struct {
	*ModStore

	mods []mod.Mod
}

func NewModList() *ModList {
	m := &ModList{
		ModStore: NewModStore(nil),
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

func (m *ModList) AddMod(newMod mod.Mod) {
	m.mods = append(m.mods, newMod)
}

func (m *ModList) AddDB(db *ModList) {
	if db == nil {
		return
	}

	newMods := utils.CopySlice(db.mods)
	m.mods = append(m.mods, newMods...)
}

func (m *ModList) List(cfg *ListCfg, names ...string) []interface{} {
	result := make([]interface{}, 0)

	mappedNames := make(map[string]bool, 0)
	for _, name := range names {
		mappedNames[name] = true
	}

	for _, mo := range m.mods {
		if _, ok := mappedNames[mo.Name()]; !ok {
			continue
		}
		if mo.Type() == mod.TypeList &&
			(cfg == nil || cfg.Flags == nil || (*cfg.Flags)&mo.Flags() == mo.Flags()) &&
			(cfg == nil || cfg.KeywordFlags == nil || mod.MatchKeywordFlags(*cfg.KeywordFlags, mo.KeywordFlags())) &&
			(cfg == nil || cfg.Source == nil || *cfg.Source == mo.GetSource()) {

			value := m.evalMod(mo, cfg)
			if value != nil {
				result = append(result, value.ValueList)
			}
		}
	}

	if m.Parent != nil {
		result = append(result, m.Parent.List(cfg, names...)...)
	}

	return result
}

func (m *ModList) Sum(modType mod.Type, cfg *ListCfg, names ...string) float64 {
	result := float64(0)

	mappedNames := make(map[string]bool, 0)
	for _, name := range names {
		mappedNames[name] = true
	}

	for _, mo := range m.mods {
		if _, ok := mappedNames[mo.Name()]; !ok {
			continue
		}

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

	if m.Parent != nil {
		result += m.Parent.Sum(modType, cfg, names...)
	}

	return result
}

func (m *ModList) More(cfg *ListCfg, names ...string) float64 {
	result := float64(1)

	mappedNames := make(map[string]bool, 0)
	for _, name := range names {
		mappedNames[name] = true
	}

	for _, mo := range m.mods {
		if _, ok := mappedNames[mo.Name()]; !ok {
			continue
		}

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

	if m.Parent != nil {
		result *= m.Parent.More(cfg, names...)
	}

	return result
}

func (m *ModList) Flag(cfg *ListCfg, names ...string) bool {
	mappedNames := make(map[string]bool, 0)
	for _, name := range names {
		mappedNames[name] = true
	}

	for _, mo := range m.mods {
		if _, ok := mappedNames[mo.Name()]; !ok {
			continue
		}

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

	if m.Parent != nil {
		if m.Parent.Flag(cfg, names...) {
			return true
		}
	}

	return false
}

func (m *ModList) Override(cfg *ListCfg, names ...string) *mod.ModValueMulti {
	mappedNames := make(map[string]bool, 0)
	for _, name := range names {
		mappedNames[name] = true
	}

	for _, mo := range m.mods {
		if _, ok := mappedNames[mo.Name()]; !ok {
			continue
		}

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

	if m.Parent != nil {
		p := m.Parent.Override(cfg, names...)
		if p != nil {
			return p
		}
	}

	return nil
}

func (m *ModList) TabulateInternal(context ModStoreFuncs, result *[]ModResult, modType mod.Type,
	cfg *ListCfg, flags mod.MFlag, keywordFlags mod.KeywordFlag, source mod.Source, modNames ...string) {

	// TODO Table
	//globalLimits := make(map[string]float64)

	for _, mMod := range m.mods {
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

	if m.Parent != nil {
		m.Parent.TabulateInternal(context, result, modType, cfg, flags, keywordFlags, source, modNames...)
	}
}

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
