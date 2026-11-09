package calculator2

import (
	"math"
	"strings"

	"github.com/Vilsol/go-pob/data/poe2"
)

// enemyConfigPlaceholders is env.configPlaceholder as far as buildDefenceEstimations reads it: the
// options' defaultPlaceholderState, the saved Placeholder elements, then the enemyIsBoss option's
// SetPlaceholder calls, then presetBossSkills'. The resist, armour and evasion placeholders enemyIsBoss
// also sets are not read here and are not ported.
//
//pob2:block:src/Classes/ConfigTab.lua:716
//pob2:block:src/Classes/ConfigTab.lua:907-913
//pob2:block:src/Modules/ConfigOptions.lua:2279-2285
func enemyConfigPlaceholders(env *Environment) map[string]float64 {
	placeholder := map[string]float64{
		"enemySpeed":           700,
		"enemyCritChance":      5,
		"enemyCritDamage":      poe2.MonsterConstants["base_critical_hit_damage_bonus"],
		"enemyPhysicalDamage":  7,
		"enemyDamageRollRange": 70,
	}
	placeholders := env.Build.Config.Placeholders
	for _, set := range env.Build.Config.ConfigSets {
		if set.ID == activeConfigSetID(env.Build) {
			placeholders = set.Placeholders
		}
	}
	for _, p := range placeholders {
		if p.Number != nil {
			placeholder[p.Name] = *p.Number
		}
	}

	boss := "Pinnacle"
	if v, ok := env.ConfigInput["enemyIsBoss"]; ok && v.IsString {
		boss = enemyIsBossInput(v.String)
	}
	setDamage := func(defaultDamage, chaosDiv float64) {
		placeholder["enemyPhysicalDamage"] = defaultDamage
		placeholder["enemyLightningDamage"] = defaultDamage
		placeholder["enemyColdDamage"] = defaultDamage
		placeholder["enemyFireDamage"] = defaultDamage
		placeholder["enemyChaosDamage"] = round(defaultDamage / chaosDiv)
	}
	setPen := func(pen float64, set bool) {
		for _, name := range []string{"enemyLightningPen", "enemyColdPen", "enemyFirePen"} {
			if set {
				placeholder[name] = pen
			} else {
				delete(placeholder, name)
			}
		}
	}

	//pob2:block:src/Modules/ConfigOptions.lua:1984-1986
	placeholder["enemySpeed"] = 700
	placeholder["enemyCritChance"] = 5
	placeholder["enemyCritDamage"] = poe2.MonsterConstants["base_critical_hit_damage_bonus"]
	switch boss {
	case "None":
		//pob2:block:src/Modules/ConfigOptions.lua:1994-2012
		defaultLevel := configEnemyLevel(env)
		setDamage(round(monsterDamage(defaultLevel)*1.5*miscConstant("normalEnemyDPSMult")), 2.5)
		delete(placeholder, "enemyPhysicalOverwhelm")
		setPen(0, false)
	case "Boss":
		//pob2:block:src/Modules/ConfigOptions.lua:2035-2053
		defaultLevel := configEnemyLevel(env)
		setDamage(round(monsterDamage(defaultLevel)*1.5*miscConstant("stdBossDPSMult")), 2.5)
		delete(placeholder, "enemyPhysicalOverwhelm")
		setPen(0, false)
	case "Pinnacle":
		//pob2:block:src/Modules/ConfigOptions.lua:2077-2093
		defaultLevel := max(configEnemyLevel(env), 82)
		setDamage(round(monsterDamage(defaultLevel)*1.5*miscConstant("pinnacleBossDPSMult")), 2.5)
		setPen(miscConstant("pinnacleBossPen"), true)
	case "Uber":
		//pob2:block:src/Modules/ConfigOptions.lua:2118-2134
		defaultLevel := max(configEnemyLevel(env), 82)
		setDamage(round(monsterDamage(defaultLevel)*1.5*miscConstant("uberBossDPSMult")), 4)
		setPen(miscConstant("uberBossPen"), true)
	}
	presetBossSkillPlaceholders(env, placeholder)
	return placeholder
}

