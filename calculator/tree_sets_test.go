package calculator

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/pob"
)

func parseCorpusBuild(t *testing.T, file string) *pob.PathOfBuilding {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(corpusDirForLoadedData(t), file))
	if err != nil {
		t.Fatal(err)
	}
	build, err := builds.ParseBuild(raw)
	if err != nil {
		t.Fatal(err)
	}
	return build
}

// TestClassAscendancyOrderMatchesTheTree: pob's per-class ascendancy order is the tree's (a spec's ascendClassId indexes it).
func TestClassAscendancyOrderMatchesTheTree(t *testing.T) {
	tree := data.TreeVersions[data.LatestTreeVersion].Tree()
	for id, want := range pob.ClassAscendancyNames() {
		var got []pob.AscendancyName
		for _, a := range tree.Classes[id].Ascendancies {
			got = append(got, a.Name)
		}
		if !slices.Equal(got, want) || tree.Classes[id].Name != pob.ClassNameByID[id] {
			t.Errorf("class %d %s: tree %s %v, pob %v", id, pob.ClassNameByID[id], tree.Classes[id].Name, got, want)
		}
	}
}

// TestTreeSpecPointsMatchPoB: The tree list's point counts.
func TestTreeSpecPointsMatchPoB(t *testing.T) {
	tree := data.TreeVersions[data.LatestTreeVersion].Tree()
	checked, clusters := 0, 0
	for _, want := range loadSidebarControlsCases(t) {
		build := parseCorpusBuild(t, want.File)
		spec := build.Tree.Specs[build.ActiveSpecIndex()]
		if slices.ContainsFunc(pob.SpecNodes(spec), func(id int64) bool { return id >= 65536 }) {
			clusters++
		}
		used, ascUsed := specAllocCounts(tree, specSubgraph(build, build.ActiveSpecIndex()+1, spec), spec)
		if used != want.Used || ascUsed != want.AscUsed {
			t.Errorf("%s: %d / %d, PoB %d / %d", want.File, used, ascUsed, want.Used, want.AscUsed)
		}
		checked++
	}
	if checked < 10 || clusters == 0 {
		t.Fatalf("only %d builds checked, %d with subgraph nodes", checked, clusters)
	}
	t.Logf("%d builds checked, %d with subgraph nodes", checked, clusters)
}

func TestTreeSpecsLabelsEachTree(t *testing.T) {
	build := parseCorpusBuild(t, "0280-elementalist-fireball.xml")
	rows := TreeSpecs(build)
	if len(rows) != 3 {
		t.Fatalf("%d rows", len(rows))
	}
	want := []string{"Until Lab", "Lab 2", "White maps"}
	for i, r := range rows {
		if r.Label != want[i] || r.ClassName != "Witch" || r.AscendClassName != "Elementalist" {
			t.Errorf("row %d = %+v", i, r)
		}
	}
	build.Tree.Specs[0].TreeVersion = "3_25_ruthless_alternate"
	if got := TreeSpecs(build)[0].Label; got != "[3.25 (ruthless alternate)] Until Lab" {
		t.Errorf("older tree label = %q", got)
	}
}

func TestNewTreeSpecStartsAtTheClassAndAscendancyStart(t *testing.T) {
	build := parseCorpusBuild(t, "0280-elementalist-fireball.xml")
	tree := data.TreeVersions[data.LatestTreeVersion].Tree()
	NewTreeSpec(build, "Fresh")
	if len(build.Tree.Specs) != 4 {
		t.Fatalf("%d specs", len(build.Tree.Specs))
	}
	s := build.Tree.Specs[3]
	if pob.SpecTitle(s) != "Fresh" || s.ClassID != pob.Witch || s.AscendClassID != 2 || s.TreeVersion != data.LatestTreeVersion {
		t.Fatalf("new spec = %+v", s)
	}
	ids := pob.SpecNodes(s)
	if len(ids) != 2 {
		t.Fatalf("nodes = %v, want the class and ascendancy starts", ids)
	}
	for _, id := range ids {
		n := tree.Nodes[strconv.FormatInt(id, 10)]
		if !isClassStartNode(n) && !(isAscendClassStartNode(n) && *n.AscendancyName == "Elementalist") {
			t.Errorf("node %d is not a start node", id)
		}
	}
	if p := TreeSpecsPoints(build)[3]; p.Used != 0 || p.AscendUsed != 0 {
		t.Errorf("a new tree uses %d / %d points", p.Used, p.AscendUsed)
	}
	if build.Tree.ActiveSpec != 2 {
		t.Errorf("New switched the active tree to %d", build.Tree.ActiveSpec)
	}
}
