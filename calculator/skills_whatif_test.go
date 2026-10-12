package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/pob"
)

// TestCalcGemIsTheBuildWithThatGem: The gem what-if is the build with that gem.
func TestCalcGemIsTheBuildWithThatGem(t *testing.T) {
	build := parseCorpusBuild(t, "0280-elementalist-fireball.xml")
	set := &build.Skills.SkillSets[build.ActiveSkillSetIndex()]
	group := build.Build.MainSocketGroup - 1
	gem := set.Skills[group].Gems[0]
	if gem.Quality == 20 {
		gem.Quality = 0
		set.Skills[group].Gems[0] = gem
	}
	original := gem.Quality
	n := len(set.Skills[group].Gems)

	misc := GetMiscCalculator(build)
	edited := gem
	edited.Quality = 20
	got := misc.CalcGem(group, 0, edited).Player.Output
	if set.Skills[group].Gems[0].Quality != original || len(set.Skills[group].Gems) != n {
		t.Fatal("the build's gem changed")
	}
	base := misc.Base().Player.Output

	set.Skills[group].Gems[0] = edited
	want := GetMiscCalculator(build).Base().Player.Output
	changed := 0
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s: %v, the build with the gem %v", k, got[k], v)
		}
		if base[k] != v {
			changed++
		}
	}
	if changed == 0 {
		t.Fatal("20 quality changes nothing: no control")
	}
}

// TestCalcConfigIsTheBuildWithThatOption: The config what-if is the build with that option.
func TestCalcConfigIsTheBuildWithThatOption(t *testing.T) {
	build := parseCorpusBuild(t, "0280-elementalist-fireball.xml")
	n := len(build.Config.Inputs)
	off := false
	input := pob.Input{Name: "conditionEnemyShocked", Boolean: &off}

	misc := GetMiscCalculator(build)
	got := misc.CalcConfig(input, false).Player.Output
	if len(build.Config.Inputs) != n {
		t.Fatal("the build's options changed")
	}
	base := misc.Base().Player.Output

	build.SetConfigOption(input)
	want := GetMiscCalculator(build).Base().Player.Output
	changed := 0
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s: %v, the build with the option %v", k, got[k], v)
		}
		if base[k] != v {
			changed++
		}
	}
	if changed == 0 {
		t.Fatal("the option changes nothing: no control")
	}
}

// TestCalcCustomModGroupIsTheBuildWithTheGroupSwitched: The custom group what-if is the build with that group switched.
func TestCalcCustomModGroupIsTheBuildWithTheGroupSwitched(t *testing.T) {
	build := parseCorpusBuild(t, "0280-elementalist-fireball.xml")
	title, on := "Test", "true"
	build.SetCustomModifierBlocks([]pob.CustomModifierBlock{{Title: &title, Enabled: &on, Text: "+1000 to maximum Life"}})

	misc := GetMiscCalculator(build)
	got := misc.CalcCustomModGroup(0).Player.Output
	if !build.GetCustomModifierBlocks()[0].IsEnabled() {
		t.Fatal("the build's group changed")
	}
	base := misc.Base().Player.Output

	off := "false"
	build.SetCustomModifierBlocks([]pob.CustomModifierBlock{{Title: &title, Enabled: &off, Text: "+1000 to maximum Life"}})
	want := GetMiscCalculator(build).Base().Player.Output
	if base["Life"] == want["Life"] {
		t.Fatal("the group changes no Life: no control")
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s: %v, the build with the group off %v", k, got[k], v)
		}
	}
}
