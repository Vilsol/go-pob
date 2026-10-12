package raw

import (
	"math"
	"slices"
	"sync"

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
	notMinionStat           []string

	statMapMutex      sync.Mutex
	calculatedStatMap *loader.ComputationCache[string, *StatMap]
	overrideStatMaps  map[string]*StatMap
}

var (
	grantedEffectCacheMutex sync.RWMutex
	grantedEffectCache      = make(map[string]*CalculatedGrantedEffect)
)

//pob:block:src/Export/Scripts/skills.lua:339-348
func padToStatMapOrder(level *CalculatedLevel, statMapOrder []string, statID string, index *int) {
	if *index < len(statMapOrder) && statMapOrder[*index] == statID {
		return
	}
	for *index < len(statMapOrder)-1 && statMapOrder[*index] != statID {
		level.Values = append(level.Values, 0)
		if len(level.StatInterpolation) < len(statMapOrder) && *index <= len(level.StatInterpolation) {
			cloned := make([]int, 0, len(level.StatInterpolation)+1)
			cloned = append(cloned, level.StatInterpolation[:*index]...)
			cloned = append(cloned, 0)
			cloned = append(cloned, level.StatInterpolation[*index:]...)
			level.StatInterpolation = cloned
		}
		*index++
	}
}

//pob:block:src/Modules/Data.lua:967-991
func GetCalculatedGrantedEffect(grantedEffect *poe.GrantedEffect) *CalculatedGrantedEffect {
	grantedEffectCacheMutex.RLock()
	cached, ok := grantedEffectCache[grantedEffect.ID]
	grantedEffectCacheMutex.RUnlock()
	if ok {
		return cached
	}

	grantedEffectCacheMutex.Lock()
	defer grantedEffectCacheMutex.Unlock()

	if cached, ok := grantedEffectCache[grantedEffect.ID]; ok {
		return cached
	}

	computed := &CalculatedGrantedEffect{
		GrantedEffect: grantedEffect,
	}
	computed.calculate()

	grantedEffectCache[grantedEffect.ID] = computed

	return computed
}

var (
	rowListMutex      sync.Mutex
	levelRowsByEffect map[int][]*poe.GrantedEffectsPerLevel
	statRowsBySet     map[int][]*poe.GrantedEffectStatSetsPerLevel
)

var interpolationBaseValues = map[int]float64{
	0: 1,
	1: float64(float32(0.06)),
}

func InterpolationBaseValue(key int) float64 {
	if value, ok := interpolationBaseValues[key]; ok {
		return value
	}
	return 1
}

// local statsPerLevel = dat("GrantedEffectStatSetsPerLevel"):GetRowList("GrantedEffectStatSets", granted.GrantedEffectStatSets)
// local perLevel = dat("GrantedEffectsPerLevel"):GetRowList("GrantedEffect", granted)
//
//	local levelRow = perLevel[indx] or perLevel[1]
//	local statRow = statsPerLevel[indx] or statsPerLevel[1]
//
//pob:block:src/Export/Scripts/skills.lua:256-264
func rowListsForEffect(granted *poe.GrantedEffect) ([]*poe.GrantedEffectsPerLevel, []*poe.GrantedEffectStatSetsPerLevel) {
	rowListMutex.Lock()
	defer rowListMutex.Unlock()
	buildRowIndexes()

	return levelRowsByEffect[granted.Key], statRowsBySet[granted.GrantedEffectStatSets]
}

