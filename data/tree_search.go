package data

import (
	"container/heap"
	"maps"
	"slices"
	"strconv"
)

type SearchState struct {
	frontier  []int64
	distances map[int64]int64
	nextHops  map[int64]int64
}

func (f *SearchState) Len() int { return len(f.frontier) }
func (f *SearchState) Less(i, j int) bool {
	iNode := f.frontier[i]
	jNode := f.frontier[j]

	if f.distances[iNode] < f.distances[jNode] {
		return true
	}

	if f.distances[iNode] > f.distances[jNode] {
		return false
	}

	return iNode < jNode
}
func (f *SearchState) Swap(i, j int) { f.frontier[i], f.frontier[j] = f.frontier[j], f.frontier[i] }
func (f *SearchState) Push(x any)    { f.frontier = append(f.frontier, x.(int64)) }
func (f *SearchState) Pop() any {
	old := f.frontier
	n := len(old)
	x := old[n-1]
	f.frontier = old[0 : n-1]
	return x
}

//pob:src/Classes/PassiveSpec.lua:537-574
func (v *TreeVersionData) CalculateAllocationPaths(activeNodes []int64, rootNodes []int64) map[int64]int64 {
	_, adjacencyMap := v.getGraph()

	state := SearchState{
		frontier:  make([]int64, len(activeNodes)+len(rootNodes)),
		distances: make(map[int64]int64, len(adjacencyMap)),
		nextHops:  make(map[int64]int64, len(adjacencyMap)),
	}

	for i, activeNode := range activeNodes {
		state.frontier[i] = activeNode
		state.distances[activeNode] = 0
		state.nextHops[activeNode] = -1
	}

	for _, rootNode := range rootNodes {
		_, isActive := state.distances[rootNode]
		if !isActive {
			state.distances[rootNode] = 1
			state.nextHops[rootNode] = -1
			if root, ok := v.Tree().Nodes[strconv.FormatInt(rootNode, 10)]; ok {
				for _, other := range slices.Concat(root.Out, root.In) {
					start, ok := v.Tree().Nodes[other]
					if id, err := strconv.ParseInt(other, 10, 64); ok && err == nil && start.ClassStartIndex != nil {
						state.nextHops[rootNode] = id
						state.nextHops[id] = -1
						break
					}
				}
			}
			state.frontier = append(state.frontier, rootNode)
		}
	}

	blockedTargets, masteryParents := v.pathRuleSets()
	tree := v.Tree()

	heap.Init(&state)
	for state.Len() > 0 {
		currentNode := heap.Pop(&state).(int64)
		currentDistance := state.distances[currentNode]

		if masteryParents[currentNode] {
			continue
		}

		for adjacency := range adjacencyMap[currentNode] {
			if blockedTargets[adjacency] || !tree.inSpec(strconv.FormatInt(adjacency, 10)) {
				continue
			}

			_, alreadyVisited := state.distances[adjacency]
			if alreadyVisited {
				continue
			}

			state.distances[adjacency] = currentDistance + 1
			state.nextHops[adjacency] = currentNode
			heap.Push(&state, adjacency)
		}
	}

	return state.nextHops
}

func (v *TreeVersionData) pathRuleSets() (map[int64]bool, map[int64]bool) {
	nodes := v.Tree().Nodes
	blockedTargets := make(map[int64]bool, 16)
	masteryParents := make(map[int64]bool, 512)
	for id, node := range nodes {
		isStart := node.ClassStartIndex != nil ||
			(node.IsAscendancyStart != nil && *node.IsAscendancyStart)
		isMastery := node.IsMastery != nil && *node.IsMastery
		if !isStart && !isMastery {
			continue
		}
		parsed, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			continue
		}
		if isStart {
			blockedTargets[parsed] = true
		}
		if isMastery {
			masteryParents[parsed] = true
		}
	}
	return blockedTargets, masteryParents
}

//pob:src/Classes/PassiveSpec.lua:844-871
func (v *TreeVersionData) CalculatePrunableNodes(activeNodes []int64, rootNodes []int64) []int64 {
	return v.prunableNodes(activeNodes, rootNodes, nil)
}

