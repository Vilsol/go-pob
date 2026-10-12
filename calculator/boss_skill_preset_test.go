package calculator

import (
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
	"github.com/Vilsol/go-pob/pob"
)

// TestPresetBossSkillDamage pins the DAMAGE and PENETRATION half of applyPresetBossSkills.
func TestPresetBossSkillDamage(t *testing.T) {
	str := func(s string) *string { return &s }
	num := func(f float64) *float64 { return &f }

	build := func(level int, inputs ...pob.Input) *pob.PathOfBuilding {
		return &pob.PathOfBuilding{
			Build:  pob.Build{Level: level},
			Config: pob.Config{Inputs: inputs},
		}
	}
	preset := func(isBoss, name string) []pob.Input {
		return []pob.Input{
			{Name: "enemyIsBoss", String: str(isBoss)},
			{Name: "presetBossSkills", String: str(name)},
		}
	}

	for _, tc := range []struct {
		name    string
		comment string
		build   *pob.PathOfBuilding
		want    map[string]float64
		absent  []string
	}{
		{
			name:    "shaper slam on none derives from the character level",
			comment: "0042-occultist-lightning-tendrils, character level 99 -> enemy level 85",
			build:   build(99, preset("None", "Shaper Slam")...),
			want:    map[string]float64{"enemyPhysicalDamage": 14388, "enemySpeed": 3510, "enemyCritChance": 0},
			absent: []string{
				"enemyLightningDamage", "enemyColdDamage", "enemyFireDamage", "enemyChaosDamage",
				"enemyLightningPen", "enemyColdPen", "enemyFirePen",
			},
		},
		{
			name:    "an enemyLevel input feeds the preset damage too",
			comment: "3277-necromancer-cremation-10, enemyLevel=83 on a level-96 character",
			build: build(96,
				pob.Input{Name: "enemyIsBoss", String: str("None")},
				pob.Input{Name: "presetBossSkills", String: str("Shaper Slam")},
				pob.Input{Name: "enemyLevel", Number: num(83)},
			),
			want: map[string]float64{"enemyPhysicalDamage": 13063},
		},
		{
			name:    "pinnacle pins the preset damage at enemy level 84",
			comment: "0873-trickster-summon-skitterbots, character level 90",
			build:   build(90, preset("Pinnacle", "Shaper Slam")...),
			want:    map[string]float64{"enemyPhysicalDamage": 13710, "enemyLevel": 84},
			absent:  []string{"enemyChaosDamage", "enemyLightningPen", "enemyColdPen", "enemyFirePen"},
		},
		{
			name: "uber pins it at 85 and takes the uber multiplier",
			comment: "character level 98, Shaper Slam UberDamageMultiplier 1.6666666666667 (14388 x 5/3). " +
				"The pin's ConfigTab:Load runs UpdateControls before BuildModList (ConfigTab.lua:886, " +
				":1297-1299); 1399-elementalist-eviscerate-10 (Uber + Sirus Meteor) confirms it in the oracle",
			build:  build(98, preset("Uber", "Shaper Slam")...),
			want:   map[string]float64{"enemyPhysicalDamage": 23980, "enemyLevel": 85},
			absent: []string{"enemyLightningPen"},
		},
		{
			name:    "a preset penetration is restored after the pen clear",
			comment: "2985-raider-vaal-caustic-arrow-3, Shaper Ball on Pinnacle",
			build:   build(97, preset("Pinnacle", "Shaper Ball")...),
			want: map[string]float64{
				"enemyColdDamage": 12944,
				"enemyColdPen":    25,
				"enemySpeed":      1400,
			},
			absent: []string{
				"enemyPhysicalDamage", "enemyFireDamage", "enemyChaosDamage",
				"enemyLightningPen", "enemyFirePen",
			},
		},
		{
			name:    "shaper ball on none",
			comment: "1458-berserker-generals-cry-3, character level 99",
			build:   build(99, preset("None", "Shaper Ball")...),
			want:    map[string]float64{"enemyColdDamage": 13584, "enemyColdPen": 25},
		},
		{
			name:    "a physical-only preset restores physical and leaves the rest cleared",
			comment: "Maven Memory Game on Pinnacle, character level 96; enemy level pins to 84",
			build:   build(96, preset("Pinnacle", "Maven Memory Game")...),
			want: map[string]float64{
				"enemyPhysicalDamage": 28923,
				"enemySpeed":          7500,
			},
			absent: []string{
				"enemyLightningDamage", "enemyColdDamage", "enemyFireDamage", "enemyChaosDamage",
				"enemyLightningPen", "enemyColdPen", "enemyFirePen",
			},
		},
		{
			name:    "the same preset one enemy level higher",
			comment: "Maven Memory Game on Uber, character level 96; enemy level pins to 85; UberDamageMultiplier 1.0086206896552",
			build:   build(96, preset("Uber", "Maven Memory Game")...),
			want: map[string]float64{
				"enemyPhysicalDamage": 30615,
				"enemyLevel":          85,
			},
		},
		{
			name: "a preset name 3.20 no longer knows leaves the enemyIsBoss values standing",
			comment: "1397-trickster-hexblast-6 saves 'Uber Atziri Flameblast', renamed to " +
				"'Atziri Flameblast' in 3.20; data.bossSkills has no such key so PoB errors on " +
				"the first field access and touches nothing",
			build: build(97, preset("Pinnacle", "Uber Atziri Flameblast")...),
			want: map[string]float64{
				"enemyPhysicalDamage":  2241,
				"enemyLightningDamage": 2241,
				"enemyChaosDamage":     896,
				"enemyLightningPen":    3,
				"enemyColdPen":         3,
				"enemyFirePen":         3,
				"enemySpeed":           700,
				"enemyCritChance":      5,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := enemyConfigPlaceholders(tc.build)
			for key, want := range tc.want {
				if have, ok := got[key]; !ok {
					t.Errorf("%s: missing (want %g) — %s", key, want, tc.comment)
				} else if have != want {
					t.Errorf("%s = %g, want %g — %s", key, have, want, tc.comment)
				}
			}
			for _, key := range tc.absent {
				if have, ok := got[key]; ok {
					t.Errorf("%s = %g, want ABSENT — %s", key, have, tc.comment)
				}
			}
		})
	}
}

