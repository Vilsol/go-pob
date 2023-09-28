package raw

import (
	"github.com/Vilsol/go-pob-data/loader"
	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/utils"
)

type CalculatedGrantedEffect struct {
	*poe.GrantedEffect

	calculatedStats         []string
	calculatedLevels        map[int]*CalculatedLevel
	calculatedConstantStats map[string]float64
	calculatedStatMap       *loader.ComputationCache[string, *StatMap]
}

var grantedEffectCache = make(map[string]*CalculatedGrantedEffect)

func GetCalculatedGrantedEffect(grantedEffect *poe.GrantedEffect) *CalculatedGrantedEffect {
	if cached, ok := grantedEffectCache[grantedEffect.ID]; ok {
		return cached
	}

	return &CalculatedGrantedEffect{
		GrantedEffect: grantedEffect,
	}
}

type CalculatedLevel struct {
	Level             int
	Values            []float64
	Cost              map[string]int
	StatInterpolation []int
	LevelRequirement  int

	ManaReservationFlat    *float64
	ManaReservationPercent *float64
	LifeReservationFlat    *float64
	LifeReservationPercent *float64
	ManaMultiplier         *float64
	DamageEffectiveness    *float64
	CritChance             *float64
	BaseMultiplier         *float64
	AttackSpeedMultiplier  *float64
	AttackTime             *float64
	Cooldown               *float64
	SoulCost               *float64
	SkillUseStorage        *float64
	SoulPreventionDuration *float64
}

type StatMap struct {
	Mods  []mod.Mod
	Value *float64
	Mult  *float64
	Div   *float64
	Base  *float64
}

func (s *StatMap) Clone() *StatMap {
	mods := make([]mod.Mod, len(s.Mods))
	for i, m := range s.Mods {
		mods[i] = m.Clone()
	}

	return &StatMap{
		Mods:  mods,
		Value: s.Value,
		Mult:  s.Mult,
		Div:   s.Div,
		Base:  s.Base,
	}
}

