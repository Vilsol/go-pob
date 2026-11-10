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

// CalcsMods is each entry's combined value over the Calcs tab's calculation's player, as formatCalcStr reads it.
// An entry whose value is not a number is absent.
//
//pob2:block:src/Modules/CalcFormat.lua:46-60
func (c *Calculator) CalcsMods(query CalcsModQuery) (map[string]float64, error) {
	env, err := c.calcs()
	if err != nil {
		return nil, err
	}
	actor := c.calcsActor(env)
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

// calcsActor is the actor the Calcs tab shows: the minion while Show Minion Stats is on and the calculation has one.
//
//pob2:block:src/Classes/CalcSectionControl.lua:249
//pob2:block:src/Classes/CalcBreakdownControl.lua:49-50
func (c *Calculator) calcsActor(env *calculator2.Environment) *calculator2.Actor {
	if calculator2.ReadCalcsInputs(c.build).ShowMinion && env.Minion != nil {
		return &env.Minion.Actor
	}
	return env.Player
}

// calcs is calcsTab.calcsEnv, the Calcs tab's own calculation after the last main one.
//
//pob2:block:src/Classes/CalcsTab.lua:512-513
func (c *Calculator) calcs() (*calculator2.Environment, error) {
	if c.env == nil {
		return nil, notCalculatedError{}
	}
	if c.calcsEnv == nil {
		c.calcsEnv = calculator2.BuildCalcsOutput(c.env, c.used)
	}
	return c.calcsEnv, nil
}

// CalcsOutputs is what the Calcs page reads from the actor the Calcs tab's calculation shows: its number, true and
// string outputs, its output.MainHand and output.OffHand tables (nil when the main skill does not attack with that
// hand), the main skill's flags and truthy skill data, the tab's inputs and skill select controls, and whether the
// player's main skill has a minion.
type CalcsOutputs struct {
	Output     map[string]float64
	Flags      map[string]bool
	Strings    map[string]string
	MainHand   map[string]float64
	OffHand    map[string]float64
	SkillFlags map[string]bool
	SkillData  map[string]bool
	Inputs     calculator2.CalcsInputs
	Controls   calculator2.SkillSelectControls
	HaveMinion bool
}

func (c *Calculator) CalcsOutputs() (*CalcsOutputs, error) {
	env, err := c.calcs()
	if err != nil {
		return nil, err
	}
	p := c.calcsActor(env)
	flags := make(map[string]bool, len(p.OutputFlags))
	for k, v := range p.OutputFlags {
		if v {
			flags[k] = true
		}
	}
	strs := make(map[string]string, len(p.OutputStrings))
	for k, v := range p.OutputStrings {
		strs[k] = v
	}
	return &CalcsOutputs{
		Output:     p.Output,
		Flags:      flags,
		Strings:    strs,
		MainHand:   p.OutputMainHand,
		OffHand:    p.OutputOffHand,
		SkillFlags: statSetFlags(p.MainSkill),
		SkillData:  truthySkillData(p.MainSkill),
		Inputs:     calculator2.ReadCalcsInputs(c.build),
		Controls:   calculator2.SkillSelect(c.build, env, env.MainSocketGroup, true),
		HaveMinion: statSetFlags(env.Player.MainSkill)["haveMinion"],
	}, nil
}

// CalcsInputs is calcsTab.input after the Calcs tab's calculation clamped its socket group.
func (c *Calculator) CalcsInputs() (calculator2.CalcsInputs, error) {
	if _, err := c.calcs(); err != nil {
		return calculator2.CalcsInputs{}, err
	}
	return calculator2.ReadCalcsInputs(c.build), nil
}

// SetCalcsSkillNumber is the tab's Socket Group dropdown.
//
//pob2:block:src/Classes/CalcsTab.lua:43-46
func (c *Calculator) SetCalcsSkillNumber(n int) {
	calculator2.SetCalcsInputNumber(c.build, "skill_number", float64(n))
}

// SetCalcsBuffMode is the tab's Calculation Mode dropdown.
//
//pob2:block:src/Classes/CalcsTab.lua:165-168
func (c *Calculator) SetCalcsBuffMode(mode string) {
	calculator2.SetCalcsInputString(c.build, "misc_buffMode", mode)
}

// SetCalcsShowMinion is the tab's Show Minion Stats check box.
//
//pob2:block:src/Classes/CalcsTab.lua:102-105
func (c *Calculator) SetCalcsShowMinion(show bool) {
	calculator2.SetCalcsInputBool(c.build, "showMinion", show)
}

// SetMainActiveSkillCalcs is the tab's Active Skill dropdown.
func (c *Calculator) SetMainActiveSkillCalcs(index int) error {
	return calculator2.SetMainActiveSkillCalcs(c.build, c.env, index)
}

// SetSkillStatSet, SetSkillStageCount and SetSkillMineCount are the Stat Set, Skill Stages and Active Mines controls
// of the sidebar (calcs false) or the Calcs tab; a nil count clears the box.
func (c *Calculator) SetSkillStatSet(calcs bool, index int) error {
	env, group, err := c.skillSelectEnv(calcs)
	if err != nil {
		return err
	}
	return calculator2.SetSkillStatSet(c.build, env, group, calcs, index)
}

func (c *Calculator) SetSkillStageCount(calcs bool, count *int) error {
	env, group, err := c.skillSelectEnv(calcs)
	if err != nil {
		return err
	}
	return calculator2.SetSkillStageCount(c.build, env, group, calcs, count)
}

func (c *Calculator) SetSkillMineCount(calcs bool, count *int) error {
	env, group, err := c.skillSelectEnv(calcs)
	if err != nil {
		return err
	}
	return calculator2.SetSkillMineCount(c.build, env, group, calcs, count)
}

// SetSkillMinion, SetSkillMinionSkill and SetSkillMinionSkillStatSet are the Minion, Minion Skill and Minion Skill Stat
// Set dropdowns of the sidebar (calcs false) or the Calcs tab.
func (c *Calculator) SetSkillMinion(calcs bool, index int) error {
	env, group, err := c.skillSelectEnv(calcs)
	if err != nil {
		return err
	}
	return calculator2.SetSkillMinion(c.build, env, group, calcs, index)
}

// SetCompareCalcsMinion is the Compare tab's Calcs header Minion dropdown.
func (c *Calculator) SetCompareCalcsMinion(index int) error {
	env, group, err := c.skillSelectEnv(true)
	if err != nil {
		return err
	}
	return calculator2.SetCompareCalcsMinion(c.build, env, group, index)
}

func (c *Calculator) SetSkillMinionSkill(calcs bool, index int) error {
	env, group, err := c.skillSelectEnv(calcs)
	if err != nil {
		return err
	}
	return calculator2.SetSkillMinionSkill(c.build, env, group, calcs, index)
}

func (c *Calculator) SetSkillMinionSkillStatSet(calcs bool, index int) error {
	env, group, err := c.skillSelectEnv(calcs)
	if err != nil {
		return err
	}
	return calculator2.SetSkillMinionSkillStatSet(c.build, env, group, calcs, index)
}

// skillSelectEnv is the calculation whose skill list the controls show, and the group they show.
func (c *Calculator) skillSelectEnv(calcs bool) (*calculator2.Environment, int, error) {
	env := c.env
	if calcs {
		var err error
		if env, err = c.calcs(); err != nil {
			return nil, 0, err
		}
	}
	if env == nil {
		return nil, 0, notCalculatedError{}
	}
	return env, env.MainSocketGroup, nil
}

// SyncCalcsSkillSelection is a compared build's load pointing its Calcs tab at its main skill.
func (c *Calculator) SyncCalcsSkillSelection() {
	calculator2.SyncCalcsSkillSelection(c.build)
}

// CompareSkillBreakdown is the Calcs view header's SkillBuffs or SkillDebuffs panel over the Calcs tab's calculation.
func (c *Calculator) CompareSkillBreakdown(name string) ([]calculator2.CompareSkillBreakdownRow, error) {
	env, err := c.calcs()
	if err != nil {
		return nil, err
	}
	return env.CompareSkillBreakdown(name), nil
}

// SetCalcsSectionCollapsed is a Calcs section's +/- toggle.
func (c *Calculator) SetCalcsSectionCollapsed(id, subsection string, collapsed bool) {
	calculator2.SetCalcsSectionCollapsed(c.build, id, subsection, collapsed)
}

// CalcsSectionCollapsed is a subsection's saved collapsed state, and whether the build saved one.
func (c *Calculator) CalcsSectionCollapsed(id, subsection string) (bool, bool) {
	return calculator2.CalcsSectionCollapsed(c.build, id, subsection)
}
