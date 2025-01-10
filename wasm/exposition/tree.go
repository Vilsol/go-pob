package exposition

import (
	"github.com/Vilsol/go-pob/data"
)

func GetRawTree(version data.TreeVersion) []byte {
	return data.TreeVersions[version].RawTree()
}

func CalculateAllocationPaths(version data.TreeVersion, activeNodes []int64, rootNodes []int64) map[int64]int64 {
	return data.TreeVersions[version].CalculateAllocationPaths(activeNodes, rootNodes)
}

// TODO: Need some algorithm that figures out which nodes would be disconnected and therefore removed if the target node is removed
// Important steps:
// 1a. On allocation calculate and store a list of adjacent nodes that are currently on a path towards a start node (pathsToStart)
// 1b. Also make sure to check all previously allocated nodes and recalculate any new start paths adjacent nodes
// 2. On Deallocate start pruning all subtrees connected to the target node that ARE NOT listed in the pathsToStart
func CalculatePrunableNodes(version data.TreeVersion, activeNodes []int64, rootNodes []int64) []int64 {
	return data.TreeVersions[version].CalculatePrunableNodes(activeNodes, rootNodes)
}
