package calculator

import (
	"math"
	"os"
	"testing"
)

// TestChillOverrideUsesSkillCfgAndMore pins CalcPerform.lua:1394.
func TestChillOverrideUsesSkillCfgAndMore(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	for _, tc := range []struct {
		build string
		want  float64
		why   string
	}{
		{"0076-occultist-vaal-cold-snap-of-power.xml", 12, "was 10 — x1.2 not applied"},
		{"0112-occultist-vaal-cold-snap-of-power-2.xml", 13, "was 10 — x1.3 not applied"},
		{"0035-elementalist-shield-charge.xml", 27, "was 21 — x1.3 not applied, floor(21*1.3)=27"},

		{"0075-elementalist-winter-orb.xml", 12, "CONTROL: 12 in BOTH states — must not move"},
	} {
		t.Run(tc.build, func(t *testing.T) {
			env := corpusEnvForLoadedData(t, tc.build)
			got, ok := env.Player.Output["CurrentChill"]
			if !ok {
				t.Fatalf("CurrentChill ABSENT")
			}
			if math.Abs(got-tc.want) > 1e-9 {
				t.Errorf("CurrentChill = %v, want %v (PoB) — %s", got, tc.want, tc.why)
			}
		})
	}
}
