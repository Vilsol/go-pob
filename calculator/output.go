package calculator

import (
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
	"github.com/Vilsol/go-pob/utils"
)

var envCache = &EnvironmentCache{}

func NewEnvironmentCache() *EnvironmentCache {
	return &EnvironmentCache{}
}

var MainOnlyOutputs = []string{"ExtraPoints", "Spec:LifeInc", "Spec:ManaInc", "Spec:ArmourInc", "Spec:EvasionInc", "Spec:EnergyShieldInc"}

//pob:src/Modules/Calcs.lua:343-474
func BuildOutputWithCache(c *Calculator, mode OutputMode, cache *EnvironmentCache) *Environment {
	env, cachedPlayerDB, cachedEnemyDB, cachedMinionDB := initEnvWith(c.PoB, cache, mode, c.AlignCalcsSkillToMain, nil)
	PerformCalc(env)
	env.CalcProps = FinalizeCalcTab(env)

	if mode == OutputModeCalcs {
		if c.main == nil {
			collectUsedTables(env)
			calcsBuffLists(env, env.MultipliersUsed)
		} else {
			calcsBuffLists(env, c.main.MultipliersUsed)
		}
		env.Player.Breakdown.cFloatSpelling()
		if env.Minion != nil {
			env.Minion.Breakdown.cFloatSpelling()
		}
	}

	if mode == OutputModeMain {
		output := env.Player.Output
		output["ExtraPoints"] = env.ModDB.Sum(mod.TypeBase, nil, "ExtraPoints")

		specCfg := &moddb.ListCfg{
			Source: utils.Ptr(mod.Source("Tree")),
		}
		output["Spec:LifeInc"] = env.ModDB.Sum(mod.TypeIncrease, specCfg, "Life")
		output["Spec:ManaInc"] = env.ModDB.Sum(mod.TypeIncrease, specCfg, "Mana")
		output["Spec:ArmourInc"] = env.ModDB.Sum(mod.TypeIncrease, specCfg, "Armour", "ArmourAndEvasion")
		output["Spec:EvasionInc"] = env.ModDB.Sum(mod.TypeIncrease, specCfg, "Evasion", "ArmourAndEvasion")
		output["Spec:EnergyShieldInc"] = env.ModDB.Sum(mod.TypeIncrease, specCfg, "EnergyShield")

		collectUsedTables(env)
	}

	fullDPS, fullDotDPS, skillDPS := calcFullDPS(c, env, OutputModeCalculator, cache, nil, &calcSpecEnv{cachedPlayerDB: cachedPlayerDB, cachedEnemyDB: cachedEnemyDB, cachedMinionDB: cachedMinionDB})
	env.Player.SkillDPS = skillDPS
	env.Player.Output["FullDPS"] = fullDPS
	env.Player.Output["FullDotDPS"] = fullDotDPS

	if mode == OutputModeMain {
		c.main = env
	}
	return env
}

//pob:src/Modules/Calcs.lua:417-420
func (c *Calculator) BuildOutput(mode OutputMode) *Environment {
	return BuildOutputWithCache(c, mode, envCache)
}
