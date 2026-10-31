package tree2

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/Vilsol/go-pob/data/assetcache"
	"github.com/Vilsol/go-pob/storage"
)

// pob2Dir resolves the pinned PoB-PoE2 checkout like the tooling: POB2_DIR, else a sibling of the main checkout.
func pob2Dir() string {
	if v := os.Getenv("POB2_DIR"); v != "" {
		return v
	}
	out, err := exec.CommandContext(context.Background(), "git", "rev-parse", "--path-format=absolute", "--git-common-dir").Output()
	if err != nil {
		return filepath.Join("..", "..", "..", "PathOfBuilding-PoE2")
	}
	common := strings.TrimSpace(string(out))
	if r, err := filepath.EvalSymlinks(common); err == nil {
		common = r
	}
	return filepath.Join(common, "..", "..", "PathOfBuilding-PoE2")
}

var loadPinTree = sync.OnceValues(func() (*PassiveTree, error) {
	raw, err := os.ReadFile(filepath.Join(pob2Dir(), "src", "TreeData", "0_5", "tree.json"))
	if err != nil {
		return nil, fmt.Errorf("reading the pinned tree: %w", err)
	}
	return Load("0_5", raw)
})

var loadDataTree = sync.OnceValues(func() (*PassiveTree, error) {
	return LoadData(context.Background(), "0_5", assetcache.AssetLoaderWrapper{Storage: storage.Get()})
})

// dataTree is the tree go-pob loads, from go-pob-data; pinTree is PoB-PoE2's own, kept as an oracle.
func dataTree(t *testing.T) *PassiveTree {
	t.Helper()
	tree, err := loadDataTree()
	if err != nil {
		t.Skipf("go-pob-data's PoE2 tree not loadable: %v", err)
	}
	return tree
}

func pinTree(t *testing.T) *PassiveTree {
	t.Helper()
	tree, err := loadPinTree()
	if errors.Is(err, fs.ErrNotExist) {
		t.Skipf("PoB-PoE2 tree.json not found under %s — set POB2_DIR", pob2Dir())
	}
	if err != nil {
		t.Fatal(err)
	}
	return tree
}

// The expected figures are PoB-PoE2's own, from loading the pinned tree headless.
func TestLoadTree0_5(t *testing.T) {
	tree := dataTree(t)

	if got := len(tree.Nodes); got != 4914 {
		t.Errorf("nodes = %d, PoB 4914", got)
	}
	if got := len(tree.Classes); got != 8 {
		t.Errorf("classes = %d, PoB 8", got)
	}
	asc := 0
	for _, c := range tree.Classes {
		asc += len(c.Classes) - 1
		if c.StartNodeID == 0 {
			t.Errorf("class %s has no start node", c.Name)
		}
		for _, a := range c.Classes[1:] {
			if a.StartNodeID == 0 {
				t.Errorf("ascendancy %s has no start node", a.Name)
			}
		}
	}
	if asc != 23 {
		t.Errorf("ascendancies = %d, PoB 23", asc)
	}
	if got := tree.Classes[11].Name; got != "Druid" {
		t.Errorf("class integerId 11 = %q, PoB Druid", got)
	}
	if ref := tree.InternalAscendNameMap["Witch3b"]; ref.AscendClass == nil || ref.AscendClass.Name != "Abyssal Lich" || ref.AscendClassID != 4 {
		t.Errorf("Witch3b = %+v, PoB Abyssal Lich at ascendClassId 4", ref)
	}
	// Abyssal Lich shares Lich's start node through the switchable start's options.
	lich, abyssal := tree.AscendNameMap["Lich"].AscendClass, tree.AscendNameMap["Abyssal Lich"].AscendClass
	if lich.StartNodeID == 0 || lich.StartNodeID != abyssal.StartNodeID {
		t.Errorf("Lich start %d, Abyssal Lich start %d: PoB gives both 23710", lich.StartNodeID, abyssal.StartNodeID)
	}

	types := map[string]int{}
	attrs := 0
	for _, n := range tree.Nodes {
		types[n.Type]++
		if n.IsAttribute {
			attrs++
			if len(n.AttributeOptions) != 3 || n.AttributeOptions[0].Dn != "Strength" || n.AttributeOptions[2].Dn != "Intelligence" {
				t.Errorf("attribute node %d options = %+v", n.ID, n.AttributeOptions)
			}
		}
	}
	if attrs != 293 {
		t.Errorf("attribute nodes = %d, PoB 293", attrs)
	}
	for typ, want := range map[string]int{"ClassStart": 6, "AscendClassStart": 22, "Socket": 18, "Keystone": 33} {
		if types[typ] != want {
			t.Errorf("%s nodes = %d, PoB %d", typ, types[typ], want)
		}
	}
	if got := len(tree.MissingConnections); got != 14 {
		t.Errorf("missing connection targets = %d, PoB prints 14 'missing node' lines", got)
	}
}

func TestTreeLinksAreSymmetric(t *testing.T) {
	tree := dataTree(t)
	count := func(ids []int, id int) int {
		c := 0
		for _, x := range ids {
			if x == id {
				c++
			}
		}
		return c
	}
	for _, n := range tree.Nodes {
		for _, o := range n.LinkedID {
			if count(n.LinkedID, o) != count(tree.Nodes[o].LinkedID, n.ID) {
				t.Fatalf("link %d-%d is not symmetric", n.ID, o)
			}
		}
	}
}

