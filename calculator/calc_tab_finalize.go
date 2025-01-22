package calculator

import (
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
}

var existingColProps map[string]ColProps

func FinalizeCalcTab(env *Environment) map[string]float64 {
	props := make(map[string]float64)

	for k, prop := range existingColProps {
		actor := env.Player
		if prop.Enemy != nil && *prop.Enemy {
			actor = env.Enemy
		}

		if actor == nil {
			continue
		}

		var modCfg *moddb.ListCfg
		if actor.MainSkill != nil {
			switch prop.Cfg {
			case "skill":
				modCfg = actor.MainSkill.SkillCfg
			case "weapon1":
				modCfg = actor.MainSkill.Weapon1Cfg
			case "weapon2":
				modCfg = actor.MainSkill.Weapon2Cfg
			case "dot":
				// TODO dot
			case "dotPhysical":
				// TODO dotPhysical
			case "dotLightning":
				// TODO dotLightning
			case "dotCold":
				// TODO dotCold
			case "dotFire":
				// TODO dotFire
			case "dotChaos":
				// TODO dotChaos
			case "bleed":
				modCfg = actor.MainSkill.BleedCfg
			case "OHbleed":
				modCfg = actor.MainSkill.OHBleedCfg
			case "poison":
				// TODO poison
			case "OHpoison":
				// TODO OHpoison
			case "ignite":
				// TODO ignite
			case "OHignite":
				// TODO OHignite
			case "decay":
				// TODO decay
			}
		}

		if modCfg != nil && prop.ModSource != nil {
			modCfg.Source = utils.Ptr(mod.Source(*prop.ModSource))
		}

		var modStore moddb.ModStoreFuncs
		if prop.Enemy != nil && *prop.Enemy {
			modStore = env.Enemy.ModDB
		} else if prop.Cfg != "" && actor.MainSkill.SkillModList != nil {
			modStore = actor.MainSkill.SkillModList
		} else {
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

func SetColProps(colProps map[string]ColProps) {
	existingColProps = colProps
}
