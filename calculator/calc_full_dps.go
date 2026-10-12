package calculator

import (
	"math"
	"slices"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/moddb"
	"github.com/Vilsol/go-pob/pob"
)

//pob:src/Modules/Calcs.lua:176-338
func calcFullDPS(c *Calculator, env *Environment, mode OutputMode, cache *EnvironmentCache, override *CalcOverride, specEnv *calcSpecEnv) (float64, float64, []SkillDPS) {
	build := c.PoB
	if build == nil || len(build.Skills.SkillSets) == 0 {
		return 0, 0, []SkillDPS{}
	}

	setIndex := activeSkillSetIndex(build.Skills)
	if setIndex < 0 {
		return 0, 0, []SkillDPS{}
	}
	groups := build.Skills.SkillSets[setIndex].Skills
	if !slices.ContainsFunc(groups, func(g pob.Skill) bool { return g.IncludeInFullDPS != nil && *g.IncludeInFullDPS }) {
		return 0, 0, []SkillDPS{}
	}

	var (
		combinedDPS float64
		f           fullDPSBuckets
		skills      = []SkillDPS{}
	)

	//pob:block:src/Modules/Calcs.lua:177
	fullEnv, cachedPlayerDB, cachedEnemyDB, cachedMinionDB := initEnvSpec(build, cache, mode, false, override, specEnv)
	fullEnv.triggerCache = env.triggerCache

	for index, activeSkill := range fullEnv.Player.ActiveSkillList {
		socketGroup, ok := activeSkill.SocketGroup.(*pob.Skill)
		if !ok || socketGroup == nil {
			continue
		}

		gi := activeSkill.SocketGroupIndex
		if gi < 1 || gi > len(groups) {
			continue
		}

		if socketGroup.IncludeInFullDPS == nil || !*socketGroup.IncludeInFullDPS {
			continue
		}

		activeSkillCount, enabled := getActiveSkillCount(activeSkill)
		if !enabled {
			continue
		}

		fullEnv.Player.MainSkill = activeSkill
		performCalc(fullEnv, true)
		recordFullDPSSkillCosts(env, index, fullEnv)
		subEnv := fullEnv
		skillName := activeSkill.ActiveEffect.GrantedEffect.Name()
		haveMinionName := false
		if minion := subEnv.Minion; minion != nil {
			mout := minion.Output
			if mout["TotalDPS"] > 0 {
				//pob:block:src/Modules/Calcs.lua:212-213
				haveMinionName = true
				minionName := ""
				if activeSkill.Minion != nil && activeSkill.Minion.MinionData != nil {
					minionName = activeSkill.Minion.MinionData.Name + ": "
				} else if minion.MinionData != nil {
					minionName = minion.MinionData.Name + ": "
				}
				skills = append(skills, SkillDPS{Name: skillName, DPS: mout["TotalDPS"], Count: activeSkillCount, Trigger: activeSkill.infoTrigger, SkillPart: minionName + activeSkill.SkillPartName})
				combinedDPS += mout["TotalDPS"] * activeSkillCount
			}
			if mout["BleedDPS"] > f.BleedDPS {
				f.BleedDPS = mout["BleedDPS"]
				f.bleedSource = skillName
			}
			if mout["IgniteDPS"] > f.IgniteDPS {
				f.IgniteDPS = mout["IgniteDPS"]
				f.igniteSource = skillName
			}
			if mout["PoisonDPS"] > 0 {
				f.PoisonDPS += mout["TotalPoisonDPS"] * activeSkillCount
			}
			if mout["ImpaleDPS"] > 0 {
				f.ImpaleDPS += mout["ImpaleDPS"] * activeSkillCount
			}
			if mout["DecayDPS"] > 0 {
				f.DecayDPS += mout["DecayDPS"]
			}
			if mout["TotalDot"] > 0 {
				f.DotDPS += mout["TotalDot"]
			}
			if mout["CullMultiplier"] > 1 && mout["CullMultiplier"] > f.CullingMulti {
				f.CullingMulti = mout["CullMultiplier"]
			}
		}

		out := subEnv.Player.Output

		if mainSkill := subEnv.Player.MainSkill; mainSkill != nil && mainSkill.Mirage != nil {
			mirage := mainSkill.Mirage
			mout := mirage.Output
			mirageCount := mirage.Count
			if mirageCount == 0 {
				mirageCount = 1
			}
			mirageCount *= activeSkillCount

			if mout["TotalDPS"] > 0 {
				//pob:block:src/Modules/Calcs.lua:251
				skills = append(skills, SkillDPS{Name: mirage.Name + " (Mirage)", DPS: mout["TotalDPS"], Count: mirageCount, SkillPart: mirage.SkillPartName})
				combinedDPS += mout["TotalDPS"] * mirageCount
			}
			if mout["BleedDPS"] > f.BleedDPS {
				f.BleedDPS = mout["BleedDPS"]
				f.bleedSource = skillName + " (Mirage)"
			}
			if mout["IgniteDPS"] > f.IgniteDPS {
				f.IgniteDPS = mout["IgniteDPS"]
				f.igniteSource = skillName + " (Mirage)"
			}
			if mout["PoisonDPS"] > 0 {
				f.PoisonDPS += mout["TotalPoisonDPS"] * mirageCount
			}
			if mout["ImpaleDPS"] > 0 {
				f.ImpaleDPS += mout["ImpaleDPS"] * mirageCount
			}
			if mout["DecayDPS"] > 0 {
				f.DecayDPS += mout["DecayDPS"]
			}
			playerDot, playerDotSet := out["TotalDot"]
			dotCanStack := mainSkill.SkillFlags[SkillFlagDotCanStack]
			if mout["TotalDot"] > 0 && (dotCanStack || (playerDotSet && playerDot == 0)) {
				stackCount := float64(1)
				if dotCanStack {
					stackCount = mirageCount
				}
				f.DotDPS += mout["TotalDot"] * stackCount
			}
			if mout["CullMultiplier"] > 1 && mout["CullMultiplier"] > f.CullingMulti {
				f.CullingMulti = mout["CullMultiplier"]
			}
			if mout["BurningGroundDPS"] > f.BurningGroundDPS {
				f.BurningGroundDPS = mout["BurningGroundDPS"]
				f.burningGroundSource = skillName + " (Mirage)"
			}
			if mout["CausticGroundDPS"] > f.CausticGroundDPS {
				f.CausticGroundDPS = mout["CausticGroundDPS"]
				f.causticGroundSource = skillName + " (Mirage)"
			}
		}

		if out["TotalDPS"] > 0 {
			//pob:block:src/Modules/Calcs.lua:288
			skillPart := activeSkill.SkillPartName
			if haveMinionName && activeSkill.infoMessage2 != "" {
				skillPart = activeSkill.infoMessage2
			}
			skills = append(skills, SkillDPS{Name: skillName, DPS: out["TotalDPS"], Count: activeSkillCount, Trigger: activeSkill.infoTrigger, SkillPart: skillPart})
			combinedDPS += out["TotalDPS"] * activeSkillCount
		}
		if out["BleedDPS"] > f.BleedDPS {
			f.BleedDPS = out["BleedDPS"]
			f.bleedSource = skillName
		}
		if out["CorruptingBloodDPS"] > f.CorruptingBloodDPS {
			f.CorruptingBloodDPS = out["CorruptingBloodDPS"]
			f.corruptingBloodSource = skillName
		}
		if out["IgniteDPS"] > f.IgniteDPS {
			f.IgniteDPS = out["IgniteDPS"]
			f.igniteSource = skillName
		}
		if out["BurningGroundDPS"] > f.BurningGroundDPS {
			f.BurningGroundDPS = out["BurningGroundDPS"]
			f.burningGroundSource = skillName
		}
		if out["PoisonDPS"] > 0 {
			f.PoisonDPS += out["TotalPoisonDPS"] * activeSkillCount
		}
		if out["CausticGroundDPS"] > f.CausticGroundDPS {
			f.CausticGroundDPS = out["CausticGroundDPS"]
			f.causticGroundSource = skillName
		}
		if out["ImpaleDPS"] > 0 {
			f.ImpaleDPS += out["ImpaleDPS"] * activeSkillCount
		}
		if out["DecayDPS"] > 0 {
			f.DecayDPS += out["DecayDPS"]
		}
		if out["TotalDot"] > 0 {
			stackCount := float64(1)
			if subEnv.Player.MainSkill != nil && subEnv.Player.MainSkill.SkillFlags[SkillFlagDotCanStack] {
				stackCount = activeSkillCount
			}
			f.DotDPS += out["TotalDot"] * stackCount
		}
		if out["CullMultiplier"] > 1 && out["CullMultiplier"] > f.CullingMulti {
			f.CullingMulti = out["CullMultiplier"]
		}

		fullEnv = reinitFullDPSEnv(fullEnv, cache, mode, cachedPlayerDB, cachedEnemyDB, cachedMinionDB)
	}

	combinedDPS, totalDotDPS := f.rollUp(combinedDPS, &skills)

	return combinedDPS, totalDotDPS, skills
}

