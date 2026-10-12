package calculator

import (
	"regexp"
	"sort"
	"strings"

	"github.com/Vilsol/go-pob/mod"
)

var (
	fireOrIgnitePattern = regexp.MustCompile(`[fi][ig][rn][ei]t?e?`)
	coldOrFreezePattern = regexp.MustCompile(`[cf][or][le][de]z?e?`)
)

//pob:src/Modules/Calcs.lua:502-511
func addUsedVar(out map[string][]mod.Mod, varName string, m mod.Mod) {
	if m.GetSource() == "Base" {
		return
	}
	out[varName] = append(out[varName], m)
}

//pob:src/Modules/Calcs.lua:512-520
func addUsedTag(out map[string][]mod.Mod, varList []string, m mod.Mod) {
	for _, varName := range varList {
		addUsedVar(out, varName, m)
	}
}

//pob:src/Modules/Calcs.lua:505-527
func (env *Environment) addModTagsUsed(isPlayer bool, m mod.Mod) {
	addUsedVar(env.ModsUsed, m.Name(), m)

	lowerName := strings.ToLower(m.Name())
	if fireOrIgnitePattern.MatchString(lowerName) {
		addUsedTag(env.EnemyConditionsUsed, []string{"Ignited", "Burning"}, m)
	}
	if coldOrFreezePattern.MatchString(lowerName) {
		addUsedTag(env.EnemyConditionsUsed, []string{"Frozen"}, m)
	}

	for _, tag := range m.Tags() {
		addUsedVar(env.TagTypesUsed, string(tag.Type()), m)

		switch t := tag.(type) {
		case *mod.IgnoreCondTag:
			return
		case *mod.ConditionTag:
			if isPlayer {
				addUsedTag(env.ConditionsUsed, t.VarList, m)
			} else {
				addUsedTag(env.MinionConditionsUsed, t.VarList, m)
			}
		case *mod.ActorConditionTag:
			if t.Actor != nil && *t.Actor == "enemy" {
				addUsedTag(env.EnemyConditionsUsed, t.VariableList, m)
			} else {
				addUsedTag(env.ConditionsUsed, t.VariableList, m)
			}
		case *mod.MultiplierTag:
			if t.TagActor == "" {
				if isPlayer {
					addUsedTag(env.MultipliersUsed, t.VariableList, m)
				}
			} else if t.TagActor == "enemy" {
				addUsedTag(env.EnemyMultipliersUsed, t.VariableList, m)
			}
		case *mod.MultiplierThresholdTag:
			if t.TagActor == "" {
				if isPlayer {
					addUsedTag(env.MultipliersUsed, thresholdVars(t), m)
				}
			} else if t.TagActor == "enemy" {
				addUsedTag(env.EnemyMultipliersUsed, thresholdVars(t), m)
			}
		case *mod.PerStatTag:
			if t.TagActor == "" {
				if isPlayer {
					addUsedTag(env.PerStatsUsed, t.StatList, m)
				}
			} else if t.TagActor == "enemy" {
				addUsedTag(env.EnemyPerStatsUsed, t.StatList, m)
			}
		case *mod.StatThresholdTag:
			if isPlayer {
				addUsedTag(env.PerStatsUsed, []string{t.Stat}, m)
			}
		}
	}
}

func (env *Environment) initUsedTables() {
	env.SkillsUsed = make(map[string]bool)
	env.ConditionsUsed = make(map[string][]mod.Mod)
	env.MultipliersUsed = make(map[string][]mod.Mod)
	env.MinionConditionsUsed = make(map[string][]mod.Mod)
	env.EnemyConditionsUsed = make(map[string][]mod.Mod)
	env.EnemyMultipliersUsed = make(map[string][]mod.Mod)
	env.PerStatsUsed = make(map[string][]mod.Mod)
	env.EnemyPerStatsUsed = make(map[string][]mod.Mod)
	env.ModsUsed = make(map[string][]mod.Mod)
	env.TagTypesUsed = make(map[string][]mod.Mod)
}

