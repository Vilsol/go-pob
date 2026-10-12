package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/pob"
)

// TestNameSpecOnlyGemResolves: a gem saved with only nameSpec is matched by FindSkillGem (SkillsTab.lua:1076-1100, :1166-1174).
func TestNameSpecOnlyGemResolves(t *testing.T) {
	requireGameData(t, "3.29")
	for name, want := range map[string]string{
		"Mana-Infused Staff": "Metadata/Items/Gems/SkillGemManaInfusedStaff",
		"fireball":           "Metadata/Items/Gems/SkillGemFireball",
		"Frostblink":         "Metadata/Items/Gems/SkillGemFrostblink",
	} {
		gemData, _ := resolveGemInstance(pob.Gem{NameSpec: name})
		if gemData == nil || gemData.GetBaseItemType() == nil || gemData.GetBaseItemType().ID != want {
			t.Errorf("%q resolved to %v, want %s", name, gemData, want)
		}
	}
	for _, name := range []string{" ", "Ancestral Protector", "CtF"} {
		if gemData, ge := resolveGemInstance(pob.Gem{NameSpec: name}); gemData != nil || ge != nil {
			t.Errorf("%q resolved, want unrecognised/ambiguous", name)
		}
	}
}

func TestManaInfusedStaffNameSpecCorpus(t *testing.T) {
	requireGameData(t, "3.29")
	env := corpusEnvForLoadedData(t, "0954-elementalist-vaal-ice-nova-5.xml")
	for k, want := range map[string]float64{"BlockChance": 49, "SpellBlockChance": 75, "ManaUnreserved": 99} {
		if got := env.Player.Output[k]; got != want {
			t.Errorf("%s = %v, want %v (PoB)", k, got, want)
		}
	}
}
