package calculator

import (
	"math"
	"os"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/pob"
)

// TestItemGrantedSkillTakesSlotSupports pins CalcSetup.lua:1794-1813 and 1502 against the pinned
// PoB on one build: a skill granted by an item takes the supports of every other group socketed in
// that item, a gem added to the granted group itself is dropped by the next calculation, and the
// gem picker's what-if (CALCULATOR mode, no 1502 wipe) still counts it.
func TestItemGrantedSkillTakesSlotSupports(t *testing.T) {
	requireGameData(t, "3.29")
	file, err := os.ReadFile("testdata/item-granted-supports/dancing-dervish.xml")
	if err != nil {
		t.Fatal(err)
	}
	build, err := builds.ParseBuild(file)
	if err != nil {
		t.Fatal(err)
	}
	near := func(what string, got, want float64) {
		t.Helper()
		if math.Abs(got-want) > 1e-6*want {
			t.Errorf("%s: minion TotalDPS %v, PoB %v", what, got, want)
		}
	}
	calc := func() *Environment {
		return BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())
	}

	env := calc()
	near("loaded", env.Minion.Output["TotalDPS"], 1891483.0676892)
	if n := len(env.Player.MainSkill.SupportList); n != 6 {
		t.Errorf("the granted skill has %d supports, PoB applies the 6 of the Weapon 1 group", n)
	}

	set := &build.Skills.SkillSets[build.ActiveSkillSetIndex()]
	gem := pob.Gem{GemID: "Metadata/Items/Gems/SupportGemMeleePhysicalDamage", NameSpec: "Melee Physical Damage", SkillID: "SupportMeleePhysicalDamage", Level: 20, Enabled: true, EnableGlobal1: true}
	near("what-if adding a support to the granted group", GetMiscCalculator(build).CalcGem(1, len(set.Skills[1].Gems), gem).Minion.Output["TotalDPS"], 2102715.8627638)

	set.Skills[1].Gems = append(set.Skills[1].Gems, gem)
	near("support added to the granted group", calc().Minion.Output["TotalDPS"], 1891483.0676892)
	if n := len(set.Skills[1].Gems); n != 1 {
		t.Errorf("the granted group holds %d gems after a calculation, PoB keeps only the granted one", n)
	}

	set.Skills[2].Enabled = false
	near("Weapon 1 support group disabled", calc().Minion.Output["TotalDPS"], 317560.80773136)
}
