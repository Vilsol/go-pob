package calculator

import (
	"math"
	"os"
	"testing"
)

// if Flag("Condition:WarcryMaxHit") then
//
//	NewMod("AreaOfEffect", "MORE", env.modDB:Sum("BASE", nil, "SeismicMoreAoE"), "Max Seismic Exert AoE")
//
// else
//
//	NewMod("AreaOfEffect", "MORE", m_floor(env.modDB:Sum("BASE", nil, "SeismicMoreAoE") / 100 * SeismicUpTimeRatio), "Avg Seismic Exert AoE")
//
// end
var seismicNaNBuilds = []string{
	"0130-slayer-sunder.xml",
	"0293-slayer-vaal-ground-slam-of-earthshaking.xml",
	"0535-slayer-vaal-ground-slam-of-earthshaking-2.xml",
	"0577-slayer-vaal-ground-slam-of-earthshaking-3.xml",
	"0612-slayer-sunder-2.xml",
	"0672-slayer-ground-slam-of-earthshaking.xml",
	"0992-slayer-vaal-ground-slam-of-earthshaking-4.xml",
}

func TestSeismicCryAoEIsFinite(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}

	exercised := 0
	for _, build := range seismicNaNBuilds {
		t.Run(build, func(t *testing.T) {
			rec, ok := oracle[build]
			if !ok {
				t.Fatalf("REFUSING: %s has no record in the oracle blob", build)
			}
			env := corpusEnvForLoadedData(t, build)

			if _, ok := env.Player.Output["SeismicUpTimeRatio"]; !ok {
				t.Fatalf("premise broken: %s no longer computes SeismicUpTimeRatio, so it does "+
					"not reach the Seismic Cry AoE block at all", build)
			}
			exercised++

			for _, k := range []string{"AreaOfEffectMod", "AreaOfEffectRadius", "AreaOfEffectRadiusMetres"} {
				got, present := env.Player.Output[k]
				if !present {
					continue
				}
				if math.IsNaN(got) || math.IsInf(got, 0) {
					t.Errorf("%s: %s = %v. The 3.20 block divided aoeImpact by SeismicExertsCount, "+
						"which is 0/0 when the build has no exerts", build, k, got)
				}
				if want, ok := rec.Output[k]; ok && (math.IsNaN(want) || math.IsInf(want, 0)) {
					t.Fatalf("REFUSING: PoB itself reports %s = %v for %s; this fixture cannot "+
						"show that OUR value became finite", k, want, build)
				}
			}
		})
	}

	if exercised == 0 {
		t.Fatal("REFUSING: not one fixture reached the Seismic Cry block; this test no longer " +
			"covers the mechanism it was written for")
	}
}

// TestAreaOfEffectRadiusMetresIsPublished pins CalcOffence.lua:362.
func TestAreaOfEffectRadiusMetresIsPublished(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}

	for _, build := range []string{
		"0006-elementalist-storm-burst.xml",
		"0007-elementalist-storm-burst-of-repulsion.xml",
		"0011-saboteur-cyclone.xml",
	} {
		t.Run(build, func(t *testing.T) {
			rec, ok := oracle[build]
			if !ok {
				t.Fatalf("REFUSING: %s has no record in the oracle blob", build)
			}
			want, ok := rec.Output["AreaOfEffectRadiusMetres"]
			if !ok || want == 0 {
				t.Fatalf("REFUSING: PoB reports no nonzero AreaOfEffectRadiusMetres for %s; an "+
					"unwritten key agrees with 0", build)
			}

			env := corpusEnvForLoadedData(t, build)
			got, present := env.Player.Output["AreaOfEffectRadiusMetres"]
			if !present {
				t.Errorf("%s: we never write AreaOfEffectRadiusMetres at all; PoB reports %v", build, want)
				return
			}
			if math.IsNaN(got) || math.IsInf(got, 0) {
				t.Errorf("%s: AreaOfEffectRadiusMetres = %v", build, got)
				return
			}
			if r := env.Player.Output["AreaOfEffectRadius"]; math.Abs(got-r/10) > 1e-9 {
				t.Errorf("%s: AreaOfEffectRadiusMetres = %v but AreaOfEffectRadius/10 = %v", build, got, r/10)
			}
			if math.Abs(got-want) > math.Abs(want)*oracleDumpTolerance {
				t.Errorf("%s: AreaOfEffectRadiusMetres = %v, PoB reports %v", build, got, want)
			}
		})
	}
}

// TestAreaOfEffectRadiusSecondaryMetresIsPublished pins CalcOffence.lua:372 and :446, the secondary and tertiary counterparts of the primary metres key above.
func TestAreaOfEffectRadiusSecondaryMetresIsPublished(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}

	for _, build := range []string{
		"0006-elementalist-storm-burst.xml",
		"0007-elementalist-storm-burst-of-repulsion.xml",
		"0010-elementalist-storm-burst-2.xml",
	} {
		t.Run(build, func(t *testing.T) {
			rec, ok := oracle[build]
			if !ok {
				t.Fatalf("REFUSING: %s has no record in the oracle blob", build)
			}
			want, ok := rec.Output["AreaOfEffectRadiusSecondaryMetres"]
			if !ok || want == 0 {
				t.Fatalf("REFUSING: PoB reports no nonzero AreaOfEffectRadiusSecondaryMetres for "+
					"%s; an unwritten key agrees with 0", build)
			}
			env := corpusEnvForLoadedData(t, build)
			got, present := env.Player.Output["AreaOfEffectRadiusSecondaryMetres"]
			if !present {
				t.Errorf("%s: we never write AreaOfEffectRadiusSecondaryMetres at all; PoB reports %v",
					build, want)
				return
			}
			if r := env.Player.Output["AreaOfEffectRadiusSecondary"]; math.Abs(got-r/10) > 1e-9 {
				t.Errorf("%s: metres = %v but AreaOfEffectRadiusSecondary/10 = %v", build, got, r/10)
			}
			if math.Abs(got-want) > math.Abs(want)*oracleDumpTolerance {
				t.Errorf("%s: AreaOfEffectRadiusSecondaryMetres = %v, PoB reports %v", build, got, want)
			}
		})
	}
}
