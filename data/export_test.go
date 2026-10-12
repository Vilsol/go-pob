package data

import "github.com/dominikbraun/graph"

func TreeGraphForTest(v *TreeVersionData) (graph.Graph[int64, int64], map[int64]map[int64]graph.Edge[int64]) {
	return v.getGraph()
}
