package tree2

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// pathsSpec is the part of a <Spec> oracle/poe2_tree_paths.lua records from PassiveSpec:Save.
type pathsSpec struct {
	Nodes         string          `json:"nodes"`
	ClassID       string          `json:"classId"`
	AscendClassID string          `json:"ascendClassId"`
	WeaponSets    json.RawMessage `json:"weaponSets"`
	StrNodes      string          `json:"strNodes"`
	DexNodes      string          `json:"dexNodes"`
	IntNodes      string          `json:"intNodes"`
}

type pathsOp struct {
	ID   int       `json:"id"`
	Spec pathsSpec `json:"spec"`
}

type pathsRow struct {
	Build   string         `json:"build"`
	Start   pathsSpec      `json:"start"`
	Granted []int          `json:"granted"`
	Leap    int            `json:"leap"`
	Dist    map[string]int `json:"dist"`
	Alloc   []pathsOp      `json:"alloc"`
	Dealloc []pathsOp      `json:"dealloc"`
}

// weaponSets reads dkjson's {} or [] for an empty table.
func (p pathsSpec) weaponSets(t *testing.T) map[int]string {
	t.Helper()
	out := map[int]string{}
	if len(p.WeaponSets) == 0 || p.WeaponSets[0] != '{' {
		return out
	}
	var m map[string]string
	if err := json.Unmarshal(p.WeaponSets, &m); err != nil {
		t.Fatal(err)
	}
	for k, v := range m {
		n, err := strconv.Atoi(strings.TrimPrefix(k, "WeaponSet"))
		if err != nil {
			t.Fatalf("weapon set key %q", k)
		}
		out[n] = v
	}
	return out
}

func (p pathsSpec) input(t *testing.T) SpecInput {
	t.Helper()
	return SpecInput{
		Nodes: &p.Nodes, ClassID: &p.ClassID, AscendClassID: &p.AscendClassID, WeaponSets: p.weaponSets(t),
		AttributeOverrides: []AttributeOverrideInput{{StrNodes: p.StrNodes, DexNodes: p.DexNodes, IntNodes: p.IntNodes}},
	}
}

// specFromSave rebuilds the spec PoB-PoE2 had after the calc: the saved spec, then the granted nodes.
type specSource func(t *testing.T, tree *PassiveTree, row pathsRow) *PassiveSpec

func specFromSave(t *testing.T, tree *PassiveTree, row pathsRow) *PassiveSpec {
	t.Helper()
	spec := NewPassiveSpec(tree)
	if err := spec.Load(row.Start.input(t)); err != nil {
		t.Fatalf("%s: %v", row.Build, err)
	}
	return grantRow(spec, row)
}

// specFromBuild loads the build file itself with its items, so jewels (alternate class starts and the
// like) take part, as the allocation corpus test does.
func specFromBuild(dir string) specSource {
	return func(t *testing.T, tree *PassiveTree, row pathsRow) *PassiveSpec {
		t.Helper()
		raw, err := os.ReadFile(filepath.Join(dir, row.Build)) //nolint:gosec // corpus file
		if err != nil {
			t.Fatal(err)
		}
		bi := readBuildItems(t, tree, raw)
		spec := NewPassiveSpec(tree)
		spec.AllocateFromNodeRadius = oracleRadiusRows
		spec.Items = func(id int) *Item { return bi.items[id] }
		if err := spec.Load(activeSpecInput(t, raw)); err != nil {
			t.Fatalf("%s: %v", row.Build, err)
		}
		return grantRow(spec, row)
	}
}

func grantRow(spec *PassiveSpec, row pathsRow) *PassiveSpec {
	if len(row.Granted) > 0 {
		granted := map[int]*SpecNode{}
		for _, id := range row.Granted {
			if n, ok := spec.Nodes[id]; ok {
				granted[id] = n
			}
		}
		spec.SetGrantedPassiveNodes(granted)
	}
	return spec
}

func saveDiff(t *testing.T, got SpecSave, want pathsSpec) string {
	t.Helper()
	var diffs []string
	field := func(name, g, w string) {
		if g != w {
			diffs = append(diffs, fmt.Sprintf("%s: ours %q, PoB %q", name, g, w))
		}
	}
	field("nodes", got.Nodes, want.Nodes)
	field("classId", strconv.Itoa(got.ClassID), want.ClassID)
	field("ascendClassId", strconv.Itoa(got.AscendClassID), want.AscendClassID)
	field("strNodes", got.StrNodes, want.StrNodes)
	field("dexNodes", got.DexNodes, want.DexNodes)
	field("intNodes", got.IntNodes, want.IntNodes)
	wantSets := want.weaponSets(t)
	for _, set := range []int{1, 2} {
		field("WeaponSet"+strconv.Itoa(set), got.WeaponSets[set], wantSets[set])
	}
	return strings.Join(diffs, "; ")
}

type pathsTally struct {
	builds, loadBad, dist, distBad, ops, opsBad, ties int
	first                                             []string
}

