package calculator

import (
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/data"
)

// TestSavedTreeJewelsFitTheirSockets: Every jewel PoB saved in a tree socket of the corpus fits that socket here.
func TestSavedTreeJewelsFitTheirSockets(t *testing.T) {
	files, err := filepath.Glob("../testdata/corpus-3.29/*.xml")
	if err != nil || len(files) == 0 {
		t.Fatalf("REFUSING: no corpus (%v)", err)
	}
	sort.Strings(files)
	tree := data.TreeVersions[data.LatestTreeVersion].Tree()
	checked, clusters := 0, 0
	for _, f := range files[:min(300, len(files))] {
		raw, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		build, err := builds.ParseBuild(raw)
		if err != nil {
			continue
		}
		i := build.ActiveSpecIndex()
		if i < 0 {
			continue
		}
		texts := map[int]string{}
		for _, it := range build.Items.Items {
			texts[it.ID] = it.Text
		}
		for _, s := range build.Tree.Specs[i].Sockets {
			text, ok := texts[s.ItemID]
			if _, main := tree.Nodes[strconv.Itoa(s.NodeID)]; !ok || !main {
				continue
			}
			item := parseItemGlobal(text)
			if item.Base == nil {
				continue
			}
			if !IsJewelValidForSocket(item, s.NodeID) {
				t.Errorf("%s: %s (%s) does not fit socket %d", filepath.Base(f), item.Name, item.BaseName, s.NodeID)
			}
			if _, ok := data.ClusterJewels.Jewels[item.BaseName]; ok {
				clusters++
			}
			checked++
		}
	}
	if checked < 100 || clusters < 10 {
		t.Fatalf("only %d jewels (%d cluster) checked", checked, clusters)
	}
	t.Logf("%d socketed jewels checked, %d of them cluster jewels", checked, clusters)
}

// TestSocketTreeJewel: 0008's tree 3: a socket offers the build's jewels and no other item.
func TestSocketTreeJewel(t *testing.T) {
	build := parseCorpusBuild(t, "0008-necromancer-vaal-summon-skeletons.xml")
	tree := data.TreeVersions[data.LatestTreeVersion].Tree()
	outer := TreeSocketItemIDs(build, 7960)
	for _, id := range []int{3, 10, 15, 5, 9, 8} {
		if !slices.Contains(outer, id) {
			t.Errorf("socket 7960 does not offer item %d (%v)", id, outer)
		}
	}
	var plain int
	for id, n := range tree.Nodes {
		if n.IsJewelSocket != nil && *n.IsJewelSocket && n.ExpansionJewel == nil && (n.Name == nil || *n.Name != "Charm Socket") {
			plain, _ = strconv.Atoi(id)
			break
		}
	}
	ids := TreeSocketItemIDs(build, plain)
	if slices.Contains(ids, 3) || slices.Contains(ids, 10) || slices.Contains(ids, 15) || !slices.Contains(ids, 5) {
		t.Errorf("plain socket %d offers %v", plain, ids)
	}
	for _, it := range build.Items.Items {
		if slices.Contains(ids, it.ID) && parseItemGlobal(it.Text).Type != "Jewel" {
			t.Errorf("socket offers item %d, not a jewel", it.ID)
		}
	}
	if SocketTreeJewel(build, plain, 3) {
		t.Error("a large cluster jewel went into a socket with no expansion slot")
	}
	if !SocketTreeJewel(build, 2311, 0) || build.SocketedJewel(2311) != 0 {
		t.Fatal("None did not empty socket 2311")
	}
	for _, s := range build.Tree.Specs[2].Sockets {
		if s.NodeID == 2311 {
			t.Fatal("an empty socket is saved")
		}
	}
	if !SocketTreeJewel(build, 2311, 12) || build.SocketedJewel(2311) != 12 {
		t.Fatal("jewel 12 did not go into socket 2311")
	}
	if SocketTreeJewel(build, 2311, 12) {
		t.Error("choosing the socket's own jewel reported a change")
	}
}