func (v *TreeVersionData) prunableNodes(activeNodes []int64, rootNodes []int64, sub *subgraphLinks) []int64 {
	_, adjacencyMap := v.getGraph()
	_, masteryParents := v.pathRuleSets()

	activeNodesSet := make(map[int64]bool, len(activeNodes))
	for _, node := range activeNodes {
		activeNodesSet[node] = true
	}

	stack := make([]int64, 0, len(rootNodes)+len(activeNodes))
	visited := make(map[int64]bool)

	for _, rootNode := range rootNodes {
		if activeNodesSet[rootNode] {
			stack = append(stack, rootNode)
		}
	}

	for len(stack) > 0 {
		currentNode := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if !visited[currentNode] {
			visited[currentNode] = true

			if masteryParents[currentNode] || sub.mastery(currentNode) {
				continue
			}

			for adjacent := range adjacencyMap[currentNode] {
				if activeNodesSet[adjacent] && !visited[adjacent] {
					stack = append(stack, adjacent)
				}
			}
			for _, adjacent := range sub.links(currentNode) {
				if activeNodesSet[adjacent] && !visited[adjacent] {
					stack = append(stack, adjacent)
				}
			}
		}
	}

	prunableNodes := make([]int64, 0, len(activeNodes))
	for _, node := range activeNodes {
		if !visited[node] {
			prunableNodes = append(prunableNodes, node)
		}
	}

	return prunableNodes
}

func (v *TreeVersionData) CalculateDependentNodes(activeNodes []int64, target int64, rootNodes []int64) []int64 {
	return v.CalculateDependentNodesWith(activeNodes, target, rootNodes, nil)
}

type SubgraphNode struct {
	ID     int64
	Type   string
	Linked []int64
}

type subgraphLinks struct {
	nodes map[int64]SubgraphNode
	extra map[int64][]int64
}

func newSubgraphLinks(subgraph []SubgraphNode) *subgraphLinks {
	if len(subgraph) == 0 {
		return nil
	}
	s := &subgraphLinks{nodes: map[int64]SubgraphNode{}, extra: map[int64][]int64{}}
	for _, n := range subgraph {
		s.nodes[n.ID] = n
		for _, other := range n.Linked {
			s.extra[n.ID] = append(s.extra[n.ID], other)
			s.extra[other] = append(s.extra[other], n.ID)
		}
	}
	return s
}

func (s *subgraphLinks) links(id int64) []int64 {
	if s == nil {
		return nil
	}
	return s.extra[id]
}

func (s *subgraphLinks) mastery(id int64) bool {
	return s != nil && s.nodes[id].Type == "Mastery"
}

func (v *TreeVersionData) CalculateDependentNodesWith(activeNodes []int64, target int64, rootNodes []int64, subgraph []SubgraphNode) []int64 {
	sub := newSubgraphLinks(subgraph)
	nodes := v.Tree().Nodes
	rootsFor := func(list []int64) []int64 {
		out := append([]int64(nil), rootNodes...)
		for _, id := range list {
			if n, ok := nodes[strconv.FormatInt(id, 10)]; ok && n.IsAscendancyStart != nil && *n.IsAscendancyStart {
				out = append(out, id)
			}
		}
		return out
	}
	before := make(map[int64]bool)
	for _, id := range v.prunableNodes(activeNodes, rootsFor(activeNodes), sub) {
		before[id] = true
	}
	node := v.SpecDependsNode
	if sub != nil {
		node = func(id int64) (DependsNode, bool) {
			d, ok := v.SpecDependsNode(id)
			if !ok {
				n, in := sub.nodes[id]
				if !in {
					return d, false
				}
				d = DependsNode{Type: n.Type}
			}
			links := slices.Clone(sub.links(id))
			slices.Sort(links)
			for _, other := range links {
				if !slices.Contains(d.Linked, other) {
					d.Linked = append(d.Linked, other)
				}
			}
			return d, true
		}
	}

	alloc := slices.Clone(activeNodes)
	for _, root := range rootNodes {
		if n, ok := v.SpecDependsNode(root); ok {
			for _, id := range append([]int64{root}, n.Linked...) {
				if o, ok := v.SpecDependsNode(id); ok && o.Type == "ClassStart" && !slices.Contains(alloc, id) {
					alloc = append(alloc, id)
				}
			}
		}
	}
	depends := SpecDepends(alloc, node, nil)[target]
	out := make([]int64, 0, len(depends))
	for _, id := range depends {
		if id != target && !before[id] {
			out = append(out, id)
		}
	}
	return out
}

