package calculator

import (
	"cmp"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/data"
)

func (b *PowerBuilder) powerTaskKeys(id int64) []string {
	if _, ok := b.power[id]; !ok {
		return []string{statsModKey(b.treeNode(id).Stats)}
	}
	n := b.treeNode(id)
	if isMasteryNode(n) && !b.alloc[id] {
		keys := make([]string, 0, len(n.MasteryEffects))
		for _, e := range n.MasteryEffects {
			keys = append(keys, "effect:"+strings.Join(e.Stats, "\n"))
		}
		return keys
	}
	if b.alloc[id] {
		return []string{b.keys[id] + "_remove"}
	}
	return []string{b.keys[id]}
}

func (b *PowerBuilder) Shard(index, count int) []int64 {
	parent := map[int64]int64{}
	var find func(int64) int64
	find = func(x int64) int64 {
		p, ok := parent[x]
		if !ok || p == x {
			return x
		}
		r := find(p)
		parent[x] = r
		return r
	}
	byKey := map[string]int64{}
	for _, id := range b.taskNodes {
		for _, k := range b.powerTaskKeys(id) {
			if o, ok := byKey[k]; ok {
				if a, c := find(id), find(o); a != c {
					parent[a] = c
				}
			} else {
				byKey[k] = id
			}
		}
	}
	var groups []int64
	size := map[int64]int{}
	for _, id := range b.taskNodes {
		r := find(id)
		if size[r] == 0 {
			groups = append(groups, r)
		}
		size[r]++
	}
	owner := map[int64]int{}
	load := make([]int, count)
	for _, r := range groups {
		s := 0
		for i := range load {
			if load[i] < load[s] {
				s = i
			}
		}
		owner[r] = s
		load[s] += size[r]
	}
	var tasks []func()
	var nodes, owned []int64
	for i, id := range b.taskNodes {
		if owner[find(id)] != index {
			continue
		}
		tasks = append(tasks, b.tasks[i])
		nodes = append(nodes, id)
		owned = append(owned, id)
	}
	b.tasks, b.taskNodes, b.next = tasks, nodes, 0
	slices.Sort(owned)
	return slices.Compact(owned)
}

func (b *PowerBuilder) Tasks() int {
	return len(b.tasks)
}

type PowerShardRows struct {
	Rows       []PowerReportEntry
	Owned      []int64
	Ascendancy []PowerReportEntry
}

func fromOwner(owner map[int64]int, id int64, s int) bool {
	o, ok := owner[id]
	return (ok && o == s) || (!ok && s == 0)
}

func shardOwners(shards []PowerShardRows) map[int64]int {
	owner := map[int64]int{}
	for s, shard := range shards {
		for _, id := range shard.Owned {
			owner[id] = s
		}
	}
	return owner
}

func MergePowerAscendancy(shards []PowerShardRows) []PowerReportEntry {
	owner := shardOwners(shards)
	var out []PowerReportEntry
	for s, shard := range shards {
		for _, e := range shard.Ascendancy {
			if fromOwner(owner, e.ID, s) {
				out = append(out, e)
			}
		}
	}
	slices.SortStableFunc(out, func(a, b PowerReportEntry) int { return cmp.Compare(a.ID, b.ID) })
	return out
}

func MergePowerReports(shards []PowerShardRows) []PowerReportEntry {
	owner := shardOwners(shards)
	tree := data.TreeVersions[data.LatestTreeVersion].Tree()
	type placed struct {
		row    PowerReportEntry
		effect int
	}
	var rows []placed
	for s, shard := range shards {
		for _, r := range shard.Rows {
			if !fromOwner(owner, r.ID, s) {
				continue
			}
			effect := 0
			if r.Type == "Mastery" && !r.Cluster {
				effect = slices.IndexFunc(tree.Nodes[strconv.FormatInt(r.ID, 10)].MasteryEffects, func(e data.MasteryEffect) bool {
					return slices.Equal(splitStatLines(e.Stats), r.Stats)
				})
			}
			rows = append(rows, placed{row: r, effect: effect})
		}
	}
	slices.SortStableFunc(rows, func(a, b placed) int {
		if a.row.Cluster != b.row.Cluster {
			if a.row.Cluster {
				return 1
			}
			return -1
		}
		if a.row.ID != b.row.ID {
			if a.row.ID < b.row.ID {
				return -1
			}
			return 1
		}
		return a.effect - b.effect
	})
	report := make([]PowerReportEntry, len(rows))
	for i, p := range rows {
		report[i] = p.row
	}
	sort.SliceStable(report, func(i, j int) bool { return report[i].Power > report[j].Power })
	return report
}