func (p *pathsTally) bad(msg string) {
	if len(p.first) < 10 {
		p.first = append(p.first, msg)
	}
}

// diffPaths replays the recorded allocations on our spec.
func diffPaths(t *testing.T, tree *PassiveTree, rows []pathsRow, load specSource) pathsTally {
	t.Helper()
	var tally pathsTally
	for _, row := range rows {
		tally.builds++
		spec := load(t, tree, row)
		if d := saveDiff(t, spec.Save(), row.Start); d != "" {
			tally.loadBad++
			tally.bad(fmt.Sprintf("%s: the loaded spec differs before any allocation: %s", row.Build, d))
			continue
		}
		for _, id := range sortedKeys(spec.Nodes) {
			node := spec.Nodes[id]
			want, ok := row.Dist[strconv.Itoa(id)]
			if !ok && !node.HasPath {
				continue
			}
			tally.dist++
			if ok != node.HasPath || len(node.Path) != want {
				tally.distBad++
				tally.bad(fmt.Sprintf("%s: node %d path: ours %v/%d, PoB %v/%d", row.Build, id, node.HasPath, len(node.Path), ok, want))
			}
		}
		for _, op := range row.Alloc {
			spec := load(t, tree, row)
			if err := spec.AllocNode(spec.Nodes[op.ID], nil); err != nil {
				t.Fatalf("%s: AllocNode(%d): %v", row.Build, op.ID, err)
			}
			tally.ops++
			got := spec.Save()
			d := saveDiff(t, got, op.Spec)
			switch {
			case d == "":
			case equalLengthRoute(t, tree, row, op, got, load):
				tally.ties++
			default:
				tally.opsBad++
				tally.bad(fmt.Sprintf("%s: AllocNode(%d): %s", row.Build, op.ID, d))
			}
		}
		for _, op := range row.Dealloc {
			spec := load(t, tree, row)
			spec.DeallocNode(spec.Nodes[op.ID])
			tally.ops++
			if d := saveDiff(t, spec.Save(), op.Spec); d != "" {
				tally.opsBad++
				tally.bad(fmt.Sprintf("%s: DeallocNode(%d): %s", row.Build, op.ID, d))
			}
		}
	}
	return tally
}

// equalLengthRoute accepts a different allocation of the same size when PoB's own route, allocated on a
// fresh spec of ours as a trace path, survives the orphan pruning and saves as PoB's: both are shortest
// routes, and PoB picks among them in pairs() order, which stock LuaJIT randomises per process.
func equalLengthRoute(t *testing.T, tree *PassiveTree, row pathsRow, op pathsOp, got SpecSave, load specSource) bool {
	t.Helper()
	if len(digits(got.Nodes)) != len(digits(op.Spec.Nodes)) {
		return false
	}
	spec := load(t, tree, row)
	inSet := map[int]bool{}
	for _, nodes := range op.Spec.weaponSets(t) {
		for _, id := range digits(nodes) {
			inSet[id] = true
		}
	}
	// PoB's route: what it allocated, and the weapon-set nodes it promoted back to the normal tree
	var route []*SpecNode
	for _, id := range digits(op.Spec.Nodes) {
		if n, ok := spec.Nodes[id]; ok && (!n.Alloc || n.AllocMode != 0 && !inSet[id]) {
			route = append(route, n)
		}
	}
	ours, _ := spec.GetEffectiveAllocationPath(spec.Nodes[op.ID], nil)
	if len(route) != len(ours) {
		return false
	}
	if err := spec.AllocNode(spec.Nodes[op.ID], route); err != nil {
		t.Fatal(err)
	}
	return saveDiff(t, spec.Save(), op.Spec) == ""
}

func readPathsRows(t *testing.T, path string) []pathsRow {
	t.Helper()
	raw, err := os.ReadFile(path) //nolint:gosec // an oracle file
	if err != nil {
		t.Fatal(err)
	}
	var rows []pathsRow
	if err := json.Unmarshal(raw, &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) == 0 {
		t.Fatalf("%s holds no builds", path)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Build < rows[j].Build })
	return rows
}

func reportPaths(t *testing.T, tally pathsTally) {
	t.Helper()
	t.Logf("builds %d (loaded differently from PoB %d), node paths %d / %d agree, alloc/dealloc %d / %d agree (%d of them an equal-length route)",
		tally.builds, tally.loadBad, tally.dist-tally.distBad, tally.dist, tally.ops-tally.opsBad, tally.ops, tally.ties)
	for _, m := range tally.first {
		t.Log(m)
	}
	if tally.ops == 0 || tally.dist == 0 {
		t.Fatal("nothing was compared")
	}
	if tally.loadBad > 0 || tally.distBad > 0 || tally.opsBad > 0 {
		t.Fail()
	}
}

