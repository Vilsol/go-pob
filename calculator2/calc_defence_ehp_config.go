package calculator2

import (
	"github.com/Vilsol/go-pob/data/poe2"
)

// enemyConfigPlaceholders is env.configPlaceholder as far as buildDefenceEstimations reads it: the
// options' defaultPlaceholderState, the saved Placeholder elements, then the enemyIsBoss option's
// SetPlaceholder calls. The resist, armour and evasion placeholders it also sets are not read here and
// are not ported, and neither is presetBossSkills.
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
	return placeholder
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
