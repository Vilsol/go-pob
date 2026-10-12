package calculator

import (
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/mod"
)

const pinnedPurposefulHarbinger = "Aura Buffs from Skills have 8% increased Effect on you for each Herald affecting you, up to 40%"

// TestTreeStatTextMatchesPinnedPoB is the gate on the fixup table.
func TestTreeStatTextMatchesPinnedPoB(t *testing.T) {
	tree := data.TreeVersions[data.TreeVersion3_18].Tree()
	node, ok := tree.Nodes["51870"]
	if !ok {
		t.Fatal("node 51870 (Purposeful Harbinger) missing from the 3_18 tree")
	}

	var got []string
	for _, line := range fixupTreeStatText(node.Stats) {
		for _, part := range strings.Split(line, "\n") {
			if part != "" {
				got = append(got, part)
			}
		}
	}
	if len(got) != 1 || got[0] != pinnedPurposefulHarbinger {
		t.Fatalf("got %q, want [%q]", got, pinnedPurposefulHarbinger)
	}
}

// TestTreeStatTextFixupsAreExhaustive pins the table in BOTH directions.
func TestTreeStatTextFixupsAreExhaustive(t *testing.T) {
	tree := data.TreeVersions[data.TreeVersion3_18].Tree()
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
			t.Errorf("fixup key never occurs in the 3_18 tree, so it can never fire: %q", key)
		}
	}
	if len(treeStatTextFixups) != 1 {
		t.Errorf("table holds %d entries; oracle/tree_stats.lua measured 2 divergent nodes of which 1 needs rewriting — re-run it before changing this", len(treeStatTextFixups))
	}
}

// TestPurposefulHarbingerParses is the behavioural half.
func TestPurposefulHarbingerParses(t *testing.T) {
	nodeMap := clusterNodeMap(data.TreeVersion3_18)
	node, ok := nodeMap["Purposeful Harbinger"]
	if !ok {
		t.Fatal("Purposeful Harbinger missing from the cluster-jewel notable map")
	}

	var mods []mod.Mod
	for _, line := range fixupTreeStatText(node.Stats) {
		entry := ParseMod(line, false)
		if strings.Trim(entry.Extra, " ") != "" {
			t.Fatalf("unparsed remainder %q for line %q", entry.Extra, line)
		}
		mods = append(mods, entry.ModList...)
	}
	if len(mods) != 1 {
		t.Fatalf("got %d mods, want 1", len(mods))
	}
	m := mods[0]
	if m.Name() != "SkillAuraEffectOnSelf" || m.Type() != mod.TypeIncrease {
		t.Fatalf("got %s %s, want SkillAuraEffectOnSelf INC", m.Name(), m.Type())
	}
	if v := m.Value(); v == nil || v.Float() != 8 {
		t.Fatalf("got value %v, want 8", m.Value())
	}
	if len(m.Tags()) != 1 {
		t.Fatalf("got %d tags, want 1", len(m.Tags()))
	}
	mult, ok := m.Tags()[0].(*mod.MultiplierTag)
	if !ok {
		t.Fatalf("got tag %T, want *mod.MultiplierTag", m.Tags()[0])
	}
	if len(mult.VariableList) != 1 || mult.VariableList[0] != "Herald" {
		t.Fatalf("got multiplier vars %v, want [Herald]", mult.VariableList)
	}
	if mult.TagGlobalLimit == nil || *mult.TagGlobalLimit != 40 {
		t.Fatalf("got global limit %v, want 40", mult.TagGlobalLimit)
	}
	if mult.TagGlobalLimitKey == nil || *mult.TagGlobalLimitKey != "PurposefulHarbinger" {
		t.Fatalf("got global limit key %v, want PurposefulHarbinger", mult.TagGlobalLimitKey)
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