// The oracle is PoB-PoE2's AllocNode/DeallocNode on the committed fixture builds
// (oracle/poe2_tree_paths.lua under the deterministic LuaJIT).
func TestAllocNodePathsVsPoB(t *testing.T) {
	tree := dataTree(t)
	reportPaths(t, diffPaths(t, tree, readPathsRows(t, filepath.Join("testdata", "tree_paths.json")), fixtureSpec))
}

// fixtureSpec loads a build whose nodes PoB had intuitive-leap sources affecting from the build file, since
// the jewels that hold those nodes are not in the saved spec.
func fixtureSpec(t *testing.T, tree *PassiveTree, row pathsRow) *PassiveSpec {
	t.Helper()
	if row.Leap > 0 {
		return specFromBuild(filepath.Join("..", "..", "calculator2", "testdata"))(t, tree, row)
	}
	return specFromSave(t, tree, row)
}

// TestAllocNodePathsVsPoBCorpus reads POE2_TREE_PATHS, oracle/poe2_tree_paths.lua's output over corpus
// builds; gated by CORPUS=true, since the corpus is local only.
func TestAllocNodePathsVsPoBCorpus(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("CORPUS=true not set")
	}
	path := os.Getenv("POE2_TREE_PATHS")
	if path == "" {
		t.Skip("POE2_TREE_PATHS not set")
	}
	tree := dataTree(t)
	reportPaths(t, diffPaths(t, tree, readPathsRows(t, path), specFromBuild(filepath.Join("..", "..", "testdata", "corpus-poe2-0.5"))))
}

// A same-size allocation that is not a route must still count as a disagreement.
func TestAllocNodePathsRejectsAWrongRoute(t *testing.T) {
	tree := dataTree(t)
	rows := readPathsRows(t, filepath.Join("testdata", "tree_paths.json"))
	var row pathsRow
	for _, r := range rows {
		if r.Leap == 0 && len(r.Alloc) > 0 {
			row = r
			break
		}
	}
	op := row.Alloc[0]
	start := map[int]bool{}
	for _, id := range digits(row.Start.Nodes) {
		start[id] = true
	}
	nodes := digits(op.Spec.Nodes)
	in := map[int]bool{}
	for _, id := range nodes {
		in[id] = true
	}
	stray := 0
	for _, id := range sortedKeys(tree.Nodes) {
		if !in[id] && tree.Nodes[id].Type == "Normal" && !tree.Nodes[id].IsAttribute && tree.Nodes[id].AscendancyName == "" {
			stray = id
		}
	}
	for i := len(nodes) - 1; i >= 0; i-- {
		if !start[nodes[i]] && nodes[i] != op.ID {
			nodes[i] = stray
			break
		}
	}
	sort.Ints(nodes)
	op.Spec.Nodes = joinInts(nodes)
	row.Alloc = []pathsOp{op}
	row.Dealloc = nil
	if tally := diffPaths(t, tree, []pathsRow{row}, specFromSave); tally.opsBad != 1 || tally.ties != 0 {
		t.Fatalf("a stray node in PoB's route was accepted: %+v", tally)
	}
}

// from_nothing_leap_attribute.xml is from_nothing.xml with attribute node 2864, which only the jewel in socket
// 26725 holds, allocated as Strength. PoB-PoE2's DeallocNode(22928), the socket's only link to the start,
// lists 2864 in node.depends through the intuitive-leap resolution and so drops its override too.
func TestDeallocNodeTakesIntuitiveLeapDependents(t *testing.T) {
	tree := dataTree(t)
	spec := specFromBuild("testdata")(t, tree, pathsRow{Build: "from_nothing_leap_attribute.xml"})
	if d := saveDiff(t, spec.Save(), pathsSpec{
		Nodes:   "1207,1913,2864,3936,4665,5710,6839,13397,16725,22928,23570,26725,27373,33556,36709,38323,38646,39274,39581,41031,46325,47175,54232,54811,55473,57703,61534,62455,63393,63579",
		ClassID: "6", AscendClassID: "0", StrNodes: "2864,13397,16725,22928,23570,27373,54811,57703", DexNodes: "1207", IntNodes: "41031",
	}); d != "" {
		t.Fatalf("loaded: %s", d)
	}
	depends := make([]int, 0, len(spec.Nodes[22928].Depends))
	for _, n := range spec.Nodes[22928].Depends {
		depends = append(depends, n.ID)
	}
	sort.Ints(depends)
	if got := joinInts(depends); got != "2864,16725,22928,26725,27373,39274,54811,57703,62455,63579" {
		t.Fatalf("depends %s", got)
	}
	spec.DeallocNode(spec.Nodes[22928])
	if d := saveDiff(t, spec.Save(), pathsSpec{
		Nodes:   "1207,1913,3936,4665,5710,6839,13397,23570,33556,36709,38323,38646,39581,41031,46325,47175,54232,55473,61534,63393",
		ClassID: "6", AscendClassID: "0", StrNodes: "13397,23570", DexNodes: "1207", IntNodes: "41031",
	}); d != "" {
		t.Fatalf("after DeallocNode(22928): %s", d)
	}
}
