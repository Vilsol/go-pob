package exposition

import (
	"encoding/json"
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

// A new skill set is the active one; switching back calculates set 1's groups again, and the save holds both sets.
func TestSkillSetsThroughTheCalculator(t *testing.T) {
	xml, err := os.ReadFile(filepath.Join("testdata", "warrior_dual_mace.xml"))
	if err != nil {
		t.Fatal(err)
	}
	calc, err := ParseBuild(string(xml))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := calc.Calculate(); err != nil {
		t.Fatal(err)
	}
	id, err := calc.NewSkillSet("Slam")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := calc.Calculate(); err != nil {
		t.Fatal(err)
	}
	if v := calc.SkillSets(); v.Active != id || len(v.Sets) != 2 || v.Sets[0].Title != "Default" || v.Sets[1].Title != "Slam" {
		t.Fatalf("skill sets %+v after New Slam (id %d)", v, id)
	}
	if err := calc.SetActiveSkillSet(1); err != nil {
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
	if want := []string{"Boneshatter", "Mace Strike"}; !slices.Equal(labels, want) {
		t.Fatalf("set 1 groups %q, PoB-PoE2 %q", labels, want)
	}
	saved, err := calc.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`activeSkillSet="1"`, `<SkillSet id="1">`, `<SkillSet id="2" title="Slam"`} {
		if !strings.Contains(saved, want) {
			t.Errorf("the save lacks %s:\n%s", want, saved)
		}
	}
}

// The sort cache and DPS reach JS for the group's new gem slot; Fork supports Spark (calculator2's gem sort oracle
// grades the values).
func TestGemSortThroughTheCalculator(t *testing.T) {
	xml, err := os.ReadFile(filepath.Join("..", "..", "calculator2", "testdata", "offence_setup.xml"))
	if err != nil {
		t.Fatal(err)
	}
	calc, err := ParseBuild(string(xml))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := calc.Calculate(); err != nil {
		t.Fatal(err)
	}
	ids := []string{"Metadata/Items/Gems/SkillGemForkSupport", "Metadata/Items/Gems/SkillGemFireball"}
	cache, err := calc.GemSortCache(1, ids)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(cache.CanSupport, []bool{true, false}) {
		t.Errorf("canSupport %v", cache.CanSupport)
	}
	r, err := calc.GemSortDPS(1, ids[:1], "CombinedDPS")
	if err != nil {
		t.Fatal(err)
	}
	if r.Base <= 0 || len(r.DPS) != 1 {
		t.Errorf("sort DPS %+v", r)
	}
	if _, err := calc.GemSortDPS(9, ids, "CombinedDPS"); err == nil {
		t.Error("a group that is not there was sorted")
	}
}

// A copied group pastes back as a new group after the last (calculator2's skills oracle grades the text and the
// pasted group).
func TestCopyPasteThroughTheCalculator(t *testing.T) {
	xml, err := os.ReadFile(filepath.Join("..", "..", "calculator2", "testdata", "offence_setup.xml"))
	if err != nil {
		t.Fatal(err)
	}
	calc, err := ParseBuild(string(xml))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := calc.PasteSocketGroup("Spark 1/0  1"); err == nil {
		t.Fatal("a paste before the first calculation was accepted")
	}
	if _, err := calc.Calculate(); err != nil {
		t.Fatal(err)
	}
	text, err := calc.CopySocketGroup(1)
	if err != nil || !strings.Contains(text, "Spark ") || !strings.Contains(text, "Fork ") {
		t.Fatalf("copied %q, %v", text, err)
	}
	groups, err := calc.SkillGroups()
	if err != nil {
		t.Fatal(err)
	}
	index, err := calc.PasteSocketGroup(text)
	if err != nil || index != len(groups)+1 {
		t.Fatalf("pasted at %d of %d groups, %v", index, len(groups), err)
	}
	if _, err := calc.Calculate(); err != nil {
		t.Fatal(err)
	}
	if again, err := calc.CopySocketGroup(index); err != nil || again != text {
		t.Errorf("the pasted group copies %q, the original %q", again, text)
	}
}

// Ctrl+Z on the Skills tab puts back the groups before the last edit, and Ctrl+Y the edit (calculator2's skills
// oracle grades the states).
func TestSkillsUndoThroughTheCalculator(t *testing.T) {
	xml, err := os.ReadFile(filepath.Join("..", "..", "calculator2", "testdata", "offence_setup.xml"))
	if err != nil {
		t.Fatal(err)
	}
	calc, err := ParseBuild(string(xml))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := calc.SkillsUndo(false); err == nil {
		t.Fatal("an undo before the first calculation was accepted")
	}
	if _, err := calc.Calculate(); err != nil {
		t.Fatal(err)
	}
	count := func() int {
		if _, err := calc.Calculate(); err != nil {
			t.Fatal(err)
		}
		groups, err := calc.SkillGroups()
		if err != nil {
			t.Fatal(err)
		}
		return len(groups)
	}
	before := count()
	if _, err := calc.NewSocketGroup(); err != nil {
		t.Fatal(err)
	}
	calc.AddSkillsUndoState()
	if n := count(); n != before+1 {
		t.Fatalf("%d groups after New, %d before", n, before)
	}
	if ok, err := calc.SkillsUndo(false); !ok || err != nil {
		t.Fatalf("undo %v %v", ok, err)
	}
	if n := count(); n != before {
		t.Errorf("%d groups after the undo, %d before New", n, before)
	}
	if ok, err := calc.SkillsUndo(true); !ok || err != nil {
		t.Fatalf("redo %v %v", ok, err)
	}
	if n := count(); n != before+1 {
		t.Errorf("%d groups after the redo, %d with New", n, before+1)
	}
	if ok, _ := calc.SkillsUndo(true); ok {
		t.Error("a second redo restored something")
	}
}

// AddGem goes after the group's last gem as the edit finds it, whatever slot the page last showed; the values are
// PoB-PoE2's after the same edits (calculator2/testdata/skills_edit_pob.jsonl).
func TestAddGemGoesAfterTheLastGem(t *testing.T) {
	jsonl, err := os.ReadFile(filepath.Join("..", "..", "calculator2", "testdata", "skills_edit_pob.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	pob := map[string]float64{}
	for _, line := range strings.Split(strings.TrimSpace(string(jsonl)), "\n") {
		var row struct {
			Step   string
			Output map[string]float64
		}
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			t.Fatal(err)
		}
		pob[row.Step] = row.Output["TotalDPS"]
	}
	xml, err := os.ReadFile(filepath.Join("..", "..", "calculator2", "testdata", "skills_edit.xml"))
	if err != nil {
		t.Fatal(err)
	}
	calc, err := ParseBuild(string(xml))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := calc.Calculate(); err != nil {
		t.Fatal(err)
	}
	if _, err := calc.NewSocketGroup(); err != nil {
		t.Fatal(err)
	}
	calc.SetMainSocketGroup(2)
	for _, step := range []struct{ id, step string }{
		{"Metadata/Items/Gems/SkillGemFireball", "add Fireball"},
		{"Metadata/Items/Gems/SkillGemElementalFocusSupport", "add Elemental Focus"},
	} {
		if err := calc.AddGem(2, step.id); err != nil {
			t.Fatal(err)
		}
		out, err := calc.Calculate()
		if err != nil {
			t.Fatal(err)
		}
		if got, want := out.Output["TotalDPS"], pob[step.step]; got != want {
			t.Errorf("%s: TotalDPS %v, PoB-PoE2 %v", step.step, got, want)
		}
	}
}
