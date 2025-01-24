package calclib

import (
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
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
func GetConvertedModTags(m mod.Mod, multiplier float64) interface{} {
	modifiers := make(map[string]string)
	/*
		for k, value in ipairs(mod) do

		if minionMods and value.type == "ActorCondition" and value.actor == "parent" then
			modifiers[k] = { type = "Condition", var = value.var }
		elseif value.limitTotal then
			-- LimitTotal can apply to 'per stat' or 'multiplier', so just copy the whole and update the limit
			local copy = copyTable(value)
			copy.limit = copy.limit * multiplier
			modifiers[k] = copy
		else
			modifiers[k] = copyTable(value)
		end

		end
	*/
	return modifiers
}
