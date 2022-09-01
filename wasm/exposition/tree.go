package exposition

import "github.com/Vilsol/go-pob/data"

func GetRawTree(version data.TreeVersion) []byte {
	return data.TreeVersions[version].RawTree()
}
