package calculator

import (
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

			value := m.EvalMod(mo, cfg)
			if value != nil {
				result = append(result, value)
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

			value := m.EvalMod(mo, cfg)
			if value != nil {
				result += value.(float64)
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

			value := m.EvalMod(mo, cfg)
			if value != nil {
				result = result * (1 + value.(float64)/100)
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

		if mo.Type() == mod.TypeMore &&
			(cfg == nil || cfg.Flags == nil || (*cfg.Flags)&mo.Flags() == mo.Flags()) &&
			(cfg == nil || cfg.KeywordFlags == nil || mod.MatchKeywordFlags(*cfg.KeywordFlags, mo.KeywordFlags())) &&
			(cfg == nil || cfg.Source == nil || *cfg.Source == mo.GetSource()) {

			value := m.EvalMod(mo, cfg)
			if value != nil && value.(bool) {
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

func (m *ModList) Override(cfg *ListCfg, names ...string) interface{} {
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

			value := m.EvalMod(mo, cfg)
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
