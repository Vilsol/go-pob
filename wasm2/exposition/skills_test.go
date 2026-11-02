package exposition

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// The warrior's maces grant Mace Strike, which the calculation lists after the saved Boneshatter group; a new
// group goes after it, and the save holds both, as PoB-PoE2's (calculator2/testdata/skills_edit_generated_pob.jsonl).
func TestSkillsEditThroughTheCalculator(t *testing.T) {
	xml, err := os.ReadFile(filepath.Join("testdata", "warrior_dual_mace.xml"))
	if err != nil {
		t.Fatal(err)
	}
	calc, err := ParseBuild(string(xml))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := calc.NewSocketGroup(); err == nil {
		t.Fatal("an edit before the first calculation was accepted")
	}
	if _, err := calc.Calculate(); err != nil {
		t.Fatal(err)
	}
	index, err := calc.NewSocketGroup()
	if err != nil {
		t.Fatal(err)
	}
	if index != 3 {
		t.Fatalf("new group at %d, PoB-PoE2 3", index)
	}
	if err := calc.SetGem(index, 1, "Metadata/Items/Gems/SkillGemRollingSlam"); err != nil {
		t.Fatal(err)
	}
	if _, err := calc.Calculate(); err != nil {
		t.Fatal(err)
	}
	groups, err := calc.SkillGroups()
	if err != nil {
		t.Fatal(err)
	}
	labels := make([]string, len(groups))
	for i, g := range groups {
		labels[i] = g.Label
	}
	if want := []string{"Boneshatter", "Mace Strike", "Rolling Slam"}; !slices.Equal(labels, want) {
		t.Fatalf("groups %q, PoB-PoE2 %q", labels, want)
	}
	if !groups[1].WeaponSetLocked || groups[2].WeaponSetLocked {
		t.Errorf("weapon sets locked: Mace Strike %v, Rolling Slam %v; PoB-PoE2 true, false", groups[1].WeaponSetLocked, groups[2].WeaponSetLocked)
	}
	if g := groups[1].Gems[0]; !g.Generated || g.CountShown {
		t.Errorf("Mace Strike's gem: generated %v, count shown %v; PoB-PoE2 true, false", g.Generated, g.CountShown)
	}
	if err := calc.RemoveGem(2, 1); err == nil {
		t.Error("removed the gem the maces grant")
	}
	if err := calc.DeleteSocketGroup(2); err == nil {
		t.Error("deleted the group the maces grant")
	}
	saved, err := calc.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`source="Default Attack"`, `nameSpec="Rolling Slam"`} {
		if !strings.Contains(saved, want) {
			t.Errorf("the save lacks %s:\n%s", want, saved)
		}
	}
	if i, j := strings.Index(saved, `source="Default Attack"`), strings.Index(saved, `nameSpec="Rolling Slam"`); i > j {
		t.Errorf("the save lists the new group before the generated one:\n%s", saved)
	}
}

// GemList is GemSelectControl's gem list: data.gems less what items and the tree grant.
func TestGemListLeavesOutGrantedSkills(t *testing.T) {
	list, err := GemList()
	if err != nil {
		t.Skipf("game data not loadable: %v", err)
	}
	byID := map[string]int{}
	for i, g := range list {
		byID[g.ID] = i
	}
	fireball, ok := byID["Metadata/Items/Gems/SkillGemFireball"]
	if !ok {
		t.Fatal("Fireball is not in the gem list")
	}
	if g := list[fireball]; g.Name != "Fireball" || g.Support || !slices.Contains(g.Tags, "fire") || g.Color != 3 {
		t.Errorf("Fireball %+v", g)
	}
	if g, ok := byID["Metadata/Items/Gems/SkillGemElementalFocusSupport"]; !ok || !list[g].Support {
		t.Error("Elemental Focus is not a support in the gem list")
	}
	if _, ok := byID["Metadata/Items/Gems/SkillGemPlayerDefaultMaceMace"]; ok {
		t.Error("the default mace attack, which a weapon grants, is in the gem list")
	}
}
