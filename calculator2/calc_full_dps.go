package calculator2

import (
	"math"
	"slices"

	"github.com/Vilsol/go-pob/moddb"
)

// SkillDPS is one output.SkillDPS row: a skill's DPS and count, or a best ailment or full DoT total. "" is nil.
type SkillDPS struct {
	Name      string
	DPS       float64
	Count     float64
	Trigger   string
	SkillPart string
	Source    string
}

// fullDPSHarvest is captureFields' snapshot of one calc pass's output.
type fullDPSHarvest map[string]float64

//pob2:block:src/Modules/Calcs.lua:149-158
var harvestFields = []string{"TotalDPS", "BleedDPS", "CorruptingBloodDPS", "IgniteDPS", "BurningGroundDPS", "PoisonDPS", "CausticGroundDPS", "ImpaleDPS", "DecayDPS", "TotalDot", "CullMultiplier"}

func captureFields(output map[string]float64) fullDPSHarvest {
	captured := fullDPSHarvest{}
	for _, field := range harvestFields {
		if v, ok := output[field]; ok {
			captured[field] = v
		}
	}
	return captured
}

type mergeStat struct {
	key, target, mode string
	scaled            bool
}

//pob2:block:src/Modules/Calcs.lua:160-170
var mergeStatsSpec = []mergeStat{
	{key: "BleedDPS", target: "bleedDPS", mode: "max"},
	{key: "CorruptingBloodDPS", target: "corruptingBloodDPS", mode: "max"},
	{key: "IgniteDPS", target: "igniteDPS", mode: "max"},
	{key: "BurningGroundDPS", target: "burningGroundDPS", mode: "max"},
	{key: "PoisonDPS", target: "poisonDPS", mode: "max"},
	{key: "CausticGroundDPS", target: "causticGroundDPS", mode: "max"},
	{key: "ImpaleDPS", target: "impaleDPS", mode: "add", scaled: true},
	{key: "DecayDPS", target: "decayDPS", mode: "add"},
	{key: "CullMultiplier", target: "cullingMulti", mode: "cull"},
}

// fullDPSActor is one pass.actors entry; dotScale 0 is nil.
type fullDPSActor struct {
	out        fullDPSHarvest
	name       string
	count      float64
	trigger    string
	skillPart  string
	sourceName string
	dotScale   float64
}

// FullDPS is calcs.calcFullDPS's fullDPS table: combinedDPS, TotalDotDPS, skills and the merged totals by target.
type FullDPS struct {
	CombinedDPS float64
	TotalDotDPS float64
	Skills      []SkillDPS
	totals      map[string]float64
}

// Merge one captured calc pass into the Full DPS totals
//
//pob2:src/Modules/Calcs.lua:173-201
func mergeFullDPSPass(fullDPS *FullDPS, sources map[string]string, actors []fullDPSActor) {
	for _, actor := range actors {
		out := actor.out
		if out["TotalDPS"] > 0 {
			fullDPS.Skills = append(fullDPS.Skills, SkillDPS{Name: actor.name, DPS: out["TotalDPS"], Count: actor.count, Trigger: actor.trigger, SkillPart: actor.skillPart})
			fullDPS.CombinedDPS += float64(out["TotalDPS"] * actor.count)
		}
		for _, stat := range mergeStatsSpec {
			value, ok := out[stat.key]
			if !ok {
				continue
			}
			switch stat.mode {
			case "max":
				if value > fullDPS.totals[stat.target] {
					fullDPS.totals[stat.target] = value
					sources[stat.target] = actor.sourceName
				}
			case "add":
				if value > 0 {
					scale := 1.0
					if stat.scaled {
						scale = actor.count
					}
					fullDPS.totals[stat.target] += float64(value * scale)
				}
			case "cull":
				if value > 1 && value > fullDPS.totals[stat.target] {
					fullDPS.totals[stat.target] = value
				}
			}
		}
		if out["TotalDot"] > 0 && actor.dotScale != 0 {
			fullDPS.totals["dotDPS"] += float64(out["TotalDot"] * actor.dotScale)
		}
	}
}