//pob:block:src/Modules/Calcs.lua:458-561
func collectUsedTables(env *Environment) {
	env.initUsedTables()

	for _, activeSkill := range env.Player.ActiveSkillList {
		for _, skillEffect := range activeSkill.EffectList {
			if skillEffect.GrantedEffect != nil {
				env.SkillsUsed[skillEffect.GrantedEffect.Name()] = true
			}
		}
		if activeSkill.Minion != nil {
			for _, minionSkill := range activeSkill.Minion.ActiveSkillList {
				if minionSkill.ActiveEffect != nil && minionSkill.ActiveEffect.GrantedEffect != nil {
					env.SkillsUsed[pobGrantedEffectID(minionSkill.ActiveEffect.GrantedEffect.ID())] = true
				}
			}
		}
	}

	collectModDBTagsUsed(env, true, env.Player)
	if env.Minion != nil {
		collectModDBTagsUsed(env, false, env.Minion.Actor)
	}

	for _, activeSkill := range env.Player.ActiveSkillList {
		for _, m := range baseSkillMods(activeSkill) {
			env.addModTagsUsed(true, m)
		}
		if activeSkill.SkillModList != nil {
			for _, m := range activeSkill.SkillModList.Mods() {
				addUsedVar(env.ModsUsed, m.Name(), m)
				for _, tag := range m.Tags() {
					addUsedVar(env.TagTypesUsed, string(tag.Type()), m)
				}
			}
		}
		if activeSkill.Minion != nil {
			for _, minionSkill := range activeSkill.Minion.ActiveSkillList {
				for _, m := range baseSkillMods(minionSkill) {
					env.addModTagsUsed(false, m)
				}
			}
		}
	}

	collectEnemyTagsUsed(env)
}

//pob:block:src/Modules/Calcs.lua:579-583
func collectModDBTagsUsed(env *Environment, isPlayer bool, actor *Actor) {
	if actor == nil || actor.ModDB == nil {
		return
	}
	for _, modName := range sortedModNames(actor.ModDB.Mods) {
		for _, m := range actor.ModDB.Mods[modName] {
			env.addModTagsUsed(isPlayer, m)
		}
	}
}

//pob:block:src/Modules/Calcs.lua:547-561
func collectEnemyTagsUsed(env *Environment) {
	if env.EnemyModDB == nil {
		return
	}
	for _, modName := range sortedModNames(env.EnemyModDB.Mods) {
		for _, m := range env.EnemyModDB.Mods[modName] {
			for _, tag := range m.Tags() {
				stop := false
				switch t := tag.(type) {
				case *mod.IgnoreCondTag:
					stop = true
				case *mod.ConditionTag:
					addUsedTag(env.EnemyConditionsUsed, t.VarList, m)
				case *mod.ActorConditionTag:
					if t.Actor != nil && (*t.Actor == "enemy" || *t.Actor == "player") {
						addUsedTag(env.ConditionsUsed, t.VariableList, m)
					} else {
						addUsedTag(env.EnemyConditionsUsed, t.VariableList, m)
					}
				case *mod.MultiplierTag:
					if t.TagActor == "" {
						addUsedTag(env.EnemyMultipliersUsed, t.VariableList, m)
					}
				case *mod.MultiplierThresholdTag:
					if t.TagActor == "" {
						addUsedTag(env.EnemyMultipliersUsed, thresholdVars(t), m)
					}
				}
				if stop {
					break
				}
			}
		}
	}
}

func pobGrantedEffectID(id string) string {
	if alias, aliased := skillFlagIDAliases[id]; aliased {
		return alias
	}
	return id
}

//pob:block:src/Modules/Calcs.lua:472
func baseSkillMods(activeSkill *ActiveSkill) []mod.Mod {
	if activeSkill == nil || activeSkill.BaseSkillModList == nil {
		return nil
	}
	return activeSkill.BaseSkillModList.Mods()
}

func sortedModNames(mods map[string][]mod.Mod) []string {
	names := make([]string, 0, len(mods))
	for name := range mods {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func thresholdVars(t *mod.MultiplierThresholdTag) []string {
	if len(t.VariableList) > 0 {
		return t.VariableList
	}
	return []string{t.Variable}
}
