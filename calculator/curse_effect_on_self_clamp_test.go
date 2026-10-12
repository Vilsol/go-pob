package calculator

import (
	"os"
	"testing"
)

// TestCurseEffectOnSelfIsClampedAtZero pins the `m_max(..., 0)` in CalcDefence.lua:1596.
func TestCurseEffectOnSelfIsClampedAtZero(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	for _, tc := range []struct {
		build string
		want  float64
		why   string
	}{
		{"0007-elementalist-storm-burst-of-repulsion.xml", 0, "was -3"},
		{"0066-hierophant-forbidden-rite-of-soul-sacrifice.xml", 0, "was -72"},
		{"0282-chieftain-incinerate-of-venting.xml", 0, "was -88.5"},

		{"0003-guardian-unknown.xml", 50, "CONTROL: positive value must be untouched"},
		{"0001-necromancer-unknown.xml", 100, "CONTROL: the default 100 must be untouched"},
	} {
		t.Run(tc.build, func(t *testing.T) {
			env := corpusEnvForLoadedData(t, tc.build)
			got, ok := env.Player.Output["CurseEffectOnSelf"]
			if !ok {
				t.Fatal("CurseEffectOnSelf ABSENT; CalcDefence.lua:1596 writes it unconditionally")
			}
			if got != tc.want {
				t.Errorf("CurseEffectOnSelf = %v, want %v (3.29 oracle) — %s", got, tc.want, tc.why)
			}
		})
	}
}