//pob2:src/Modules/Calcs.lua:254-259
func getSocketGroupOverride(main *Environment, override calcOverride, socketGroup *SocketGroup) calcOverride {
	skillOverride := override
	skillOverride.WeaponSet = socketGroup.UsingSkillSet
	skillOverride.MainSocketGroup = slices.Index(main.SocketGroupList, socketGroup) + 1
	return skillOverride
}

//pob2:src/Modules/Calcs.lua:261-268
func findActiveSkillInEnv(env *Environment, sourceSkill *ActiveSkill) *ActiveSkill {
	for _, candidate := range env.Player.ActiveSkillList {
		if candidate.SocketGroup == sourceSkill.SocketGroup && candidate.ActiveEffect.SrcInstance == sourceSkill.ActiveEffect.SrcInstance &&
			candidate.ActiveEffect.GrantedEffect.ID == sourceSkill.ActiveEffect.GrantedEffect.ID {
			return candidate
		}
	}
	return nil
}

// usesFullDPS reports whether any socket group is included in Full DPS; calcFullDPS finds no skill otherwise.
func usesFullDPS(main *Environment) bool {
	return slices.ContainsFunc(main.SocketGroupList, func(g *SocketGroup) bool { return g.IncludeInFullDPS })
}

// initFullDPSEnv is calcs.initEnv(build, mode, override) over the build main calculated.
func initFullDPSEnv(main *Environment, mode string, override calcOverride) *Environment {
	env := newWeaponSetEnv(main, mode, override)
	initEnv(env)
	return env
}

// initFullDPSEnvSpec is calcs.initEnv(build, mode, override, specEnv) over the build main calculated; a specEnv
// that names an env rebuilds that env in place.
func initFullDPSEnvSpec(main *Environment, mode string, override calcOverride, spec *specEnv) (*Environment, *moddb.ModDB, *moddb.ModDB, *moddb.ModDB) {
	env := spec.env
	if env == nil {
		env = newWeaponSetEnv(main, mode, override)
	}
	cachedPlayerDB, cachedEnemyDB, cachedMinionDB := initEnvSpec(env, spec)
	return env, cachedPlayerDB, cachedEnemyDB, cachedMinionDB
}