func (g *CalculatedGrantedEffect) calculate() {
	if g.calculatedStats != nil || g.calculatedLevels != nil {
		return
	}

	g.calculatedLevels = make(map[int]*CalculatedLevel)

	statMap := make(map[string]int)

	statsPerLevel := g.GetEffectStatSetsPerLevel()
	effectsPerLevel := g.GetEffectsPerLevel()
	for index, levelRow := range effectsPerLevel {
		statRow := statsPerLevel[index]
		level := &CalculatedLevel{
			Level:             levelRow.Level,
			StatInterpolation: make([]int, 0),
			Values:            make([]float64, 0),
			Cost:              make(map[string]int),
			LevelRequirement:  levelRow.PlayerLevelReq,
		}

		for i, cost := range levelRow.GetCostTypes() {
			level.Cost[cost.ID] = levelRow.CostAmounts[i]
		}

		if levelRow.ManaReservationFlat != 0 {
			level.ManaReservationFlat = utils.Ptr(float64(levelRow.ManaReservationFlat))
		}
		if levelRow.ManaReservationPercent != 0 {
			level.ManaReservationPercent = utils.Ptr(float64(levelRow.ManaReservationPercent) / 100)
		}
		if levelRow.LifeReservationFlat != 0 {
			level.LifeReservationFlat = utils.Ptr(float64(levelRow.LifeReservationFlat))
		}
		if levelRow.LifeReservationPercent != 0 {
			level.LifeReservationPercent = utils.Ptr(float64(levelRow.LifeReservationPercent) / 100)
		}
		if levelRow.CostMultiplier != 100 {
			level.ManaMultiplier = utils.Ptr(float64(levelRow.CostMultiplier - 100))
		}
		if statRow.DamageEffectiveness != 0 {
			level.DamageEffectiveness = utils.Ptr(float64(statRow.DamageEffectiveness)/10000 + 1)
		}
		if statRow.AttackCritChance != 0 {
			level.CritChance = utils.Ptr(float64(statRow.AttackCritChance) / 100)
		}
		if statRow.OffhandCritChance != 0 {
			level.CritChance = utils.Ptr(float64(statRow.OffhandCritChance) / 100)
		}
		if statRow.BaseMultiplier != 0 {
			level.BaseMultiplier = utils.Ptr(float64(statRow.BaseMultiplier)/10000 + 1)
		}
		if levelRow.AttackSpeedMultiplier != 0 {
			level.AttackSpeedMultiplier = utils.Ptr(float64(levelRow.AttackSpeedMultiplier))
		}
		if levelRow.AttackTime != 0 {
			level.AttackTime = utils.Ptr(float64(levelRow.AttackTime))
		}
		if levelRow.Cooldown != 0 {
			level.Cooldown = utils.Ptr(float64(levelRow.Cooldown) / 1000)
		}
		if levelRow.VaalSouls != 0 {
			level.SoulCost = utils.Ptr(float64(levelRow.VaalSouls))
		}
		if levelRow.VaalStoredUses != 0 {
			level.SkillUseStorage = utils.Ptr(float64(levelRow.VaalStoredUses))
		}
		if levelRow.SoulGainPreventionDuration != 0 {
			level.SoulPreventionDuration = utils.Ptr(float64(levelRow.SoulGainPreventionDuration) / 1000)
		}

		level.StatInterpolation = statRow.StatInterpolations

		resolveInterpolation := true
		injectConstantValuesIntoEachLevel := false

		for i, stat := range statRow.GetFloatStats() {
			if _, ok := statMap[stat.ID]; !ok {
				statMap[stat.ID] = len(g.calculatedStats)
				g.calculatedStats = append(g.calculatedStats, stat.ID)
			}

			if resolveInterpolation {
				level.Values = append(level.Values, float64(statRow.BaseResolvedValues[i]))
				level.StatInterpolation[i] = 1
			} else {
				// TODO table.insert(level, statRow.FloatStatsValues[i] / math.max(statRow.InterpolationBases[i].Value, 0.00001) )
			}
		}

		if injectConstantValuesIntoEachLevel {
			/*
				TODO injectConstantValuesIntoEachLevel
				for i, stat in ipairs(granted.GrantedEffectStatSets.ConstantStats) do
					if not statMap[stat.Id] then
						statMap[stat.Id] = #skill.stats + #skill.constantStats + 1
						table.insert(skill.stats, { id = stat.Id })
					end
					table.insert(level, granted.GrantedEffectStatSets.ConstantStatsValues[i])
					table.insert(level.statInterpolation, #statRow.FloatStats + 1, 1)
				end
			*/
		}

		for i, stat := range statRow.GetAdditionalStats() {
			if _, ok := statMap[stat.ID]; !ok {
				statMap[stat.ID] = len(g.calculatedStats)
				g.calculatedStats = append(g.calculatedStats, stat.ID)
			}
			level.Values = append(level.Values, float64(statRow.AdditionalStatsValues[i]))
		}

		for _, stat := range statRow.GetAdditionalBooleanStats() {
			if _, ok := statMap[stat.ID]; !ok {
				statMap[stat.ID] = len(g.calculatedStats)
				g.calculatedStats = append(g.calculatedStats, stat.ID)
			}
		}

		g.calculatedLevels[level.Level] = level
	}

	grantedEffectStatSet := g.GetGrantedEffectStatSet()

	for _, stat := range grantedEffectStatSet.GetImplicitStats() {
		if _, ok := statMap[stat.ID]; !ok {
			statMap[stat.ID] = len(g.calculatedStats)
			g.calculatedStats = append(g.calculatedStats, stat.ID)
		}
	}

	g.calculatedConstantStats = make(map[string]float64)
	for i, stat := range grantedEffectStatSet.GetConstantStats() {
		g.calculatedConstantStats[stat.ID] = float64(grantedEffectStatSet.ConstantStatsValues[i])
	}

	// TODO Pull manually defined stat additions
	g.calculatedStatMap = loader.NewComputationCache[string, *StatMap](func(key string) *StatMap {
		oldMap := SkillStatMap[key]
		if oldMap != nil {
			newMap := oldMap.Clone()
			for i, m := range newMap.Mods {
				newMap.Mods[i] = processMod(g, m)
			}
			return newMap
		}
		return nil
	})
}

func processMod(grantedEffect *CalculatedGrantedEffect, m mod.Mod) mod.Mod {
	out := m.Source(mod.Source("Skill:" + grantedEffect.ID))

	/*
		TODO processMod
		if type(mod.value) == "table" and mod.value.mod then
			mod.value.mod.source = "Skill:"..grantedEffect.id
		end
		for _, tag in ipairs(mod) do
			if tag.type == "GlobalEffect" then
				grantedEffect.hasGlobalEffect = true
				break
			end
		end
	*/

	return out
}

func (g *CalculatedGrantedEffect) GetCalculatedStats() []string {
	g.calculate()
	return g.calculatedStats
}

func (g *CalculatedGrantedEffect) GetCalculatedLevels() map[int]*CalculatedLevel {
	g.calculate()
	return g.calculatedLevels
}

func (g *CalculatedGrantedEffect) GetCalculatedConstantStats() map[string]float64 {
	g.calculate()
	return g.calculatedConstantStats
}

func (g *CalculatedGrantedEffect) GetCalculatedStatMap() *loader.ComputationCache[string, *StatMap] {
	g.calculate()
	return g.calculatedStatMap
}
