package mcp

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/data"
)

// TestDeallocateWithoutDependentsSaysWhatPoBDrops: PoB's spec drops nodes cut off from the class start
// (PassiveSpec.lua:311), so with_dependents:false cannot keep them; the edit says how many it leaves cut off.
func TestDeallocateWithoutDependentsSaysWhatPoBDrops(t *testing.T) {
	s := testServer(t)
	h := call(t, s, "new_build", map[string]any{"class": "Templar"})["handle"].(string)
	call(t, s, "edit_tree", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "allocate", "node": "34098"})})
	hd, _ := s.get(h)
	b, _ := hd.parseRev(hd.cur)
	v := specTreeVersion(b)
	starts := startNodes(b, v)
	var first int64
	for _, id := range b.Build.PassiveNodes {
		if !slices.Contains(starts, id) && len(data.TreeVersions[v].CalculateDependentNodes(slices.Clone(b.Build.PassiveNodes), id, classRoots(b, v))) > 0 {
			first = id
			break
		}
	}
	res := call(t, s, "edit_tree", map[string]any{"handle": h, "dry_run": true, "ops": ops(map[string]any{"op": "deallocate", "node": first, "with_dependents": false})})
	if !strings.Contains(fmt.Sprint(res["applied"]), "cut off") {
		t.Fatalf("deallocate %d without dependents: %v", first, res["applied"])
	}
}