// presetBossSkill is data.bossSkills[val] for the presetBossSkills input, with ConfigTab:Load's
// pre-3.20 "Uber " prefix dropped. A name data.bossSkills lacks errors in PoB before any write.
//
//pob2:block:src/Classes/ConfigTab.lua:896-897
//pob2:block:src/Modules/ConfigOptions.lua:2190-2195
func presetBossSkill(env *Environment) (poe2.BossSkill, string, bool, bool) {
	v, ok := env.ConfigInput["presetBossSkills"]
	if !ok || !v.IsString {
		return poe2.BossSkill{}, "", false, false
	}
	val := strings.TrimPrefix(v.String, "Uber ")
	bossData, ok := poe2.BossSkills[val]
	if !ok {
		return poe2.BossSkill{}, "", false, false
	}
	boss := "Pinnacle"
	if b, ok := env.ConfigInput["enemyIsBoss"]; ok && b.IsString {
		boss = enemyIsBossInput(b.String)
	}
	isUber := boss == "Uber"
	if bossData.EarlierUber && boss == "Pinnacle" {
		isUber = true
	}
	return bossData, val, isUber, true
}

// presetBossSkillPlaceholders is presetBossSkills' SetPlaceholder half.
//
//pob2:block:src/Modules/ConfigOptions.lua:2196-2241
func presetBossSkillPlaceholders(env *Environment, placeholder map[string]float64) {
	bossData, _, isUber, ok := presetBossSkill(env)
	if !ok {
		return
	}
	for _, name := range []string{"enemyPhysicalDamage", "enemyLightningDamage", "enemyColdDamage", "enemyFireDamage", "enemyChaosDamage"} {
		delete(placeholder, name)
	}

	rollRangeMult := math.Min(math.Max(configNumberOr(env, placeholder, "enemyDamageRollRange", 70), 0), 100)
	enemyLevel := configEnemyLevel(env)
	for damageType, damageMult := range bossData.DamageMultipliers {
		if isUber && bossData.UberDamageMultiplier.Set {
			placeholder["enemy"+damageType+"Damage"] = round(monsterDamage(enemyLevel) * (damageMult[0] + float64(rollRangeMult*damageMult[1])) * bossData.UberDamageMultiplier.Value)
		} else {
			placeholder["enemy"+damageType+"Damage"] = round(monsterDamage(enemyLevel) * (damageMult[0] + float64(rollRangeMult*damageMult[1])))
		}
	}

	for _, name := range []string{"enemyPhysicalOverwhelm", "enemyLightningPen", "enemyColdPen", "enemyFirePen"} {
		delete(placeholder, name)
	}
	setValue := func(name string, v poe2.BossSkillValue) {
		if v.Set {
			placeholder[name] = v.Value
		} else {
			delete(placeholder, name)
		}
	}
	for penType, pen := range bossData.DamagePenetrations {
		if uberPen, ok := bossData.UberDamagePenetrations[penType]; isUber && ok {
			setValue("enemy"+penType, uberPen)
		} else {
			setValue("enemy"+penType, pen)
		}
	}

	if isUber && bossData.UberSpeed.Set {
		placeholder["enemySpeed"] = bossData.UberSpeed.Value
	} else if bossData.Speed.Set {
		placeholder["enemySpeed"] = bossData.Speed.Value
	}
	if bossData.CritChance.Set {
		placeholder["enemyCritChance"] = bossData.CritChance.Value
	}
}

// monsterDamage is data.monsterDamageTable[level].
func monsterDamage(level int) float64 {
	return poe2.MonsterDamageTable[level-1]
}

// configInputNumber is tonumber(env.configInput[name]).
func configInputNumber(env *Environment, name string) (float64, bool) {
	v, ok := env.ConfigInput[name]
	if !ok || !v.IsNumber {
		return 0, false
	}
	return v.Number, true
}

// configNumberOr is `env.configInput[name] or env.configPlaceholder[name] or def`.
func configNumberOr(env *Environment, placeholder map[string]float64, name string, def float64) float64 {
	if v, ok := configInputNumber(env, name); ok {
		return v
	}
	if v, ok := placeholder[name]; ok {
		return v
	}
	return def
}