// TestPresetBossSkillMods pins the MOD half of ConfigOptions.lua:2246's apply.
func TestPresetBossSkillMods(t *testing.T) {
	apply, dispatched := configurationsWithBuild["presetBossSkills"]
	testza.AssertTrue(t, dispatched, "presetBossSkills has no `configurationsWithBuild` handler")
	ordered := false
	for _, name := range configOrder {
		if name == "presetBossSkills" {
			ordered = true
			break
		}
	}
	testza.AssertTrue(t, ordered, "presetBossSkills is not in configOrder")

	flagSet := func(list *moddb.ModList, name string) bool {
		for _, m := range list.Mods() {
			if m.Name() == name && m.Type() == mod.TypeFlag {
				return true
			}
		}
		return false
	}
	baseValue := func(list *moddb.ModList, name string) (float64, bool) {
		for _, m := range list.Mods() {
			if m.Name() == name && m.Type() == mod.TypeBase {
				return m.Value().Float(), true
			}
		}
		return 0, false
	}

	for _, tc := range []struct {
		name        string
		preset      string
		active      bool
		wantEnemy   map[string]float64
		wantFlags   []string
		wantNoFlags []string
	}{
		{
			name:   "sirus meteor carries its base conversion stats",
			preset: "Sirus Meteor",
			active: true,
			wantEnemy: map[string]float64{
				"PhysicalDamageSkillConvertToChaos":     25,
				"PhysicalDamageSkillConvertToFire":      25,
				"PhysicalDamageSkillConvertToLightning": 25,
			},
		},
		{
			name:        "exarch ball carries its base flags",
			preset:      "Exarch Ball",
			active:      true,
			wantFlags:   []string{"CannotBeBlocked", "CannotBeDodged", "CannotBeSuppressed"},
			wantNoFlags: []string{"CannotBeEvaded"},
		},
		{
			name:        "shaper slam is uber-only and contributes no enemy mods",
			preset:      "Shaper Slam",
			active:      true,
			wantNoFlags: []string{"CannotBeBlocked", "CannotBeDodged", "CannotBeEvaded", "CannotBeSuppressed"},
		},
		{
			name:        "a preset with no additionalStats contributes none",
			preset:      "Shaper Beam",
			active:      true,
			wantNoFlags: []string{"CannotBeBlocked", "CannotBeDodged", "CannotBeSuppressed"},
		},
		{
			name:        "None sets nothing at all",
			preset:      "None",
			active:      false,
			wantNoFlags: []string{"CannotBeBlocked"},
		},
		{
			name:        "a preset upstream no longer knows sets nothing at all",
			preset:      "Uber Atziri Flameblast",
			active:      false,
			wantNoFlags: []string{"CannotBeBlocked"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			player, enemy := moddb.NewModList(), moddb.NewModList()
			apply(tc.preset, player, enemy, &pob.PathOfBuilding{})

			testza.AssertEqual(t, tc.active, flagSet(player, "BossSkillActive"),
				"BossSkillActive for preset %q", tc.preset)

			for name, want := range tc.wantEnemy {
				got, ok := baseValue(enemy, name)
				testza.AssertTrue(t, ok, "enemy %s absent for preset %q", name, tc.preset)
				testza.AssertEqual(t, want, got, "enemy %s for preset %q", name, tc.preset)
			}
			for _, name := range tc.wantFlags {
				testza.AssertTrue(t, flagSet(enemy, name),
					"enemy flag %s not set for preset %q", name, tc.preset)
			}
			for _, name := range tc.wantNoFlags {
				testza.AssertFalse(t, flagSet(enemy, name),
					"enemy flag %s set for preset %q, which does not declare it", name, tc.preset)
			}
		})
	}
}
