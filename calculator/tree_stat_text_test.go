package calculator

import (
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/mod"
)

var pinnedTreeStatText = map[string]string{
	"18054": "Non-Damaging Elemental Ailments you inflict spread to nearby enemies within 2 metres",
	"27602": "25% of Damage taken Recouped as Life, Mana and Energy Shield",
}

// TestTreeStatTextMatchesPinnedPoB is the gate on the fixup table.
func TestTreeStatTextMatchesPinnedPoB(t *testing.T) {
	tree := data.TreeVersions[data.LatestTreeVersion].Tree()
	for id, want := range pinnedTreeStatText {
		node, ok := tree.Nodes[id]
		if !ok {
			t.Fatalf("node %s missing from the %s tree", id, data.LatestTreeVersion)
		}
		got := fixupTreeStatText(node.Stats)
		if len(got) == 0 || got[0] != want {
			t.Errorf("node %s: got %q, want first line %q", id, got, want)
		}
	}
}

// TestTreeStatTextFixupsAreExhaustive pins the table in BOTH directions.
func TestTreeStatTextFixupsAreExhaustive(t *testing.T) {
	tree := data.TreeVersions[data.LatestTreeVersion].Tree()
	seen := map[string]bool{}
	for _, node := range tree.Nodes {
		for _, line := range node.Stats {
			if _, ok := treeStatTextFixups[line]; ok {
				seen[line] = true
			}
		}
	}
	for key := range treeStatTextFixups {
		if !seen[key] {
			t.Errorf("fixup key never occurs in the %s tree, so it can never fire: %q", data.LatestTreeVersion, key)
		}
	}
	if len(treeStatTextFixups) != len(pinnedTreeStatText) {
		t.Errorf("table holds %d entries, pinnedTreeStatText %d — re-run `task tree-tables` before changing this", len(treeStatTextFixups), len(pinnedTreeStatText))
	}
}

// TestPurposefulHarbingerStaysUnparsed pins the pin's 3.29 wording as UNPARSED.
func TestPurposefulHarbingerStaysUnparsed(t *testing.T) {
	nodeMap := clusterNodeMap(data.LatestTreeVersion)
	node, ok := nodeMap["Purposeful Harbinger"]
	if !ok {
		t.Fatal("Purposeful Harbinger missing from the cluster-jewel notable map")
	}

	lines := fixupTreeStatText(node.Stats)
	if len(lines) != 1 {
		t.Fatalf("got %d stat lines, want 1", len(lines))
	}
	parts := strings.Split(lines[0], "\n")
	for _, line := range append(parts, strings.Join(parts, " ")) {
		entry := ParseMod(line, false)
		if strings.Trim(entry.Extra, " ") == "" {
			t.Errorf("%q parsed to %d mods; the pin's ModCache leaves it unparsed", line, len(entry.ModList))
		}
	}
}

// TestSupremeEgoWordingIsBehaviourNeutral records the OTHER measured tree-text divergence and why it needs no fixup.
func TestSupremeEgoWordingIsBehaviourNeutral(t *testing.T) {
	ours := ParseMod("40% more Mana Reservation of Aura Skills", false)
	pob := ParseMod("40% more Mana Reservation of Skills", false)

	if len(ours.ModList) != 1 || len(pob.ModList) != 1 {
		t.Fatalf("got %d / %d mods, want 1 each", len(ours.ModList), len(pob.ModList))
	}
	a, b := ours.ModList[0], pob.ModList[0]
	if a.Name() != b.Name() || a.Type() != b.Type() || a.Value().Float() != b.Value().Float() {
		t.Fatalf("mods differ: %s %s %v vs %s %s %v", a.Name(), a.Type(), a.Value(), b.Name(), b.Type(), b.Value())
	}
	if len(a.Tags()) != 1 || len(b.Tags()) != 1 {
		t.Fatalf("got %d / %d tags, want 1 each", len(a.Tags()), len(b.Tags()))
	}
	at, aok := a.Tags()[0].(*mod.SkillTypeTag)
	bt, bok := b.Tags()[0].(*mod.SkillTypeTag)
	if !aok || !bok {
		t.Fatalf("got tags %T / %T, want *mod.SkillTypeTag", a.Tags()[0], b.Tags()[0])
	}
	if strings.Join(at.SkillTypeList, ",") != strings.Join(bt.SkillTypeList, ",") || at.Negative != bt.Negative {
		t.Fatalf("skill-type tags differ: %v/%v vs %v/%v", at.SkillTypeList, at.Negative, bt.SkillTypeList, bt.Negative)
	}
}
