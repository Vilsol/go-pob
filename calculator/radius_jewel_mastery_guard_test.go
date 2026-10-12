package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// TestRadiusJewelSkipsMasteryNodes pins CalcSetup.lua:126 and :145 — the `rad.nodes[node.id].type ~= "Mastery"` guard that sits on BOTH radius-jewel passes.
func TestRadiusJewelSkipsMasteryNodes(t *testing.T) {
	yes := true

	run := func(t *testing.T, funcType string, node data.Node) int {
		t.Helper()
		env := &Environment{
			AllocatedNodes: map[string]data.Node{"7": node},
			Spec:           &PassiveSpec{TreeVersion: data.LatestTreeVersion},
		}
		calls := 0
		jewel := &radiusJewel{
			nodes:    map[string]bool{"7": true},
			funcType: funcType,
			nodeID:   "7",
			data:     &radiusJewelData{},
			fn: func(_ *radiusJewelNode, out *moddb.ModList, _ *radiusJewelData) {
				calls++
				out.AddMod(mod.NewFloat("Life", mod.TypeBase, 100).Source("Jewel"))
			},
		}
		buildModListForConqueredNode(env, "7", node, nodeRewrite{}, []*radiusJewel{jewel})
		return calls
	}

	mastery := data.Node{IsMastery: &yes}
	notable := data.Node{IsNotable: &yes}

	for _, funcType := range []string{"Other", "Threshold"} {
		t.Run(funcType+"/mastery is skipped", func(t *testing.T) {
			if got := run(t, funcType, mastery); got != 0 {
				t.Errorf("jewel func ran %d times on a Mastery node, want 0 — CalcSetup.lua:%s",
					got, map[string]string{"Other": "126", "Threshold": "145"}[funcType])
			}
		})
		t.Run(funcType+"/notable still runs", func(t *testing.T) {
			if got := run(t, funcType, notable); got != 1 {
				t.Errorf("jewel func ran %d times on a Notable node, want 1 — the guard is on "+
					"Mastery alone, not on the pass", got)
			}
		})
	}
}
