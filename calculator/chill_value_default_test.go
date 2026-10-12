package calculator

import (
	"math"
	"os"
	"testing"
)

// TestChillFlagUsesNoDefaultValue pins CalcPerform.lua:835-848.
//
//	local chillValue = m_max(modDB:Sum("BASE", nil, "SelfChillOverride"),
//	                         modDB:Override(nil, "ChillVal")) or ailmentData.Chill.default
func TestChillFlagUsesNoDefaultValue(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	cases := []struct {
		build string
		want  float64
		note  string
	}{
		{"0087-juggernaut-ground-slam.xml", 0.77777777777778, "chilled 10 via the ailment path"},
		{"0091-elementalist-winter-orb-2.xml", 0.82352941176471, "chilled 15 via the ailment path"},
	}

	for _, tc := range cases {
		t.Run(tc.build, func(t *testing.T) {
			env := corpusEnvForLoadedData(t, tc.build)
			got, ok := env.Player.Output["enemySkillTime"]
			if !ok {
				t.Fatal("enemySkillTime absent")
			}
			if math.IsNaN(got) || math.IsInf(got, 0) {
				t.Fatalf("enemySkillTime = %v — a doubled chill can drive enemy action speed to "+
					"0, and this divides by it", got)
			}
			if math.Abs(got-tc.want) > math.Abs(tc.want)*1e-9 {
				t.Errorf("enemySkillTime = %v, PoB %v (%s)", got, tc.want, tc.note)
			}
		})
	}
}
