package calculator

import (
	"math"
	"os"
	"sync"
	"testing"

	"github.com/Vilsol/go-pob/data"
)

// TestImpaleDurationModOutput pins CalcOffence.lua:1605-1606.
func TestImpaleDurationModOutput(t *testing.T) {
	t.Parallel()

	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	for _, tc := range []struct {
		build string
		want  float64
	}{
		{"0001-necromancer-unknown.xml", 1},
		{"0070-champion-frostblink.xml", 1.5},
	} {
		t.Run(tc.build, func(t *testing.T) {
			env := corpusEnvForLoadedData(t, tc.build)
			got, ok := env.Player.Output["ImpaleDurationMod"]
			if !ok {
				t.Fatal("ImpaleDurationMod ABSENT; PoB publishes it on every corpus build")
			}
			if math.Abs(got-tc.want) > 0.0005 {
				t.Errorf("ImpaleDurationMod = %v, want %v (from the 3.29 oracle blob)", got, tc.want)
			}
		})
	}
}

// TestImpaleDurationMatchesLuaBitsExactly: CalcOffence.lua:1607-1608.
//
//	local ImpaleDurationBase = data.characterConstants["impaled_debuff_base_duration_ms"] / 1000
//	output.ImpaleDuration = m_ceil(ImpaleDurationBase * output.ImpaleDurationMod * data.misc.ServerTickRate * debuffDurationMult) / data.misc.ServerTickRate
func TestImpaleDurationMatchesLuaBitsExactly(t *testing.T) {
	t.Parallel()

	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}
	for _, build := range []string{
		"0004-trickster-lightning-tendrils-of-eccentricity.xml",
		"0005-occultist-vaal-double-strike.xml",
		"0006-elementalist-storm-burst.xml",
	} {
		t.Run(build, func(t *testing.T) {
			rec, ok := oracle[build]
			if !ok {
				t.Fatalf("REFUSING: %s has no record in the oracle blob", build)
			}
			want, has := rec.Output["ImpaleDuration"]
			if !has || want == 0 {
				t.Fatalf("REFUSING: PoB reports no nonzero ImpaleDuration for %s; an unwritten "+
					"key agrees with 0 and the case cannot discriminate", build)
			}
			env := corpusEnvForLoadedData(t, build)
			got := env.Player.Output["ImpaleDuration"]
			if got != want {
				t.Errorf("%s: ImpaleDuration = %.17g, PoB reports %.17g (delta %.17g, one tick is %.17g)",
					build, got, want, got-want, 1/data.ServerTickRate)
			}
		})
	}
}

// TestImpaleDurationBaseIsARuntimeDivision pins the Go-constant-folding trap directly.
func TestImpaleDurationBaseIsARuntimeDivision(t *testing.T) {
	t.Parallel()

	const folded = data.ImpaledDebuffBaseDurationMs / 1000
	runtimeDivided := float64(data.ImpaledDebuffBaseDurationMs) / 1000
	if float64(folded) != runtimeDivided {
		t.Fatalf("ImpaledDebuffBaseDurationMs/1000 differs between constant folding (%.17g) and "+
			"runtime division (%.17g); the production expression must use the runtime form and "+
			"this constant now needs a LuaJIT-bit pin of its own",
			float64(folded), runtimeDivided)
	}
	if runtimeDivided != 8 {
		t.Fatalf("impale duration base is %v, not 8 -- PoB's impaled_debuff_base_duration_ms "+
			"moved and every assertion in this file was written against 8000ms", runtimeDivided)
	}
}

var _ = registerCorpusCheck("TestImpaleDurationIsExactOnMostBuilds", func() corpusCheck { return &impaleDurationCheck{} })

type impaleDurationCheck struct {
	mu          sync.Mutex
	seen, exact int
}

func (c *impaleDurationCheck) visit(_ string, rec oracleRecord, _ bool, env *Environment) {
	want, has := rec.Output["ImpaleDuration"]
	if !has || want == 0 || env == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.seen++
	if env.Player.Output["ImpaleDuration"] == want {
		c.exact++
	}
}

// TestImpaleDurationIsExactOnMostBuilds is the evidence shape the operator asked for: landing on PoB's value for builds the gate never flagged.
func TestImpaleDurationIsExactOnMostBuilds(t *testing.T) {
	t.Parallel()

	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	c := corpusWalkCheck[*impaleDurationCheck](t)
	seen, exact := c.seen, c.exact
	if seen == 0 {
		t.Fatal("REFUSING: no build reports a nonzero ImpaleDuration, so this sweep measured nothing")
	}
	if exact*10 < seen*9 {
		t.Errorf("only %d of %d builds match PoB's ImpaleDuration EXACTLY (<90%%); the port was "+
			"at 1892/1977 when written, so something upstream has moved", exact, seen)
	}
	t.Logf("ImpaleDuration exact on %d of %d builds (%.1f%%)", exact, seen, 100*float64(exact)/float64(seen))
}
