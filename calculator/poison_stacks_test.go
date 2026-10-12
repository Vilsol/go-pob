package calculator

import (
	"math"
	"os"
	"testing"
)

// TestPoisonStacksIsPublished pins CalcOffence.lua:4596-4621 and :4740.
func TestPoisonStacksIsPublished(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}

	for _, tc := range []struct {
		build string
		arm   string
	}{
		{"0059-occultist-viper-strike.xml", "plain, high stacks"},
		{"0047-elementalist-manabond.xml", "plain, small stacks"},
		{"0334-elementalist-penance-brand-11.xml", "Condition:NonPoisonedOnly"},
		{"2271-ladder-ZeratorRules_4850.POOGABOOGAA.xml", "showAverage"},
		{"7501-ladder-karzakk_7077.Krzktrap.xml", "showAverage, trap"},
	} {
		t.Run(tc.build, func(t *testing.T) {
			rec, ok := oracle[tc.build]
			if !ok {
				t.Fatalf("REFUSING: %s has no record in the oracle blob", tc.build)
			}
			want, ok := rec.Output["PoisonStacks"]
			if !ok || want == 0 {
				t.Fatalf("REFUSING: PoB reports no nonzero PoisonStacks for %s", tc.build)
			}

			env := corpusEnvForLoadedData(t, tc.build)

			if wantDur, ok := rec.Output["PoisonDuration"]; ok && wantDur != 0 {
				gotDur := env.Player.Output["PoisonDuration"]
				if math.Abs(gotDur-wantDur) > math.Abs(wantDur)*oracleDumpTolerance {
					t.Fatalf("REFUSING: %s is contaminated — PoisonDuration is %v against PoB's "+
						"%v, and it multiplies the stacks expression directly",
						tc.build, gotDur, wantDur)
				}
			}

			if wantChance, ok := rec.Output["PoisonChance"]; ok && wantChance != 0 {
				gotChance := env.Player.Output["PoisonChance"]
				if math.Abs(gotChance-wantChance) > math.Abs(wantChance)*oracleDumpTolerance {
					t.Fatalf("REFUSING: %s is contaminated — PoisonChance is %v against PoB's %v, "+
						"which is the unported :4590-4593 local, not this block",
						tc.build, gotChance, wantChance)
				}
			}

			got, present := env.Player.Output["PoisonStacks"]
			if !present {
				t.Errorf("%s: we never write PoisonStacks at all; PoB reports %v", tc.build, want)
				return
			}
			if math.Abs(got-want) > math.Abs(want)*oracleDumpTolerance {
				t.Errorf("%s [%s]: PoisonStacks = %v, PoB reports %v (ratio %.4f)",
					tc.build, tc.arm, got, want, got/want)
			}
		})
	}
}

// TestTotalPoisonStacksIsAVestige pins the OTHER half of the same divergence.
func TestTotalPoisonStacksIsAVestige(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}

	checked := 0
	for build, rec := range oracle {
		v, ok := rec.Output["TotalPoisonStacks"]
		if !ok {
			continue
		}
		if v != 0 {
			t.Fatalf("REFUSING: PoB reports TotalPoisonStacks = %v for %s — upstream has grown a "+
				"producer and this test's premise is void", v, build)
			return
		}
		checked++
		if checked > 5 {
			break
		}

		env := corpusEnvForLoadedData(t, build)
		if got, present := env.Player.Output["TotalPoisonStacks"]; present && got != 0 {
			t.Errorf("%s: TotalPoisonStacks = %v where PoB reports 0; the pin has no producer "+
				"for this key and ours is a v2.18.1 leftover", build, got)
		}
	}
	if checked == 0 {
		t.Fatalf("REFUSING: no build in the oracle carries TotalPoisonStacks, so this test " +
			"asserts nothing. Either the key left PoB's dumps or the blob changed shape.")
	}
}
