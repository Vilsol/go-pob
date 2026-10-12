package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/pob"
)

// TestEnemyConfigPlaceholders pins enemyConfigPlaceholders against PoB's own configPlaceholder table.
func TestEnemyConfigPlaceholders(t *testing.T) {
	str := func(s string) *string { return &s }
	num := func(f float64) *float64 { return &f }

	build := func(level int, inputs []pob.Input, placeholders []pob.Input) *pob.PathOfBuilding {
		return &pob.PathOfBuilding{
			Build:  pob.Build{Level: level},
			Config: pob.Config{Inputs: inputs, Placeholders: placeholders},
		}
	}
	boss := func(v string) []pob.Input {
		return []pob.Input{{Name: "enemyIsBoss", String: str(v)}}
	}
	bossLevel := func(v string, lvl float64) []pob.Input {
		return []pob.Input{
			{Name: "enemyIsBoss", String: str(v)},
			{Name: "enemyLevel", Number: num(lvl)},
		}
	}

	for _, tc := range []struct {
		name    string
		build   *pob.PathOfBuilding
		want    map[string]float64
		absent  []string
		comment string
	}{
		{
			name:    "boss caps the enemy level at MaxEnemyLevel",
			comment: "0001-juggernaut-molten-strike, character level 98",
			build:   build(98, boss("Boss"), nil),
			want: map[string]float64{
				"enemyPhysicalDamage":  1176,
				"enemyLightningDamage": 1176,
				"enemyColdDamage":      1176,
				"enemyFireDamage":      1176,
				"enemyChaosDamage":     470,
				"enemySpeed":           700,
				"enemyCritChance":      5,
				"enemyCritDamage":      30,
				"enemyDamageRollRange": 70,
				"enemyLightningResist": 40,
				"enemyChaosResist":     25,
			},
			absent: []string{"enemyLevel", "enemyLightningPen"},
		},
		{
			name:    "boss below the cap uses the character level",
			comment: "5702-raider-poisonous-concoction-11, character level 65",
			build:   build(65, boss("Boss"), nil),
			want: map[string]float64{
				"enemyPhysicalDamage": 438,
				"enemyChaosDamage":    175,
			},
		},
		{
			name:    "an explicit enemyLevel input wins over the character level",
			comment: "1320-raider-frost-blades-5, enemyLevel=83 on a level-100 character",
			build:   build(100, bossLevel("Boss", 83), nil),
			want: map[string]float64{
				"enemyPhysicalDamage": 1068,
				"enemyChaosDamage":    427,
			},
		},
		{
			name:    "a non-positive enemyLevel input falls through to the character level",
			comment: "3497-deadeye-barrage-11, enemyLevel=-1 on a level-100 character",
			build:   build(100, bossLevel("Boss", -1), nil),
			want: map[string]float64{
				"enemyPhysicalDamage": 1176,
				"enemyChaosDamage":    470,
			},
		},
		{
			name:    "none derives its damage from the character level too",
			comment: "0007-ascendant-hexblast, character level 90",
			build:   build(90, boss("None"), nil),
			want: map[string]float64{
				"enemyPhysicalDamage": 1294,
				"enemySpeed":          700,
			},
			absent: []string{
				"enemyLightningDamage", "enemyColdDamage", "enemyFireDamage",
				"enemyChaosDamage", "enemyLevel", "enemyLightningResist",
			},
		},
		{
			name:    "none on a level-1 character",
			comment: "3600-inquisitor-determination",
			build:   build(1, boss("None"), nil),
			want:    map[string]float64{"enemyPhysicalDamage": 7},
		},
		{
			name:    "an enemyLevel input above the cap is clamped",
			comment: "2351-inquisitor-spark-57, enemyLevel=86",
			build:   build(100, bossLevel("None", 86), nil),
			want:    map[string]float64{"enemyPhysicalDamage": 1294},
		},
		{
			name:    "pinnacle pins the level at 84 and divides chaos by 2.5",
			comment: "0004-saboteur-lightning-trap, character level 97",
			build:   build(97, boss("Pinnacle"), nil),
			want: map[string]float64{
				"enemyPhysicalDamage":  2241,
				"enemyLightningDamage": 2241,
				"enemyChaosDamage":     896,
				"enemyLevel":           84,
				"enemyLightningPen":    3,
				"enemyLightningResist": 50,
				"enemyChaosResist":     30,
			},
		},
		{
			name:    "uber pins the level at 85 and divides chaos by 4",
			comment: "0020-deadeye-fireball, character level 100",
			build:   build(100, boss("Uber"), nil),
			want: map[string]float64{
				"enemyPhysicalDamage": 3044,
				"enemyChaosDamage":    761,
				"enemyLevel":          85,
				"enemyLightningPen":   8,
			},
		},
		{
			name:    "the boss handler overwrites a stale saved speed and crit chance",
			comment: "0483-elementalist-creeping-frost-2 saves enemySpeed=25000, enemyCritChance=0",
			build: build(93, boss("Pinnacle"), []pob.Input{
				{Name: "enemySpeed", Number: num(25000)},
				{Name: "enemyCritChance", Number: num(0)},
				{Name: "enemyCritDamage", Number: num(30)},
			}),
			want: map[string]float64{
				"enemySpeed":          700,
				"enemyCritChance":     5,
				"enemyCritDamage":     30,
				"enemyPhysicalDamage": 2241,
			},
		},
		{
			name: "a preset overwrites the speed and crit the boss handler just set",
			comment: "Maven Memory Game on Pinnacle; the preset handler runs after enemyIsBoss " +
				"because varList orders it that way",
			build: build(100, []pob.Input{
				{Name: "enemyIsBoss", String: str("Pinnacle")},
				{Name: "presetBossSkills", String: str("Maven Memory Game")},
			}, nil),
			want: map[string]float64{
				"enemySpeed": 7500, "enemyCritChance": 5, "enemyPhysicalDamage": 28923,
			},
			absent: []string{"enemyChaosDamage", "enemyLightningPen"},
		},
		{
			name: "isUber follows the saved enemyIsBoss, so Uber Shaper Slam takes UberSpeed",
			comment: "the pin's ConfigTab:Load ends in SetActiveConfigSet, which runs UpdateControls " +
				"before BuildModList (ConfigTab.lua:886, :1297-1299), so the dropdown selIndex is " +
				"already Uber; 1399-elementalist-eviscerate-10 (Uber + Sirus Meteor) confirms the " +
				"oracle takes the Uber branch",
			build: build(100, []pob.Input{
				{Name: "enemyIsBoss", String: str("Uber")},
				{Name: "presetBossSkills", String: str("Shaper Slam")},
			}, nil),
			want: map[string]float64{"enemySpeed": 1755, "enemyCritChance": 0},
		},
		{
			name:    "a preset name 3.20 no longer knows is a no-op",
			comment: "1397-trickster-hexblast-6 saves the renamed 'Uber Atziri Flameblast'; PoB errors on bossData.earlierUber and leaves every enemy placeholder untouched",
			build: build(100, []pob.Input{
				{Name: "enemyIsBoss", String: str("Pinnacle")},
				{Name: "presetBossSkills", String: str("Uber Atziri Flameblast")},
			}, nil),
			want: map[string]float64{
				"enemySpeed":          700,
				"enemyCritChance":     5,
				"enemyPhysicalDamage": 2241,
			},
		},
		{
			//	if varData.defaultIndex then
			name: "an enemyIsBoss default overwrites a saved placeholder",
			comment: "ConfigTab.lua:1237-1238 writes the defaultIndex value into input, so the " +
				"handler runs even with no <Input> of its own",
			build: build(93, nil, []pob.Input{
				{Name: "enemySpeed", Number: num(25000)},
			}),
			want: map[string]float64{
				"enemySpeed":           700,
				"enemyCritChance":      5,
				"enemyDamageRollRange": 70,
				"enemyPhysicalDamage":  2241,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := enemyConfigPlaceholders(tc.build)
			for key, want := range tc.want {
				if have, ok := got[key]; !ok {
					t.Errorf("%s: %s missing, want %g (%s)", tc.name, key, want, tc.comment)
				} else if have != want {
					t.Errorf("%s: %s = %g, want %g (%s)", tc.name, key, have, want, tc.comment)
				}
			}
			for _, key := range tc.absent {
				if have, ok := got[key]; ok {
					t.Errorf("%s: %s = %g, want it absent (%s)", tc.name, key, have, tc.comment)
				}
			}
		})
	}
}
