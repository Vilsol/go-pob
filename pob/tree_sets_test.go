package pob

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"testing"
)

func title(s string) *string { return &s }

func threeSpecs() *PathOfBuilding {
	return &PathOfBuilding{
		Build: Build{ClassName: "Witch", AscendClassName: "Elementalist", PassiveNodes: []int64{54447, 1, 2}},
		Tree: Tree{ActiveSpec: 2, Specs: []Spec{
			{Title: title("Until Lab"), ClassID: Witch, AscendClassID: 2, NodesAttr: "54447,7"},
			{Title: title("Lab 2"), ClassID: Witch, AscendClassID: 2, NodesAttr: "54447,1,2", MasteryEffects: "{63268,6216}",
				Sockets: []Socket{{NodeID: 26725, ItemID: 4}}, Overrides: []Override{{NodeID: 3, DN: "Acrobatics"}}},
			{ClassID: Marauder, AscendClassID: 1, NodesAttr: "47175,9,10"},
		}},
	}
}

func TestSetActiveSpecLoadsItsClassAndNodes(t *testing.T) {
	b := threeSpecs()
	b.SetActiveSpec(3)
	if b.Tree.ActiveSpec != 3 {
		t.Fatalf("ActiveSpec = %d, want 3", b.Tree.ActiveSpec)
	}
	if b.Build.ClassName != "Marauder" || b.Build.AscendClassName != "Juggernaut" {
		t.Fatalf("class = %s / %s, want Marauder / Juggernaut", b.Build.ClassName, b.Build.AscendClassName)
	}
	if got := strconv.Itoa(len(b.Build.PassiveNodes)); got != "3" || b.Build.PassiveNodes[0] != 47175 {
		t.Fatalf("PassiveNodes = %v, want the spec's 47175,9,10", b.Build.PassiveNodes)
	}
	b.SetActiveSpec(9)
	if b.Tree.ActiveSpec != 3 {
		t.Fatalf("ActiveSpec = %d, want 3 (m_min(specId, #specList))", b.Tree.ActiveSpec)
	}
	b.SetActiveSpec(1)
	if b.Build.ClassName != "Witch" || b.Build.AscendClassName != "Elementalist" || len(b.Build.PassiveNodes) != 2 {
		t.Fatalf("tree 1: %s / %s %v", b.Build.ClassName, b.Build.AscendClassName, b.Build.PassiveNodes)
	}
}

// TestAllocationStaysWithItsSpec: An allocation lands in the active spec only, so switching away and back keeps both trees.
func TestAllocationStaysWithItsSpec(t *testing.T) {
	b := threeSpecs()
	b.AllocateNodes([]int64{5})
	b.SetActiveSpec(1)
	b.SetActiveSpec(2)
	if got := b.Tree.Specs[1].NodesAttr; got != "54447,1,2,5" {
		t.Fatalf("spec 2 nodes = %q", got)
	}
	if got := b.Tree.Specs[0].NodesAttr; got != "54447,7" {
		t.Fatalf("spec 1 nodes = %q, changed by an allocation on spec 2", got)
	}
	if len(b.Build.PassiveNodes) != 4 {
		t.Fatalf("PassiveNodes = %v after switching back", b.Build.PassiveNodes)
	}
}

func TestCopySpecIsADeepCopyAtTheEnd(t *testing.T) {
	b := threeSpecs()
	b.CopySpec(2, "Lab 2 copy")
	if len(b.Tree.Specs) != 4 || SpecTitle(b.Tree.Specs[3]) != "Lab 2 copy" {
		t.Fatalf("specs = %d, last %q", len(b.Tree.Specs), SpecTitle(b.Tree.Specs[3]))
	}
	c := b.Tree.Specs[3]
	if c.NodesAttr != "54447,1,2" || c.MasteryEffects != "{63268,6216}" || c.ClassID != Witch || c.AscendClassID != 2 {
		t.Fatalf("copy = %+v", c)
	}
	c.Sockets[0].ItemID = 9
	c.Overrides[0].DN = "x"
	if b.Tree.Specs[1].Sockets[0].ItemID != 4 || b.Tree.Specs[1].Overrides[0].DN != "Acrobatics" {
		t.Fatal("the copy shares its jewels or tattoos with the original")
	}
	if b.Tree.ActiveSpec != 2 {
		t.Fatalf("copying changed the active tree to %d", b.Tree.ActiveSpec)
	}
}

