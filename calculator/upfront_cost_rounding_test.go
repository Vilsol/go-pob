package calculator

import "testing"

// TestUpfrontCostRoundedAfterEfficiency pins CalcOffence.lua:1835-1838 and :1852-1855.
func TestUpfrontCostRoundedAfterEfficiency(t *testing.T) {
	requireGameData(t, "3.29")
	cases := []struct {
		build, stat string
		want        float64
	}{
		{"0046-elementalist-arc.xml", "ManaCost", 133},
		{"0010-elementalist-storm-burst-2.xml", "ManaCost", 3},
		{"0043-pathfinder-toxic-rain.xml", "LifeCost", 11},
	}
	for _, c := range cases {
		env := corpusEnvForLoadedData(t, c.build)
		if got := env.Player.Output[c.stat]; got != c.want {
			t.Errorf("%s %s = %v, want %v", c.build, c.stat, got, c.want)
		}
	}
}
