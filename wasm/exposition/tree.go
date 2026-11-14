package exposition

import (
	"fmt"

	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/pob"
)

func CalcTreeVersion(build *pob.PathOfBuilding) pob.TreeVersion {
	if build == nil {
		return data.LatestTreeVersion
	}
	return calculator.CalcTreeVersion(build)
}

func LatestTreeVersion() pob.TreeVersion {
	return data.LatestTreeVersion
}

func TreeLoaded(version pob.TreeVersion) bool {
	v, ok := data.TreeVersions[version]
	return ok && v.RawLoaded()
}

// TreeTab.lua:513-516 refuses a tree version it has no data for
func treeVersion(version pob.TreeVersion) (*data.TreeVersionData, error) {
	v, ok := data.TreeVersions[version]
	if !ok {
		return nil, fmt.Errorf("unknown passive tree version %q", version)
	}
	return v, nil
}

func GetRawTree(version pob.TreeVersion) ([]byte, error) {
	v, err := treeVersion(version)
	if err != nil {
		return nil, err
	}
	return v.RawTree(), nil
}

func CalculateAllocationPaths(version pob.TreeVersion, activeNodes []int64, rootNodes []int64) (map[int64]int64, error) {
	v, err := treeVersion(version)
	if err != nil {
		return nil, err
	}
	return v.CalculateAllocationPaths(activeNodes, rootNodes), nil
}

func CalculatePrunableNodes(version pob.TreeVersion, activeNodes []int64, rootNodes []int64) ([]int64, error) {
	v, err := treeVersion(version)
	if err != nil {
		return nil, err
	}
	return v.CalculatePrunableNodes(activeNodes, rootNodes), nil
}

func CalculateDependentNodes(version pob.TreeVersion, activeNodes []int64, target int64, rootNodes []int64) ([]int64, error) {
	v, err := treeVersion(version)
	if err != nil {
		return nil, err
	}
	return v.CalculateDependentNodes(activeNodes, target, rootNodes), nil
}

func CalculateDependentNodesWith(version pob.TreeVersion, activeNodes []int64, target int64, rootNodes []int64, subgraph []data.SubgraphNode) ([]int64, error) {
	v, err := treeVersion(version)
	if err != nil {
		return nil, err
	}
	return v.CalculateDependentNodesWith(activeNodes, target, rootNodes, subgraph), nil
}
