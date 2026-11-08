package calculator2

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/data/tree2"
	"github.com/Vilsol/go-pob/pob2"
)

// nodeCompareRow is one row of testdata/node_compare_pob.jsonl.gz, which testdata/node_compare_oracle.lua writes from
// PoB-PoE2's node tooltips: per build, a row with the base pass, then one row per node. A calculation's output holds
// only the stats that differ from the base pass (null for one it lacks).
type nodeCompareRow struct {
	Build string         `json:"build"`
	Base  map[string]any `json:"base"`
	Node  int            `json:"node"`
	Alloc bool           `json:"alloc"`
	Lines []string       `json:"lines"`
	Calcs []struct {
		Add       []int          `json:"add"`
		Remove    []int          `json:"remove"`
		RemoveIDs []int          `json:"removeIds"`
		Output    map[string]any `json:"output"`
	} `json:"calcs"`
}

func readNodeCompareRows(t *testing.T) []nodeCompareRow {
	t.Helper()
	f, err := os.Open(filepath.Join("testdata", "node_compare_pob.jsonl.gz"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	var rows []nodeCompareRow
	sc := bufio.NewScanner(zr)
	sc.Buffer(nil, 1<<24)
	for sc.Scan() {
		var r nodeCompareRow
		if err := json.Unmarshal(sc.Bytes(), &r); err != nil {
			t.Fatal(err)
		}
		rows = append(rows, r)
	}
	return rows
}

func nodeCompareBuild(t *testing.T, name string) *MiscCalculator {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", name+".xml"))
	if err != nil {
		t.Fatal(err)
	}
	build, err := pob2.ParseBuild(raw)
	if err != nil {
		t.Fatal(err)
	}
	tab := NewItemsTab(build, loadTestTree(t), nil)
	env, err := BuildOutput(build, Options{Tree: loadTestTree(t), Items: tab})
	if err != nil {
		t.Fatal(err)
	}
	return NewMiscCalculator(env)
}

// overlayOutput is the base pass with a calculation's differences applied.
func overlayOutput(base, diff map[string]any) map[string]any {
	out := make(map[string]any, len(base))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range diff {
		if v == nil {
			delete(out, k)
		} else {
			out[k] = v
		}
	}
	return out
}

// stripColours drops PoB's colour codes.
func stripColours(s string) string { return colorCodeRe.ReplaceAllString(s, "") }

func TestNodeComparesMatchPoB(t *testing.T) {
	rows := readNodeCompareRows(t)
	var misc *MiscCalculator
	var base map[string]any
	// a stat the base pass already gets wrong is a defect of the calculation, not of the what-if; it is left out
	var baseOff []string
	nodes, calcs, stats, left, pathLines := 0, 0, 0, 0, 0
	for _, r := range rows {
		if r.Node == 0 {
			misc, base = nodeCompareBuild(t, r.Build), r.Base
			baseOff = outputDiffs(misc.Base.Player.Output, base)
			if len(baseOff) > 0 {
				t.Logf("%s: base pass differs on %v, left out", r.Build, baseOff)
				left += len(baseOff)
			}
			continue
		}
		nodes++
		node := misc.main.Spec.Nodes[r.Node]
		if node == nil {
			t.Fatalf("%s: no node %d", r.Build, r.Node)
		}
		if node.Alloc != r.Alloc {
			t.Errorf("%s %d: alloc %v, PoB-PoE2 %v", r.Build, r.Node, node.Alloc, r.Alloc)
		}
		got := NodeCompares(misc, node, nil)
		// the tooltip calculates the node, then its path when that is longer than the node; a path block is the second
		var envs []*Environment
		var adds, removes [][]int
		for _, b := range got.Blocks {
			envs = append(envs, b.Env)
		}
		path := node.Path
		if node.Alloc {
			path = node.Depends
		}
		if len(path) > 1 && len(got.Blocks) == 1 {
			t.Logf("%s %d: path calculated but not shown", r.Build, r.Node)
		}
		if len(r.Calcs) != len(envs) {
			t.Errorf("%s %d: %d calculations, PoB-PoE2 ran %d", r.Build, r.Node, len(envs), len(r.Calcs))
			continue
		}
		for i, c := range r.Calcs {
			calcs++
			want := overlayOutput(base, c.Output)
			stats += len(want)
			d := slices.DeleteFunc(outputDiffs(envs[i].Player.Output, want), func(k string) bool { return slices.Contains(baseOff, k) })
			if len(d) > 0 {
				t.Errorf("%s %d calc %d (add %v remove %v): %d stats differ: %.300v", r.Build, r.Node, i, c.Add, c.Remove, len(d), d)
			}
			for k, v := range c.Output {
				if _, has := envs[i].Player.Output[k]; v == nil && has {
					t.Errorf("%s %d calc %d: %s is not in PoB-PoE2's output", r.Build, r.Node, i, k)
				}
			}
			adds = append(adds, c.Add)
			removes = append(removes, c.Remove)
		}
		// the overrides: the node alone, then its path or dependants
		wantAdd, wantRemove := [][]int{}, [][]int{}
		one := []int{node.ID}
		var pathIDs []int
		for _, n := range path {
			pathIDs = append(pathIDs, n.ID)
		}
		sort.Ints(pathIDs)
		if node.Alloc {
			wantAdd, wantRemove = append(wantAdd, []int{}), append(wantRemove, one)
			if len(path) > 1 {
				wantAdd, wantRemove = append(wantAdd, []int{}), append(wantRemove, pathIDs)
			}
		} else {
			wantAdd, wantRemove = append(wantAdd, one), append(wantRemove, []int{})
			if len(path) > 1 {
				wantAdd, wantRemove = append(wantAdd, pathIDs), append(wantRemove, []int{})
			}
		}
		for i := range adds {
			if i < len(wantAdd) && (!slices.Equal(adds[i], wantAdd[i]) || !slices.Equal(removes[i], wantRemove[i])) {
				t.Errorf("%s %d calc %d: PoB-PoE2 adds %v removes %v, ours %v / %v", r.Build, r.Node, i, adds[i], removes[i], wantAdd[i], wantRemove[i])
			}
		}
		// PoB-PoE2 shows the headers of blocks with a displayed stat difference (graded with the stat lines on the
		// page), else the no-change line
		var headers []string
		for _, b := range got.Blocks {
			headers = append(headers, b.Header)
		}
		// the pathing lines, the gold lines left out
		var wantPath []string
		for _, l := range r.Lines {
			l = stripColours(l)
			if strings.Contains(l, "points to node") || strings.Contains(l, "To reach this node") || strings.Contains(l, "points gained from") {
				wantPath = append(wantPath, l)
			}
		}
		pathLines++
		if gotPath := NodePathLines(misc.main.Spec, node, nil); strings.Join(gotPath, "|") != strings.Join(wantPath, "|") {
			t.Errorf("%s %d: path lines\n got %q\nwant %q", r.Build, r.Node, gotPath, wantPath)
		}
		for _, l := range r.Lines {
			l = stripColours(l)
			switch {
			case strings.HasSuffix(l, "will give you:"):
				if !slices.Contains(headers, l) {
					t.Errorf("%s %d: PoB-PoE2 header %q, ours %q", r.Build, r.Node, l, headers)
				}
			case strings.HasPrefix(l, "No changes") || strings.HasSuffix(l, "will cause no changes"):
				if l != got.NoChange {
					t.Errorf("%s %d: no-change line %q, ours %q", r.Build, r.Node, l, got.NoChange)
				}
			}
		}
	}
	t.Logf("%d nodes, %d calculations, %d stats, %d path line sets graded", nodes, calcs, stats, pathLines)
	if nodes == 0 || calcs == 0 {
		t.Fatal("nothing graded")
	}
	// the base-pass stats left out, a ceiling
	if left > 4 {
		t.Errorf("%d base-pass stats differ, more than the 4 known", left)
	}
}

func TestTraceStep(t *testing.T) {
	spec := nodeCompareBuild(t, "warrior_items").main.Spec
	var target *tree2.SpecNode
	for _, id := range sortedNodeIDs(spec.Nodes) {
		if n := spec.Nodes[id]; !n.Alloc && n.HasPath && len(n.Path) == 3 {
			target = n
			break
		}
	}
	if target == nil {
		t.Fatal("no node three points away")
	}
	// the first hover starts the trace with the node's path, from the tree out
	trace, hover := TraceStep(nil, target)
	if hover != target || len(trace) != 3 || trace[2] != target || trace[0] != target.Path[2] {
		t.Fatalf("first hover: %v %v", nodeIDsOf(trace), hover)
	}
	// a node linked to the path's end extends it; hovering the end again changes nothing
	var next *tree2.SpecNode
	for _, l := range target.Linked {
		if !slices.Contains(trace, l) && l.HasPath {
			next = l
			break
		}
	}
	if next == nil {
		t.Fatal("no node linked past the target")
	}
	trace2, hover := TraceStep(trace, next)
	if hover != next || len(trace2) != 4 || trace2[3] != next || len(trace) != 3 {
		t.Fatalf("extend: %v %v (before %v)", nodeIDsOf(trace2), hover, nodeIDsOf(trace))
	}
	if same, hover := TraceStep(trace2, next); hover != next || len(same) != 4 {
		t.Fatalf("hover the end: %v %v", nodeIDsOf(same), hover)
	}
	// stepping back onto a linked node already on the path moves it to the end
	back, hover := TraceStep(trace2, target)
	if hover != target || len(back) != 4 || back[3] != target || back[2] != next {
		t.Fatalf("step back: %v %v", nodeIDsOf(back), hover)
	}
	// a node not linked to the end is not highlighted and leaves the trace alone
	far := trace[0]
	if slices.Contains(far.Linked, trace2[3]) {
		t.Skip("the path's start is linked to its end")
	}
	if kept, hover := TraceStep(trace2, far); hover != nil || len(kept) != 4 {
		t.Fatalf("unlinked: %v %v", nodeIDsOf(kept), hover)
	}
}

func nodeIDsOf(list []*tree2.SpecNode) []int {
	ids := make([]int, 0, len(list))
	for _, n := range list {
		ids = append(ids, n.ID)
	}
	return ids
}
