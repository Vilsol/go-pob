package exposition

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/data/raw"
)

// TestSetActiveTreeSpecMatchesTheSavedTree: Switching 0280 from "Lab 2" (tree 2) to "White maps" (tree 3) calculates as the same build saved with tree 3 active.
func TestSetActiveTreeSpecMatchesTheSavedTree(t *testing.T) {
	initItemData(t)
	const file = "0280-elementalist-fireball.xml"
	build := corpusBuild(t, file)
	if build.Tree.ActiveSpec != 2 {
		t.Fatalf("REFUSING: 0280 opens on tree %d, want 2", build.Tree.ActiveSpec)
	}
	xml, err := os.ReadFile(filepath.Join("..", "..", "testdata", "corpus-"+raw.LatestVersion, file))
	if err != nil {
		t.Fatal(err)
	}
	saved, err := builds.ParseBuild([]byte(strings.Replace(string(xml), `<Tree activeSpec="2">`, `<Tree activeSpec="3">`, 1)))
	if err != nil || saved.Tree.ActiveSpec != 3 {
		t.Fatalf("REFUSING: the tree-3 copy did not load (%v)", err)
	}

	SetActiveTreeSpec(build, 3)
	list := GetTreeSpecs(build)
	if list.Active != 3 || len(list.Specs) != 3 || list.Specs[2].Title != "White maps" {
		t.Fatalf("GetTreeSpecs = %+v", list)
	}
	got, want := GetMiscCalculator(build).Base().Output, GetMiscCalculator(saved).Base().Output
	for _, k := range []string{"Life", "Mana", "EnergyShield", "FireResist", "TotalEHP", "CombinedDPS"} {
		if got[k] != want[k] {
			t.Errorf("%s: %v after switching, %v saved on tree 3", k, got[k], want[k])
		}
	}
	tree2 := GetMiscCalculator(corpusBuild(t, file)).Base().Output
	if got["Life"] == tree2["Life"] && got["CombinedDPS"] == tree2["CombinedDPS"] {
		t.Error("vacuity: tree 2 and tree 3 calculate the same Life and DPS")
	}
}
