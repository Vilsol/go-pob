package builds

import (
	"os"
	"testing"
)

// TestSavedBuildSummaryOfAPoE2Save: the list reads a PoB-PoE2 save's <Build> header as BuildListHelpers.lua:27-34 (PoB-PoE2) does, and says which game it is.
func TestSavedBuildSummaryOfAPoE2Save(t *testing.T) {
	poe2, err := os.ReadFile("../wasm2/exposition/testdata/sorceress_spark.xml")
	if err != nil {
		t.Fatal(err)
	}
	spec := `<Tree activeSpec="1"><Spec nodes="" treeVersion="3_25"/></Tree>`
	cases := []struct {
		name string
		raw  string
		want SavedBuild
	}{
		{"PoE2 save", string(poe2), SavedBuild{ClassName: "Sorceress", Level: 40, MainSkill: "Spark", Game: "poe2"}},
		{"PoE1 save", `<PathOfBuilding><Build level="7" className="Witch" ascendClassName="Necromancer" targetVersion="3_0"/>` + spec + `</PathOfBuilding>`, SavedBuild{ClassName: "Witch", Level: 7, Ascendancy: "Necromancer"}},
		{"PoE2 save from before the root rename", `<PathOfBuilding><Build level="3" className="Monk" targetVersion="0_1"/>` + spec + `</PathOfBuilding>`, SavedBuild{ClassName: "Monk", Level: 3, Game: "poe2-legacy"}},
	}
	for _, c := range cases {
		got, err := SavedBuildSummary(c.raw)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if got != c.want {
			t.Errorf("%s: got %+v, want %+v", c.name, got, c.want)
		}
	}
}
