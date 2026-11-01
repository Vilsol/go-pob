package exposition

import (
	"github.com/Vilsol/go-pob/calculator2"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
	"github.com/Vilsol/go-pob/utils"
)

// CalcsModEntry is one CalcSections.lua cell entry a {p:mod:n} format combines.
type CalcsModEntry struct {
	Cfg       string
	ModSource string
	Enemy     bool
	ModName   []string
	ModType   string
	Actor     string
}

// CalcsModQuery holds the entries by the frontend's cell entry id.
type CalcsModQuery struct {
	Entries map[string]CalcsModEntry
}

// mainSkillCfg is actor.mainSkill[cfg.."Cfg"]; calculator2 keeps the dot<type>Cfg by damage type.
func mainSkillCfg(skill *calculator2.ActiveSkill, name string) *moddb.ListCfg {
	switch name {
	case "skill":
		return skill.SkillCfg
	case "weapon1":
		return skill.Weapon1Cfg
	case "weapon2":
		return skill.Weapon2Cfg
	case "dot":
		return skill.DotCfg
	case "decay":
		return skill.DecayCfg
	case "dotPhysical", "dotLightning", "dotCold", "dotFire", "dotChaos":
		return skill.DotTypeCfg[name[len("dot"):]]
	}
	return skill.AilmentCfg[name+"Cfg"]
}

// CalcsMods is each entry's combined value over the last calculation's player, as formatCalcStr reads it.
// An entry whose value is not a number is absent.
//
//pob2:block:src/Modules/CalcFormat.lua:46-60
func (c *Calculator) CalcsMods(query CalcsModQuery) (map[string]float64, error) {
	if c.env == nil {
		return nil, notCalculatedError{}
	}
	actor := c.env.Player
	out := make(map[string]float64, len(query.Entries))
	for k, sectionData := range query.Entries {
		modCfg := &moddb.ListCfg{}
		if sectionData.Cfg != "" && actor.MainSkill != nil {
			if cfg := mainSkillCfg(actor.MainSkill, sectionData.Cfg); cfg != nil {
				// Lua writes source and actor onto the skill's own cfg; no entry with a cfg sets either
				copied := *cfg
				modCfg = &copied
			}
		}
		if sectionData.ModSource != "" {
			modCfg.Source = utils.Ptr(mod.Source(sectionData.ModSource))
		}
		if sectionData.Actor != "" {
			modCfg.Actor = sectionData.Actor
		}
		var modVal *mod.ModValueMulti
		switch {
		case sectionData.Enemy && actor.Enemy != nil && actor.Enemy.ModDB != nil:
			modVal = actor.Enemy.ModDB.Combine(mod.Type(sectionData.ModType), modCfg, sectionData.ModName...)
		case sectionData.Cfg != "" && actor.MainSkill != nil && actor.MainSkill.SkillModList != nil:
			modVal = actor.MainSkill.SkillModList.Combine(mod.Type(sectionData.ModType), modCfg, sectionData.ModName...)
		case actor.ModDB != nil:
			modVal = actor.ModDB.Combine(mod.Type(sectionData.ModType), modCfg, sectionData.ModName...)
		}
		if modVal != nil && modVal.Type() == mod.ModValueMultiTypeFloat {
			out[k] = modVal.Float()
		}
	}
	return out, nil
}

// CalcsOutputs is what the Calcs page reads from the last calculation's player beside Output: its true outputs and
// its output.MainHand and output.OffHand tables, nil when the main skill does not attack with that hand.
type CalcsOutputs struct {
	Flags    map[string]bool
	MainHand map[string]float64
	OffHand  map[string]float64
}

func (c *Calculator) CalcsOutputs() (*CalcsOutputs, error) {
	if c.env == nil {
		return nil, notCalculatedError{}
	}
	p := c.env.Player
	flags := make(map[string]bool, len(p.OutputFlags))
	for k, v := range p.OutputFlags {
		if v {
			flags[k] = true
		}
	}
	return &CalcsOutputs{Flags: flags, MainHand: p.OutputMainHand, OffHand: p.OutputOffHand}, nil
}
