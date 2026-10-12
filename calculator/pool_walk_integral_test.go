package calculator

import (
	"fmt"
	"math"
	"os"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestPoolWalkPoolsAreIntegral pins CalcDefence.lua:411-413.
func TestPoolWalkPoolsAreIntegral(t *testing.T) {
	t.Parallel()

	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	dir, names := corpusNamesForLoadedData(t)

	walked, stepped, violating := 0, 0, 0
	var firstExamples []string

	for _, name := range names {
		rawXML, err := os.ReadFile(dir + "/" + name)
		if err != nil {
			t.Fatalf("REFUSING: %v", err)
		}
		build, err := builds.ParseBuild(rawXML)
		if err != nil {
			t.Fatalf("REFUSING: %s did not parse: %v", name, err)
		}
		env, _, _, _ := InitEnv(build, corpusCacheForTest(t), OutputModeMain)
		env.TracePoolWalk = true
		PerformCalc(env)
		if len(env.PoolWalkTrace) == 0 {
			continue
		}
		walked++
		bad := ""
		for _, st := range env.PoolWalkTrace {
			stepped++
			for _, p := range []struct {
				name string
				got  float64
			}{
				{"Life", st.LifeOut},
				{"Mana", st.ManaOut},
				{"EnergyShield", st.EnergyShieldOut},
			} {
				if bad == "" && p.got != math.Floor(p.got) {
					bad = fmt.Sprintf("%s: %s after the drain = %v at hit %d", name, p.name, p.got, st.N)
				}
			}
		}
		if bad != "" {
			violating++
			if len(firstExamples) < 5 {
				firstExamples = append(firstExamples, bad)
			}
		}
	}

	fmt.Printf("=== POOL WALK INTEGRALITY === %d builds ran the walk, %d hits recorded, "+
		"%d builds carry a fractional pool\n", walked, stepped, violating)

	if walked == 0 || stepped == 0 {
		t.Fatalf("REFUSING: %d builds walked over %d hits — an assertion over an empty trace "+
			"asserts nothing", walked, stepped)
	}
	for _, ex := range firstExamples {
		t.Errorf("fractional pool survives the drain — PoB floors it at CalcDefence.lua:411-413: %s", ex)
	}
	if violating > 0 {
		t.Errorf("%d of %d walked builds carry a fractional pool into the next hit (first %d shown)",
			violating, walked, len(firstExamples))
	}
}
