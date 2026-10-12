package calculator

import (
	"slices"
	"strconv"
	"testing"

	"github.com/Vilsol/go-pob/pob"
)

func hasNode(build *pob.PathOfBuilding, id int64) bool {
	return slices.Contains(build.Build.PassiveNodes, id)
}

func startOf(t *testing.T, class pob.ClassName) int64 {
	t.Helper()
	classID, _ := pob.ClassIDByName(class)
	id, err := strconv.ParseInt(classStartNodeID(specTree(), classID), 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func ascendancyNodes(build *pob.PathOfBuilding) int {
	tree := specTree()
	n := 0
	for _, id := range build.Build.PassiveNodes {
		if node, ok := tree.Nodes[nodeKey(id)]; ok && node.AscendancyName != nil && *node.AscendancyName != "" {
			n++
		}
	}
	return n
}

// TestChangeClassAsksThenResets: A full Witch tree is not connected to the Marauder's start.
func TestChangeClassAsksThenResets(t *testing.T) {
	build := parseCorpusBuild(t, "0243-necromancer-raise-spectre.xml")
	before := slices.Clone(build.Build.PassiveNodes)
	if got := ChangeClass(build, "Marauder", ClassAsk); !got.NeedsConfirm || got.Changed || got.ClassName != "Marauder" {
		t.Fatalf("ask: %+v", got)
	}
	if build.Build.ClassName != "Witch" || !slices.Equal(build.Build.PassiveNodes, before) {
		t.Fatalf("asking changed the build: %s, %d nodes", build.Build.ClassName, len(build.Build.PassiveNodes))
	}
	if got := ChangeClass(build, "Marauder", ClassContinue); !got.Changed {
		t.Fatalf("continue: %+v", got)
	}
	if build.Build.ClassName != "Marauder" || build.Build.AscendClassName != "None" {
		t.Fatalf("class %s / %s", build.Build.ClassName, build.Build.AscendClassName)
	}
	if want := []int64{startOf(t, "Marauder")}; !slices.Equal(build.Build.PassiveNodes, want) {
		t.Fatalf("nodes after reset = %v, want %v", build.Build.PassiveNodes, want)
	}
	if spec := build.Tree.Specs[build.ActiveSpecIndex()]; spec.NodesAttr != strconv.FormatInt(startOf(t, "Marauder"), 10) {
		t.Fatalf("spec nodes %q", spec.NodesAttr)
	}
}

// TestChangeClassConnectPathKeepsTheTree: Connect Path allocates a path to the new class's start and keeps the whole tree.
func TestChangeClassConnectPathKeepsTheTree(t *testing.T) {
	build := parseCorpusBuild(t, "0243-necromancer-raise-spectre.xml")
	used := CountAllocNodes(build)
	if ascendancyNodes(build) == 0 {
		t.Fatal("fixture has no ascendancy nodes")
	}
	if IsClassConnected(build, pob.Shadow) {
		t.Skip("fixture already reaches the Shadow")
	}
	if got := ChangeClass(build, "Shadow", ClassConnect); !got.Changed {
		t.Fatalf("connect: %+v", got)
	}
	if build.Build.ClassName != "Shadow" || !hasNode(build, startOf(t, "Shadow")) || hasNode(build, startOf(t, "Witch")) {
		t.Fatalf("start nodes: class %s, %v", build.Build.ClassName, build.Build.PassiveNodes)
	}
	if ascendancyNodes(build) != 0 {
		t.Fatalf("%d ascendancy nodes kept after the ascendancy reset", ascendancyNodes(build))
	}
	connected := CountAllocNodes(build)
	if connected <= used {
		t.Fatalf("points %d after connecting, %d before", connected, used)
	}

	if !IsClassConnected(build, pob.Witch) {
		t.Fatal("the tree is not connected back to the Witch")
	}
	if got := ChangeClass(build, "Witch", ClassAsk); !got.Changed {
		t.Fatalf("connected change asked: %+v", got)
	}
	if CountAllocNodes(build) != connected || !hasNode(build, startOf(t, "Witch")) {
		t.Fatalf("points %d after a connected change, want %d", CountAllocNodes(build), connected)
	}
}

// TestChangeClassEmptyTreeNeedsNoConfirm: An empty tree changes class without asking.
func TestChangeClassEmptyTreeNeedsNoConfirm(t *testing.T) {
	build := parseCorpusBuild(t, "0243-necromancer-raise-spectre.xml")
	build.DeallocateNodes(slices.Clone(build.Build.PassiveNodes))
	if got := ChangeClass(build, "Duelist", ClassAsk); !got.Changed || build.Build.ClassName != "Duelist" {
		t.Fatalf("%+v, class %s", got, build.Build.ClassName)
	}
}

// TestSelectAscendClassPrunesTheOldAscendancy: Switching ascendancy prunes the old one's nodes; the main tree stays.
func TestSelectAscendClassPrunesTheOldAscendancy(t *testing.T) {
	build := parseCorpusBuild(t, "0243-necromancer-raise-spectre.xml")
	used := CountAllocNodes(build)
	SelectAscendClass(build, "Occultist")
	if build.Build.AscendClassName != "Occultist" || ascendancyNodes(build) != 1 {
		t.Fatalf("%s with %d ascendancy nodes", build.Build.AscendClassName, ascendancyNodes(build))
	}
	if CountAllocNodes(build) != used {
		t.Fatalf("main tree %d, was %d", CountAllocNodes(build), used)
	}
}

// TestClickAscendancyNode: Clicking a node of another of the class's ascendancies switches to it and allocates its path.
func TestClickAscendancyNode(t *testing.T) {
	tree := specTree()
	pick := func(ascendancy string) int64 {
		for _, n := range tree.Nodes {
			if n.AscendancyName != nil && *n.AscendancyName == ascendancy && !isAscendClassStartNode(n) && n.Skill != nil &&
				(n.IsMultipleChoiceOption == nil || !*n.IsMultipleChoiceOption) && n.IsNotable != nil && *n.IsNotable {
				return *n.Skill
			}
		}
		t.Fatalf("no %s notable", ascendancy)
		return 0
	}

	build := parseCorpusBuild(t, "0243-necromancer-raise-spectre.xml")
	used := CountAllocNodes(build)
	elem := pick("Elementalist")
	if got := ClickAscendancyNode(build, elem, ClassAsk); !got.Changed || !got.Handled {
		t.Fatalf("same class: %+v", got)
	}
	if build.Build.AscendClassName != "Elementalist" || !hasNode(build, elem) || CountAllocNodes(build) != used {
		t.Fatalf("%s, clicked allocated %v, points %d / %d", build.Build.AscendClassName, hasNode(build, elem), CountAllocNodes(build), used)
	}

	jugg := pick("Juggernaut")
	if got := ClickAscendancyNode(build, jugg, ClassAsk); !got.NeedsConfirm || !got.Handled || got.ClassName != "Marauder" {
		t.Fatalf("other class: %+v", got)
	}
	if got := ClickAscendancyNode(build, jugg, ClassContinue); !got.Changed {
		t.Fatalf("continue: %+v", got)
	}
	if build.Build.ClassName != "Marauder" || build.Build.AscendClassName != "Juggernaut" || !hasNode(build, jugg) || CountAllocNodes(build) != 0 {
		t.Fatalf("%s / %s, clicked %v, points %d", build.Build.ClassName, build.Build.AscendClassName, hasNode(build, jugg), CountAllocNodes(build))
	}

	if got := ClickAscendancyNode(build, startOf(t, "Witch"), ClassAsk); got.Handled {
		t.Fatalf("main tree node handled: %+v", got)
	}
}

// TestPruneOrphansKeepsTreesPoBSaved: PoB prunes a tree when it loads it.
func TestPruneOrphansKeepsTreesPoBSaved(t *testing.T) {
	_, names := corpusNamesForLoadedData(t)
	checked := 0
	for i, name := range names {
		if i%20 != 0 {
			continue
		}
		build := parseCorpusBuild(t, name)
		if build.ActiveSpecIndex() < 0 {
			continue
		}
		before := len(build.Build.PassiveNodes)
		pruneOrphans(build)
		if after := len(build.Build.PassiveNodes); after != before {
			t.Errorf("%s: %d of %d nodes pruned", name, before-after, before)
		}
		checked++
	}
	if checked < 500 {
		t.Fatalf("only %d builds checked", checked)
	}
}

// TestWardenIsTheRangersAscendancy: The Ranger's Warden carries the ascendancy id "Raider"; "Warden" is a legacy alternate's id, whose nodes PoB drops.
func TestWardenIsTheRangersAscendancy(t *testing.T) {
	build := parseCorpusBuild(t, "0243-necromancer-raise-spectre.xml")
	ChangeClass(build, "Ranger", ClassContinue)
	SelectAscendClass(build, "Warden")
	tree := specTree()
	var starts []string
	for _, id := range build.Build.PassiveNodes {
		if n := tree.Nodes[nodeKey(id)]; isAscendClassStartNode(n) {
			starts = append(starts, *n.AscendancyName)
		}
	}
	if !slices.Equal(starts, []string{"Raider"}) {
		t.Fatalf("ascendancy starts %v", starts)
	}
	NewTreeSpec(build, "fresh")
	if nodes := pob.SpecNodes(build.Tree.Specs[len(build.Tree.Specs)-1]); len(nodes) != 2 || tree.Nodes[nodeKey(nodes[1])].AscendancyName == nil || *tree.Nodes[nodeKey(nodes[1])].AscendancyName != "Raider" {
		t.Fatalf("a new Warden tree starts at %v", nodes)
	}
}
