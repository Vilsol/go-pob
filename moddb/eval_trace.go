package moddb

import "github.com/Vilsol/go-pob/mod"

type TagStep struct {
	Tag    mod.Tag
	Before *mod.ModValueMulti
	After  *mod.ModValueMulti
}

type EvalTrace struct {
	Initial *mod.ModValueMulti
	Final   *mod.ModValueMulti
	Steps   []TagStep
}

func (t EvalTrace) Excluded() bool {
	return t.Final == nil
}

func TraceEvalMod(context ModStoreFuncs, m mod.Mod, cfg *ListCfg) EvalTrace {
	out := EvalTrace{Initial: m.Value()}

	s := storeOf(context)
	if s == nil {
		out.Final = out.Initial
		return out
	}

	value := out.Initial
	for _, raw := range m.Tags() {
		next := s.evalTag(raw, value, cfg)
		out.Steps = append(out.Steps, TagStep{Tag: raw, Before: value, After: next})
		value = next
		if value == nil {
			return out
		}
	}

	out.Final = value
	return out
}

func storeOf(f ModStoreFuncs) *ModStore {
	switch v := f.(type) {
	case *ModDB:
		return v.ModStore
	case *ModList:
		return v.ModStore
	}
	return nil
}

func GetStat(context ModStoreFuncs, stat string, cfg *ListCfg) float64 {
	s := storeOf(context)
	if s == nil {
		return 0
	}
	return s.getStat(stat, cfg)
}

func RelatedModDB(context ModStoreFuncs, name string) ModStoreFuncs {
	s := storeOf(context)
	if s == nil {
		return nil
	}
	return s.relatedModDB(name)
}

func CfgAdmits(m mod.Mod, cfg *ListCfg) (bool, bool, bool) {
	return cfgFlags(cfg)&m.Flags() == m.Flags(),
		mod.MatchKeywordFlags(cfgKeywordFlags(cfg), m.KeywordFlags()),
		cfg == nil || cfg.Source == nil || sourcePrefixMatch(m.GetSource(), *cfg.Source)
}
