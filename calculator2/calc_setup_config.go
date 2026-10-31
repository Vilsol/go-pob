package calculator2

import (
	"github.com/Vilsol/go-pob/moddb"
)

// configApply is the arguments every ConfigOptions apply function receives, plus the config set's
// placeholder table, which enemyIsBoss rewrites through SetPlaceholder.
type configApply struct {
	env          *Environment
	modList      *moddb.ModList
	enemyModList *moddb.ModList
	placeholder  map[string]float64
}

type configVar struct {
	name               string
	varType            string
	defaultState       *ConfigValue
	defaultPlaceholder *float64
	apply              func(c *configApply, val ConfigValue)
}

func configNumber(v float64) *ConfigValue { return &ConfigValue{Number: v, IsNumber: true} }
func configString(v string) *ConfigValue  { return &ConfigValue{String: v, IsString: true} }
func configBool(v bool) *ConfigValue      { return &ConfigValue{Boolean: v, IsBoolean: true} }
func configPlaceholder(v float64) *float64 {
	return &v
}

// configPlaceholders is the active config set's placeholder table after Load: each option's
// defaultPlaceholderState, then the saved Placeholder elements.
//
//pob2:block:src/Classes/ConfigTab.lua:1323-1331
//pob2:block:src/Classes/ConfigTab.lua:907-913
func configPlaceholders(env *Environment) map[string]float64 {
	placeholder := map[string]float64{}
	for _, v := range configVarList {
		if v.defaultPlaceholder != nil {
			placeholder[v.name] = *v.defaultPlaceholder
		}
	}
	placeholders := env.Build.Config.Placeholders
	for _, set := range env.Build.Config.ConfigSets {
		if set.ID == activeConfigSetID(env.Build) {
			placeholders = set.Placeholders
		}
	}
	for _, p := range placeholders {
		if p.Number != nil {
			placeholder[p.Name] = *p.Number
		}
	}
	return placeholder
}

// applyConfigVars is the varList loop of ConfigTab:BuildModList: an option's input (its saved value, else
// its defaultState or defaultIndex value) and, for the number types, its placeholder when the input is
// unset.
//
//pob2:block:src/Classes/ConfigTab.lua:1177-1199
func applyConfigVars(env *Environment, modList, enemyModList *moddb.ModList) {
	c := &configApply{env: env, modList: modList, enemyModList: enemyModList, placeholder: configPlaceholders(env)}
	for _, v := range configVarList {
		input, set := env.ConfigInput[v.name]
		if !set && v.defaultState != nil {
			input, set = *v.defaultState, true
		}
		switch v.varType {
		case "check":
			if set && (!input.IsBoolean || input.Boolean) {
				v.apply(c, ConfigValue{Boolean: true, IsBoolean: true})
			}
		case "count", "integer", "countAllowZero", "float":
			if set && input.IsNumber && (input.Number != 0 || v.varType != "count") {
				v.apply(c, input)
			} else if p, ok := c.placeholder[v.name]; ok && (p != 0 || v.varType != "count") {
				v.apply(c, ConfigValue{Number: p, IsNumber: true})
			}
		case "list", "text":
			if set {
				v.apply(c, input)
			}
		}
	}
}
