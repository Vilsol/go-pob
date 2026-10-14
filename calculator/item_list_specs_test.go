package calculator

import (
	"encoding/json"
	"os"
	"slices"
	"strconv"
	"testing"

	"github.com/Vilsol/go-pob/pob"
)

// TestDeleteJewelEveryTreeAgainstPoB deletes each jewel socketed in an allocated socket of any tree.
func TestDeleteJewelEveryTreeAgainstPoB(t *testing.T) {
	initItemsTabData(t)
	raw, err := os.ReadFile("../testdata/items-ui/item_delete_tree_specs.json")
	if err != nil {
		t.Fatalf("REFUSING: testdata/items-ui/item_delete_tree_specs.json: %v", err)
	}
	var want map[string]struct {
		Active int
		Before [][]int64
		After  map[string][][]int64
	}
	if err := json.Unmarshal(raw, &want); err != nil || len(want) < 2 {
		t.Fatalf("REFUSING: the fixture does not decode: %v", err)
	}
	specNodes := func(build *pob.PathOfBuilding) [][]int64 {
		out := make([][]int64, len(build.Tree.Specs))
		for k, spec := range build.Tree.Specs {
			nodes := pob.SpecNodes(spec)
			if k == build.ActiveSpecIndex() {
				nodes = build.Build.PassiveNodes
			}
			out[k] = sortedNodeIDs(nodes)
		}
		return out
	}
	known := map[string]bool{"0526-luminary-freezing-pulse.xml 92 4": true}
	cases, inactive := 0, 0
	for file, w := range want {
		build := loadCorpusBuild(t, file)
		if build.ActiveSpecIndex() != w.Active-1 {
			t.Fatalf("%s: our active tree is %d, the pin's %d", file, build.ActiveSpecIndex()+1, w.Active)
		}
		for k, before := range specNodes(build) {
			if !slices.Equal(before, w.Before[k]) {
				t.Fatalf("%s tree %d: ours loads %d nodes, the pin's %d", file, k+1, len(before), len(w.Before[k]))
			}
		}
		for id, after := range w.After {
			cases++
			build := loadCorpusBuild(t, file)
			n, _ := strconv.Atoi(id)
			if !DeleteItems(build, []int{n}) {
				t.Fatalf("%s: item %d not deleted", file, n)
			}
			for k, got := range specNodes(build) {
				if k != w.Active-1 && !slices.Equal(w.Before[k], after[k]) {
					inactive++
				}
				key := file + " " + id + " " + strconv.Itoa(k+1)
				if known[key] {
					if slices.Equal(got, after[k]) {
						t.Errorf("%s: a known divergence now matches the pin; drop it", key)
					}
					continue
				}
				if !slices.Equal(got, after[k]) {
					t.Errorf("%s delete %s, tree %d: ours drops %v and keeps %v that the pin drops", file, id, k+1, setMinus(after[k], got), setMinus(got, after[k]))
				}
			}
		}
	}
	if cases < 40 || inactive < 20 {
		t.Fatalf("graded only %d deletes, %d of them changing an inactive tree", cases, inactive)
	}
}
