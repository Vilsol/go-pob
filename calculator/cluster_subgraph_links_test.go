package calculator

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/pob"
)

var clusterCensusJewels = []struct{ name, text string }{
	{"small keystone", "Rarity: UNIQUE\nNatural Affinity\nSmall Cluster Jewel\nItem Level: 79\nImplicits: 0\nAdds Nature's Patience\nCorrupted"},
	{"small 3", "Rarity: RARE\nArmageddon Ice\nSmall Cluster Jewel\nItem Level: 85\nImplicits: 2\n{crafted}Adds 3 Passive Skills\n" +
		"{crafted}Added Small Passive Skills grant: 6% increased Mana Reservation Efficiency of Skills\n1 Added Passive Skill is Self-Control"},
	{"small 2", "Rarity: MAGIC\nSmall Cluster Jewel\nItem Level: 85\nImplicits: 2\n{crafted}Adds 2 Passive Skills\n" +
		"{crafted}Added Small Passive Skills grant: 6% increased Mana Reservation Efficiency of Skills"},
	{"medium 4+socket", "Rarity: RARE\nMaelstrom Essence\nMedium Cluster Jewel\nItem Level: 82\nImplicits: 4\n{crafted}Adds 4 Passive Skills\n" +
		"{crafted}1 Added Passive Skill is a Jewel Socket\n{crafted}Added Small Passive Skills grant: Minions deal 10% increased Damage while you are affected by a Herald\n" +
		"1 Added Passive Skill is Cult-Leader\n1 Added Passive Skill is Pure Agony"},
	{"large voices", "Rarity: UNIQUE\nVoices\nLarge Cluster Jewel\nItem Level: 84\nImplicits: 3\n{crafted}Adds 3 Passive Skills\n" +
		"{crafted}Added Small Passive Skills grant: 12% increased Physical Damage\nAdds 1 Jewel Socket Passive Skill\nAdds 3 Small Passive Skills which grant nothing"},
	{"large 8+2 sockets", "Rarity: RARE\nApocalypse Eye\nLarge Cluster Jewel\nItem Level: 83\nImplicits: 3\n{crafted}Adds 8 Passive Skills\n" +
		"{crafted}2 Added Passive Skills are Jewel Sockets\n{crafted}Added Small Passive Skills grant: 12% increased Physical Damage\n" +
		"1 Added Passive Skill is Battle-Hardened\n1 Added Passive Skill is Furious Assault\n1 Added Passive Skill is Master the Fundamentals"},
}

// TestClusterSubgraphsConnectToTheirSocket: PassiveSpec.lua:2094, :2341: every subgraph links its entrance to the socket it is in.
func TestClusterSubgraphsConnectToTheirSocket(t *testing.T) {
	requireGameData(t, "3.29")
	tree := data.TreeVersions[data.LatestTreeVersion].Tree()
	build := parseCorpusBuild(t, "0021-assassin-blade-flurry.xml")
	build.Items.Items = nil
	for i, j := range clusterCensusJewels {
		build.Items.Items = append(build.Items.Items, pob.Item{ID: i + 1, Text: j.text})
	}
	spec := &build.Tree.Specs[max(build.Tree.ActiveSpec-1, 0)]

	var broken []string
	cases := 0
	var walk func(chain []pob.Socket)
	walk = func(chain []pob.Socket) {
		spec.Sockets = chain
		ids := make([]string, len(chain))
		for i, s := range chain {
			ids[i] = strconv.Itoa(s.NodeID)
		}
		spec.NodesAttr = strings.Join(ids, ",")
		nodes := TreeClusterNodes(build)
		adjacent := map[int64][]int64{}
		for _, n := range nodes {
			for _, o := range n.Linked {
				adjacent[n.ID] = append(adjacent[n.ID], o)
				adjacent[o] = append(adjacent[o], n.ID)
			}
		}
		seen := map[int64]bool{int64(chain[0].NodeID): true}
		queue := []int64{int64(chain[0].NodeID)}
		for len(queue) > 0 {
			id := queue[0]
			queue = queue[1:]
			for _, o := range adjacent[id] {
				if !seen[o] {
					seen[o] = true
					queue = append(queue, o)
				}
			}
		}
		label := make([]string, len(chain))
		for i, s := range chain {
			label[i] = fmt.Sprintf("%d:%s", s.NodeID, clusterCensusJewels[s.ItemID-1].name)
		}
		cases++
		drawn := 0
		for _, n := range nodes {
			if n.Type == "Mastery" {
				continue
			}
			drawn++
			if !seen[n.ID] {
				broken = append(broken, fmt.Sprintf("%s: %s %d %q", strings.Join(label, " > "), n.Type, n.ID, n.Name))
			}
		}
		if drawn == 0 {
			broken = append(broken, strings.Join(label, " > ")+": no subgraph")
		}
		if len(chain) >= 3 {
			return
		}
		for _, n := range nodes {
			if n.Type != "Socket" || slices.ContainsFunc(chain, func(s pob.Socket) bool { return int64(s.NodeID) == n.ID }) {
				continue
			}
			size := func(itemID int) int {
				return data.ClusterJewels.Jewels[PeekItem(clusterCensusJewels[itemID-1].text).BaseName].SizeIndex
			}
			smallest := size(slices.MinFunc(chain, func(a, b pob.Socket) int { return size(a.ItemID) - size(b.ItemID) }).ItemID)
			for i, j := range clusterCensusJewels {
				if size(i+1) < smallest && IsJewelValidForSocket(PeekItem(j.text), int(n.ID)) {
					walk(append(slices.Clone(chain), pob.Socket{NodeID: int(n.ID), ItemID: i + 1}))
				}
			}
		}
	}
	for _, slot := range tree.JewelSlots {
		socket := tree.Nodes[strconv.FormatInt(slot, 10)]
		if socket.ExpansionJewel == nil || socket.ExpansionJewel.Size != 2 {
			continue
		}
		for i := range clusterCensusJewels {
			walk([]pob.Socket{{NodeID: int(slot), ItemID: i + 1}})
		}
	}
	t.Logf("%d socket chains", cases)
	if cases < 100 {
		t.Fatalf("REFUSING: only %d socket chains", cases)
	}
	if len(broken) > 0 {
		t.Errorf("%d nodes not reachable from their socket:\n%s", len(broken), strings.Join(broken[:min(len(broken), 60)], "\n"))
	}
}
