package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/storage"
)

// TestSkillCfgNameKeepsSummonSkeletonsPlural pins CalcActiveSkill.lua:457 at the 3.29 pin: skillCfg.skillName strips a leading "Vaal " and nothing else.
func TestSkillCfgNameKeepsSummonSkeletonsPlural(t *testing.T) {
	if err := poe.InitializeAll(raw.DataContext(t.Context()), raw.LatestVersion, raw.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}
	for id, want := range map[string]string{
		"SummonSkeletons":     "Summon Skeletons",
		"VaalSummonSkeletons": "Summon Skeletons",
	} {
		ge := poe.GrantedEffectByID(id)
		if ge == nil {
			t.Fatalf("REFUSING: %q is not a granted effect in the loaded data — wrong fixture", id)
		}
		if got := activeSkillCfgName(&GrantedEffect{Raw: ge}); got != want {
			t.Errorf("%s: skillCfg.skillName = %q, want %q", id, got, want)
		}
	}
}
