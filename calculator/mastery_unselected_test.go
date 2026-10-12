package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

const lifeMasteryNode = "4139"

func masteryTestEnv(selections map[int64]int64) *Environment {
	if selections == nil {
		selections = map[int64]int64{}
	}
	return &Environment{Spec: &PassiveSpec{
		TreeVersion:       data.TreeVersion3_18,
		MasterySelections: selections,
	}}
}

func countMods(ml *moddb.ModList, name string, modType mod.Type, value float64) int {
	n := 0
	for _, m := range ml.Mods() {
		if m.Name() != name || m.Type() != modType {
			continue
		}
		if v := m.Value(); v != nil && v.Float() == value {
			n++
		}
	}
	return n
}

// TestUnselectedMasteryContributesAllEffects pins PassiveSpec.lua:860-861.
func TestUnselectedMasteryContributesAllEffects(t *testing.T) {
	tree := data.TreeVersions[data.TreeVersion3_18].Tree()
	node, ok := tree.Nodes[lifeMasteryNode]
	if !ok {
		t.Fatalf("node %s missing from the 3_18 tree", lifeMasteryNode)
	}
	if node.IsMastery == nil || !*node.IsMastery {
		t.Fatalf("node %s is not a mastery", lifeMasteryNode)
	}
	if len(node.MasteryEffects) < 2 {
		t.Fatalf("node %s carries %d effects, want several", lifeMasteryNode, len(node.MasteryEffects))
	}

	ml := buildModListForNode(masteryTestEnv(nil), lifeMasteryNode, node)

	if n := countMods(ml, "Life", mod.TypeBase, 50); n != 1 {
		t.Errorf("Life BASE 50 (+50 to maximum Life effect): got %d, want 1", n)
	}
	if n := countMods(ml, "Life", mod.TypeIncrease, 10); n != 1 {
		t.Errorf("Life INC 10 (10%% increased maximum Life effect): got %d, want 1", n)
	}
}

// TestSelectedMasteryContributesOnlyThatEffect is the other half.
func TestSelectedMasteryContributesOnlyThatEffect(t *testing.T) {
	tree := data.TreeVersions[data.TreeVersion3_18].Tree()
	node := tree.Nodes[lifeMasteryNode]

	var chosen int64
	for _, effect := range node.MasteryEffects {
		for _, stat := range effect.Stats {
			if stat == "+50 to maximum Life" {
				chosen = effect.Effect
			}
		}
	}
	if chosen == 0 {
		t.Fatal("no Life Mastery effect grants +50 to maximum Life")
	}

	ml := buildModListForNode(masteryTestEnv(map[int64]int64{4139: chosen}), lifeMasteryNode, node)

	if n := countMods(ml, "Life", mod.TypeBase, 50); n != 1 {
		t.Errorf("Life BASE 50: got %d, want 1", n)
	}
	if n := countMods(ml, "Life", mod.TypeIncrease, 10); n != 0 {
		t.Errorf("Life INC 10 from an UNSELECTED effect: got %d, want 0", n)
	}
}