func TestRenameSpec(t *testing.T) {
	b := threeSpecs()
	b.RenameSpec(3, "Maps")
	if SpecTitle(b.Tree.Specs[2]) != "Maps" || SpecTitle(b.Tree.Specs[0]) != "Until Lab" {
		t.Fatalf("titles %q %q", SpecTitle(b.Tree.Specs[2]), SpecTitle(b.Tree.Specs[0]))
	}
}

// TestDeleteSpecMatchesPoB: delete_spec.tsv is the pinned PoB's own OnSelDelete run over every delete in lists up to 5 long.
func TestDeleteSpecMatchesPoB(t *testing.T) {
	f, err := os.Open("testdata/delete_spec.tsv")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rows := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		cols := strings.Split(scanner.Text(), "\t")
		if len(cols) != 5 {
			t.Fatalf("bad row %q", scanner.Text())
		}
		n, _ := strconv.Atoi(cols[0])
		active, _ := strconv.Atoi(cols[1])
		deleted, _ := strconv.Atoi(cols[2])
		b := &PathOfBuilding{Tree: Tree{ActiveSpec: active}}
		for i := 1; i <= n; i++ {
			b.Tree.Specs = append(b.Tree.Specs, Spec{Title: title(strconv.Itoa(i))})
		}
		b.DeleteSpec(deleted)
		order := make([]string, 0, n)
		for _, s := range b.Tree.Specs {
			order = append(order, SpecTitle(s))
		}
		if got := strings.Join(order, ","); got != cols[3] || strconv.Itoa(b.Tree.ActiveSpec) != cols[4] {
			t.Errorf("n=%d active=%d delete %d: order %s active %d, PoB %s active %s", n, active, deleted, got, b.Tree.ActiveSpec, cols[3], cols[4])
		}
		rows++
	}
	if rows != 55 {
		t.Fatalf("read %d rows, want 55", rows)
	}
}

// TestSetClassAndAscendancyReachTheActiveSpec: PoB saves and loads the class from the spec (PassiveSpec.lua:200-201), so the class select must reach it.
func TestSetClassAndAscendancyReachTheActiveSpec(t *testing.T) {
	b := threeSpecs()
	b.SetClass("Marauder")
	if s := b.Tree.Specs[1]; s.ClassID != Marauder || s.AscendClassID != 0 {
		t.Fatalf("after SetClass: classId %d ascendClassId %d, want 1 / 0", s.ClassID, s.AscendClassID)
	}
	b.SetAscendancy("Chieftain")
	if s := b.Tree.Specs[1]; s.AscendClassID != 3 {
		t.Fatalf("after SetAscendancy: ascendClassId %d, want 3", s.AscendClassID)
	}
	b.SetAscendancy("None")
	if s := b.Tree.Specs[1]; s.AscendClassID != 0 {
		t.Fatalf("after SetAscendancy(None): ascendClassId %d, want 0", s.AscendClassID)
	}
	if b.Tree.Specs[0].ClassID != Witch {
		t.Fatal("SetClass changed an inactive spec")
	}
}

// TestCopyCompareSpecCopiesTheComparedTree: CompareTab.lua:1331-1368: the compared build's active tree joins this build's trees as "<title> (Compared)".
func TestCopyCompareSpecCopiesTheComparedTree(t *testing.T) {
	a := threeSpecs()
	b := threeSpecs()
	b.Tree.ActiveSpec = 2
	a.CopyCompareSpec(b, false)
	if len(a.Tree.Specs) != 4 || a.Tree.ActiveSpec != 2 {
		t.Fatalf("%d trees, active %d", len(a.Tree.Specs), a.Tree.ActiveSpec)
	}
	got := a.Tree.Specs[3]
	if SpecTitle(got) != "Lab 2 (Compared)" || got.NodesAttr != "54447,1,2" || got.MasteryEffects != "{63268,6216}" || got.ClassID != Witch {
		t.Fatalf("copied %+v", got)
	}
	if len(got.Sockets) != 0 || len(got.Overrides) != 1 {
		t.Fatalf("jewels %v, tattoos %v", got.Sockets, got.Overrides)
	}
	b.Tree.ActiveSpec = 3
	a.CopyCompareSpec(b, true)
	if a.Tree.ActiveSpec != 5 || SpecTitle(a.Tree.Specs[4]) != "Default (Compared)" || a.Build.ClassName != "Marauder" {
		t.Fatalf("and use: active %d, %q, %s", a.Tree.ActiveSpec, SpecTitle(a.Tree.Specs[4]), a.Build.ClassName)
	}
}