// calcFullDPS runs every skill of a group included in Full DPS as the main skill and sums their DPS. main is the
// build: its socket group list, items and tree. The per-skill result cache (specEnv.fullDPSCache) and perform's
// skipEHP are not ported; every skill is calculated, and with its EHP. The mirage branch is dropped, as calcs.mirages
// never runs.
//
//pob2:src/Modules/Calcs.lua:270-486
//nolint:gocyclo // mirrors the Lua
func calcFullDPS(main *Environment, mode string, override calcOverride) *FullDPS {
	fullDPS := &FullDPS{Skills: []SkillDPS{}, totals: map[string]float64{}}
	// Go must differ: initEnv is skipped when no group can be included, which finds no skill either
	if !usesFullDPS(main) {
		return fullDPS
	}
	// Go must differ: the first env caches its own base databases rather than taking the caller's
	fullEnv, cachedPlayerDB, cachedEnemyDB, cachedMinionDB := initFullDPSEnvSpec(main, mode, override, &specEnv{})

	sources := map[string]string{}
	initialActiveSkillList := fullEnv.Player.ActiveSkillList

	for activeSkillIndex, sourceSkill := range initialActiveSkillList {
		activeSkill := sourceSkill
		if activeSkill.SocketGroup == nil || !activeSkill.SocketGroup.IncludeInFullDPS {
			continue
		}
		skillEnv := fullEnv
		groupSet := activeSkill.SocketGroup.UsingSkillSet
		if groupSet != fullEnv.WeaponSet {
			skillEnv = fullEnv.WeaponSetEnvs[groupSet]
			if skillEnv == nil {
				skillEnv = initFullDPSEnv(main, mode, getSocketGroupOverride(main, override, activeSkill.SocketGroup))
			}
		}
		if found := findActiveSkillInEnv(skillEnv, sourceSkill); found != nil {
			activeSkill = found
		}
		activeSkillCount, enabled := getActiveSkillCount(activeSkill)
		if !enabled {
			continue
		}
		skillEnv.Player.MainSkill = activeSkill
		perform(skillEnv)
		usedEnv := skillEnv
		skillName := getActiveSkillDisplayName(activeSkill)
		dotCanStack := activeSkill.ActiveEffect.StatSet != nil && activeSkill.ActiveEffect.StatSet.SkillFlags["DotCanStack"]
		var actors []fullDPSActor
		var minionOut fullDPSHarvest
		var infoMessage2 string
		// PoB-PoE2's env.minion is set whenever activeSkill.minion is
		if usedEnv.Minion != nil {
			minionOut = captureFields(usedEnv.Minion.Output)
			minionNamePrefix := ""
			switch {
			case activeSkill.Minion != nil && activeSkill.Minion.MinionData != nil:
				minionNamePrefix = activeSkill.Minion.MinionData.Name + ": "
			case usedEnv.Minion.MinionData != nil:
				minionNamePrefix = usedEnv.Minion.MinionData.Name + ": "
			}
			actors = append(actors, fullDPSActor{
				out:        minionOut,
				name:       skillName,
				count:      activeSkillCount,
				skillPart:  minionNamePrefix + activeSkill.SkillPartName,
				sourceName: skillName,
				dotScale:   1,
			})
			// This is a fix to prevent Absolution spell hit from being counted multiple times when increasing minions count
			if activeSkill.ActiveEffect.GrantedEffect.Name == "Absolution" && skillEnv.ModDB.Flag(nil, "Condition:AbsolutionSkillDamageCountedOnce") {
				activeSkillCount = 1
				infoMessage2 = "Skill Damage"
			}
		}

		playerOut := captureFields(usedEnv.Player.Output)

		minionContributed := minionOut["TotalDPS"] > 0
		skillPart := activeSkill.SkillPartName
		if minionContributed && infoMessage2 != "" {
			skillPart = infoMessage2
		}
		dotScale := 1.0
		if dotCanStack {
			dotScale = activeSkillCount
		}
		actors = append(actors, fullDPSActor{
			out:        playerOut,
			name:       skillName,
			count:      activeSkillCount,
			skillPart:  skillPart,
			sourceName: skillName,
			dotScale:   dotScale,
		})
		mergeFullDPSPass(fullDPS, sources, actors)

		var nextSkill *ActiveSkill
		for _, candidate := range initialActiveSkillList[activeSkillIndex+1:] {
			if candidate.SocketGroup != nil && candidate.SocketGroup.IncludeInFullDPS {
				nextSkill = candidate
				break
			}
		}
		if nextSkill != nil {
			if fullEnv.WeaponSetEnvs != nil || nextSkill.SocketGroup.UsingSkillSet != fullEnv.WeaponSet {
				// Per-set databases cannot be reused across a set transition, and paired
				// environments share auxiliary skill objects mutated by perform().
				fullEnv, cachedPlayerDB, cachedEnemyDB, cachedMinionDB = initFullDPSEnvSpec(main, mode, getSocketGroupOverride(main, override, nextSkill.SocketGroup), &specEnv{})
			} else {
				accelerationTbl := accelerate{
					nodeAlloc:         true,
					requirementsItems: true,
					requirementsGems:  true,
					skills:            true,
					everything:        true,
				}
				fullEnv, _, _, _ = initFullDPSEnvSpec(main, mode, override, &specEnv{cachedPlayerDB: cachedPlayerDB, cachedEnemyDB: cachedEnemyDB, cachedMinionDB: cachedMinionDB, env: fullEnv, accelerate: accelerationTbl})
			}
		}
	}

	// Re-Add ailment DPS components
	//pob2:block:src/Modules/Calcs.lua:439-483
	t := fullDPS.totals
	fullDPS.TotalDotDPS = 0
	if t["bleedDPS"] > 0 {
		fullDPS.Skills = append(fullDPS.Skills, SkillDPS{Name: "Best Bleed DPS", DPS: t["bleedDPS"], Count: 1, Source: sources["bleedDPS"]})
		fullDPS.TotalDotDPS += t["bleedDPS"]
	}
	if t["corruptingBloodDPS"] > 0 {
		fullDPS.Skills = append(fullDPS.Skills, SkillDPS{Name: "Best Corr. Blood DPS", DPS: t["corruptingBloodDPS"], Count: 1, Source: sources["corruptingBloodDPS"]})
		fullDPS.TotalDotDPS += t["corruptingBloodDPS"]
	}
	if t["igniteDPS"] > 0 {
		fullDPS.Skills = append(fullDPS.Skills, SkillDPS{Name: "Best Ignite DPS", DPS: t["igniteDPS"], Count: 1, Source: sources["igniteDPS"]})
		fullDPS.TotalDotDPS += t["igniteDPS"]
	}
	if t["burningGroundDPS"] > 0 {
		fullDPS.Skills = append(fullDPS.Skills, SkillDPS{Name: "Best Burning Ground DPS", DPS: t["burningGroundDPS"], Count: 1, Source: sources["burningGroundDPS"]})
		fullDPS.TotalDotDPS += t["burningGroundDPS"]
	}
	if t["poisonDPS"] > 0 {
		fullDPS.Skills = append(fullDPS.Skills, SkillDPS{Name: "Best Poison DPS", DPS: t["poisonDPS"], Count: 1, Source: sources["poisonDPS"]})
		fullDPS.TotalDotDPS += t["poisonDPS"]
	}
	if t["causticGroundDPS"] > 0 {
		fullDPS.Skills = append(fullDPS.Skills, SkillDPS{Name: "Best Caustic Ground DPS", DPS: t["causticGroundDPS"], Count: 1, Source: sources["causticGroundDPS"]})
		fullDPS.TotalDotDPS += t["causticGroundDPS"]
	}
	if t["impaleDPS"] > 0 {
		fullDPS.Skills = append(fullDPS.Skills, SkillDPS{Name: "Full Impale DPS", DPS: t["impaleDPS"], Count: 1})
		fullDPS.CombinedDPS += t["impaleDPS"]
	}
	if t["decayDPS"] > 0 {
		fullDPS.Skills = append(fullDPS.Skills, SkillDPS{Name: "Full Decay DPS", DPS: t["decayDPS"], Count: 1})
		fullDPS.TotalDotDPS += t["decayDPS"]
	}
	if t["dotDPS"] > 0 {
		fullDPS.Skills = append(fullDPS.Skills, SkillDPS{Name: "Full DoT DPS", DPS: t["dotDPS"], Count: 1})
		fullDPS.TotalDotDPS += t["dotDPS"]
	}
	fullDPS.TotalDotDPS = math.Min(fullDPS.TotalDotDPS, miscConstant("DotDpsCap"))
	fullDPS.CombinedDPS += fullDPS.TotalDotDPS
	if t["cullingMulti"] > 0 {
		cullingDPS := float64(fullDPS.CombinedDPS * (t["cullingMulti"] - 1))
		fullDPS.Skills = append(fullDPS.Skills, SkillDPS{Name: "Full Culling DPS", DPS: cullingDPS, Count: 1})
		fullDPS.CombinedDPS += cullingDPS
	}

	return fullDPS
}

// setFullDPS is the output.SkillDPS, FullDPS and FullDotDPS a caller adds from calcFullDPS.
func setFullDPS(env *Environment, fullDPS *FullDPS) {
	env.Player.SkillDPS = fullDPS.Skills
	env.Player.Output["FullDPS"] = fullDPS.CombinedDPS
	env.Player.Output["FullDotDPS"] = fullDPS.TotalDotDPS
}

// SortSkillDPS is the sidebar's in-place sort of output.SkillDPS by dps * count, the order Save then writes.
//
//pob2:block:src/Modules/Build.lua:2391-2394
func SortSkillDPS(rows []SkillDPS) {
	luaSort(rows, func(a, b SkillDPS) bool { return a.DPS*a.Count > b.DPS*b.Count })
}
