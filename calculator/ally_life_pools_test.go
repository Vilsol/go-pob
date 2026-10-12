package calculator

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestAllyLifePools covers the totem arm of PoB's ally-life pools.
func TestAllyLifePools(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	corpusDir, _ := corpusNamesForLoadedData(t)

	for _, tc := range []struct {
		name  string
		build string
		keys  []string
	}{
		{"nearest-totem", "0045-hierophant-spark-of-the-nova.xml",
			[]string{"TotalTotemLife", "TotemAllyDamageMitigation", "PhysicalTotalHitPool"}},
		{"vaal-rejuvenation-totem", "0016-gladiator-dual-strike-of-ambidexterity.xml",
			[]string{"TotalVaalRejuvenationTotemLife", "PhysicalTotalHitPool"}},
		{"companionship-mitigation", "0060-assassin-icicle-mine.xml",
			[]string{"MinionAllyDamageMitigation"}},
		{"companionship-life", "0270-warden-elemental-hit-of-the-spectrum.xml",
			[]string{"TotalMinionLife", "PhysicalTotalHitPool"}},
		{"stone-golem-main-minion", "1782-ladder-Gruber788_5683.Wefuckingworbing.xml",
			[]string{"TotalStoneGolemLife"}},
		{"no-ally-redirect-control", "0004-trickster-lightning-tendrils-of-eccentricity.xml",
			[]string{"PhysicalTotalHitPool"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec, ok := oracle[tc.build]
			if !ok {
				t.Fatalf("REFUSING: %s is not in the oracle blob", tc.build)
			}
			file, err := os.ReadFile(filepath.Join(corpusDir, tc.build))
			if err != nil {
				t.Fatalf("%v", err)
			}
			build, err := builds.ParseBuild(file)
			if err != nil {
				t.Fatalf("%v", err)
			}
			env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())

			if tc.name == "no-ally-redirect-control" {
				for _, ally := range allyLifePoolList {
					if ally.redirect == "" {
						continue
					}
					if got := env.Player.Output[ally.mitigation]; got != 0 {
						t.Errorf("%s: %s = %v on a build with no ally redirect, want 0",
							tc.build, ally.mitigation, got)
					}
				}
			}

			for _, key := range tc.keys {
				want := rec.Output[key]
				if want == 0 {
					t.Fatalf("REFUSING: %s publishes no %s; the fixture has stopped exercising it",
						tc.build, key)
				}
				got := env.Player.Output[key]
				if math.Abs(got-want) > math.Abs(want)*oracleDumpTolerance {
					t.Errorf("%s: %s = %v, want %v (PoB)", tc.build, key, got, want)
				}
			}
		})
	}
}
