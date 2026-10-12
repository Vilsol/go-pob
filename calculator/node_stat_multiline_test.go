package calculator

import (
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// TestNodeStatEmbeddedNewlineIsSplit: PoB's PassiveTree:ProcessStats splits a stat entry containing a literal newline into separate lines before parsing.
func TestNodeStatEmbeddedNewlineIsSplit(t *testing.T) {
	node, ok := data.TreeVersions[data.LatestTreeVersion].Tree().Nodes["56116"]
	testza.AssertTrue(t, ok, "node 56116 (Magebane) must exist in the tree")
	testza.AssertEqual(t, 1, len(node.Stats), "Magebane still ships its stats as one newline-joined entry")

	env := &Environment{}
	ml := moddb.NewModList()
	processStats(env, ml, "56116", "Passive Node (Magebane)", node.Stats)

	testza.AssertTrue(t, ml.Flag(nil, "NoDexBonusToEvasion"), "first line must parse")

	var suppression mod.Mod
	for _, m := range ml.Mods() {
		if m.Name() == "SpellSuppressionChance" {
			suppression = m
		}
	}
	testza.AssertNotNil(t, suppression, "second line must parse into a SpellSuppressionChance mod")
}

// TestNodeStatCombineFallback: The combine fallback itself: neither line parses alone, joined with a space they do.
func TestNodeStatCombineFallback(t *testing.T) {
	name := "combine"
	node := data.Node{
		Name: &name,
		Stats: []string{
			"10% increased Attack",
			"and Cast Speed",
		},
	}

	env := &Environment{}
	ml := buildModListForNode(env, "1", node)

	found := 0
	for _, m := range ml.Mods() {
		if m.Name() == "Speed" {
			found++
		}
	}
	testza.AssertTrue(t, found > 0, "the two lines must be recovered by the combine fallback")
}
