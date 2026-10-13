package calclib

import (
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
	"github.com/Vilsol/go-pob/utils"
)

// Calculate and combine INC/MORE modifiers for the given modifier names
//
//pob:src/Modules/CalcTools.lua:15-18
func Mod(modStore moddb.ModStoreFuncs, cfg *moddb.ListCfg, names ...string) float64 {
	return (1 + (moddb.SumOf(modStore, mod.TypeIncrease, cfg, names...))/100) * moddb.MoreOf(modStore, cfg, names...)
}

//pob:src/Modules/CalcTools.lua:20-29
func Mods(modStore moddb.ModStoreFuncs, cfg *moddb.ListCfg, names ...string) (float64, float64) {
	return 1 + moddb.SumOf(modStore, mod.TypeIncrease, cfg, names...)/100, moddb.MoreOf(modStore, cfg, names...)
}

// Calculate value
//
//pob:src/Modules/CalcTools.lua:31-39
func Val(modStore moddb.ModStoreFuncs, name string, cfgs ...*moddb.ListCfg) float64 {
	var cfg *moddb.ListCfg
	if len(cfgs) > 0 {
		cfg = cfgs[0]
	}

	baseVal := moddb.SumOf(modStore, mod.TypeBase, cfg, name)
	if baseVal != 0 {
		return baseVal * Mod(modStore, cfg, name)
	}
	return 0
}

// Correct the tags on conversion with multipliers so they carry over correctly
//
//pob:src/Modules/CalcTools.lua:244-264
func GetConvertedModTags(m mod.Mod, multiplier float64, minionMods bool) []mod.Tag {
	modifiers := make([]mod.Tag, len(m.Tags()))

	for k, value := range m.Tags() {
		if actorCondition, ok := value.(*mod.ActorConditionTag); ok && minionMods && actorCondition.Actor != nil && *actorCondition.Actor == "parent" {
			modifiers[k] = mod.Condition(actorCondition.VariableList...)
			continue
		}

		// LimitTotal can apply to 'per stat' or 'multiplier', so just copy the whole and update the limit.
		switch x := value.(type) {
		case *mod.MultiplierTag:
			c := *x
			if c.TagLimitTotal && c.TagLimit != nil {
				c.TagLimit = utils.Ptr(*c.TagLimit * multiplier)
			}
			modifiers[k] = &c
		case *mod.PerStatTag:
			c := *x
			if c.TagLimitTotal && c.TagLimit != nil {
				c.TagLimit = utils.Ptr(*c.TagLimit * multiplier)
			}
			modifiers[k] = &c
		default:
			modifiers[k] = value
		}
	}

	return modifiers
}
