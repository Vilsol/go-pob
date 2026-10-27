package calculator2

import (
	"math"
	"sort"
	"strings"
	"sync"

	"github.com/Vilsol/go-pob/data/poe2"
	"github.com/Vilsol/go-pob/data/poe2skills"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

var (
	posInf = math.Inf(1)
	negInf = math.Inf(-1)
)

//pob2:src/Modules/CalcTools.lua:16-19
func calcLibMod(modStore moddb.ModStoreFuncs, cfg *moddb.ListCfg, names ...string) float64 {
	inc, more := calcLibMods(modStore, cfg, names...)
	return inc * more
}

//pob2:src/Modules/CalcTools.lua:26-47
func calcLibMods(modStore moddb.ModStoreFuncs, cfg *moddb.ListCfg, names ...string) (float64, float64) {
	return 1 + modStore.Sum(mod.TypeIncrease, cfg, names...)/100, modStore.More(cfg, names...)
}

//pob2:src/Modules/CalcTools.lua:50-57
func calcLibVal(modStore moddb.ModStoreFuncs, name string, cfg *moddb.ListCfg) float64 {
	baseVal := modStore.Sum(mod.TypeBase, cfg, name)
	if baseVal != 0 {
		return baseVal * calcLibMod(modStore, cfg, name)
	}
	return 0
}

// GemData is what calcLib.gemIsType reads of a gem (gem.gemData or the gem itself).
type GemData interface {
	GemName() string
	GemTags() map[string]bool
}

// gemCorrupted is a gem that carries `gem.corrupted`.
type gemCorrupted interface {
	GemCorrupted() bool
}

//pob2:src/Modules/CalcTools.lua:136-149
func gemIsType(gem GemData, typ string, includeTransfigured bool) bool {
	tags := gem.GemTags()
	name := strings.ToLower(gem.GemName())
	corrupted := false
	if c, ok := gem.(gemCorrupted); ok {
		corrupted = c.GemCorrupted()
	}
	return typ == "all" ||
		(typ == "corrupted" && corrupted) ||
		(typ == "elemental" && (tags["fire"] || tags["cold"] || tags["lightning"])) ||
		(typ == "aoe" && tags["area"]) ||
		(typ == "trap or mine" && (tags["trap"] || tags["mine"])) ||
		((typ == "active skill" || typ == "grants_active_skill" || typ == "skill") && tags["grants_active_skill"] && !tags["support"]) ||
		(typ == "non-vaal" && !tags["vaal"]) ||
		typ == name ||
		typ == strings.TrimPrefix(name, "vaal ") ||
		(includeTransfigured && isGemIDSame(gem.GemName(), typ)) ||
		((typ != "active skill" && typ != "grants_active_skill" && typ != "skill") && tags[typ])
}

var (
	gemForBaseNameOnce sync.Once
	gemForBaseName     map[string]*poe2.Gem
)

// PoB fills data.gemForBaseName in pairs() order, so of two colliding names an arbitrary one wins;
// ids are walked sorted here.
//
//pob2:block:src/Modules/Data.lua:987-995
func gemByBaseName(name string) *poe2.Gem {
	gemForBaseNameOnce.Do(func() {
		gemForBaseName = make(map[string]*poe2.Gem, len(poe2.Gems))
		gems := make([]*poe2.Gem, len(poe2.Gems))
		for i := range poe2.Gems {
			gems[i] = &poe2.Gems[i]
		}
		sort.Slice(gems, func(i, j int) bool { return gems[i].ID < gems[j].ID })
		for _, gem := range gems {
			baseName := gem.Name
			if gem.Support && gem.GrantedEffectID != "SupportBarrage" {
				baseName += " Support"
			}
			gemForBaseName[strings.ToLower(baseName)] = gem
			if gem.BaseTypeName != "" && gem.BaseTypeName != baseName {
				gemForBaseName[strings.ToLower(gem.BaseTypeName)] = gem
			}
		}
	})
	return gemForBaseName[strings.ToLower(name)]
}

// The generated gem table carries no vaalGem flag, so dropVaal is not ported.
//
//pob2:src/Modules/CalcTools.lua:259-272
func getGameIDFromGemName(gemName string) string {
	gem := gemByBaseName(gemName)
	if gem == nil {
		return ""
	}
	return gem.GameID
}

//pob2:src/Modules/CalcTools.lua:279-283
func isGemIDSame(gemName, typeName string) bool {
	a := getGameIDFromGemName(gemName)
	return a != "" && a == getGameIDFromGemName(typeName)
}

//pob2:block:src/Modules/Data.lua:482-603
var highPrecisionMods = map[string]map[mod.Type]int{
	"CritChance": {mod.TypeBase: 2}, "SelfCritChance": {mod.TypeBase: 2},
	"LifeRegenPercent": {mod.TypeBase: 2}, "ManaRegenPercent": {mod.TypeBase: 2},
	"EnergyShieldRegenPercent": {mod.TypeBase: 2}, "WardRegenPercent": {mod.TypeBase: 2},
	"LifeRegen": {mod.TypeBase: 1}, "ManaRegen": {mod.TypeBase: 1},
	"EnergyShieldRegen": {mod.TypeBase: 1}, "WardRegen": {mod.TypeBase: 1}, "RageRegen": {mod.TypeBase: 1},
	"LifeDegenPercent": {mod.TypeBase: 2}, "ManaDegenPercent": {mod.TypeBase: 2},
	"EnergyShieldDegenPercent": {mod.TypeBase: 2},
	"LifeDegen":                {mod.TypeBase: 1}, "ManaDegen": {mod.TypeBase: 1}, "EnergyShieldDegen": {mod.TypeBase: 1},
	"DamageLifeLeech": {mod.TypeBase: 2}, "PhysicalDamageLifeLeech": {mod.TypeBase: 2},
	"ElementalDamageLifeLeech": {mod.TypeBase: 2}, "FireDamageLifeLeech": {mod.TypeBase: 2},
	"ColdDamageLifeLeech": {mod.TypeBase: 2}, "LightningDamageLifeLeech": {mod.TypeBase: 2},
	"ChaosDamageLifeLeech": {mod.TypeBase: 2},
	"DamageManaLeech":      {mod.TypeBase: 2}, "PhysicalDamageManaLeech": {mod.TypeBase: 2},
	"ElementalDamageManaLeech": {mod.TypeBase: 2}, "FireDamageManaLeech": {mod.TypeBase: 2},
	"ColdDamageManaLeech": {mod.TypeBase: 2}, "LightningDamageManaLeech": {mod.TypeBase: 2},
	"ChaosDamageManaLeech":    {mod.TypeBase: 2},
	"DamageEnergyShieldLeech": {mod.TypeBase: 2}, "PhysicalDamageEnergyShieldLeech": {mod.TypeBase: 2},
	"ElementalDamageEnergyShieldLeech": {mod.TypeBase: 2}, "FireDamageEnergyShieldLeech": {mod.TypeBase: 2},
	"ColdDamageEnergyShieldLeech": {mod.TypeBase: 2}, "LightningDamageEnergyShieldLeech": {mod.TypeBase: 2},
	"ChaosDamageEnergyShieldLeech": {mod.TypeBase: 2},
	"SupportManaMultiplier":        {mod.TypeMore: 4}, "ReservationMultiplier": {mod.TypeMore: 4},
}

// BaseFlagData is a skill gem whose granted effect has stat sets; StatSetBaseFlags is statSets[1].baseFlags.
type BaseFlagData interface {
	StatSetBaseFlags() map[string]bool
}

//pob2:block:src/Classes/ModStore.lua:990-995
func gemBaseFlags(cfg *moddb.ListCfg) map[string]bool {
	if cfg == nil {
		return nil
	}
	if gem, ok := cfg.GemData.(BaseFlagData); ok {
		return gem.StatSetBaseFlags()
	}
	return cfg.BaseFlags
}

// kernelRules is how PoB-PoE2's ModDB and ModStore read the mods PoE1's kernel also reads; every root
// database carries it.
var kernelRules = newRules()

func newRules() *moddb.Rules {
	return &moddb.Rules{
		BaseFlags:                  gemBaseFlags,
		LimitNeg:                   true,
		SumWholeSource:             true,
		ReplaceIgnoresReplaced:     true,
		ScaleLevelFloor:            true,
		ManaUnreservedNegativeDead: true,
		GameIDFromGemName:          getGameIDFromGemName,
		GemIsType: func(gem any, keyword string) bool {
			g, ok := gem.(GemData)
			return ok && gemIsType(g, keyword, false)
		},
		HighPrecisionMods: highPrecisionMods,
	}
}

// validateGemLevel is calcLib.validateGemLevel: the level a gem instance or effect keeps, limited to the
// levels its granted effect has. next(levels) is the lowest level here.
//
//pob2:src/Modules/CalcTools.lua:60-76
func validateGemLevel(grantedEffect *poe2skills.GrantedEffect, gemData *poe2skills.Gem, level int) int {
	if grantedEffect == nil {
		grantedEffect = gemData.GrantedEffect
	}
	if grantedEffect.Levels[level] == nil {
		level = max(1, level)
		if n := levelCount(grantedEffect); n > 0 {
			level = min(n, level)
		}
	}
	if grantedEffect.Levels[level] == nil && gemData != nil && gemData.Data.NaturalMaxLevel != 0 {
		level = int(gemData.Data.NaturalMaxLevel)
	}
	if grantedEffect.Levels[level] == nil {
		level = lowestLevel(grantedEffect)
	}
	return level
}

// doesTypeExpressionMatch evaluates a skill type postfix expression.
//
//pob2:src/Modules/CalcTools.lua:78-105
func doesTypeExpressionMatch(checkTypes []string, skillTypes, minionTypes map[string]bool) bool {
	stack := make([]bool, 0, len(checkTypes))
	for _, skillType := range checkTypes {
		switch skillType {
		case "OR":
			other := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			stack[len(stack)-1] = stack[len(stack)-1] || other
		case "AND":
			other := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			stack[len(stack)-1] = stack[len(stack)-1] && other
		case "NOT":
			stack[len(stack)-1] = !stack[len(stack)-1]
		default:
			stack = append(stack, skillTypes[skillType] || minionTypes[skillType])
		}
	}
	for _, v := range stack {
		if v {
			return true
		}
	}
	return false
}

// canGrantedEffectSupportActiveSkill checks if the given support skill can support the given active skill.
//
//pob2:src/Modules/CalcTools.lua:107-133
func canGrantedEffectSupportActiveSkill(env *Environment, grantedEffect *poe2skills.GrantedEffect, activeSkill *ActiveSkill) bool {
	active := activeSkill.ActiveEffect
	if active.GrantedEffect.CannotBeSupported {
		return false
	}
	if grantedEffect.SupportGemsOnly && active.GemData == nil {
		return false
	}

	// Special case for things like Forbidden Shako or Hungry Loop with  for example Prismatic Burst and another compatible support
	if env.effectFromItem(grantedEffect) && grantedEffect.Support && (env.effectFromItem(active.GrantedEffect) || strings.HasPrefix(active.GrantedEffect.ModSource, "Item") || (active.SrcInstance != nil && active.SrcInstance.FromItem)) {
		return false
	}

	effectiveSkillTypes := activeSkill.SkillTypes
	var effectiveMinionTypes map[string]bool
	if activeSkill.SummonSkill != nil {
		effectiveSkillTypes = activeSkill.SummonSkill.SkillTypes
	}
	if !grantedEffect.IgnoreMinionTypes {
		effectiveMinionTypes = activeSkill.MinionSkillTypes
		if activeSkill.SummonSkill != nil {
			effectiveMinionTypes = activeSkill.SummonSkill.MinionSkillTypes
		}
	}

	// if the activeSkill is a Minion's skill like "Default Attack", use minion's skillTypes instead for exclusions
	// otherwise compare support to activeSkill directly
	if len(grantedEffect.ExcludeSkillTypes) > 0 && doesTypeExpressionMatch(grantedEffect.ExcludeSkillTypes, effectiveSkillTypes, nil) {
		return false
	}
	if grantedEffect.IsTrigger && activeSkill.Actor.Enemy.Player != activeSkill.Actor {
		return false
	}
	return len(grantedEffect.RequireSkillTypes) == 0 || doesTypeExpressionMatch(grantedEffect.RequireSkillTypes, effectiveSkillTypes, effectiveMinionTypes)
}

// getGemStatRequirement is the in-game formula.
//
//pob2:src/Modules/CalcTools.lua:151-158
func getGemStatRequirement(level, multi float64, isSupport bool) float64 {
	if multi == 0 || isSupport {
		return 0
	}
	req := round((5+(level-3)*1.7)*math.Pow(multi/100, 0.9)) + 4
	if req < 8 {
		return 0
	}
	return req
}