//pob:block:src/Classes/PassiveTree.lua:521-552,src/Classes/PassiveTree.lua:603-628
func (v *TreeVersionData) SpecDependsNode(id int64) (DependsNode, bool) {
	n, ok := v.Tree().Nodes[strconv.FormatInt(id, 10)]
	if !ok {
		return DependsNode{}, false
	}
	out := DependsNode{Type: "Normal"}
	switch {
	case n.ClassStartIndex != nil:
		out.Type = "ClassStart"
	case n.IsAscendancyStart != nil && *n.IsAscendancyStart:
		out.Type = "AscendClassStart"
	case n.IsMastery != nil && *n.IsMastery:
		out.Type = "Mastery"
	case n.IsJewelSocket != nil && *n.IsJewelSocket:
		out.Type = "Socket"
	case n.IsKeystone != nil && *n.IsKeystone:
		out.Type = "Keystone"
	case n.IsNotable != nil && *n.IsNotable:
		out.Type = "Notable"
	}
	if n.AscendancyName != nil {
		out.Ascendancy = *n.AscendancyName
	}
	for _, key := range append(slices.Clone(n.Out), n.In...) {
		if other, err := strconv.ParseInt(key, 10, 64); err == nil {
			out.Linked = append(out.Linked, other)
		}
	}
	return out, true
}

type DependsNode struct {
	Type       string
	Ascendancy string
	Linked     []int64
}

type LeapJewel struct {
	Socket       int64
	LeapLike     bool
	KeystoneOnly bool
	Radius       map[int64]bool
	Escape       map[int64]bool
}

type dependsState struct {
	DependsNode
	id               int64
	alloc            bool
	visited          bool
	connectedToStart bool
	depends          []int64
	leaps            []int64
}

