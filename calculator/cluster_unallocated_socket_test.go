package calculator

import (
	"strconv"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/pob"
)

// TestClusterSubgraphOnlyUnderAllocatedSocket pins PassiveSpec.lua:1849 (upstream #10272).
func TestClusterSubgraphOnlyUnderAllocatedSocket(t *testing.T) {
	requireGameData(t, "3.29")
	build := parseCorpusBuild(t, "0021-assassin-blade-flurry.xml")
	if len(TreeClusterNodes(build)) == 0 {
		t.Fatal("CONTROL FAILED: the fixture builds no subgraph")
	}
	spec := &build.Tree.Specs[max(build.Tree.ActiveSpec-1, 0)]
	sockets := map[string]bool{}
	for _, s := range spec.Sockets {
		sockets[strconv.Itoa(s.NodeID)] = true
	}
	var kept []string
	for _, id := range pob.SpecNodes(*spec) {
		if !sockets[strconv.FormatInt(id, 10)] {
			kept = append(kept, strconv.FormatInt(id, 10))
		}
	}
	spec.NodesAttr = strings.Join(kept, ",")
	if got := TreeClusterNodes(build); len(got) != 0 {
		t.Errorf("%d subgraph nodes with every socket deallocated, want 0", len(got))
	}
}
