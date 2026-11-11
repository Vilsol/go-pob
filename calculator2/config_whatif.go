package calculator2

import (
	"maps"
	"slices"

	"github.com/Vilsol/go-pob/pob2"
)

// enemySkillPlaceholders are the placeholders enemyConfigPlaceholders keeps for the defence estimations, which the
// enemyIsBoss and presetBossSkills options set and clear.
var enemySkillPlaceholders = []string{
	"enemySpeed", "enemyCritChance", "enemyCritDamage", "enemyDamageRollRange",
	"enemyPhysicalDamage", "enemyLightningDamage", "enemyColdDamage", "enemyFireDamage", "enemyChaosDamage",
	"enemyPhysicalOverwhelm", "enemyLightningPen", "enemyColdPen", "enemyFirePen",
}

var enemySizeRadius = map[string]float64{"Small": 2, "Medium": 3, "Large": 5, "Huge": 11}

// ConfigPlaceholders is each option control's placeholder after the calculation's BuildModList: the config set's
// placeholder table as the options left it, with the enemy level placeholder enemyIsBoss sets, and the enemy radius
// enemySizePreset shows on its control without storing it.
//
//pob2:block:src/Classes/ConfigTab.lua:1051-1052
//pob2:block:src/Modules/ConfigOptions.lua:780-794
//pob2:block:src/Modules/ConfigOptions.lua:1994-1995
//pob2:block:src/Modules/ConfigOptions.lua:2035-2036
//pob2:block:src/Modules/ConfigOptions.lua:2077-2078
//pob2:block:src/Modules/ConfigOptions.lua:2118-2119
func ConfigPlaceholders(env *Environment) map[string]float64 {
	out := maps.Clone(env.configPlaceholder)
	if out == nil {
		out = map[string]float64{}
	}
	enemy := enemyConfigPlaceholders(env)
	for _, name := range enemySkillPlaceholders {
		if v, ok := enemy[name]; ok {
			out[name] = v
		} else {
			delete(out, name)
		}
	}
	boss := "Pinnacle"
	if v, ok := env.ConfigInput["enemyIsBoss"]; ok && v.IsString {
		boss = enemyIsBossInput(v.String)
	}
	switch boss {
	case "None", "Boss":
		out["enemyLevel"] = float64(env.Build.Build.Level)
	case "Pinnacle", "Uber":
		out["enemyLevel"] = 82
	}
	size := "Medium"
	if v, ok := env.ConfigInput["enemySizePreset"]; ok && v.IsString {
		size = v.String
	}
	if r, ok := enemySizeRadius[size]; ok {
		out["enemyRadius"] = r
	}
	return out
}

// CalcConfig is an option tooltip's calcFunc() with input[var] set to value and the mod lists rebuilt around it.
//
//pob2:block:src/Classes/ConfigTab.lua:814-823
func (m *MiscCalculator) CalcConfig(name string, value ConfigValue) *Environment {
	main := *m.main
	main.ConfigInput = maps.Clone(m.main.ConfigInput)
	main.ConfigInput[name] = value
	return m.over(&main)
}

// CalcCustomModGroup is a custom modifier group's enable check tooltip: calcFunc() with the group toggled; nil for no
// such group.
//
//pob2:block:src/Classes/ConfigTab.lua:66-73
func (m *MiscCalculator) CalcCustomModGroup(i int) *Environment {
	build := *m.main.Build
	build.Config.ConfigSets = slices.Clone(build.Config.ConfigSets)
	blocks := slices.Clone(customModifierBlocks(&build))
	if i < 0 || i >= len(blocks) {
		return nil
	}
	enabled := blocks[i].Enabled != nil && !*blocks[i].Enabled
	blocks[i].Enabled = &enabled
	id := activeConfigSetID(&build)
	if k := slices.IndexFunc(build.Config.ConfigSets, func(s pob2.ConfigSet) bool { return s.ID == id }); k >= 0 {
		build.Config.ConfigSets[k].CustomModifierBlocks = blocks
	} else {
		build.Config.CustomModifierBlocks = blocks
	}
	main := *m.main
	main.Build = &build
	return m.over(&main)
}

// over is calcFunc() over the build as main holds it, the base pass's Full DPS use kept.
func (m *MiscCalculator) over(main *Environment) *Environment {
	return (&MiscCalculator{main: main, usedFullDPS: m.usedFullDPS}).calc(calcOverride{}, true)
}
