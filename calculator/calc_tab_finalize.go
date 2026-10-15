package calculator

import (
	"strings"
	"sync"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
	"github.com/Vilsol/go-pob/utils"
)

type ColProps struct {
	Cfg       string
	ModSource *string
	Enemy     *bool
	ModName   []string
	ModType   mod.Type
	Actor     string
}

var (
	colPropsMu       sync.RWMutex
	existingColProps map[string]ColProps
)

//pob:block:src/Modules/CalcFormat.lua:45-61
func FinalizeCalcTab(env *Environment) map[string]float64 {
	props := make(map[string]float64)

	colPropsMu.RLock()
	colProps := existingColProps
	colPropsMu.RUnlock()

	//pob:block:src/Classes/CalcSectionControl.lua:190
	shown := env.Player
	if env.Mode == OutputModeCalcs && env.Build != nil && env.Build.CalcsInputBool("showMinion") && env.Minion != nil && env.Minion.Actor != nil {
		shown = env.Minion.Actor
	}
	for k, prop := range colProps {
		actor := shown
		if actor == nil {
			continue
		}

		var modCfg *moddb.ListCfg
		if prop.Cfg != "" && actor.MainSkill != nil {
			modCfg = mainSkillCfg(actor.MainSkill, prop.Cfg)
		}
		if modCfg != nil {
			cfgCopy := *modCfg
			modCfg = &cfgCopy
		} else {
			modCfg = &moddb.ListCfg{}
		}
		if prop.ModSource != nil {
			modCfg.Source = utils.Ptr(mod.Source(*prop.ModSource))
		}
		if prop.Actor != "" {
			modCfg.Actor = prop.Actor
		}

		var modStore moddb.ModStoreFuncs
		switch {
		case prop.Enemy != nil && *prop.Enemy && actor.Enemy != nil && actor.Enemy.ModDB != nil:
			modStore = actor.Enemy.ModDB
		case prop.Cfg != "" && actor.MainSkill != nil && actor.MainSkill.SkillModList != nil:
			modStore = actor.MainSkill.SkillModList
		default:
			modStore = actor.ModDB
		}

		if modStore == nil {
			continue
		}

		data := modStore.Combine(prop.ModType, modCfg, prop.ModName...)
		if data != nil {
			props[k] = data.Float()
		}
	}

	return props
}

func mainSkillCfg(skill *ActiveSkill, name string) *moddb.ListCfg {
	switch name {
	case "skill":
		return skill.SkillCfg
	case "weapon1":
		return skill.Weapon1Cfg
	case "weapon2":
		return skill.Weapon2Cfg
	case "bleed":
		return skill.BleedCfg
	case "OHbleed":
		return skill.OHBleedCfg
	case "poison":
		return skill.PoisonCfg
	case "OHpoison":
		return skill.OHPoisonCfg
	case "ignite":
		return skill.IgniteCfg
	case "OHignite":
		return skill.OHIgniteCfg
	case "decay":
		return skill.DecayCfg
	case "dot":
		return skill.DotCfg
	}
	if damageType, ok := strings.CutPrefix(name, "dot"); ok {
		return skill.DotTypeCfg[data.DamageType(damageType)]
	}
	return nil
}

//pob:block:src/Modules/CalcSections.lua:117-122
func SetColProps(colProps map[string]ColProps) {
	colPropsMu.Lock()
	existingColProps = colProps
	colPropsMu.Unlock()
}
