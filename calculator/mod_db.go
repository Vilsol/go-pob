package calculator

import (
	"go-pob/calculator/mod"
	"go-pob/utils"
)

type ModDB struct {
	*ModStore

	Mods map[string][]mod.Mod
}

func NewModDB() *ModDB {
	return &ModDB{
		ModStore: NewModStore(nil),
		Mods:     make(map[string][]mod.Mod),
	}
}

func (m *ModDB) Clone() *ModDB {
	if m == nil {
		return nil
	}

	out := NewModDB()
	out.AddDB(m)
	out.ModStore = m.ModStore.Clone()
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

				value := m.EvalMod(mo, cfg)
				if value != nil {
					result = append(result, value)
				}
			}
		}
	}

	if m.Parent != nil {
		// TODO Parent
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

				value := m.EvalMod(mo, cfg)
				if value != nil {
					result += value.(float64)
				}
			}
		}
	}

	if m.Parent != nil {
		// TODO Parent
	}

	return result
}
