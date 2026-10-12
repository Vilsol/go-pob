package calculator

import (
	"math"
	"testing"
)

// TestCalcsFirstDivergingRungNamesTheEarliestCause is the control for the damage chain's attributor.
func TestCalcsFirstDivergingRungNamesTheEarliestCause(t *testing.T) {
	clean := func() (map[string]float64, map[string]float64) {
		m := map[string]float64{
			"FireMinBase": 100, "FireMaxBase": 200,
			"FireEffMult": 1.5,
			"FireMin":     300, "FireMax": 600,
			"FireHitAverage": 450,
		}
		cp := map[string]float64{}
		for k, v := range m {
			cp[k] = v
		}
		return m, cp
	}

	for _, tc := range []struct {
		name   string
		break_ func(got map[string]float64)
		want   string
	}{
		{"agreeing chain names nothing", func(map[string]float64) {}, ""},
		{"wrong base is named at the base", func(g map[string]float64) {
			g["FireMinBase"] = 50
			g["FireMin"] = 150
			g["FireHitAverage"] = 375
		}, "MinBase"},
		{"clean base, wrong enemy multiplier", func(g map[string]float64) {
			g["FireEffMult"] = 1.0
			g["FireMin"] = 200
			g["FireMax"] = 400
			g["FireHitAverage"] = 300
		}, "EffMult"},
		{"clean base and EffMult, wrong modifier product", func(g map[string]float64) {
			g["FireMin"] = 450
			g["FireMax"] = 900
			g["FireHitAverage"] = 675
		}, "F(modifiers)"},
		{"every input clean, the aggregate alone is wrong", func(g map[string]float64) {
			g["FireHitAverage"] = 999
		}, "HitAverage"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			want, got := clean()
			tc.break_(got)
			if r := calcsFirstDivergingRung(want, got, "Fire"); r != tc.want {
				t.Errorf("first diverging rung = %q, want %q", r, tc.want)
			}
		})
	}

	t.Run("absent rungs are skipped, not scored", func(t *testing.T) {
		want := map[string]float64{"FireMinBase": 100, "FireHitAverage": 450}
		got := map[string]float64{"FireMinBase": 100, "FireHitAverage": 999}
		if r := calcsFirstDivergingRung(want, got, "Fire"); r != "HitAverage" {
			t.Errorf("got %q, want HitAverage — absent Min/Max/EffMult must not be scored", r)
		}
	})

	t.Run("a zero on PoB's side is not a comparison", func(t *testing.T) {
		want := map[string]float64{"FireMinBase": 0, "FireHitAverage": 450}
		got := map[string]float64{"FireMinBase": 77, "FireHitAverage": 450}
		if r := calcsFirstDivergingRung(want, got, "Fire"); r != "" {
			t.Errorf("got %q, want \"\" — a zero in the dump carries no claim", r)
		}
	})
}

func calcsFirstDivergingRung(want, got map[string]float64, dt string) string {
	const tol = 0.005

	agrees := func(key string) (bool, bool) {
		w, present := want[key]
		if !present || w == 0 {
			return false, false
		}
		return true, math.Abs(got[key]-w) <= math.Abs(w)*tol
	}

	for _, rung := range []string{"MinBase", "MaxBase", "EffMult"} {
		if compared, ok := agrees(dt + rung); compared && !ok {
			return rung
		}
	}

	base, okB := want[dt+"MinBase"]
	em, okE := want[dt+"EffMult"]
	mn, okM := want[dt+"Min"]
	if okB && okE && okM && base != 0 && em != 0 {
		ourBase, ourEm := got[dt+"MinBase"], got[dt+"EffMult"]
		if ourBase != 0 && ourEm != 0 {
			wantF := mn / (base * em)
			gotF := got[dt+"Min"] / (ourBase * ourEm)
			if wantF != 0 && math.Abs(gotF-wantF) > math.Abs(wantF)*tol {
				return "F(modifiers)"
			}
		}
	}

	for _, rung := range []string{"Min", "Max", "HitAverage"} {
		if compared, ok := agrees(dt + rung); compared && !ok {
			return rung
		}
	}
	return ""
}
