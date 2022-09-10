package exposition

import "github.com/Vilsol/go-pob/data"

func GetRawTree(version data.TreeVersion) []byte {
	return data.TreeVersions[version].RawTree()
}

func CalculateTreePath(version data.TreeVersion, activeNodes []int64, target int64) []int64 {
	return data.TreeVersions[version].CalculateTreePath(activeNodes, target)
}
