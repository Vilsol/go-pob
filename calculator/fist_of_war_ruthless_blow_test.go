package calculator

import (
	"math"
	"os"
	"testing"
)

// TestFistOfWarAndRuthlessBlowUseTheirRealModNames pins CalcOffence.lua:2889-2945.
func TestFistOfWarAndRuthlessBlowUseTheirRealModNames(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	t.Run("ruthless blow ailment multiplier reaches its mod", func(t *testing.T) {
		for _, tc := range []struct {
			build string
			want  float64
		}{
			{"0069-berserker-boneshatter.xml", 1.94},
			{"0093-ascendant-frostblink.xml", 1.98},
			{"0104-gladiator-boneshatter-of-complex-trauma.xml", 1.97},
		} {
			env := corpusEnvForLoadedData(t, tc.build)
			mh := env.Player.OutputTable[OutTableMainHand]
			got, ok := mh["RuthlessBlowAilmentMultiplier"]
			if !ok {
				t.Errorf("%s: MainHand.RuthlessBlowAilmentMultiplier absent — this key was "+
					"published as RuthlessBlowBleedMultiplier, a name PoB does not use", tc.build)
				continue
			}
			if math.Abs(got-tc.want) > 1e-9 {
				t.Errorf("%s: MainHand.RuthlessBlowAilmentMultiplier = %v, PoB %v — a 1 here "+
					"means the sum is reading a mod name nothing produces", tc.build, got, tc.want)
			}
		}
	})

	t.Run("fist of war publishes DamageEffect, not Hit/Ailment", func(t *testing.T) {
		env := corpusEnvForLoadedData(t, "0065-chieftain-tectonic-slam.xml")
		mh := env.Player.OutputTable[OutTableMainHand]
		if _, ok := mh["FistOfWarDamageEffect"]; !ok {
			t.Error("MainHand.FistOfWarDamageEffect absent; PoB publishes it and we published " +
				"FistOfWarHitEffect / FistOfWarAilmentEffect")
		}
		for _, dead := range []string{"FistOfWarHitEffect", "FistOfWarAilmentEffect", "RuthlessBlowBleedEffect"} {
			if _, ok := mh[dead]; ok {
				t.Errorf("%s is still published; PoB has no such output at this pin and a "+
					"consumer left reading it would silently keep the old semantics", dead)
			}
		}
	})
}
