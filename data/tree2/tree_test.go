package tree2

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
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
func TestLoadPinnedTree0_5(t *testing.T) {
	tree := pinTree(t)

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
	tree := pinTree(t)
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
