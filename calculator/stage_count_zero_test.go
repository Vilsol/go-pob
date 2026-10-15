package calculator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/mod"
)

// TestSavedStageCountZeroRaisesToMinimumStage pins CalcActiveSkill.lua:639: a saved skillStageCount="0" is not nil.
func TestSavedStageCountZeroRaisesToMinimumStage(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(corpusDirForLoadedData(t), "15467-necromancer-animate-guardians-weapon.xml"))
	if err != nil {
		t.Fatal(err)
	}
	build, err := builds.ParseBuild(raw)
	if err != nil {
		t.Fatal(err)
	}
	set := &build.Skills.SkillSets[8]
	if g := set.Skills[4].Gems[2]; set.ID != 9 || g.NameSpec != "Frost Shield" || !g.StageCountSet() || g.SkillStageCount != 0 {
		t.Fatalf("REFUSING: fixture moved: set %d %q stage %d", set.ID, g.NameSpec, g.SkillStageCount)
	}
	build.Skills.ActiveSkillSet = 9
	build.Build.MainSocketGroup = 5
	for active := 1; active <= 4; active++ {
		set.Skills[4].MainActiveSkill = active
		env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())
		ms := env.Player.MainSkill
		if ms == nil || ms.ActiveEffect.GrantedEffect.Name() != "Frost Shield" {
			continue
		}
		if limit := ms.SkillModList.Sum(mod.TypeBase, ms.SkillCfg, "Multiplier:FrostShieldMaxStages"); limit != 4 {
			t.Fatalf("REFUSING: FrostShieldMaxStages %v, want 4", limit)
		}
		minimum := ms.SkillModList.Sum(mod.TypeBase, ms.SkillCfg, "Multiplier:FrostShieldMinimumStage")
		if got, want := ms.SkillModList.Sum(mod.TypeBase, ms.SkillCfg, "Multiplier:FrostShieldStage"), 1+minimum; got != want {
			t.Errorf("Multiplier:FrostShieldStage = %v, want %v (m_max(0, 1 + MinimumStage))", got, want)
		}
		return
	}
	t.Fatal("REFUSING: Frost Shield never became the main skill")
}
