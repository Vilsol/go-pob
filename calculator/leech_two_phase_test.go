package calculator

import (
	"math"
	"os"
	"strings"
	"sync"
	"testing"
)

var _ = registerCorpusCheck("TestLeechZeroingRunsBeforeTheCompoundForm", func() corpusCheck { return &leechTwoPhaseCheck{} })

type leechTwoPhaseCheck struct {
	mu                                   sync.Mutex
	builds, population, nonzero, control int
	log                                  corpusCheckLog
}

func (c *leechTwoPhaseCheck) visit(b string, _ oracleRecord, _ bool, env *Environment) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.builds++
	if env == nil {
		return
	}
	if env.Player.Output["LifeLeechInstanceRate"] > 0 {
		c.control++
	}
	if !env.Player.ModDB.Flag(nil, "ImmortalAmbition") &&
		!env.Player.ModDB.Flag(nil, "UnaffectedByNonInstantLifeLeech") {
		return
	}
	c.population++
	oracle, err := oracleForLoadedData()
	if err != nil {
		c.log.fatalf(b, "REFUSING: cannot load the oracle blob: %v", err)
		return
	}
	rec, ok := oracle[b]
	if !ok {
		c.log.fatalf(b, "REFUSING: %s has no record in the oracle blob", b)
		return
	}
	for k, want := range rec.Output {
		if !strings.Contains(k, "Leech") {
			continue
		}
		if want != 0 {
			c.nonzero++
		}
		if got := env.Player.Output[k]; math.Abs(got-want) > math.Abs(want)*oracleDumpTolerance {
			c.log.errorf(b, "%s: %s = %v, PoB reports %v", b, k, got, want)
		}
	}
}

// TestLeechZeroingRunsBeforeTheCompoundForm: PoB writes the leech rates TWICE, and the zeroing runs between the two writes.
func TestLeechZeroingRunsBeforeTheCompoundForm(t *testing.T) {
	t.Parallel()

	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	c := corpusWalkCheck[*leechTwoPhaseCheck](t)
	c.log.report(t)
	population, nonzero, control, names := c.population, c.nonzero, c.control, c.builds

	if control == 0 {
		t.Fatal("REFUSING: no build has a nonzero LifeLeechInstanceRate, so this sweep is not " +
			"reaching the leech calculation and its zeroes mean nothing")
	}
	if population > 0 && nonzero == 0 {
		t.Fatalf("REFUSING: %d builds carry the zeroing flags but PoB reports no nonzero leech "+
			"output on any of them, so agreement here grades nothing", population)
	}
	t.Logf("%d builds zero non-instant leech; %d nonzero PoB leech outputs graded; control %d of %d builds",
		population, nonzero, control, names)
}

// TestLeechPerHitStillMatchesTheOracle guards the restructure itself.
func TestLeechPerHitStillMatchesTheOracle(t *testing.T) {
	t.Parallel()

	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}
	checked := 0
	for _, build := range []string{
		"0017-deadeye-spectral-throw-of-materialising.xml",
		"0021-assassin-blade-flurry.xml",
		"0048-warden-frost-blades.xml",
	} {
		rec, ok := oracle[build]
		if !ok {
			t.Fatalf("REFUSING: %s has no record in the oracle blob", build)
		}
		env := corpusEnvForLoadedData(t, build)
		for _, k := range []string{"LifeLeechRate", "LifeLeechPerHit", "ManaLeechRate", "ManaLeechPerHit"} {
			want, present := rec.Output[k]
			if !present || want == 0 {
				continue
			}
			checked++
			if got := env.Player.Output[k]; math.Abs(got-want) > math.Abs(want)*oracleDumpTolerance {
				t.Errorf("%s: %s = %v, PoB reports %v", build, k, got, want)
			}
		}
	}
	if checked == 0 {
		t.Fatal("REFUSING: no fixture had a nonzero leech rate to compare")
	}
}
