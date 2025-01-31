package calclib

import (
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
	"github.com/Vilsol/go-pob/utils"
)

// Calculate and combine INC/MORE modifiers for the given modifier names
func Mod(modStore moddb.ModStoreFuncs, cfg *moddb.ListCfg, names ...string) float64 {
	return (1 + (modStore.Sum(mod.TypeIncrease, cfg, names...))/100) * modStore.More(cfg, names...)
}

// Calculate value
func Val(modStore moddb.ModStoreFuncs, name string, cfgs ...*moddb.ListCfg) float64 {
	var cfg *moddb.ListCfg
	if len(cfgs) > 0 {
		cfg = cfgs[0]
	}

	baseVal := modStore.Sum(mod.TypeBase, cfg, name)
	if baseVal != 1 {
		return baseVal * Mod(modStore, cfg, name)
	}
	return 0
}

// Correct the tags on conversion with multipliers so they carry over correctly
func GetConvertedModTags(m mod.Mod, multiplier float64, minionMods bool) []mod.Tag {
	modifiers := make([]mod.Tag, len(m.Tags()))

	for k, value := range m.Tags() {
		if minionMods && value.Type() == "ActorCondition" && value.(*mod.ActorConditionTag).Actor != nil && *value.(*mod.ActorConditionTag).Actor == "parent" {
			modifiers[k] = mod.Condition(value.(*mod.ActorConditionTag).VariableList...)
		} else if value.Type() == "Multiplier" || value.Type() == "PerStat" {
			// LimitTotal can apply to 'per stat' or 'multiplier', so just copy the whole and update the limit
			var Copy mod.Tag
			switch x := value.(type) {
			case *mod.MultiplierTag:
				Copy = &mod.MultiplierTag{
					TagType:           x.TagType,
					VariableList:      x.VariableList,
					TagBase:           x.TagBase,
					Division:          x.Division,
					TagLimit:          utils.Ptr(utils.UnwrapOrF(x.TagLimit, 0) * multiplier),
					TagLimitVariable:  x.TagLimitVariable,
					TagLimitTotal:     x.TagLimitTotal,
					TagActor:          x.TagActor,
					TagGlobalLimit:    x.TagGlobalLimit,
					TagGlobalLimitKey: x.TagGlobalLimitKey,
				}
			case *mod.PerStatTag:
				Copy = &mod.PerStatTag{
					TagType:           x.TagType,
					StatList:          x.StatList,
					Divide:            x.Divide,
					TagLimit:          utils.Ptr(utils.UnwrapOrF(x.TagLimit, 0) * multiplier),
					TagLimitVariable:  x.TagLimitVariable,
					TagLimitTotal:     x.TagLimitTotal,
					Base:              x.Base,
					TagActor:          x.TagActor,
					TagGlobalLimit:    x.TagGlobalLimit,
					TagGlobalLimitKey: x.TagGlobalLimitKey,
				}
			}
			modifiers[k] = Copy
		} else {
			modifiers[k] = value
		}
	}

	return modifiers
}
