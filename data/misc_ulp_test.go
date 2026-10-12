package data

import (
	"math"
	"testing"
)

// TestULPClassIsReachableAndNecessary pins the one measured instance the ULP class exists for.
func TestULPClassIsReachableAndNecessary(t *testing.T) {
	const pobServerTickRate = 30.303030303030301

	const folded = 1 / 0.033

	if got := ulpsApart(folded, pobServerTickRate); got != 1 {
		t.Fatalf("REFUSING: the ServerTickRate constant-folding gap is %d ULP, expected 1.\n"+
			"    goconst=%.17g (bits %016x) pob=%.17g (bits %016x)\n"+
			"    Either the reference moved or ulpsApart is wrong — and if this is 0, the ULP class\n"+
			"    has no reachable member and the whole classification is dead code.",
			got, float64(folded), math.Float64bits(folded), pobServerTickRate, math.Float64bits(pobServerTickRate))
	}

	if ulpsApart(folded, pobServerTickRate) > ulpClassLimit {
		t.Errorf("ulpClassLimit = %d is below the one measured instance the class exists for; "+
			"a real evaluation-order gap would be reported as transcription drift", ulpClassLimit)
	}

	if a, b := math.Float64bits(ServerTickRate), math.Float64bits(pobServerTickRate); a != b {
		t.Errorf("ServerTickRate = %.17g (bits %016x), PoB has %.17g (bits %016x).\n"+
			"    It must stay a RUNTIME division (`1 / float64(ServerTickTime)`): a const expression\n"+
			"    folds to a different double and moves m_ceil(x * rate) by a whole server tick.",
			ServerTickRate, a, pobServerTickRate, b)
	}
}

// TestULPsApartRefusesToFoldIncomparableValues pins the arms that must NOT report a small number.
func TestULPsApartRefusesToFoldIncomparableValues(t *testing.T) {
	for _, c := range []struct {
		name string
		a, b float64
	}{
		{"opposite signs", math.SmallestNonzeroFloat64, -math.SmallestNonzeroFloat64},
		{"NaN", math.NaN(), 1},
		{"Inf", math.Inf(1), math.MaxFloat64},
	} {
		if got := ulpsApart(c.a, c.b); got != math.MaxUint64 {
			t.Errorf("%s: ulpsApart = %d, want MaxUint64 — anything smaller would classify a "+
				"genuinely incomparable pair as ULP noise and pass it silently", c.name, got)
		}
	}
	if got := ulpsApart(1.5, 1.5); got != 0 {
		t.Errorf("identical values reported %d ULP apart", got)
	}
}
