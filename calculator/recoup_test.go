package calculator

import (
	"math"
	"os"
	"testing"
)

var recoupBuilds = []string{
	"0006-elementalist-storm-burst.xml",
	"0011-saboteur-cyclone.xml",
	"0018-berserker-heavy-strike.xml",
}

func TestRecoupRatesArePublished(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}

	sawSplit := false
	for _, build := range recoupBuilds {
		t.Run(build, func(t *testing.T) {
			rec, ok := oracle[build]
			if !ok {
				t.Fatalf("REFUSING: %s has no record in the oracle blob", build)
			}
			want, ok := rec.Output["anyRecoup"]
			if !ok || want == 0 {
				t.Fatalf("REFUSING: PoB reports no nonzero anyRecoup for %s; an unwritten key "+
					"agrees with 0 and the case cannot discriminate", build)
			}
			if rec.Output["LifeRecoup"] != want {
				sawSplit = true
			}

			env := corpusEnvForLoadedData(t, build)
			for _, k := range []string{"anyRecoup", "LifeRecoup", "ManaRecoup", "EnergyShieldRecoup"} {
				w, present := rec.Output[k]
				if !present {
					continue
				}
				got, ourPresent := env.Player.Output[k]
				if !ourPresent {
					t.Errorf("%s: we never write %s at all; PoB reports %v", build, k, w)
					continue
				}
				if math.Abs(got-w) > math.Abs(w)*oracleDumpTolerance {
					t.Errorf("%s: %s = %v, PoB reports %v", build, k, got, w)
				}
			}
		})
	}
	if !sawSplit {
		t.Fatal("REFUSING: every fixture has anyRecoup == LifeRecoup, so a port that aliased the " +
			"two would pass; the set has lost its discriminating case")
	}
}

// TestRecoupModsParse pins the producers directly.
func TestRecoupModsParse(t *testing.T) {
	for _, c := range []struct {
		line  string
		names []string
	}{
		{"15% of Damage taken Recouped as Life", []string{"LifeRecoup"}},
		{"10% of Damage taken Recouped as Mana", []string{"ManaRecoup"}},
		{"10% of Damage taken Recouped as Energy Shield", []string{"EnergyShieldRecoup"}},
		{"25% of Damage taken Recouped as Life, Mana and Energy Shield",
			[]string{"LifeRecoup", "EnergyShieldRecoup", "ManaRecoup"}},
		{"20% of Physical Damage taken Recouped as Life", []string{"PhysicalLifeRecoup"}},
		{"20% of Fire Damage taken Recouped as Life", []string{"FireLifeRecoup"}},
	} {
		t.Run(c.line, func(t *testing.T) {
			e := ParseMod(c.line, false)
			if e.Extra != "" {
				t.Fatalf("%q left Extra=%q, so processModLine would DROP the line entirely", c.line, e.Extra)
			}
			if len(e.ModList) != len(c.names) {
				t.Fatalf("%q yielded %d mods, want %d", c.line, len(e.ModList), len(c.names))
			}
			got := map[string]bool{}
			for _, m := range e.ModList {
				got[m.Name()] = true
			}
			for _, n := range c.names {
				if !got[n] {
					t.Errorf("%q did not yield %q (got %v)", c.line, n, got)
				}
			}
		})
	}
}

var recoupRecoveryBuilds = []string{
	"0030-saboteur-cyclone-2.xml",
	"0166-occultist-unknown-3.xml",
	"0277-reliquarian-blade-blast-2.xml",
	"0471-elementalist-vortex.xml",
}

func TestRecoupRecoveryMatchesPoBWhereItsInputsDo(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}

	for _, build := range recoupRecoveryBuilds {
		t.Run(build, func(t *testing.T) {
			rec, ok := oracle[build]
			if !ok {
				t.Fatalf("REFUSING: %s has no record in the oracle blob", build)
			}
			env := corpusEnvForLoadedData(t, build)

			wPhys := rec.Output["PhysicalRecoupableDamageTaken"]
			if wPhys == 0 {
				t.Fatalf("REFUSING: PoB reports PhysicalRecoupableDamageTaken 0 for %s", build)
			}
			got, present := env.Player.Output["PhysicalRecoupableDamageTaken"]
			if !present {
				t.Fatalf("%s: PhysicalRecoupableDamageTaken is not written at all; the recoup "+
					"tracking in the max-hit loop is gone", build)
			}
			if math.Abs(got-wPhys) > math.Abs(wPhys)*oracleDumpTolerance {
				t.Skipf("input drifted: PhysicalRecoupableDamageTaken = %v, PoB %v -- the max-hit "+
					"loop no longer agrees here, so this fixture cannot isolate the recoup port", got, wPhys)
			}

			for _, k := range []string{
				"TotalLifeRecoupRecovery", "LifeRecoupRecoveryMax",
			} {
				want, present := rec.Output[k]
				if !present || want == 0 {
					continue
				}
				got, ourPresent := env.Player.Output[k]
				if !ourPresent {
					t.Errorf("%s: we never write %s at all; PoB reports %v", build, k, want)
					continue
				}
				if math.Abs(got-want) > math.Abs(want)*oracleDumpTolerance {
					t.Errorf("%s: %s = %v, PoB reports %v", build, k, got, want)
				}
			}

			recoupTime := 4.0
			if env.Player.ModDB.Flag(nil, "3SecondLifeRecoup") || env.Player.ModDB.Flag(nil, "3SecondRecoup") {
				recoupTime = 3
			}
			pseudoDur := env.Player.Output["PhysicalDamageMitigatedLifePseudoRecoupDuration"]
			if pseudoDur == 0 {
				pseudoDur = 4
			}
			wantMax := env.Player.Output["TotalLifeRecoupRecovery"]/recoupTime +
				env.Player.Output["TotalLifePseudoRecoup"]/pseudoDur
			if got := env.Player.Output["LifeRecoupRecoveryMax"]; math.Abs(got-wantMax) > 1e-6 {
				t.Errorf("%s: LifeRecoupRecoveryMax = %v but total/%v + pseudo/%v = %v",
					build, got, recoupTime, pseudoDur, wantMax)
			}

			ehp := env.Player.Output["EHPSurvivalTime"]
			wantAvg := env.Player.Output["TotalLifeRecoupRecovery"]/(ehp+recoupTime) +
				env.Player.Output["TotalLifePseudoRecoup"]/(ehp+pseudoDur)
			if got := env.Player.Output["LifeRecoupRecoveryAvg"]; math.Abs(got-wantAvg) > 1e-6 {
				t.Errorf("%s: LifeRecoupRecoveryAvg = %v but total/(EHPSurvivalTime %v + %v) + "+
					"pseudo/(... + %v) = %v", build, got, ehp, recoupTime, pseudoDur, wantAvg)
			}
		})
	}
}
