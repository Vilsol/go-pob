package calculator

import (
	"math"
	"os"
	"testing"
)

// TestLeechZeroingRunsBeforeTheCompoundForm: PoB writes the leech rates TWICE, and the zeroing runs between the two writes.
func TestLeechZeroingRunsBeforeTheCompoundForm(t *testing.T) {
	t.Parallel()

	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	_, names := corpusNamesForLoadedData(t)
	immortalAmbition, nonInstant, control := 0, 0, 0
	for _, b := range names {
		env := corpusEnvForLoadedData(t, b)
		if env.Player.ModDB.Flag(nil, "ImmortalAmbition") {
			immortalAmbition++
		}
		if env.Player.ModDB.Flag(nil, "UnaffectedByNonInstantLifeLeech") {
			nonInstant++
		}
		if env.Player.Output["LifeLeechInstanceRate"] > 0 {
			control++
		}
	}

	if control == 0 {
		t.Fatal("REFUSING: no build has a nonzero LifeLeechInstanceRate, so this sweep is not " +
			"reaching the leech calculation and its zeroes mean nothing")
	}

	if immortalAmbition > 0 || nonInstant > 0 {
		t.Fatalf("the MEASURED ZERO recorded for this change no longer holds: ImmortalAmbition on "+
			"%d builds and UnaffectedByNonInstantLifeLeech on %d. The two-phase order now has an "+
			"observable population and needs a real oracle assertion instead of this census.",
			immortalAmbition, nonInstant)
	}
	t.Logf("ImmortalAmbition %d, UnaffectedByNonInstantLifeLeech %d, control %d of %d builds",
		immortalAmbition, nonInstant, control, len(names))
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
