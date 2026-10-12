package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/mod"
)

// TestMissingTreeNodesAreAbsentFromOurTree is the premise check: a synthetic node must never shadow one the graded tree carries.
func TestMissingTreeNodesAreAbsentFromOurTree(t *testing.T) {
	tree := data.TreeVersions[data.LatestTreeVersion].Tree()
	for id := range missingTreeNodes {
		if _, ok := tree.Nodes[id]; ok {
			t.Errorf("node %s is now in the tree; drop it from missingTreeNodes rather than shadowing it", id)
		}
	}
}

// TestIndomitableResolveResolves: a Forbidden Flame/Flesh grant resolves by name through the graded tree.
func TestIndomitableResolveResolves(t *testing.T) {
	version := data.TreeVersions[data.LatestTreeVersion]
	id := version.AscendancyMap()["indomitable resolve"]
	if id != "52435" {
		t.Fatalf(`ascendancyMap["indomitable resolve"] = %q, want "52435"`, id)
	}

	ml := buildModListForNode(masteryTestEnv(nil), id, resolveTreeNode(version.Tree(), id))
	if n := countMods(ml, "DamageTaken", mod.TypeMore, -25); n != 1 {
		t.Errorf("DamageTaken MORE -25 (Take 25%% less Damage): got %d, want 1", n)
	}
	if n := len(ml.Mods()); n != 1 {
		t.Errorf("got %d mods, want 1 — PoB's dump for 157-berserker-blade-flurry carries only DamageTaken MORE -25", n)
	}
}

// TestSoulDrinkerResolves: Soul Drinker is an ordinary Trickster notable in the graded tree (src/TreeData/3_29/tree.lua:84000-84020), two stats.
func TestSoulDrinkerResolves(t *testing.T) {
	version := data.TreeVersions[data.LatestTreeVersion]
	id := version.AscendancyMap()["soul drinker"]
	if id != "48999" {
		t.Fatalf(`ascendancyMap["soul drinker"] = %q, want "48999"`, id)
	}
	ml := buildModListForNode(masteryTestEnv(nil), id, resolveTreeNode(version.Tree(), id))
	if n := countMods(ml, "DamageEnergyShieldLeech", mod.TypeBase, 2); n != 1 {
		t.Errorf("DamageEnergyShieldLeech BASE 2: got %d, want 1", n)
	}
}

// TestHarnessTheVoidResolves: PoB grants Harness the Void (Saboteur.
func TestHarnessTheVoidResolves(t *testing.T) {
	version := data.TreeVersions[data.LatestTreeVersion]
	if _, ok := version.AscendancyMap()["harness the void"]; ok {
		t.Fatal("the graded tree now carries Harness the Void; missingTreeNodes must drop it")
	}
	id, ok := missingTreeNodeIDByName["harness the void"]
	if !ok || id != "57331" {
		t.Fatalf(`missingTreeNodeIDByName["harness the void"] = %q, want "57331"`, id)
	}
	ml := buildModListForNode(masteryTestEnv(nil), id, resolveTreeNode(version.Tree(), id))
	if n := len(ml.Mods()); n != 3 {
		t.Errorf("got %d mods, want 3 (one per chance line)", n)
	}
}
