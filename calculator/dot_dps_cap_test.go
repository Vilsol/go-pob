package calculator

import (
	"math"
	"os"
	"testing"

	"github.com/Vilsol/go-pob/data"
)

// TestDotDpsCapClampsTheMinionActor pins CalcOffence.lua's `m_min(..., data.misc.DotDpsCap)` sites.
//
//pob:block:src/Modules/CalcOffence.lua:6251
func TestDotDpsCapClampsTheMinionActor(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}

	const build = "0001-necromancer-unknown.xml"
	rec, ok := oracle[build]
	if !ok {
		t.Fatalf("REFUSING: %s has no record in the oracle blob", build)
	}

	cases := []struct {
		table string
		key   string
		pob   map[string]float64
		ours  func(env *Environment) map[string]float64
	}{
		{"Minion.", "TotalDotDPS", rec.Minion, func(e *Environment) map[string]float64 { return e.Minion.Output }},
		{"Minion.", "TotalPoisonDPS", rec.Minion, func(e *Environment) map[string]float64 { return e.Minion.Output }},
		{"Minion.MainHand.", "TotalPoisonDPS", rec.MinionMain, func(e *Environment) map[string]float64 {
			return e.Minion.OutputTable[OutTableMainHand]
		}},
	}

	env := corpusEnvForLoadedData(t, build)
	if env.Minion == nil || env.Minion.Actor == nil {
		t.Fatal("REFUSING: we build no minion for this build, so no Minion. table is graded and " +
			"this fixture measures nothing")
	}

	graded := 0
	for _, c := range cases {
		want, ok := c.pob[c.key]
		if !ok {
			t.Fatalf("REFUSING: PoB publishes no %s%s for %s", c.table, c.key, build)
		}
		if want != data.DotDpsCap {
			t.Fatalf("REFUSING: PoB's %s%s is %v, not DotDpsCap %v — this fixture no longer sits "+
				"on the clamp and cannot distinguish a capped port from an uncapped one",
				c.table, c.key, want, float64(data.DotDpsCap))
		}
		got := c.ours(env)[c.key]
		if math.Abs(got-want) > math.Abs(want)*oracleDumpTolerance {
			t.Errorf("%s%s = %v, PoB reports %v (ratio %.5f) — DotDpsCap is %v",
				c.table, c.key, got, want, got/want, float64(data.DotDpsCap))
		}
		graded++
	}

	if graded != len(cases) {
		t.Fatalf("REFUSING: graded %d of %d keys", graded, len(cases))
	}
}
