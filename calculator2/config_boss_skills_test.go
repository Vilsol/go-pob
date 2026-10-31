package calculator2

import (
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/pob2"
)

// The expected placeholders are PoB-PoE2's ConfigTab:BuildModList run under LuaJIT on the pin, with
// these enemyIsBoss / presetBossSkills inputs; a nil there is an absent key here.
func TestPresetBossSkillPlaceholders(t *testing.T) {
	for _, c := range []struct {
		boss, preset string
		level, roll  float64
		want         map[string]float64
	}{
		{boss: "Uber", preset: "Shaper Slam", level: 90, want: map[string]float64{"enemyPhysicalDamage": 9835, "enemySpeed": 1755, "enemyCritChance": 0}},
		{boss: "Pinnacle", preset: "Atziri Flameblast", level: 90, want: map[string]float64{"enemyFireDamage": 30733, "enemyFirePen": 10, "enemySpeed": 25000, "enemyCritChance": 0}},
		{boss: "Pinnacle", preset: "Maven Fireball", level: 90, roll: 100, want: map[string]float64{"enemyFireDamage": 7946, "enemySpeed": 3000, "enemyCritChance": 5}},
		{boss: "Uber", preset: "Maven Fireball", level: 90, want: map[string]float64{"enemyFireDamage": 14497, "enemyFirePen": 30, "enemySpeed": 3000, "enemyCritChance": 5}},
		{boss: "None", preset: "Sirus Meteor", level: 70, want: map[string]float64{"enemyPhysicalDamage": 15086, "enemySpeed": 1500, "enemyCritChance": 5}},
	} {
		build := &pob2.PathOfBuilding2{}
		build.Build.Level = int(c.level)
		env := &Environment{Build: build, ConfigInput: map[string]ConfigValue{
			"enemyIsBoss":      {String: c.boss, IsString: true},
			"presetBossSkills": {String: c.preset, IsString: true},
		}}
		if c.roll > 0 {
			env.ConfigInput["enemyDamageRollRange"] = ConfigValue{Number: c.roll, IsNumber: true}
		}
		p := enemyConfigPlaceholders(env)
		for _, name := range []string{"enemyPhysicalDamage", "enemyLightningDamage", "enemyColdDamage", "enemyFireDamage", "enemyChaosDamage",
			"enemyFirePen", "enemyColdPen", "enemyLightningPen", "enemyPhysicalOverwhelm", "enemySpeed", "enemyCritChance"} {
			got, ok := p[name]
			want, wantOK := c.want[name]
			if ok != wantOK || got != want {
				t.Errorf("%s %s: %s = %v (set %v), PoB %v (set %v)", c.boss, c.preset, name, got, ok, want, wantOK)
			}
		}
	}
}

// PoB-PoE2 under LuaJIT: Uber Shaper Slam writes enemyDamageType Melee, the uber additionalStats as
// enemy FLAG mods and BossSkillActive; Uber's armour and evasion placeholders scale by data.bossStats.
func TestPresetBossSkillsApply(t *testing.T) {
	env := configTestEnv(t, `<Input name="enemyIsBoss" string="Uber"/><Input name="presetBossSkills" string="Shaper Slam"/>`)
	testza.AssertEqual(t, ConfigValue{String: "Melee", IsString: true}, env.ConfigInput["enemyDamageType"])
	testza.AssertTrue(t, len(env.ModDB.Mods["BossSkillActive"]) == 1)
	for _, name := range []string{"CannotBeDodged", "CannotBeEvaded", "CannotBeSuppressed", "CannotBeBlocked"} {
		testza.AssertEqual(t, []configModRow{{Type: mod.TypeFlag, Source: "BossSkillAdditionalData"}}, configModRows(env.Enemy.ModDB.Mods[name]), name)
	}
	var armour, evasion []configModRow
	for _, row := range configModRows(env.Enemy.ModDB.Mods["Armour"]) {
		if row.Source == "Config" {
			armour = append(armour, row)
		}
	}
	for _, row := range configModRows(env.Enemy.ModDB.Mods["Evasion"]) {
		if row.Source == "Config" {
			evasion = append(evasion, row)
		}
	}
	testza.AssertEqual(t, []configModRow{{Type: mod.TypeBase, Value: 6719, Source: "Config"}}, armour)
	testza.AssertEqual(t, []configModRow{{Type: mod.TypeBase, Value: 1097, Source: "Config"}}, evasion)
}
