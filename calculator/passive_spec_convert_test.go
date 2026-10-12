package calculator

import (
	"slices"
	"testing"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/pob"
)

// TestConvertSpecCopyKeepsATreeOnTheSameVersion: Copy + Convert of a tree PoB saved on the latest version keeps every node.
func TestConvertSpecCopyKeepsATreeOnTheSameVersion(t *testing.T) {
	build := parseCorpusBuild(t, "0005-occultist-vaal-double-strike.xml")
	i := build.ActiveSpecIndex()
	before := build.Tree.Specs[i]
	n := len(build.Tree.Specs)
	if err := ConvertSpecToVersion(build, data.LatestTreeVersion, false); err != nil {
		t.Fatal(err)
	}
	if len(build.Tree.Specs) != n+1 || build.Tree.ActiveSpec != i+2 {
		t.Fatalf("%d trees (was %d), active %d", len(build.Tree.Specs), n, build.Tree.ActiveSpec)
	}
	got := build.Tree.Specs[i+1]
	if pob.SpecTitle(got) != pob.SpecTitle(before) || got.TreeVersion != data.LatestTreeVersion || len(got.Sockets) != len(before.Sockets) || got.MasteryEffects == "" {
		t.Fatalf("converted %q %s, %d sockets", pob.SpecTitle(got), got.TreeVersion, len(got.Sockets))
	}
	if !slices.Equal(sortedIDs(pob.SpecNodes(got)), sortedIDs(pob.SpecNodes(before))) {
		t.Fatalf("%d nodes, the tree had %d", len(pob.SpecNodes(got)), len(pob.SpecNodes(before)))
	}
	if !slices.Equal(pob.SpecNodes(build.Tree.Specs[i]), pob.SpecNodes(before)) {
		t.Fatal("the original tree changed")
	}
}

// TestConvertReplacesAndConvertAll: Convert replaces the tree in place; "Convert all" puts every tree on the version and keeps the active one active.
func TestConvertReplacesAndConvertAll(t *testing.T) {
	build := parseCorpusBuild(t, "0004-trickster-lightning-tendrils-of-eccentricity.xml")
	n := len(build.Tree.Specs)
	build.SetActiveSpec(2)
	title := pob.SpecTitle(build.Tree.Specs[1])
	if build.Tree.Specs[1].TreeVersion == data.LatestTreeVersion {
		t.Fatal("fixture's second tree is already on the latest version")
	}
	if err := ConvertSpecToVersion(build, data.LatestTreeVersion, true); err != nil {
		t.Fatal(err)
	}
	if len(build.Tree.Specs) != n || build.Tree.ActiveSpec != 2 || build.Tree.Specs[1].TreeVersion != data.LatestTreeVersion || pob.SpecTitle(build.Tree.Specs[1]) != title {
		t.Fatalf("%d trees, active %d, tree 2 on %s", len(build.Tree.Specs), build.Tree.ActiveSpec, build.Tree.Specs[1].TreeVersion)
	}
	build.SetActiveSpec(3)
	if err := ConvertAllSpecsToVersion(build, data.LatestTreeVersion); err != nil {
		t.Fatal(err)
	}
	for k, s := range build.Tree.Specs {
		if s.TreeVersion != data.LatestTreeVersion {
			t.Errorf("tree %d on %s", k+1, s.TreeVersion)
		}
	}
	if len(build.Tree.Specs) != n || build.Tree.ActiveSpec != 3 {
		t.Fatalf("%d trees, active %d", len(build.Tree.Specs), build.Tree.ActiveSpec)
	}
}