//pob:block:src/Classes/PassiveSpec.lua:1067-1095,src/Classes/PassiveSpec.lua:1133-1183,src/Classes/PassiveSpec.lua:1438-1592
func SpecDepends(allocNodes []int64, node func(int64) (DependsNode, bool), jewels []LeapJewel) map[int64][]int64 {
	nodes := map[int64]*dependsState{}
	get := func(id int64) *dependsState {
		if n, ok := nodes[id]; ok {
			return n
		}
		d, ok := node(id)
		if !ok {
			nodes[id] = nil
			return nil
		}
		n := &dependsState{DependsNode: d, id: id}
		nodes[id] = n
		return n
	}
	alloc := slices.Clone(allocNodes)
	slices.Sort(alloc)
	alloc = slices.Compact(alloc)
	for _, id := range alloc {
		if n := get(id); n != nil {
			n.alloc = true
		}
	}
	jewels = slices.Clone(jewels)
	slices.SortFunc(jewels, func(a, b LeapJewel) int { return int(a.Socket - b.Socket) })
	isStart := func(n *dependsState) bool { return n.Type == "ClassStart" || n.Type == "AscendClassStart" }

	for _, id := range alloc {
		n := get(id)
		if n == nil {
			continue
		}
		if n.Type != "ClassStart" && n.Type != "Socket" && n.Ascendancy == "" {
			for _, j := range jewels {
				if s := get(j.Socket); s == nil || !s.alloc {
					continue
				}
				if j.Radius[id] && j.LeapLike && (!j.KeystoneOnly || n.Type == "Keystone") {
					n.leaps = append(n.leaps, j.Socket)
				}
				if j.Escape[id] {
					n.leaps = append(n.leaps, j.Socket)
				}
			}
		}
		n.depends = []int64{id}
	}

	inRadius := func(n *dependsState) []int64 {
		var out []int64
		for _, j := range jewels {
			if j.Socket != n.id {
				continue
			}
			for _, id := range alloc {
				if j.LeapLike && j.Radius[id] {
					out = append(out, id)
				}
			}
			for _, id := range alloc {
				if j.Escape[id] {
					out = append(out, id)
				}
			}
		}
		return out
	}

	var visited []*dependsState
	var findStart func(n *dependsState) bool
	findStart = func(n *dependsState) bool {
		n.visited = true
		visited = append(visited, n)
		for _, oid := range n.Linked {
			other := get(oid)
			if other == nil {
				continue
			}
			startIndex := len(visited)
			if other.alloc && (isStart(other) || (!other.visited && n.Type != "Mastery" && findStart(other))) {
				if n.Ascendancy != "" && other.Ascendancy == "" {
					for _, v := range visited[startIndex:] {
						v.visited = false
					}
					visited = visited[:startIndex]
				} else {
					return true
				}
			}
		}
		return false
	}

	potentialDeps := map[int64][]int64{}
	intuitiveLeaps := map[int64][]int64{}
	for _, id := range alloc {
		n := get(id)
		if n == nil {
			continue
		}
		n.visited = true
		n.connectedToStart = false
		anyStartFound := isStart(n)
		for _, oid := range n.Linked {
			other := get(oid)
			if other == nil || !other.alloc || slices.Contains(n.depends, oid) {
				continue
			}
			if isStart(other) {
				anyStartFound = true
				n.connectedToStart = true
			} else if findStart(other) {
				anyStartFound = true
				n.connectedToStart = true
				for _, v := range visited {
					v.visited = false
				}
				visited = visited[:0]
			} else {
				depIds := map[int64]bool{}
				for _, v := range visited {
					if len(v.leaps) == 0 {
						depIds[v.id] = true
					}
				}
				for _, v := range visited {
					if v.Type == "Mastery" {
						otherPath := false
						allocatedLinkCount := 0
						for _, lid := range v.Linked {
							if l := get(lid); l != nil && l.alloc {
								allocatedLinkCount++
							}
						}
						if allocatedLinkCount > 1 {
							for _, lid := range v.Linked {
								if l := get(lid); l != nil && l.alloc && !depIds[lid] {
									otherPath = true
								}
							}
						}
						if !otherPath {
							n.depends = append(n.depends, v.id)
						}
					} else {
						if len(v.leaps) > 0 {
							potentialDeps[v.id] = append(potentialDeps[v.id], id)
						} else {
							n.depends = append(n.depends, v.id)
						}
						intuitiveLeaps[id] = append(intuitiveLeaps[id], inRadius(v)...)
					}
					v.visited = false
				}
				visited = visited[:0]
			}
		}
		n.visited = false
		if !anyStartFound {
			for _, dep := range n.depends {
				for _, j := range jewels {
					if s := get(j.Socket); s != nil && s.alloc && ((j.LeapLike && j.Radius[dep]) || j.Escape[dep]) {
						intuitiveLeaps[j.Socket] = append(intuitiveLeaps[j.Socket], dep)
						break
					}
				}
			}
		}
	}

	for _, id := range slices.Sorted(maps.Keys(potentialDeps)) {
		potential := get(id)
		seen := map[int64]bool{}
		for _, nid := range potentialDeps[id] {
			n := get(nid)
			allDep := true
			for _, provider := range potential.leaps {
				if !slices.Contains(n.depends, provider) {
					allDep = false
				}
			}
			if allDep && !seen[nid] {
				n.depends = append(n.depends, id)
				seen[nid] = true
			}
		}
	}

	for _, id := range slices.Sorted(maps.Keys(intuitiveLeaps)) {
		n := get(id)
		seen := map[int64]bool{}
		for _, did := range intuitiveLeaps[id] {
			dep := get(did)
			if dep.connectedToStart {
				continue
			}
			allDep := true
			for _, leap := range dep.leaps {
				if !slices.Contains(n.depends, leap) {
					allDep = false
					break
				}
			}
			if allDep && !seen[did] {
				n.depends = append(n.depends, did)
				seen[did] = true
			}
		}
	}

	out := make(map[int64][]int64, len(alloc))
	for _, id := range alloc {
		if n := get(id); n != nil {
			out[id] = n.depends
		}
	}
	return out
}

//pob:block:src/Classes/PassiveSpec.lua:44
func (t *Tree) inSpec(key string) bool {
	node, ok := t.Nodes[key]
	if !ok || node.Group == nil || boolVal(node.IsProxy) {
		return false
	}
	if group, ok := t.Groups[strconv.FormatInt(*node.Group, 10)]; ok && boolVal(group.IsProxy) {
		return false
	}
	return node.ExpansionJewel == nil || node.ExpansionJewel.Parent == nil
}
