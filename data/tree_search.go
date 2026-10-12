package data

import (
	"container/heap"
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

			if masteryParents[currentNode] {
				continue
			}

			for adjacent := range adjacencyMap[currentNode] {
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

//pob:block:src/Classes/PassiveSpec.lua:1440-1497
func (v *TreeVersionData) CalculateDependentNodes(activeNodes []int64, target int64, rootNodes []int64) []int64 {
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

	rest := make([]int64, 0, len(activeNodes))
	for _, id := range activeNodes {
		if id != target {
			rest = append(rest, id)
		}
	}

	before := make(map[int64]bool)
	for _, id := range v.CalculatePrunableNodes(activeNodes, rootsFor(activeNodes)) {
		before[id] = true
	}
	after := v.CalculatePrunableNodes(rest, rootsFor(rest))
	out := make([]int64, 0, len(after))
	for _, id := range after {
		if !before[id] {
			out = append(out, id)
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
