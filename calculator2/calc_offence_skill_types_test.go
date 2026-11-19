package calculator2

import (
	"math"
	"os"
	"testing"

	"github.com/Vilsol/go-pob/pob2"
)

// PoB-PoE2 widens HazardRearmChance past its cap, and takes the average EffectiveCooldown, only when it builds a
// breakdown (CalcOffence.lua:1650-1652, 1822-1823): the main output and GetMiscCalculator's base output for the
// fixture's Spearfield explosion (group 1) and Frost Bomb (group 2).
func TestSkillTypeOutputsFollowBreakdown(t *testing.T) {
	raw, err := os.ReadFile("testdata/skill_type_breakdown_gates.xml")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		group      int
		stat       string
		main, calc float64
	}{
		{1, "HazardRearmChance", 150, 100},
		{2, "EffectiveCooldown", 4.2042, 6.006},
	} {
		build, err := pob2.ParseBuild(raw)
		if err != nil {
			t.Fatal(err)
		}
		build.Build.MainSocketGroup = c.group
		tree := loadTestTree(t)
		env, err := BuildOutput(build, Options{Tree: tree, Items: NewItemsTab(build, tree, nil)})
		if err != nil {
			t.Fatal(err)
		}
		if got := env.Player.Output[c.stat]; math.Abs(got-c.main) > 1e-9 {
			t.Errorf("main %s %v, PoB-PoE2 %v", c.stat, got, c.main)
		}
		if got := NewMiscCalculator(env).Base.Player.Output[c.stat]; math.Abs(got-c.calc) > 1e-9 {
			t.Errorf("calculator %s %v, PoB-PoE2 %v", c.stat, got, c.calc)
		}
	}
}
