package data

import (
	"testing"

	"github.com/Vilsol/go-pob/config"
	"github.com/Vilsol/go-pob/data/raw"
)

func init() {
	config.InitLogging(false)

	if err := raw.InitializeAll(raw.LatestVersion, nil); err != nil {
		panic(err)
	}
}

func TestLoadTreeGraph(t *testing.T) {
	TreeVersions[TreeVersion3_18].getGraph()
}

func BenchmarkGraphSearch(b *testing.B) {
	TreeVersions[TreeVersion3_18].getGraph()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		TreeVersions[TreeVersion3_18].CalculateTreePath([]int64{48828, 55373, 2151, 47062, 15144, 62103}, 23881)
	}
}
