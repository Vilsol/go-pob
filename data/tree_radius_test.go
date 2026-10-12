package data

import (
	"encoding/json"
	"math"
	"os"
	"sort"
	"strconv"
	"testing"

	"github.com/MarvinJWendt/testza"
)

const jewelRadiusExpected = "../testdata/oracle/jewel_radius_expected.json"

type expectedJewelRadius struct {
	TreeVersion    string  `json:"treeVersion"`
	OrbitRadii     []int64 `json:"orbitRadii"`
	SkillsPerOrbit []int64 `json:"skillsPerOrbit"`
	JewelRadius    []struct {
		Inner float64 `json:"inner"`
		Outer float64 `json:"outer"`
	} `json:"jewelRadius"`
	NodePositions map[string]struct {
		X      float64 `json:"x"`
		Y      float64 `json:"y"`
		Ascend bool    `json:"ascend"`
	} `json:"nodePositions"`
	SocketCount int `json:"socketCount"`
	Sockets     map[string]struct {
		X             float64   `json:"x"`
		Y             float64   `json:"y"`
		NodesInRadius [][]int64 `json:"nodesInRadius"`
	} `json:"sockets"`
}

func TestJewelRadiusNodesAgainstPoB(t *testing.T) {
	raw, err := os.ReadFile(jewelRadiusExpected)
	testza.AssertNoError(t, err)
	var want expectedJewelRadius
	testza.AssertNoError(t, json.Unmarshal(raw, &want))

	tree := TreeVersions[LatestTreeVersion].Tree()

	testza.AssertEqual(t, want.OrbitRadii, tree.Constants.OrbitRadii, "orbitRadii differ; trees are not the same export")
	testza.AssertEqual(t, want.SkillsPerOrbit, tree.Constants.SkillsPerOrbit, "skillsPerOrbit differ")
	testza.AssertEqual(t, len(want.JewelRadius), len(JewelRadius), "jewel radius count")
	for i, r := range want.JewelRadius {
		testza.AssertEqual(t, r.Inner, JewelRadius[i].Inner, "radius %d inner", i)
		testza.AssertEqual(t, r.Outer, JewelRadius[i].Outer, "radius %d outer", i)
	}

	ascend := make(map[int64]bool)
	for key, np := range want.NodePositions {
		if np.Ascend {
			id, err := strconv.ParseInt(key, 10, 64)
			testza.AssertNoError(t, err)
			ascend[id] = true
		}
	}
	testza.AssertGreater(t, len(ascend), 0, "expected some ascendancy nodes in the dump")

	got := tree.JewelRadiusNodes()
	testza.AssertEqual(t, want.SocketCount, len(got), "socket count")

	for socketKey, ws := range want.Sockets {
		socketID, err := strconv.ParseInt(socketKey, 10, 64)
		testza.AssertNoError(t, err)

		gs, ok := got[socketID]
		testza.AssertTrue(t, ok, "socket %d missing from go-pob", socketID)
		if !ok {
			continue
		}

		testza.AssertEqual(t, len(ws.NodesInRadius), len(gs), "socket %d radius count", socketID)
		if len(ws.NodesInRadius) != len(gs) {
			continue
		}

		for ri := range ws.NodesInRadius {
			var wantIDs []int64
			for _, id := range ws.NodesInRadius[ri] {
				if !ascend[id] {
					wantIDs = append(wantIDs, id)
				}
			}
			gotIDs := append([]int64(nil), gs[ri]...)
			sort.Slice(wantIDs, func(a, b int) bool { return wantIDs[a] < wantIDs[b] })
			sort.Slice(gotIDs, func(a, b int) bool { return gotIDs[a] < gotIDs[b] })

			testza.AssertEqual(t, len(wantIDs), len(gotIDs), "socket %d radius %d node count", socketID, ri)
			if len(wantIDs) == len(gotIDs) {
				for i := range wantIDs {
					testza.AssertEqual(t, wantIDs[i], gotIDs[i], "socket %d radius %d node[%d]", socketID, ri, i)
				}
			}
		}
	}
}

// TestAscendancyPositionsStillDivergeFromPoB is the retiring condition for includedInRadius's ascendancy arm, made executable.
func TestAscendancyPositionsStillDivergeFromPoB(t *testing.T) {
	raw, err := os.ReadFile(jewelRadiusExpected)
	testza.AssertNoError(t, err)
	var want expectedJewelRadius
	testza.AssertNoError(t, json.Unmarshal(raw, &want))

	tree := TreeVersions[LatestTreeVersion].Tree()
	geom := tree.geometry()

	type delta struct{ dx, dy float64 }
	var mainMatch, mainDiffer, ascMatch, ascDiffer int
	offsets := map[int64][]delta{}

	for key, wp := range want.NodePositions {
		id, err := strconv.ParseInt(key, 10, 64)
		testza.AssertNoError(t, err)
		got, ok := geom.positions[id]
		if !ok || !got.valid {
			continue
		}
		matches := math.Abs(got.x-wp.X) < 1e-6 && math.Abs(got.y-wp.Y) < 1e-6
		if !wp.Ascend {
			if matches {
				mainMatch++
			} else {
				mainDiffer++
			}
			continue
		}
		if matches {
			ascMatch++
		} else {
			ascDiffer++
		}
		if node, ok := tree.Nodes[key]; ok && node.Group != nil {
			offsets[*node.Group] = append(offsets[*node.Group], delta{got.x - wp.X, got.y - wp.Y})
		}
	}

	testza.AssertGreater(t, mainMatch, 0, "no main-tree positions compared; the fixture or the tree is not what this test thinks")
	testza.AssertEqual(t, 0, mainDiffer, "%d main-tree node positions differ from PoB — this is our orbit math, not a data divergence, and includedInRadius's ascendancy arm does not excuse it", mainDiffer)

	testza.AssertGreater(t, ascDiffer, 0, "expected ascendancy nodes in the fixture")
	testza.AssertEqual(t, 0, ascMatch,
		"%d of %d ascendancy node positions now AGREE with PoB. go-pob-data's tree export has been "+
			"fixed (wholly or partly). Re-measure, and if all of them agree, DELETE the ascendancy arm "+
			"in includedInRadius AND the `ascend` filter in TestJewelRadiusNodesAgainstPoB — leaving "+
			"them narrows every socket's radius set against a tree that no longer needs it.",
		ascMatch, ascMatch+ascDiffer)

	varying := 0
	for _, ds := range offsets {
		for _, d := range ds[1:] {
			if math.Abs(d.dx-ds[0].dx) > 1e-6 || math.Abs(d.dy-ds[0].dy) > 1e-6 {
				varying++
				break
			}
		}
	}
	testza.AssertEqual(t, 0, varying, "%d ascendancy groups have a NON-constant per-node offset from PoB; the divergence is no longer purely a group coordinate", varying)
}
