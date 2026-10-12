package exposition

import (
	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/pob"
)

func CalcTreeVersion() pob.TreeVersion {
	return data.LatestTreeVersion
}

func GetRawTree(version pob.TreeVersion) []byte {
	return data.TreeVersions[version].RawTree()
}

func CalculateAllocationPaths(version pob.TreeVersion, activeNodes []int64, rootNodes []int64) map[int64]int64 {
	return data.TreeVersions[version].CalculateAllocationPaths(activeNodes, rootNodes)
}

func CalculatePrunableNodes(version pob.TreeVersion, activeNodes []int64, rootNodes []int64) []int64 {
	return data.TreeVersions[version].CalculatePrunableNodes(activeNodes, rootNodes)
}

func CalculateDependentNodes(version pob.TreeVersion, activeNodes []int64, target int64, rootNodes []int64) []int64 {
	return data.TreeVersions[version].CalculateDependentNodes(activeNodes, target, rootNodes)
}
