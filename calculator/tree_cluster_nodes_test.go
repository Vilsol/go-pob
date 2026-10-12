package calculator

import (
	"bufio"
	"encoding/json"
	"math"
	"os"
	"slices"
	"testing"
)

type clusterOracleNode struct {
	ID     int64   `json:"id"`
	Type   string  `json:"type"`
	Name   string  `json:"name"`
	Icon   string  `json:"icon"`
	Proxy  int64   `json:"proxy"`
	O      int64   `json:"o"`
	Oidx   int64   `json:"oidx"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Linked []int64 `json:"linked"`
}

// TestTreeClusterNodesMatchPoB: The active spec's cluster subgraph nodes as PoB draws them.
func TestTreeClusterNodesMatchPoB(t *testing.T) {
	f, err := os.Open("../testdata/cluster-subgraph/oracle.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1<<20), 1<<22)
	builds := 0
	tree := specTree()
	for scanner.Scan() {
		var want struct {
			File  string              `json:"file"`
			Nodes []clusterOracleNode `json:"nodes"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &want); err != nil {
			t.Fatal(err)
		}
		builds++
		got := TreeClusterNodes(parseCorpusBuild(t, want.File))
		byID := make(map[int64]TreeClusterNode, len(got))
		for _, n := range got {
			byID[n.ID] = n
		}
		if len(got) != len(want.Nodes) {
			t.Errorf("%s: %d nodes, PoB %d", want.File, len(got), len(want.Nodes))
		}
		for _, w := range want.Nodes {
			g, ok := byID[w.ID]
			if !ok {
				t.Errorf("%s: node %d (%s %s) missing", want.File, w.ID, w.Type, w.Name)
				continue
			}
			linked := slices.Clone(g.Linked)
			slices.Sort(linked)
			if w.Linked == nil {
				w.Linked = []int64{}
			}
			if linked == nil {
				linked = []int64{}
			}
			proxyGroup := tree.Nodes[nodeKey(w.Proxy)].Group
			if proxyGroup == nil || g.Group != *proxyGroup || g.Icon != w.Icon {
				t.Errorf("%s: node %d group %d icon %q, PoB proxy %d icon %q", want.File, w.ID, g.Group, g.Icon, w.Proxy, w.Icon)
			}
			if g.Type != w.Type || g.Name != w.Name || g.Orbit != w.O || g.OrbitIndex != w.Oidx ||
				math.Abs(g.X-w.X) > 0.01 || math.Abs(g.Y-w.Y) > 0.01 || !slices.Equal(linked, w.Linked) {
				t.Errorf("%s: node %d\n got  %s %q o%d/%d (%.2f, %.2f) %v\n want %s %q o%d/%d (%.2f, %.2f) %v",
					want.File, w.ID, g.Type, g.Name, g.Orbit, g.OrbitIndex, g.X, g.Y, linked,
					w.Type, w.Name, w.O, w.Oidx, w.X, w.Y, w.Linked)
			}
		}
	}
	if builds == 0 {
		t.Fatal("REFUSING: no oracle rows")
	}
}

// TestTreeSpecClusterNodes: The compared tree's subgraphs are its own: a tree with no jewels has none, whichever tree is active.
func TestTreeSpecClusterNodes(t *testing.T) {
	build := parseCorpusBuild(t, "0021-assassin-blade-flurry.xml")
	own := TreeClusterNodes(build)
	if len(own) == 0 {
		t.Fatal("control: the build has no cluster nodes")
	}
	active := build.Tree.ActiveSpec
	bare := build.Tree.Specs[active-1]
	bare.Sockets = nil
	build.Tree.Specs = append(build.Tree.Specs, bare)
	if n := len(TreeSpecClusterNodes(build, len(build.Tree.Specs))); n != 0 {
		t.Fatalf("a tree with no jewels: %d nodes", n)
	}
	build.Tree.ActiveSpec = len(build.Tree.Specs)
	if got := TreeSpecClusterNodes(build, active); len(got) != len(own) || build.Tree.ActiveSpec != len(build.Tree.Specs) {
		t.Fatalf("the jewelled tree while another is active: %d nodes, want %d", len(got), len(own))
	}
}
