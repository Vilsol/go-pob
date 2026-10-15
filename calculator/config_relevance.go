package calculator

import (
	"maps"
	"slices"
	"strings"
	"sync"

	raw2 "github.com/Vilsol/go-pob/data/raw"
)

type ConfigRelevance struct {
	Conds              []string
	MinionConds        []string
	EnemyConds         []string
	CondTrue           []string
	Mults              []string
	EnemyMults         []string
	Stats              []string
	EnemyStats         []string
	TagTypes           []string
	Flags              []string
	Mods               []string
	Skills             []string
	TransfiguredSkills []string
	SkillFlags         []string
	SkillData          []string
}

func keepIf(names []string, pass func(string) bool) []string {
	out := make([]string, 0)
	for _, name := range names {
		if pass(name) {
			out = append(out, name)
		}
	}
	return out
}

func usedIn[T any](m map[string]T) func(string) bool {
	return func(name string) bool {
		_, ok := m[name]
		return ok
	}
}

//pob:src/Classes/ConfigTab.lua:401-662
func (env *Environment) ConfigRelevance(query ConfigRelevance) ConfigRelevance {
	player := env.Player
	mainSkill := player.MainSkill
	return ConfigRelevance{
		Conds:       keepIf(query.Conds, usedIn(env.ConditionsUsed)),
		MinionConds: keepIf(query.MinionConds, usedIn(env.MinionConditionsUsed)),
		EnemyConds:  keepIf(query.EnemyConds, usedIn(env.EnemyConditionsUsed)),
		CondTrue: keepIf(query.CondTrue, func(name string) bool {
			return player.ModDB.Conditions[name]
		}),
		Mults:      keepIf(query.Mults, usedIn(env.MultipliersUsed)),
		EnemyMults: keepIf(query.EnemyMults, usedIn(env.EnemyMultipliersUsed)),
		Stats: keepIf(query.Stats, func(name string) bool {
			return usedIn(env.PerStatsUsed)(name) || usedIn(env.EnemyMultipliersUsed)(name)
		}),
		EnemyStats: keepIf(query.EnemyStats, usedIn(env.EnemyPerStatsUsed)),
		TagTypes:   keepIf(query.TagTypes, usedIn(env.TagTypesUsed)),
		Flags: keepIf(query.Flags, func(name string) bool {
			if mainSkill == nil {
				return false
			}
			return mainSkill.SkillFlags[SkillFlag(name)] || mainSkill.SkillModList.Flag(nil, name)
		}),
		Mods:   keepIf(query.Mods, usedIn(env.ModsUsed)),
		Skills: keepIf(query.Skills, func(name string) bool { return env.SkillsUsed[name] }),
		TransfiguredSkills: keepIf(query.TransfiguredSkills, func(name string) bool {
			if gameIDFromGemName(name, true) == "" {
				return false
			}
			for skill := range env.SkillsUsed {
				if isGemIDSame(skill, name, true) {
					return true
				}
			}
			return false
		}),
		SkillFlags: keepIf(query.SkillFlags, func(name string) bool {
			for _, activeSkill := range player.ActiveSkillList {
				if activeSkill.SkillFlags[SkillFlag(name)] {
					return true
				}
			}
			return false
		}),
		SkillData: keepIf(query.SkillData, func(name string) bool {
			for _, activeSkill := range player.ActiveSkillList {
				if skillDataSet(activeSkill.SkillData, name) {
					return true
				}
			}
			return false
		}),
	}
}

func skillDataSet(skillData *SkillData, name string) bool {
	if skillData == nil {
		return false
	}
	switch name {
	case "explodeCorpse":
		return skillData.ExplodeCorpse
	}
	return false
}

var gemRowForBaseName = sync.OnceValue(func() map[string]raw2.PoBGem {
	out := make(map[string]raw2.PoBGem, len(raw2.Gems)+len(raw2.GemVariants))
	add := func(row raw2.PoBGem) {
		name := row.Name
		if row.Tags["support"] && row.GrantedEffectID != "SupportBarrage" {
			name += " Support"
		}
		if _, ok := out[strings.ToLower(name)]; !ok {
			out[strings.ToLower(name)] = row
		}
	}
	for _, id := range slices.Sorted(maps.Keys(raw2.Gems)) {
		add(raw2.Gems[id])
	}
	for _, id := range slices.Sorted(maps.Keys(raw2.GemVariants)) {
		add(raw2.GemVariants[id])
	}
	return out
})

//pob:src/Modules/CalcTools.lua:279-292
func gameIDFromGemName(gemName string, dropVaal bool) string {
	row, ok := gemRowForBaseName()[strings.ToLower(gemName)]
	if !ok {
		return ""
	}
	if dropVaal && row.Tags["vaal"] && row.SecondaryGrantedEffectID != "" {
		return raw2.GemGameIDForGrantedEffectID[row.SecondaryGrantedEffectID]
	}
	return raw2.GemGameIDForGrantedEffectID[row.GrantedEffectID]
}

//pob:src/Modules/CalcTools.lua:299-303
func isGemIDSame(gemName string, typeName string, dropVaal bool) bool {
	gemNameID := gameIDFromGemName(gemName, dropVaal)
	typeID := gameIDFromGemName(typeName, dropVaal)
	return gemNameID != "" && typeID != "" && gemNameID == typeID
}