func TestLoadRefusesAnOrbitOutsideTheTable(t *testing.T) {
	raw := []byte(`{"classes":[],"constants":{"orbitRadii":[0],"orbitAnglesByOrbit":[[0]]},
		"groups":[{"x":0,"y":0,"orbits":[0],"nodes":[1]}],
		"nodes":{"1":{"skill":1,"group":1,"orbit":3,"orbitIndex":0,"name":"x","stats":[],"connections":[]}}}`)
	if _, err := Load("test", raw); err == nil || !strings.Contains(err.Error(), "orbit 3") {
		t.Fatalf("err = %v, want an orbit refusal", err)
	}
}

// TestDataTreeMatchesPinnedTree compares every node as go-pob reads it from go-pob-data's tree and from
// PoB-PoE2's tree.json. The two stat lines that differ are PoB's: the game gives 4367 three feathers.
func TestDataTreeMatchesPinnedTree(t *testing.T) {
	got, want := dataTree(t), pinTree(t)
	wantStats := map[int][]string{
		4367: {
			"Dodging can expend up to 3 Owl Feathers, granting Primal Bounty 100% more",
			"Empowerment effect per additional Feather expended",
			"Gain Owl Feathers 50% faster",
		},
		37778: {"20% reduced Spirit Reservation Efficiency of Skills", "40% increased Reservation Efficiency of Minion Skills"},
	}
	if len(got.Nodes) != len(want.Nodes) || len(got.Groups) != len(want.Groups) || len(got.Classes) != len(want.Classes) {
		t.Fatalf("nodes/groups/classes = %d/%d/%d, PoB %d/%d/%d", len(got.Nodes), len(got.Groups), len(got.Classes),
			len(want.Nodes), len(want.Groups), len(want.Classes))
	}
	near := func(a, b float64) bool { return math.Abs(a-b) <= 1e-6 }
	// PoB's tree.json rounds floats to 14 significant digits; go-pob-data's keeps them whole.
	gc, wc := got.Constants, want.Constants
	gc.OrbitAnglesByOrbit, wc.OrbitAnglesByOrbit = nil, nil
	if !reflect.DeepEqual(gc, wc) || !reflect.DeepEqual(got.JewelSlots, want.JewelSlots) ||
		!reflect.DeepEqual(got.MissingConnections, want.MissingConnections) {
		t.Errorf("constants, jewel slots or missing connections differ")
	}
	if len(got.OrbitAnglesByOrbit) != len(want.OrbitAnglesByOrbit) {
		t.Fatalf("orbits = %d, PoB %d", len(got.OrbitAnglesByOrbit), len(want.OrbitAnglesByOrbit))
	}
	for o, angles := range want.OrbitAnglesByOrbit {
		if len(got.OrbitAnglesByOrbit[o]) != len(angles) {
			t.Fatalf("orbit %d angles = %d, PoB %d", o, len(got.OrbitAnglesByOrbit[o]), len(angles))
		}
		for i, a := range angles {
			if !near(got.OrbitAnglesByOrbit[o][i], a) {
				t.Errorf("orbit %d angle %d = %v, PoB %v", o, i, got.OrbitAnglesByOrbit[o][i], a)
			}
		}
	}
	radiusIDs := func(n *Node) [][]int {
		out := make([][]int, len(n.NodesInRadius))
		for i, m := range n.NodesInRadius {
			for id := range m {
				out[i] = append(out[i], id)
			}
			sort.Ints(out[i])
		}
		return out
	}
	strip := func(n Node) Node {
		n.Group, n.NodesInRadius, n.RawOptions = nil, nil, nil
		n.X, n.Y, n.Angle = 0, 0, 0
		return n
	}
	for id, w := range want.Nodes {
		g := got.Nodes[id]
		if g == nil {
			t.Errorf("node %d missing", id)
			continue
		}
		if !near(g.X, w.X) || !near(g.Y, w.Y) || !near(g.Angle, w.Angle) {
			t.Errorf("node %d at (%v,%v,%v), PoB (%v,%v,%v)", id, g.X, g.Y, g.Angle, w.X, w.Y, w.Angle)
		}
		if !reflect.DeepEqual(radiusIDs(g), radiusIDs(w)) {
			t.Errorf("node %d nodes in radius differ", id)
		}
		gs, ws := strip(*g), strip(*w)
		if stats, ok := wantStats[id]; ok {
			if !reflect.DeepEqual(g.Sd, stats) {
				t.Errorf("node %d stats = %q, game %q", id, g.Sd, stats)
			}
			if reflect.DeepEqual(g.Sd, w.Sd) {
				t.Errorf("node %d stats equal PoB's %q; drop it from the expected differences", id, w.Sd)
			}
			gs.Stats, gs.Sd, ws.Stats, ws.Sd = nil, nil, nil, nil
		}
		if !reflect.DeepEqual(gs, ws) {
			t.Errorf("node %d differs:\n got %+v\nwant %+v", id, gs, ws)
		}
	}
	for id, w := range want.Classes {
		g := got.Classes[id]
		if g == nil || g.Name != w.Name || g.StartNodeID != w.StartNodeID || len(g.Classes) != len(w.Classes) {
			t.Errorf("class %d = %+v, PoB %+v", id, g, w)
			continue
		}
		for i := range w.Classes {
			if *g.Classes[i] != *w.Classes[i] {
				t.Errorf("class %d ascendancy %d = %+v, PoB %+v", id, i, *g.Classes[i], *w.Classes[i])
			}
		}
	}
}
