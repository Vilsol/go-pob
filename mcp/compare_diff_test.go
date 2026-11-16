package mcp

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/pob"
)

// TestGemEditsDuplicateGems: three Raise Spectre gems at levels 20, 21 and 20 compared with themselves, or reordered,
// have no edits; raising only the third names one level change; a fourth copy is one added gem.
func TestGemEditsDuplicateGems(t *testing.T) {
	gems := func(levels ...int) []pob.Gem {
		out := make([]pob.Gem, 0, len(levels))
		for _, l := range levels {
			out = append(out, pob.Gem{NameSpec: "Raise Spectre", Level: l, Quality: 20, Enabled: true})
		}
		return out
	}
	names := func(g []pob.Gem) []string {
		out := make([]string, 0, len(g))
		for _, x := range g {
			out = append(out, x.NameSpec)
		}
		return out
	}
	for _, b := range [][]pob.Gem{gems(20, 21, 20), gems(21, 20, 20)} {
		if e := gemEdits(gems(20, 21, 20), b); len(e) != 0 {
			t.Fatalf("false edits: %v", e)
		}
	}
	if e := gemEdits(gems(20, 21, 20), gems(20, 21, 21)); fmt.Sprint(e) != "[Raise Spectre level 20 -> 21]" {
		t.Fatalf("edits = %v", e)
	}
	a, b := names(gems(20, 21, 20)), names(gems(20, 21, 20, 20))
	if add, rem := gemNameDiff(b, a), gemNameDiff(a, b); fmt.Sprint(add) != "[Raise Spectre]" || len(rem) != 0 {
		t.Fatalf("added %v removed %v", add, rem)
	}
}

// TestCompareBuildsSocketRecolour: recolouring a body armour's sockets is its own item change, named in the summary.
func TestCompareBuildsSocketRecolour(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	body := "Rarity: RARE\nDoom Coat\nAstral Plate\nSockets: %s\n+90 to maximum Life"
	for _, sockets := range []string{"R-R-G", "B-B-B"} {
		call(t, s, "edit_items", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "equip_text", "slot": "Body Armour", "text": fmt.Sprintf(body, sockets)})})
	}
	res := call(t, s, "compare_builds", map[string]any{"a": h + "@1", "b": h})
	items := res["structure"].(map[string]any)["items_changed"].([]any)
	if len(items) != 1 || items[0].(map[string]any)["sockets"] != "R-R-G -> B-B-B" {
		t.Fatalf("items_changed = %v", items)
	}
	steps := asList(call(t, s, "compare_builds", map[string]any{"a": h + "@1", "b": h, "steps": true})["steps"])
	if sum := steps[0].(map[string]any)["changes"].(string); !strings.Contains(sum, "sockets R-R-G -> B-B-B") {
		t.Fatalf("summary = %s", sum)
	}
}
