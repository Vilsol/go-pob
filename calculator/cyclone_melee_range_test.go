package calculator

import (
	"os"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestCycloneAdditionalMeleeRangeMatchesPin covers act_dex.lua:4701-4714.
func TestCycloneAdditionalMeleeRangeMatchesPin(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	for _, tc := range []struct {
		build string
		want  float64
	}{
		{"0215-slayer-cyclone-of-tumult", 7},
		{"0609-slayer-cyclone-of-tumult-5", 10},
		{"0167-slayer-cyclone", 7},
		{"0234-champion-cyclone", 3},
	} {
		env := corpusEnvForLoadedData(t, tc.build)
		if env.Player.MainSkill == nil {
			t.Errorf("%s: no main skill", tc.build)
			continue
		}
		got := env.Player.MainSkill.SkillModList.Sum(mod.TypeBase, nil, "Multiplier:AdditionalMeleeRange")
		if got != tc.want {
			t.Errorf("%s: Multiplier:AdditionalMeleeRange = %v, want %v (the pinned PoB's value)",
				tc.build, got, tc.want)
		}
	}
}
