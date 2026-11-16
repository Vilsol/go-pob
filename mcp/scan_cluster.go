package mcp

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/pob"
)

const notablePrefix = "1 Added Passive Skill is "

var tagRe = strings.NewReplacer("{crafted}", "", "{fractured}", "")

// clusterSkill is the cluster jewel skill whose enchant lines an item carries.
func clusterSkill(cj data.ClusterJewel, text string) (data.ClusterJewelSkill, bool) {
	lines := map[string]bool{}
	for _, l := range strings.Split(text, "\n") {
		lines[strings.TrimSpace(tagRe.Replace(l))] = true
	}
	for _, sk := range cj.Skills {
		if len(sk.Enchant) > 0 && !slices.ContainsFunc(sk.Enchant, func(e string) bool { return !lines[e] }) {
			return sk, true
		}
	}
	return data.ClusterJewelSkill{}, false
}

// clusterNotables are the notables a cluster jewel with this skill can roll: the base's "1 Added Passive Skill is"
// affixes weighted for the skill's tag.
func clusterNotables(item *calculator.Item, sk data.ClusterJewelSkill) []string {
	var out []string
	for _, m := range affixPool(item) {
		if len(m.Lines) != 1 || !strings.HasPrefix(m.Lines[0], notablePrefix) {
			continue
		}
		k := slices.Index(m.WeightKey, sk.Tag)
		if k < 0 || k >= len(m.WeightVal) || m.WeightVal[k] <= 0 {
			continue
		}
		if n := strings.TrimPrefix(m.Lines[0], notablePrefix); !slices.Contains(out, n) {
			out = append(out, n)
		}
	}
	return out
}

func replaceItemText(id int, text string) change {
	return func(c *pob.PathOfBuilding) {
		for i := range c.Items.Items {
			if c.Items.Items[i].ID == id {
				c.Items.Items[i].Text = text
			}
		}
	}
}

// jewelSubgraph is the cluster subgraph a jewel socket opens in c, the nodes PoB places in the socket's proxy group
// (PassiveSpec.lua:1806-1882), without the subgraphs of jewels socketed inside it: its nodes and their links, both ways.
func jewelSubgraph(c *pob.PathOfBuilding, socket int64) (map[int64]calculator.TreeClusterNode, map[int64][]int64) {
	adj := map[int64][]int64{}
	nodes := map[int64]calculator.TreeClusterNode{}
	for _, cn := range calculator.TreeClusterNodes(c) {
		nodes[cn.ID] = cn
		for _, l := range cn.Linked {
			adj[cn.ID] = append(adj[cn.ID], l)
			adj[l] = append(adj[l], cn.ID)
		}
	}
	sub := map[int64]calculator.TreeClusterNode{}
	for _, entrance := range adj[socket] {
		n, ok := nodes[entrance]
		if !ok {
			continue
		}
		for id, o := range nodes {
			if o.Group == n.Group {
				sub[id] = o
			}
		}
		break
	}
	return sub, adj
}

// clusterSwap is the change that swaps notable old for n on the cluster jewel item id socketed at socket: the jewel's
// text with the notable replaced and, where the rebuilt subgraph puts a notable the jewel had allocated (n in old's
// place) on a node the tree does not allocate, the nodes that allocate it (the shortest path from the allocated part
// of the jewel's subgraph).
func clusterSwap(b *pob.PathOfBuilding, socket int64, id int, lines []string, old, n string) (change, []int64) {
	next := make([]string, len(lines))
	for i, l := range lines {
		next[i] = strings.Replace(l, notablePrefix+old, notablePrefix+n, 1)
	}
	allocated := map[int64]bool{}
	for _, a := range b.Build.PassiveNodes {
		allocated[a] = true
	}
	// the notables allocated before, with old's place going to n
	want := map[string]bool{}
	before, _ := jewelSubgraph(b, socket)
	for nid, node := range before {
		if node.Type == "Notable" && allocated[nid] {
			want[map[bool]string{true: n, false: node.Name}[node.Name == old]] = true
		}
	}
	swap := replaceItemText(id, strings.Join(next, "\n"))
	c := cloneBuild(b)
	swap(c)
	sub, adj := jewelSubgraph(c, socket)
	var path []int64
	for _, nid := range slices.Sorted(maps.Keys(sub)) {
		if !want[sub[nid].Name] || sub[nid].Type != "Notable" || allocated[nid] {
			continue
		}
		prev := map[int64]int64{nid: nid}
		queue := []int64{nid}
		for len(queue) > 0 && !allocated[queue[0]] {
			at := queue[0]
			queue = queue[1:]
			for _, o := range adj[at] {
				if _, seen := prev[o]; !seen && (o == socket || sub[o].ID != 0) {
					prev[o] = at
					queue = append(queue, o)
				}
			}
		}
		if len(queue) == 0 {
			continue
		}
		for at := prev[queue[0]]; ; at = prev[at] {
			allocated[at] = true
			path = append(path, at)
			if at == nid {
				break
			}
		}
	}
	if len(path) == 0 {
		return swap, nil
	}
	return func(c *pob.PathOfBuilding) {
		swap(c)
		c.AllocateNodes(path)
	}, path
}

// clusters ranks swapping each notable of a socketed cluster jewel for another its skill can roll, by a full
// calculation of the build with the jewel's text changed: the subgraph is rebuilt from the new text and keeps the
// allocated node ids, as PoB's does; a row whose notables land on unallocated nodes allocates them and counts the
// points.
func (sc *scanCtx) clusters(deadline time.Time) []map[string]any {
	texts := map[int]string{}
	for _, it := range sc.b.Items.Items {
		texts[it.ID] = it.Text
	}
	var rows []map[string]any
	for _, sock := range activeSpec(sc.b).Sockets {
		text, ok := texts[sock.ItemID]
		if !ok {
			continue
		}
		item := calculator.ParseItem(text)
		cj, ok := data.ClusterJewels.Jewels[item.BaseName]
		if !ok {
			continue
		}
		sk, ok := clusterSkill(cj, text)
		if !ok {
			continue
		}
		lines := strings.Split(text, "\n")
		var current []string
		for _, l := range lines {
			n, ok := strings.CutPrefix(strings.TrimSpace(tagRe.Replace(l)), notablePrefix)
			if _, notable := data.ClusterJewels.NotableSortOrder[n]; ok && notable {
				current = append(current, n)
			}
		}
		slot := fmt.Sprintf("Jewel %d", sock.NodeID)
		var jewelRows []map[string]any
		for _, old := range current {
			for _, n := range clusterNotables(item, sk) {
				if slices.Contains(current, n) {
					continue
				}
				if time.Now().After(deadline) {
					sc.partial = true
					return append(rows, bestPerSlot(jewelRows, sc.in.PerSlot)...)
				}
				ch, path := clusterSwap(sc.b, int64(sock.NodeID), sock.ItemID, lines, old, n)
				row := sc.row(fmt.Sprintf("cluster|%d|%s|%s", sock.NodeID, old, n), slot, n, evalChange(sc.b, ch), ch)
				row["replaces"] = old
				if len(path) > 0 {
					row["points"] = len(path)
				}
				if sc.keep(row) {
					jewelRows = append(jewelRows, row)
				}
			}
		}
		rows = append(rows, bestPerSlot(jewelRows, sc.in.PerSlot)...)
	}
	return rows
}
