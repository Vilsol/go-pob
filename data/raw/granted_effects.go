package raw

import (
	"github.com/Vilsol/go-pob/cache"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/utils"
)

type GrantedEffect struct {
	ID                    string `json:"Id"`
	IsSupport             bool   `json:"IsSupport"`
	SupportTypes          []int  `json:"AllowedActiveSkillTypes"`
	SupportGemLetter      string `json:"SupportGemLetter"`
	Attribute             int    `json:"Attribute"`
	AddTypes              []int  `json:"AddedActiveSkillTypes"`
	ExcludeTypes          []int  `json:"ExcludedActiveSkillTypes"`
	SupportsGemsOnly      bool   `json:"SupportsGemsOnly"`
	CannotBeSupported     bool   `json:"CannotBeSupported"`
	CastTime              int    `json:"CastTime"`
	ActiveSkill           *int   `json:"ActiveSkill"`
	IgnoreMinionTypes     bool   `json:"IgnoreMinionTypes"`
	AddMinionTypes        []int  `json:"AddedMinionActiveSkillTypes"`
	Animation             *int   `json:"Animation"`
	WeaponRestrictions    []int  `json:"SupportWeaponRestrictions"`
	PlusVersionOf         *int   `json:"RegularVariant"`
	GrantedEffectStatSets int    `json:"StatSet"`
	Key                   int    `json:"_key"`

	calculatedStats         []string
	calculatedLevels        map[int]*CalculatedLevel
	calculatedConstantStats map[string]float64
	calculatedStatMap       *cache.ComputationCache[string, *StatMap]
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

var GrantedEffects []*GrantedEffect
var GrantedEffectsMap map[int]*GrantedEffect

var grantedEffectsByIDMap map[string]*GrantedEffect

func InitializeGrantedEffects(version string) error {
	if err := InitHelper(version, "GrantedEffects", &GrantedEffects); err != nil {
		return err
	}

	GrantedEffectsMap = make(map[int]*GrantedEffect)
	for _, i := range GrantedEffects {
		GrantedEffectsMap[i.Key] = i
	}

	grantedEffectsByIDMap = make(map[string]*GrantedEffect)
	for _, i := range GrantedEffects {
		grantedEffectsByIDMap[i.ID] = i
	}

	return nil
}

func GrantedEffectByID(id string) *GrantedEffect {
	return grantedEffectsByIDMap[id]
}

func (g *GrantedEffect) GetActiveSkill() *ActiveSkill {
	if g.ActiveSkill == nil {
		return nil
	}

	return ActiveSkillsMap[*g.ActiveSkill]
}

func (g *GrantedEffect) GetEffectsPerLevel() map[int]*GrantedEffectsPerLevel {
	return grantedEffectsPerLevelsByIDMap[g.Key]
}

func (g *GrantedEffect) GetEffectStatSetsPerLevel() map[int]*GrantedEffectStatSetsPerLevel {
	return grantedEffectStatSetsPerLevelsByIDMap[g.Key]
}

func (g *GrantedEffect) GetEffectQualityStats() map[int]*GrantedEffectQualityStat {
	return grantedEffectQualityStatsByIDMap[g.Key]
}

func (g *GrantedEffect) HasGlobalEffect() bool {
	// TODO HasGlobalEffect
	return false
}

func (g *GrantedEffect) Levels() map[int]*GrantedEffectsPerLevel {
	return grantedEffectsPerLevelsByIDMap[g.Key]
}

func (g *GrantedEffect) GetGrantedEffectStatSet() *GrantedEffectStatSet {
	return GrantedEffectStatSetsMap[g.GrantedEffectStatSets]
}

func (g *GrantedEffect) Calculate() {
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
	g.calculatedStatMap = cache.NewComputationCache[string, *StatMap](func(key string) *StatMap {
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

func processMod(grantedEffect *GrantedEffect, m mod.Mod) mod.Mod {
	out := m.Source(mod.Source("Skill:" + grantedEffect.GetActiveSkill().ID))

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

func (g *GrantedEffect) GetCalculatedStats() []string {
	g.Calculate()
	return g.calculatedStats
}

func (g *GrantedEffect) GetCalculatedLevels() map[int]*CalculatedLevel {
	g.Calculate()
	return g.calculatedLevels
}

func (g *GrantedEffect) GetCalculatedConstantStats() map[string]float64 {
	g.Calculate()
	return g.calculatedConstantStats
}

func (g *GrantedEffect) GetCalculatedStatMap() *cache.ComputationCache[string, *StatMap] {
	g.Calculate()
	return g.calculatedStatMap
}
