package calculator

import (
	"slices"
	"testing"

	"github.com/Vilsol/go-pob/data"
)

// TestTreeJewelRings: 0005 sockets Impossible Escape.
func TestTreeJewelRings(t *testing.T) {
	build := parseCorpusBuild(t, "0005-occultist-vaal-double-strike.xml")
	rings := TreeJewelRings(build, nil)
	tree := specTree()
	var arts []string
	onKeystone := 0
	for _, r := range rings {
		n := tree.Nodes[nodeKey(r.NodeID)]
		switch {
		case n.IsKeystone != nil && *n.IsKeystone:
			onKeystone++
			if r.Inner != 150 {
				t.Errorf("Impossible Escape's inner %v", r.Inner)
			}
		case n.IsJewelSocket == nil || !*n.IsJewelSocket:
			t.Errorf("ring on node %d, neither a socket nor a keystone", r.NodeID)
		}
		if r.Art != "" {
			arts = append(arts, r.Art)
		}
	}
	if onKeystone != 1 || !slices.Contains(arts, "Lethal Pride") {
		t.Fatalf("%d rings, %d on a keystone, art %v", len(rings), onKeystone, arts)
	}
}

// TestTreeJewelRingsCompared: PassiveTreeView.lua:1235: a socket allocated only in the compared tree still draws its ring.
func TestTreeJewelRingsCompared(t *testing.T) {
	build := parseCorpusBuild(t, "0005-occultist-vaal-double-strike.xml")
	all := TreeJewelRings(build, nil)
	var socket int64
	for _, r := range all {
		if r.Art == "Lethal Pride" {
			socket = r.NodeID
		}
	}
	build.Build.PassiveNodes = slices.DeleteFunc(build.Build.PassiveNodes, func(id int64) bool { return id == socket })
	if n := len(TreeJewelRings(build, nil)); n != len(all)-1 {
		t.Fatalf("unallocated socket: %d rings, want %d", n, len(all)-1)
	}
	if n := len(TreeJewelRings(build, []int64{socket})); n != len(all) {
		t.Fatalf("socket allocated in the compared tree: %d rings, want %d", n, len(all))
	}
}

// TestTreeSocketRadii: Hovering a socket: the full-disc radii for a jewel, the annuli for Thread of Hope; a radius's nodes include the smaller one's.
func TestTreeSocketRadii(t *testing.T) {
	build := parseCorpusBuild(t, "0016-gladiator-dual-strike-of-ambidexterity.xml")
	var hope, other int64
	for _, s := range build.Tree.Specs[build.ActiveSpecIndex()].Sockets {
		if item, ok := socketedJewel(build, int64(s.NodeID)); ok {
			if item.JewelRadiusLabel == "Variable" {
				hope = int64(s.NodeID)
			} else if _, ok := itemJewelRadiusIndex(item); ok {
				other = int64(s.NodeID)
			}
		}
	}
	if hope == 0 {
		t.Fatal("fixture has no Thread of Hope")
	}
	annuli := TreeSocketRadii(build, hope)
	if len(annuli) != 5 || annuli[0].Inner == 0 || len(annuli[0].Nodes) == 0 {
		t.Fatalf("Thread of Hope: %+v", annuli)
	}
	empty := int64(0)
	for _, s := range treeSockets(specTree()) {
		if _, ok := socketedJewel(build, s); !ok && s != other {
			empty = s
			break
		}
	}
	discs := TreeSocketRadii(build, empty)
	if len(discs) != 5 || discs[0].Inner != 0 || discs[0].Outer != data.JewelRadius[0].Outer {
		t.Fatalf("an empty socket: %+v", discs)
	}
	for _, id := range discs[0].Nodes {
		if !slices.Contains(discs[1].Nodes, id) {
			t.Fatalf("node %d in the small radius, not the medium", id)
		}
	}
	if TreeSocketRadii(build, startOf(t, "Duelist")) != nil {
		t.Fatal("a class start has radii")
	}
}

// TestTreeJewelOverlays: Each allocated socket's jewel art.
func TestTreeJewelOverlays(t *testing.T) {
	build := parseCorpusBuild(t, "0005-occultist-vaal-double-strike.xml")
	overlays := TreeJewelOverlays(build)
	arts := map[string]bool{}
	for _, o := range overlays {
		arts[o.Art] = true
	}
	if !arts["JewelSocketActiveLegion"] || len(overlays) < 3 {
		t.Fatalf("%+v", overlays)
	}
	first := overlays[0].NodeID
	build.DeallocateNodes([]int64{first})
	if slices.ContainsFunc(TreeJewelOverlays(build), func(x TreeJewelOverlay) bool { return x.NodeID == first }) {
		t.Fatalf("socket %d draws its jewel unallocated", first)
	}
}
