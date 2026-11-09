package calculator2

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Vilsol/go-pob/pob2"
)

// TestGemFoundByNameWhenItsIDIsUnknown loads gems whose saved gemId no gem has; PoB-PoE2 finds them by name
// (SkillsTab.lua:1335-1343) and exports their buffs (testdata/oracle/poe2_party_export.json).
func TestGemFoundByNameWhenItsIDIsUnknown(t *testing.T) {
	xml, err := os.ReadFile(filepath.Join("testdata", "party_giver.xml"))
	if err != nil {
		t.Fatal(err)
	}
	build, err := pob2.ParseBuild(xml)
	if err != nil {
		t.Fatal(err)
	}
	tree := loadTestTree(t)
	env, err := BuildOutput(build, Options{Tree: tree, Items: NewItemsTab(build, tree, nil)})
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(env.Player.ActiveSkillList))
	for _, s := range env.Player.ActiveSkillList {
		names = append(names, s.ActiveEffect.GrantedEffect.Name)
	}
	for _, want := range []string{"Defiance Banner", "Elemental Weakness", "Infernal Cry"} {
		if !slices.Contains(names, want) {
			t.Errorf("%s is not an active skill: %q", want, names)
		}
	}
}
