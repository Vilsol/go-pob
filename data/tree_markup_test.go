package data

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
)

func TestTreeTextDropsGGGMarkup(t *testing.T) {
	fixture := `{"nodes":{"1":{"skill":1,"stats":["Grants Level 20 [SummonAbyssalConstructs::20|Summon Bone Feeders]","10% chance to [Blind] Enemies","a \"[Onslaught]\" b"],"out":["2","3"]}}}`
	var tree Tree
	if err := json.Unmarshal(escapeTreeGGGStrings([]byte(fixture)), &tree); err != nil {
		t.Fatal(err)
	}
	want := []string{"Grants Level 20 Summon Bone Feeders", "10% chance to Blind Enemies", `a "Onslaught" b`}
	if got := tree.Nodes["1"].Stats; !slices.Equal(got, want) {
		t.Errorf("stats = %q, want %q", got, want)
	}
	if got := tree.Nodes["1"].Out; !slices.Equal(got, []string{"2", "3"}) {
		t.Errorf("out = %q, want [2 3]", got)
	}
}

func TestTree330NodeTextHasNoGGGMarkup(t *testing.T) {
	v := TreeVersions[TreeVersion3_30]
	node, ok := v.Tree().Nodes["16287"]
	if !ok {
		t.Fatal("node 16287 missing from the 3.30 tree")
	}
	if want := "Triggers Level 20 Summon Bone Feeders when Allocated"; !slices.Contains(node.Stats, want) {
		t.Errorf("node 16287 stats = %q, want %q", node.Stats, want)
	}
	if strings.Contains(string(v.RawTree()), "[SummonAbyssalConstructs::20|") {
		t.Error("raw tree still carries GGG markup")
	}
}
