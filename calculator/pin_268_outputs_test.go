package calculator

import (
	"os"
	"testing"
)

// TestPin268OutputsMatchThePin grades one corpus build per calculation PoB 2.68.1 changed against the value the
// pinned PoB dumps for it.
func TestPin268OutputsMatchThePin(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	for _, tc := range []struct {
		build, stat string
		want        float64
	}{
		// CalcPerform.lua:666 — fortification stacks default to the maximum
		{"0209-elementalist-lightning-tendrils-of-eccentricity.xml", "FortificationStacks", 20},
		{"0029-inquisitor-spark-of-the-nova.xml", "TotalEHP", 244015.89480553},
		// CalcPerform.lua:134, CalcDefence.lua:1228-1233
		{"0209-elementalist-lightning-tendrils-of-eccentricity.xml", "MaximumLifeManaDifference", 3139},
		{"0209-elementalist-lightning-tendrils-of-eccentricity.xml", "MissingUnreservedLifePercent", 45},
		// CalcPerform.lua:3984 — gem level is the active effect's level
		{"0862-elementalist-treeskill-2.xml", "GemLevel", 20},
		// CalcSetup.lua:17-31 — socket colour quality from the equipped item
		{"10796-ladder-Lotzkat_5271.WeedCarriedByMerc.xml", "GemQuality", 10},
		// CalcOffence.lua:2938-2940 — pact uptime counts non-repeated casts
		{"0182-assassin-freezing-pulse.xml", "BeidatUpTimeRatio", 100},
		// CalcOffence.lua:2301-2304 — trap time floored at its cooldown
		{"0022-saboteur-blade-trap-of-greatswords.xml", "TotalDotDPS", 1391.349407175},
		// CalcOffence.lua:3505-3523 — partially lucky hits average the lucky and unlucky rolls
		{"0781-hierophant-fireball.xml", "LightningHitAverage", 14883.1},
		{"0532-necromancer-summon-holy-relic-of-conviction-7.xml", "ColdMaximumHitTaken", 50270},
	} {
		t.Run(tc.build+"/"+tc.stat, func(t *testing.T) {
			env := defenceEnv(t, tc.build)
			if got := env.Player.Output[tc.stat]; !oracleClose(got, tc.want, oracleDumpTolerance) {
				t.Errorf("%s = %v, PoB %v", tc.stat, got, tc.want)
			}
		})
	}
}