func buildRowIndexes() {
	if levelRowsByEffect != nil {
		return
	}

	levelRowsByEffect = make(map[int][]*poe.GrantedEffectsPerLevel)
	for _, row := range poe.GrantedEffectsPerLevels {
		levelRowsByEffect[row.GrantedEffect] = append(levelRowsByEffect[row.GrantedEffect], row)
	}

	statRowsBySet = make(map[int][]*poe.GrantedEffectStatSetsPerLevel)
	for _, row := range poe.GrantedEffectStatSetsPerLevels {
		statRowsBySet[row.StatSet] = append(statRowsBySet[row.StatSet], row)
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
	PvPDamageMultiplier    *float64
	SoulCost               *float64
	SkillUseStorage        *float64
	StoredUses             *float64
	SoulPreventionDuration *float64
}

func interpolationOnes(n int) []int {
	out := make([]int, n)
	for i := range out {
		out[i] = 1
	}
	return out
}

type StatMap struct {
	Mods      []mod.Mod
	Value     *float64
	Mult      *float64
	Div       *float64
	Base      *float64
	SkillFlag string

	Groups []*StatMap
}

func (s *StatMap) Clone() *StatMap {
	mods := make([]mod.Mod, len(s.Mods))
	for i, m := range s.Mods {
		mods[i] = m.Clone()
	}

	var groups []*StatMap
	if s.Groups != nil {
		groups = make([]*StatMap, len(s.Groups))
		for i, g := range s.Groups {
			groups[i] = g.Clone()
		}
	}

	return &StatMap{
		Mods:      mods,
		Value:     s.Value,
		Mult:      s.Mult,
		Div:       s.Div,
		Base:      s.Base,
		SkillFlag: s.SkillFlag,
		Groups:    groups,
	}
}

//pob:block:src/Export/Scripts/skills.lua:281-374
func (g *CalculatedGrantedEffect) calculate() {
	if g.calculatedStats != nil || g.calculatedLevels != nil {
		return
	}

	g.calculatedLevels = make(map[int]*CalculatedLevel)

	statMap := make(map[string]int)
	statMapOrder := make([]string, 0)

	effectsPerLevel, statsPerLevel := rowListsForEffect(g.GrantedEffect)

	if hidden, ok := SkillHiddenLevels[g.ID]; ok {
		g.calculatedStats = append(make([]string, 0, len(hidden.Stats)), hidden.Stats...)
		g.calculatedConstantStats = make(map[string]float64)
		for level, values := range hidden.Levels {
			g.calculatedLevels[level] = &CalculatedLevel{
				Level:             level,
				StatInterpolation: interpolationOnes(len(values)),
				Values:            append(make([]float64, 0, len(values)), values...),
				Cost:              make(map[string]int),
			}
		}
		g.calculatedStatMap = loader.NewComputationCache[string, *StatMap](func(key string) *StatMap {
			oldMap := SkillStatMap[key]
			if oldMap == nil {
				return nil
			}
			newMap := oldMap.Clone()
			for i, m := range newMap.Mods {
				newMap.Mods[i] = processMod(g, m, key)
			}
			return newMap
		})
		return
	}

	if len(effectsPerLevel) == 0 {
		g.calculatedStats = make([]string, 0)
		g.calculatedConstantStats = make(map[string]float64)
		g.calculatedLevels[1] = &CalculatedLevel{
			Level:             1,
			StatInterpolation: make([]int, 0),
			Values:            make([]float64, 0),
			Cost:              make(map[string]int),
		}
		g.calculatedStatMap = loader.NewComputationCache[string, *StatMap](func(key string) *StatMap {
			oldMap := SkillStatMap[key]
			if oldMap == nil {
				return nil
			}
			newMap := oldMap.Clone()
			for i, m := range newMap.Mods {
				newMap.Mods[i] = processMod(g, m, key)
			}
			return newMap
		})
		return
	}

	//pob:block:src/Export/Scripts/skills.lua:262-267
	for index := 0; index < max(len(effectsPerLevel), len(statsPerLevel)); index++ {
		if len(effectsPerLevel) == 0 || len(statsPerLevel) == 0 {
			break
		}
		levelRow := effectsPerLevel[0]
		if index < len(effectsPerLevel) {
			levelRow = effectsPerLevel[index]
		}
		statRow := statsPerLevel[0]
		if index < len(statsPerLevel) {
			statRow = statsPerLevel[index]
		}

		levelNumber, levelReq := levelRow.Level, levelRow.PlayerLevelReq
		if len(effectsPerLevel) == 1 {
			levelNumber, levelReq = statRow.GemLevel, statRow.PlayerLevelReq
		}

		level := &CalculatedLevel{
			Level:             levelNumber,
			StatInterpolation: make([]int, 0),
			Values:            make([]float64, 0),
			Cost:              make(map[string]int),
			LevelRequirement:  levelReq,
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
		if levelRow.PvPDamageMultiplier != 0 {
			level.PvPDamageMultiplier = utils.Ptr(float64(levelRow.PvPDamageMultiplier))
		}
		if levelRow.VaalSouls != 0 {
			level.SoulCost = utils.Ptr(float64(levelRow.VaalSouls))
			level.Cost["Soul"] = levelRow.VaalSouls
		}
		if levelRow.VaalStoredUses != 0 {
			level.SkillUseStorage = utils.Ptr(float64(levelRow.VaalStoredUses))
		}
		if levelRow.StoredUses != 0 {
			level.StoredUses = utils.Ptr(float64(levelRow.StoredUses))
		}
		if levelRow.SoulGainPreventionDuration != 0 {
			level.SoulPreventionDuration = utils.Ptr(float64(levelRow.SoulGainPreventionDuration) / 1000)
		}

		level.StatInterpolation = statRow.StatInterpolations

		injectConstantValuesIntoEachLevel := false

		statMapOrderIndex := 0

		for i, stat := range statRow.GetFloatStats() {
			if _, ok := statMap[stat.ID]; !ok || index == 0 {
				statMap[stat.ID] = len(g.calculatedStats)
				g.calculatedStats = append(g.calculatedStats, stat.ID)
				if index == 0 {
					statMapOrder = append(statMapOrder, stat.ID)
					//pob:block:src/Export/Scripts/skills.lua:333-335
					if stat.CannotGrantToMinion {
						g.notMinionStat = append(g.notMinionStat, stat.ID)
					}
				}
			} else {
				padToStatMapOrder(level, statMapOrder, stat.ID, &statMapOrderIndex)
			}
			statMapOrderIndex++

			level.Values = append(level.Values,
				float64(float32(statRow.FloatStatsValues[i]))/math.Max(InterpolationBaseValue(statRow.InterpolationBases[i]), 0.00001))
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
				if index == 0 {
					statMapOrder = append(statMapOrder, stat.ID)
					//pob:block:src/Export/Scripts/skills.lua:388-390
					if stat.CannotGrantToMinion {
						g.notMinionStat = append(g.notMinionStat, stat.ID)
					}
				}
			} else {
				padToStatMapOrder(level, statMapOrder, stat.ID, &statMapOrderIndex)
			}
			statMapOrderIndex++
			level.Values = append(level.Values, float64(statRow.AdditionalStatsValues[i]))
		}

		for _, stat := range statRow.GetAdditionalBooleanStats() {
			if _, ok := statMap[stat.ID]; !ok {
				statMap[stat.ID] = len(g.calculatedStats)
				g.calculatedStats = append(g.calculatedStats, stat.ID)
				//pob:block:src/Export/Scripts/skills.lua:411-413
				if stat.CannotGrantToMinion {
					g.notMinionStat = append(g.notMinionStat, stat.ID)
				}
			}
		}

		g.calculatedLevels[level.Level] = level
	}

	grantedEffectStatSet := g.GetGrantedEffectStatSet()
	if grantedEffectStatSet == nil {
		g.calculatedConstantStats = make(map[string]float64)
		return
	}

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

	g.calculatedStatMap = loader.NewComputationCache[string, *StatMap](func(key string) *StatMap {
		oldMap := SkillStatMap[key]
		if oldMap != nil {
			newMap := oldMap.Clone()
			for i, m := range newMap.Mods {
				newMap.Mods[i] = processMod(g, m, key)
			}
			return newMap
		}
		return nil
	})
}

//pob:src/Modules/Data.lua:68-92
func processMod(grantedEffect *CalculatedGrantedEffect, m mod.Mod, statName string) mod.Mod {
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

	notMinionStat := false
	if grantedEffect.notMinionStat != nil && statName != "" && (grantedEffect.IsSupport || slices.Contains(grantedEffectSkillTypes(grantedEffect.GrantedEffect), "Buff")) {
		for _, notMinionStatName := range grantedEffect.notMinionStat {
			if notMinionStatName == statName {
				notMinionStat = true
			}
		}
	}
	if notMinionStat {
		out = out.Tag(mod.ActorCondition("parent").Neg(true))
	}

	return out
}

func grantedEffectSkillTypes(grantedEffect *poe.GrantedEffect) []string {
	if names, ok := SkillTypes[grantedEffect.ID]; ok {
		return names
	}
	var out []string
	if as := grantedEffect.GetActiveSkill(); as != nil {
		for _, t := range as.GetActiveSkillTypes() {
			out = append(out, t.ID)
		}
	}
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

func (g *CalculatedGrantedEffect) GetNotMinionStat() []string {
	g.calculate()
	return g.notMinionStat
}

//pob:block:src/Export/Scripts/skills.lua:158-163
func (g *CalculatedGrantedEffect) GetStatSet() *poe.GrantedEffectStatSet {
	return g.GetGrantedEffectStatSet()
}

//pob:src/Modules/Data.lua:951-963
func (g *CalculatedGrantedEffect) GetStatMapEntry(key string) *StatMap {
	g.calculate()
	if perSkill := SkillStatMaps[g.ID]; perSkill != nil {
		if override := perSkill[key]; override != nil {
			g.statMapMutex.Lock()
			defer g.statMapMutex.Unlock()
			if newMap, ok := g.overrideStatMaps[key]; ok {
				return newMap
			}
			newMap := override.Clone()
			for i, m := range newMap.Mods {
				newMap.Mods[i] = processMod(g, m, key)
			}
			for _, grp := range newMap.Groups {
				for i, m := range grp.Mods {
					grp.Mods[i] = processMod(g, m, key)
				}
			}
			if g.overrideStatMaps == nil {
				g.overrideStatMaps = map[string]*StatMap{}
			}
			g.overrideStatMaps[key] = newMap
			return newMap
		}
	}
	g.statMapMutex.Lock()
	entry := g.calculatedStatMap.Get(key)
	g.statMapMutex.Unlock()

	return entry
}