//pob:block:src/Modules/Calcs.lua:327-335
func reinitFullDPSEnv(fullEnv *Environment, cache *EnvironmentCache, mode OutputMode, cachedPlayerDB, cachedEnemyDB, cachedMinionDB moddb.ModStoreFuncs) *Environment {
	next, _, _, _ := initEnvSpec(fullEnv.Build, cache, mode, false, fullEnv.override, &calcSpecEnv{
		cachedPlayerDB: cachedPlayerDB,
		cachedEnemyDB:  cachedEnemyDB,
		cachedMinionDB: cachedMinionDB,
		env:            fullEnv,
		accelerate: calcAccelerate{
			nodeAlloc:         true,
			requirementsItems: true,
			requirementsGems:  true,
			skills:            true,
			everything:        true,
		},
	})
	return next
}

type fullDPSBuckets struct {
	BleedDPS           float64
	CorruptingBloodDPS float64
	IgniteDPS          float64
	BurningGroundDPS   float64
	CausticGroundDPS   float64

	PoisonDPS float64
	ImpaleDPS float64
	DecayDPS  float64
	DotDPS    float64

	CullingMulti float64

	bleedSource           string
	corruptingBloodSource string
	igniteSource          string
	burningGroundSource   string
	causticGroundSource   string
}

