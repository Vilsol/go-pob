package raw

import (
	"slices"
	"testing"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/storage"
)

func carriesNotMinionTag(m *StatMap) bool {
	for _, md := range m.Mods {
		for _, tag := range md.Tags() {
			if ac, ok := tag.(*mod.ActorConditionTag); ok && ac.Actor != nil && *ac.Actor == "parent" && ac.Negative && len(ac.VariableList) == 0 {
				return true
			}
		}
	}
	return false
}

// TestNotMinionStatTagsSupportAndBuffStats: The lists and the tag are the pin's: sup_str.lua SupportLifetap and spectre.lua EDSHarvestMinerHammerSmoke.
//
//pob:block:src/Modules/Data.lua:81-91
func TestNotMinionStatTagsSupportAndBuffStats(t *testing.T) {
	if err := poe.InitializeAll(DataContext(t.Context()), LatestVersion, AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}
	lifetap := GetCalculatedGrantedEffect(poe.GrantedEffectByID("SupportLifetap"))
	if got, want := lifetap.GetNotMinionStat(), []string{"support_lifetap_damage_+%_final_while_buffed", "support_lifetap_spent_life_threshold"}; !slices.Equal(got, want) {
		t.Errorf("SupportLifetap notMinionStat = %v, PoB %v", got, want)
	}
	m := lifetap.GetStatMapEntry("support_lifetap_damage_+%_final_while_buffed")
	if m == nil || len(m.Mods) == 0 {
		t.Fatal("REFUSING: Lifetap's stat map is empty")
	}
	if !carriesNotMinionTag(m) {
		t.Errorf("Lifetap's MORE does not carry { type = \"ActorCondition\", actor = \"parent\", neg = true }")
	}

	smoke := GetCalculatedGrantedEffect(poe.GrantedEffectByID("EDSHarvestMinerHammerSmoke"))
	if got, want := smoke.GetNotMinionStat(), []string{"spell_minimum_base_physical_damage", "spell_maximum_base_physical_damage"}; !slices.Equal(got, want) {
		t.Errorf("EDSHarvestMinerHammerSmoke notMinionStat = %v, PoB %v", got, want)
	}
	m = smoke.GetStatMapEntry("spell_minimum_base_physical_damage")
	if m == nil || len(m.Mods) == 0 {
		t.Fatal("REFUSING: the control's stat map is empty")
	}
	if carriesNotMinionTag(m) {
		t.Errorf("EDSHarvestMinerHammerSmoke's damage carries the notMinionStat tag; PoB tags only supports and buffs")
	}
}
