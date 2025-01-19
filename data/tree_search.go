package data

import "container/heap"

type SearchState struct {
	frontier  []int64
	distances map[int64]int64
	nextHops  map[int64]int64
}

// heap.Interface implementation to maintain SearchState.frontier as a
// priority queue sorted by distance from active nodes.
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

	// We want results to be consistent every time we run the algorithm,
	// so we use the node ID as a tiebreaker
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

// Calculates the next hop you should take to traverse a shortest path from any
// arbitrary node to the nearest active node. Active nodes map to -1. Disconnected
// tree nodes will not appear in the result.
//
// The algorithm will always pick consistent paths through the tree each time it is
// run when there are multiple shortest paths (this property is important to prevent
// the skill tree UI from flip-flopping between options as users allocate nodes).
//
// Requires a single BFS of the tree, + a heap push/pop pair per node.
// Time complexity: O(V * log(V) + E)
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
			// A root node is reachable, but if it isn't already
			// active, we'd need to spend 1 distance allocating it.
			state.distances[rootNode] = 1
			state.nextHops[rootNode] = -1
			state.frontier = append(state.frontier, rootNode)
		}
	}

	// This is a BFS, but using a priority queue that sorts by both
	// distance *and* node ID. The use of node ID ensures that we
	// produce stable output when there are multiple shortest paths,
	// despite the arbitrary iteration order of neighbors in the
	// adjacency map.
	heap.Init(&state)
	for state.Len() > 0 {
		currentNode := heap.Pop(&state).(int64)
		currentDistance := state.distances[currentNode]

		for adjacency := range adjacencyMap[currentNode] {
			_, alreadyVisited := state.distances[adjacency]
			if alreadyVisited {
				// Visiting in heap order means that we can assume
				// that any path we already found is no longer than
				// this new path.
				continue
			}

			state.distances[adjacency] = currentDistance + 1
			state.nextHops[adjacency] = currentNode
			heap.Push(&state, adjacency)
		}
	}

	return state.nextHops
}

func (v *TreeVersionData) CalculatePrunableNodes(activeNodes []int64, rootNodes []int64) []int64 {
	_, adjacencyMap := v.getGraph()

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
