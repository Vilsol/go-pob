package data_test

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/data/raw"
)

// TestDependentNodesMasteryOtherPath: PoB's node.depends of 42917 on 0215.
func TestDependentNodesMasteryOtherPath(t *testing.T) {
	v := data.TreeVersions[data.LatestTreeVersion]
	nodes := v.Tree().Nodes
	xml, err := os.ReadFile(filepath.Join("..", "testdata", "corpus-"+raw.LatestVersion, "0215-slayer-cyclone-of-tumult.xml"))
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	build, err := builds.ParseBuild(xml)
	if err != nil {
		t.Fatal(err)
	}
	spec := build.Tree.Specs[max(build.Tree.ActiveSpec, 1)-1]
	var active []int64
	for _, s := range strings.Split(spec.NodesAttr, ",") {
		if id, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64); err == nil {
			active = append(active, id)
		}
	}
	var roots []int64
	for _, n := range nodes {
		if n.ClassStartIndex != nil && *n.ClassStartIndex == int64(spec.ClassID) {
			for _, key := range append(slices.Clone(n.Out), n.In...) {
				if o, ok := nodes[key]; ok && o.Skill != nil && o.AscendancyName == nil {
					roots = append(roots, *o.Skill)
				}
			}
		}
	}
	if !slices.Contains(active, 62416) || !slices.Contains(active, 42917) {
		t.Fatalf("REFUSING: 0215 no longer allocates 62416 and 42917")
	}
	got := v.CalculateDependentNodes(active, 42917, roots)
	slices.Sort(got)
	if want := []int64{16754, 39761, 48282, 51559}; !slices.Equal(got, want) {
		t.Errorf("depends of 42917 = %v, PoB %v", got, want)
	}
}
