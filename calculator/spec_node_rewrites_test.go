package calculator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"testing"
)

// TestSpecNodeRewritesAgainstPoB grades each rewritten node's name and lines against PoB's spec.nodes after a load.
func TestSpecNodeRewritesAgainstPoB(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "testdata", "spec-node-rewrites", "pob.json"))
	if err != nil {
		t.Fatal(err)
	}
	var oracle map[string]map[string]struct {
		Name  string   `json:"name"`
		Stats []string `json:"stats"`
	}
	if err := json.Unmarshal(raw, &oracle); err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(oracle))
	for name := range oracle {
		names = append(names, name)
	}
	sort.Strings(names)
	var vaalNames []string
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			want := oracle[name]
			if len(want) == 0 {
				t.Fatal("REFUSING: PoB rewrote no node")
			}
			env := GetMiscCalculator(parseCorpusBuild(t, name)).Base()
			got := map[string]SpecNodeRewrite{}
			for _, r := range SpecNodeRewrites(env) {
				got[strconv.FormatInt(r.NodeID, 10)] = r
			}
			tree := env.Spec.Tree()
			var diffs []string
			for id, w := range want {
				g, ok := got[id]
				if !ok {
					diffs = append(diffs, fmt.Sprintf("%s missing, PoB %q %q", id, w.Name, w.Stats))
					continue
				}
				if isMasteryNode(tree.Nodes[id]) {
					if g.Name != w.Name {
						diffs = append(diffs, fmt.Sprintf("%s mastery ours %q, PoB %q", id, g.Name, w.Name))
					}
					continue
				}
				sameLines := slices.Equal(g.Stats, w.Stats) || (vaalName(w.Name) && slices.Equal(sorted(g.Stats), sorted(w.Stats)))
				switch {
				case !sameLines:
					diffs = append(diffs, fmt.Sprintf("%s ours %q %q, PoB %q %q", id, g.Name, g.Stats, w.Name, w.Stats))
				case g.Name != w.Name && vaalName(g.Name) && vaalName(w.Name):
					vaalNames = append(vaalNames, name+"/"+id)
				case g.Name != w.Name:
					diffs = append(diffs, fmt.Sprintf("%s ours %q, PoB %q", id, g.Name, w.Name))
				}
			}
			for id, g := range got {
				if _, ok := want[id]; !ok {
					diffs = append(diffs, fmt.Sprintf("%s extra %q %q", id, g.Name, g.Stats))
				}
			}
			sort.Strings(diffs)
			for _, d := range diffs {
				t.Error(d)
			}
			t.Logf("%d nodes, %d differ", len(want), len(diffs))
		})
	}
	if want := []string{"0006-elementalist-storm-burst.xml/11420"}; !slices.Equal(vaalNames, want) {
		t.Errorf("Vaal names differing from PoB: %v, want %v", vaalNames, want)
	}
}

func vaalName(name string) bool {
	return name == "Might of the Vaal" || name == "Legacy of the Vaal"
}

func sorted(lines []string) []string {
	out := slices.Clone(lines)
	slices.Sort(out)
	return out
}
