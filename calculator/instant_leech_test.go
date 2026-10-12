package calculator

import (
	"math"
	"os"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

var instantLeechBuilds = []string{
	"0028-occultist-forbidden-rite-of-soul-sacrifice.xml",
	"0058-saboteur-cyclone-3.xml",
	"0062-occultist-crackling-lance.xml",
}

func TestInstantLeechIsAProportion(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}

	sawPartial := false
	for _, build := range instantLeechBuilds {
		t.Run(build, func(t *testing.T) {
			rec, ok := oracle[build]
			if !ok {
				t.Fatalf("REFUSING: %s has no record in the oracle blob", build)
			}
			env := corpusEnvForLoadedData(t, build)

			for _, k := range []string{
				"LifeLeechInstantProportion",
				"EnergyShieldLeechInstantProportion",
				"ManaLeechInstantProportion",
			} {
				want, present := rec.Output[k]
				if !present || want == 0 {
					continue
				}
				if want > 0 && want < 1 {
					sawPartial = true
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
		})
	}
	if !sawPartial {
		t.Fatal("REFUSING: no fixture has a proportion strictly between 0 and 1, so the whole set " +
			"would pass against the boolean model this change replaced")
	}
}

// TestInstantLeechModsAreBase pins the producer side: a regression to flag() would leave the consumer summing 0 and every proportion silently back at zero.
func TestInstantLeechModsAreBase(t *testing.T) {
	for _, c := range []struct {
		line string
		name string
	}{
		{"Life Leech from Hits with this Weapon is instant", "InstantLifeLeech"},
		{"Mana Leech from Hits with this Weapon is instant", "InstantManaLeech"},
		{"25% of Leech is instant", "InstantLifeLeech"},
		{"25% of Leech is instant", "InstantEnergyShieldLeech"},
		{"25% of Life Leech is instant", "InstantLifeLeech"},
	} {
		t.Run(c.line, func(t *testing.T) {
			e := ParseMod(c.line, false)
			if e.Extra != "" {
				t.Fatalf("%q left Extra=%q, so the line would be dropped", c.line, e.Extra)
			}
			found := false
			for _, m := range e.ModList {
				if m.Name() != c.name {
					continue
				}
				found = true
				if m.Type() != mod.TypeBase {
					t.Errorf("%q yields %s as %v; PoB makes it BASE 100 and the consumer Sums it",
						c.line, c.name, m.Type())
				}
			}
			if !found {
				t.Errorf("%q did not yield %q at all", c.line, c.name)
			}
		})
	}
}
