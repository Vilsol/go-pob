package calculator

import (
	"math"
	"os"
	"strings"
	"testing"
)

// TestLeechZeroingRunsBeforeTheCompoundForm: PoB writes the leech rates TWICE, and the zeroing runs between the two writes.
func TestLeechZeroingRunsBeforeTheCompoundForm(t *testing.T) {
	t.Parallel()

	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}
	_, names := corpusNamesForLoadedData(t)
	population, nonzero, control := 0, 0, 0
	for _, b := range names {
		env := corpusEnvForLoadedData(t, b)
		if env.Player.Output["LifeLeechInstanceRate"] > 0 {
			control++
		}
		if !env.Player.ModDB.Flag(nil, "ImmortalAmbition") &&
			!env.Player.ModDB.Flag(nil, "UnaffectedByNonInstantLifeLeech") {
			continue
		}
		population++
		rec, ok := oracle[b]
		if !ok {
			t.Fatalf("REFUSING: %s has no record in the oracle blob", b)
		}
		for k, want := range rec.Output {
			if !strings.Contains(k, "Leech") {
				continue
			}
			if want != 0 {
				nonzero++
			}
			if got := env.Player.Output[k]; math.Abs(got-want) > math.Abs(want)*oracleDumpTolerance {
				t.Errorf("%s: %s = %v, PoB reports %v", b, k, got, want)
			}
		}
	}

	if control == 0 {
		t.Fatal("REFUSING: no build has a nonzero LifeLeechInstanceRate, so this sweep is not " +
			"reaching the leech calculation and its zeroes mean nothing")
	}
	if population > 0 && nonzero == 0 {
		t.Fatalf("REFUSING: %d builds carry the zeroing flags but PoB reports no nonzero leech "+
			"output on any of them, so agreement here grades nothing", population)
	}
	t.Logf("%d builds zero non-instant leech; %d nonzero PoB leech outputs graded; control %d of %d builds",
		population, nonzero, control, len(names))
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
