package data_test

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/data/raw"
)

// TestPrunableNodesOnRealBuilds: The tree page's deallocate preview (and the click that follows it) removes a node plus what it alone connects.
func TestPrunableNodesOnRealBuilds(t *testing.T) {
	names := []string{"0243-necromancer-raise-spectre.xml", "0041-pathfinder-vaal-fireball.xml", "0029-inquisitor-spark-of-the-nova.xml"}
	v := data.TreeVersions[data.LatestTreeVersion]
	nodes := v.Tree().Nodes
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join("..", "testdata", "corpus-"+raw.LatestVersion, name))
			if err != nil {
				t.Fatalf("REFUSING: %v", err)
			}
			build, err := builds.ParseBuild(raw)
			if err != nil {
				t.Fatal(err)
			}
			spec := build.Tree.Specs[max(build.Tree.ActiveSpec, 1)-1]
			active := make([]int64, 0, 128)
			for _, s := range strings.Split(spec.NodesAttr, ",") {
				if id, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64); err == nil {
					if _, ok := nodes[strconv.FormatInt(id, 10)]; ok {
						active = append(active, id)
					}
				}
			}
			if len(active) < 20 {
				t.Fatalf("REFUSING: only %d allocated nodes resolved on %s", len(active), data.LatestTreeVersion)
			}
			roots := make([]int64, 0, 2)
			for id, n := range nodes {
				if n.ClassStartIndex != nil && *n.ClassStartIndex == int64(spec.ClassID) {
					roots = append(roots, mustID(t, id))
				}
			}
			pruneRoots := func(list []int64) []int64 {
				out := append([]int64(nil), roots...)
				for _, id := range list {
					if n := nodes[strconv.FormatInt(id, 10)]; n.IsAscendancyStart != nil && *n.IsAscendancyStart {
						out = append(out, id)
					}
				}
				return out
			}

			base := map[int64]bool{}
			for _, id := range v.CalculatePrunableNodes(active, pruneRoots(active)) {
				base[id] = true
			}
			t.Logf("%d allocated nodes are already cut off (cluster jewel inner sockets and what hangs off them)", len(base))

			_, adj := data.TreeGraphForTest(v)
			set := make(map[int64]bool, len(active))
			for _, id := range active {
				set[id] = true
			}
			for _, id := range active {
				n := nodes[strconv.FormatInt(id, 10)]
				if n.ClassStartIndex != nil || (n.IsAscendancyStart != nil && *n.IsAscendancyStart) || (n.IsMastery != nil && *n.IsMastery) {
					continue
				}
				nb := 0
				for a := range adj[id] {
					if set[a] {
						nb++
					}
				}
				if nb != 1 {
					continue
				}
				if got := v.CalculateDependentNodes(active, id, roots); len(got) != 0 {
					t.Errorf("removing leaf %d (%v) also takes %d nodes: %v", id, *n.Name, len(got), got)
				}
				break
			}

			found := false
			for _, id := range active {
				deps := v.CalculateDependentNodes(active, id, roots)
				if len(deps) < 2 {
					continue
				}
				found = true
				for _, d := range deps {
					if !set[d] || base[d] || d == id {
						t.Errorf("removing %d takes %d, which is not an allocated, reachable, other node", id, d)
					}
				}
				break
			}
			if !found {
				t.Errorf("CONTROL: no allocated node carries a branch of 2+ nodes, so the leaf check proves nothing")
			}
		})
	}
}

func mustID(t *testing.T, s string) int64 {
	t.Helper()
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	return id
}
