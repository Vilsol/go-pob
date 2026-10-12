package calculator

import "testing"

// TestAilmentsNeverFromCritKeepTheCritPass pins CalcOffence.lua:4975-5021.
func TestAilmentsNeverFromCritKeepTheCritPass(t *testing.T) {
	for _, name := range []string{"0006-elementalist-storm-burst.xml", "0007-elementalist-storm-burst-of-repulsion.xml"} {
		t.Run(name, func(t *testing.T) {
			assertBuildStats(t, name, []string{"IgniteDPS", "IgniteDamage"}, nil)
		})
	}
}
