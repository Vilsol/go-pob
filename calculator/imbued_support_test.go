package calculator

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestImbuedSupportIsAddedToItsSlotsGroups pins CalcSetup.lua:1557-1565 together with the SkillsTab.lua:1437-1448 map it reads.
func TestImbuedSupportIsAddedToItsSlotsGroups(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	oracle, err := loadOutputOracle("../testdata/oracle-3.29/output-3.29.jsonl.gz")
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	corpusDir, _ := corpusNamesForLoadedData(t)

	run := func(t *testing.T, name string) *Environment {
		t.Helper()
		file, err := os.ReadFile(filepath.Join(corpusDir, name))
		if err != nil {
			t.Fatalf("%v", err)
		}
		build, err := builds.ParseBuild(file)
		if err != nil {
			t.Fatalf("%v", err)
		}
		return BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())
	}

	for _, tc := range []struct {
		build string
		want  string
	}{
		{"0114-occultist-icicle-mine-of-fanning.xml", "SupportPowerChargeOnCritical"},
		{"0553-slayer-cyclone-2.xml", "SupportChanceToPoison"},
	} {
		t.Run("mechanism/"+tc.build, func(t *testing.T) {
			env := run(t, tc.build)
			if env.Player.MainSkill == nil {
				t.Fatalf("%s: no main skill", tc.build)
			}
			found := false
			for _, effect := range env.Player.MainSkill.EffectList {
				if effect.GrantedEffect != nil && effect.GrantedEffect.ID() == tc.want {
					found = true
					break
				}
			}
			if !found {
				ids := make([]string, 0, len(env.Player.MainSkill.EffectList))
				for _, effect := range env.Player.MainSkill.EffectList {
					if effect.GrantedEffect != nil {
						ids = append(ids, effect.GrantedEffect.ID())
					}
				}
				t.Fatalf("%s: main skill does not carry %s; effects were %v", tc.build, tc.want, ids)
			}
		})
	}

	for _, tc := range []struct {
		build string
		stat  string
	}{
		{"0114-occultist-icicle-mine-of-fanning.xml", "AverageDamage"},
		{"0114-occultist-icicle-mine-of-fanning.xml", "ManaCost"},
		{"0553-slayer-cyclone-2.xml", "CritMultiplier"},
	} {
		t.Run("value/"+tc.build+"/"+tc.stat, func(t *testing.T) {
			rec, ok := oracle[tc.build]
			if !ok {
				t.Fatalf("REFUSING: %s is not in the oracle blob", tc.build)
			}
			want, ok := rec.Output[tc.stat]
			if !ok || want == 0 {
				t.Fatalf("REFUSING: %s publishes no %s; the fixture has moved", tc.build, tc.stat)
			}
			got := run(t, tc.build).Player.Output[tc.stat]
			if math.Abs(got-want) > math.Abs(want)*1e-9 {
				t.Errorf("%s: %s = %v, want %v (PoB)", tc.build, tc.stat, got, want)
			}
		})
	}
}
