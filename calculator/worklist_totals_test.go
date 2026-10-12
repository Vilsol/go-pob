package calculator

import (
	"strings"
	"testing"
)

// TestWorklistTotalsCarriesTheGradedTotalAndActorSplit pins the file that fixes the 1.755x pp inflation in `task residual-map`.
func TestWorklistTotalsCarriesTheGradedTotalAndActorSplit(t *testing.T) {
	perStat := map[string]*statTally{
		"Life":              {seen: 10, ok: 10},
		"TotalDPS":          {seen: 10, ok: 6},
		"MainHand.TotalDPS": {seen: 4, ok: 1},
		"Minion.Life":       {seen: 6, ok: 6},
		"Minion.TotalDPS":   {seen: 6, ok: 2, nan: 1},
		"Enemy.Life":        {seen: 2, ok: 0},
	}
	got := worklistTotalsTSV(perStat)

	want := strings.Join([]string{
		"scope\tseen\tok\tmiss\tnan",
		"ALL\t38\t25\t13\t1",
		"Enemy.\t2\t0\t2\t0",
		"MainHand.\t4\t1\t3\t0",
		"Minion.\t12\t8\t4\t1",
		"player\t20\t16\t4\t0",
		"",
	}, "\n")
	if got != want {
		t.Errorf("worklistTotalsTSV mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}