//pob:block:src/Modules/Calcs.lua:340-385
func (f fullDPSBuckets) rollUp(combinedDPS float64, skills *[]SkillDPS) (float64, float64) {
	var totalDotDPS float64
	if f.BleedDPS > 0 {
		*skills = append(*skills, SkillDPS{Name: "Best Bleed DPS", DPS: f.BleedDPS, Count: 1, Source: f.bleedSource})
		totalDotDPS += f.BleedDPS
	}
	if f.CorruptingBloodDPS > 0 {
		*skills = append(*skills, SkillDPS{Name: "Corrupting Blood DPS", DPS: f.CorruptingBloodDPS, Count: 1, Source: f.corruptingBloodSource})
		totalDotDPS += f.CorruptingBloodDPS
	}
	if f.IgniteDPS > 0 {
		*skills = append(*skills, SkillDPS{Name: "Best Ignite DPS", DPS: f.IgniteDPS, Count: 1, Source: f.igniteSource})
		totalDotDPS += f.IgniteDPS
	}
	if f.BurningGroundDPS > 0 {
		*skills = append(*skills, SkillDPS{Name: "Best Burning Ground DPS", DPS: f.BurningGroundDPS, Count: 1, Source: f.burningGroundSource})
		totalDotDPS += f.BurningGroundDPS
	}
	if f.PoisonDPS > 0 {
		poisonDPS := math.Min(f.PoisonDPS, data.DotDpsCap)
		*skills = append(*skills, SkillDPS{Name: "Full Poison DPS", DPS: poisonDPS, Count: 1})
		totalDotDPS += poisonDPS
	}
	if f.CausticGroundDPS > 0 {
		*skills = append(*skills, SkillDPS{Name: "Best Caustic Ground DPS", DPS: f.CausticGroundDPS, Count: 1, Source: f.causticGroundSource})
		totalDotDPS += f.CausticGroundDPS
	}
	if f.ImpaleDPS > 0 {
		*skills = append(*skills, SkillDPS{Name: "Full Impale DPS", DPS: f.ImpaleDPS, Count: 1})
		combinedDPS += f.ImpaleDPS
	}
	if f.DecayDPS > 0 {
		*skills = append(*skills, SkillDPS{Name: "Full Decay DPS", DPS: f.DecayDPS, Count: 1})
		totalDotDPS += f.DecayDPS
	}
	if f.DotDPS > 0 {
		*skills = append(*skills, SkillDPS{Name: "Full DoT DPS", DPS: f.DotDPS, Count: 1})
		totalDotDPS += f.DotDPS
	}
	totalDotDPS = math.Min(totalDotDPS, data.DotDpsCap)
	combinedDPS += totalDotDPS
	if f.CullingMulti > 0 {
		cullingDPS := combinedDPS * (f.CullingMulti - 1)
		*skills = append(*skills, SkillDPS{Name: "Full Culling DPS", DPS: cullingDPS, Count: 1})
		combinedDPS += cullingDPS
	}
	return combinedDPS, totalDotDPS
}

//pob:src/Modules/Calcs.lua:154-178
func getActiveSkillCount(activeSkill *ActiveSkill) (float64, bool) {
	socketGroup, ok := activeSkill.SocketGroup.(*pob.Skill)
	if !ok || socketGroup == nil {
		return 1, true
	}

	if socketGroup.GroupCount != nil {
		return float64(*socketGroup.GroupCount), true
	}

	if activeSkill.ActiveEffect == nil || activeSkill.ActiveEffect.SrcInstance == nil ||
		activeSkill.ActiveEffect.GemData == nil || activeSkill.ActiveEffect.GrantedEffect == nil {
		return 1, true
	}
	gem := activeSkill.ActiveEffect.SrcInstance
	gemData := activeSkill.ActiveEffect.GemData

	count := 1.0
	if gem.Count != nil {
		count = float64(*gem.Count)
	}

	//pob:block:src/Modules/Data.lua:1078-1081
	var effects []*poe.GrantedEffect
	if _, bare := resolveGemInstance(*gem); bare != nil {
		effects = []*poe.GrantedEffect{bare}
	} else {
		effects = gemData.GetGrantedEffects()
	}
	if secondary := secondaryGrantedEffectOf(gemData, gem.SkillID, gem.VariantID); secondary != nil {
		if len(effects) == 0 || effects[0] != secondary {
			effects = append(effects, secondary)
		}
	}
	raw := activeSkill.ActiveEffect.GrantedEffect.Raw
	if gemData.IsVaalGem {
		switch {
		case len(effects) > 0 && effects[0] == raw:
			return count, gem.EnableGlobal1
		case len(effects) > 1 && effects[1] == raw:
			return count, gem.EnableGlobal2
		}
		return 1, true
	}

	if len(effects) > 0 && effects[0] == raw && !effects[0].IsSupport {
		return count, true
	}
	if len(effects) > 1 && effects[1] == raw {
		return count, true
	}
	return 1, true
}
