package mcp

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/pob"
)

func TestAscendancyNamesNeverResolveToAnotherAscendancysNode(t *testing.T) {
	s := testServer(t)
	type row struct {
		class pob.ClassName
		asc   pob.AscendancyName
	}
	var rows []row
	for id, ascs := range pob.ClassAscendancyNames() {
		for _, a := range ascs {
			rows = append(rows, row{pob.ClassNameByID[id], a})
		}
	}
	for _, r := range rows {
		t.Run(string(r.asc), func(t *testing.T) {
			h := call(t, s, "new_build", map[string]any{"class": string(r.class)})["handle"].(string)
			out, e := callRaw(t, s, "edit_tree", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "allocate", "node": string(r.asc)})})
			if e == nil {
				res := map[string]any{}
				_ = json.Unmarshal(out, &res)
				b, _ := s.handles[h].parseRev(1)
				for _, id := range b.Build.PassiveNodes {
					if n, ok := treeNodeByID(data.LatestTreeVersion, id); ok && n.Ascendancy != "" {
						t.Fatalf("allocate %q took %s's node %d: %v", r.asc, n.Ascendancy, id, res["applied"])
					}
				}
			} else if e.Code != CodeAmbiguous || !slices.ContainsFunc(e.Candidates, func(c any) bool {
				m, _ := c.(map[string]any)
				return m["op"] == "set_ascendancy" && m["ascendancy"] == string(r.asc)
			}) {
				t.Fatalf("allocate %q: %s %s %v", r.asc, e.Code, e.Message, e.Candidates)
			}

			res := call(t, s, "edit_tree", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "set_ascendancy", "ascendancy": string(r.asc)})})
			b, _ := s.handles[h].parseRev(int(res["revision"].(float64)))
			if b.Build.AscendClassName != r.asc {
				t.Fatalf("set_ascendancy %q gave %s", r.asc, b.Build.AscendClassName)
			}

			w := call(t, s, "new_build", map[string]any{"class": "Witch"})["handle"].(string)
			if r.class == "Witch" {
				w = call(t, s, "new_build", map[string]any{"class": "Marauder"})["handle"].(string)
			}
			res = call(t, s, "edit_tree", map[string]any{"handle": w, "ops": ops(map[string]any{"op": "set_class", "class": string(r.asc)})})
			b, _ = s.handles[w].parseRev(int(res["revision"].(float64)))
			if b.Build.ClassName != r.class || b.Build.AscendClassName != r.asc {
				t.Fatalf("set_class %q gave %s/%s", r.asc, b.Build.ClassName, b.Build.AscendClassName)
			}
		})
	}
}

func TestAscendantSameNamedNodesAreAmbiguous(t *testing.T) {
	s := testServer(t)
	h := call(t, s, "new_build", map[string]any{"class": "Scion", "ascendancy": "Ascendant"})["handle"].(string)
	e := callErr(t, s, "edit_tree", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "allocate", "node": "Juggernaut"})})
	if e.Code != CodeAmbiguous || len(e.Candidates) < 2 {
		t.Fatalf("Juggernaut on an Ascendant: %s %s %v", e.Code, e.Message, e.Candidates)
	}
	res := call(t, s, "edit_tree", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "allocate", "node": "Berserker"})})
	if fmt.Sprint(res["applied"]) == "[]" {
		t.Fatal("an Ascendant could not take its Berserker node")
	}
}

func TestDeallocatingAnUnallocatedNodeIsANoOp(t *testing.T) {
	s := testServer(t)
	h := call(t, s, "new_build", map[string]any{"class": "Templar"})["handle"].(string)
	res := call(t, s, "edit_tree", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "deallocate", "node": "Heart of Oak"})})
	applied := fmt.Sprint(res["applied"])
	if !strings.Contains(applied, "Heart of Oak") || !strings.Contains(applied, "not allocated") || points(res)["used"] != float64(0) {
		t.Fatalf("deallocating an unallocated node: applied %v, used %v", applied, points(res)["used"])
	}
}
