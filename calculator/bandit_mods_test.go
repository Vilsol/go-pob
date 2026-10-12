package calculator

import (
	"math"
	"os"
	"sort"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

func banditSourcedMods(t *testing.T, env *Environment) map[string]float64 {
	t.Helper()
	got := map[string]float64{}
	for name, mods := range env.Player.ModDB.Mods {
		for _, m := range mods {
			if m.GetSource() != "Bandit" {
				continue
			}
			if m.Value().Type() != mod.ModValueMultiTypeFloat {
				t.Fatalf("Bandit-sourced %q is not a numeric mod (%v); this test cannot compare it",
					name, m.Value().Type())
			}
			got[name] += m.Value().Float()
		}
	}
	return got
}

func TestBanditGrantsExactlyPoBsMods(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	for _, c := range []struct {
		bandit string
		build  string
		builds int
		want   map[string]float64
	}{
		{"Alira", "0004-trickster-lightning-tendrils-of-eccentricity.xml", 222,
			map[string]float64{"ElementalResist": 15}},
		{"Kraityn", "0046-elementalist-arc.xml", 29,
			map[string]float64{"MovementSpeed": 8}},
		{"Oak", "0016-gladiator-dual-strike-of-ambidexterity.xml", 12,
			map[string]float64{"Life": 40}},
		{"None", "0001-necromancer-unknown.xml", 1563,
			map[string]float64{"ExtraPoints": 1}},
		{"Eramir", "0013-inquisitor-divine-ire.xml", 157,
			map[string]float64{"ExtraPoints": 1}},
	} {
		t.Run(c.bandit, func(t *testing.T) {
			env := corpusEnvForLoadedData(t, c.build)
			if got := env.Build.Build.Bandit; got != c.bandit && !(c.bandit == "None" && got == "") {
				t.Fatalf("premise broken: %s has bandit %q, not %q; this case no longer covers "+
					"that branch (population was %d builds)", c.build, got, c.bandit, c.builds)
			}

			got := banditSourcedMods(t, env)

			names := make([]string, 0, len(got))
			for n := range got {
				names = append(names, n)
			}
			sort.Strings(names)

			for n, want := range c.want {
				if v, ok := got[n]; !ok {
					t.Errorf("%s: no Bandit-sourced %q at all (PoB grants %v); Bandit mods present: %v",
						c.bandit, n, want, names)
				} else if math.Abs(v-want) > 1e-9 {
					t.Errorf("%s: Bandit %q = %v, PoB grants %v", c.bandit, n, v, want)
				}
			}
			for _, n := range names {
				if _, expected := c.want[n]; !expected {
					t.Errorf("%s: we grant a Bandit-sourced %q = %v that PoB does not grant at all "+
						"(CalcSetup.lua:549-558 grants only %v)", c.bandit, n, got[n], c.want)
				}
			}
		})
	}
}
