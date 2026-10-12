package moddb

import (
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/mod"
)

// TestPercentStatCeilsItsProduct: ModStore.lua:532 closes the PercentStat branch with.
//
//	value = m_ceil(value * mult + (tag.base or 0))
func TestPercentStatCeilsItsProduct(t *testing.T) {
	eval := func(t *testing.T, life, pct, value float64) float64 {
		t.Helper()
		store := NewModDB()
		cfg := &ListCfg{SkillStats: map[string]float64{"Life": life}}
		got := store.evalPercentStatTag(mod.NewModValueFloat(value), cfg, mod.PercentStat("Life", pct))
		return got.ValueFloat
	}

	for _, tc := range []struct {
		name string
		life float64
		pct  float64
		want float64
	}{
		{"a non-integral product rounds UP", 4321, 5, 217},
		{"an exact product is untouched", 4000, 5, 200},
		{"a small fraction still rounds up", 4321, 1, 44},
	} {
		t.Run(tc.name, func(t *testing.T) {
			testza.AssertEqual(t, tc.want, eval(t, tc.life, tc.pct, 1))
		})
	}

	t.Run("a negative product ceils toward zero", func(t *testing.T) {
		testza.AssertEqual(t, -216.0, eval(t, 4321, 5, -1))
	})

	t.Run("the stat is actually read", func(t *testing.T) {
		testza.AssertEqual(t, 44.0, eval(t, 4321, 1, 1))
		testza.AssertEqual(t, 87.0, eval(t, 4321, 1, 2))
	})
}
