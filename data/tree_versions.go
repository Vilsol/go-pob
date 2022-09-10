package data

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/andybalholm/brotli"
	"github.com/dominikbraun/graph"
	"github.com/goccy/go-json"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/Vilsol/go-pob/cache"
)

type TreeVersion string

const (
	TreeVersion3_10 = TreeVersion("3_10")
	TreeVersion3_11 = TreeVersion("3_11")
	TreeVersion3_12 = TreeVersion("3_12")
	TreeVersion3_13 = TreeVersion("3_13")
	TreeVersion3_14 = TreeVersion("3_14")
	TreeVersion3_15 = TreeVersion("3_15")
	TreeVersion3_16 = TreeVersion("3_16")
	TreeVersion3_17 = TreeVersion("3_17")
	TreeVersion3_18 = TreeVersion("3_18")
)

const LatestTreeVersion = TreeVersion3_18
const DefaultTreeVersion = TreeVersion3_10

type TreeVersionData struct {
	Display      string
	Num          float64
	URL          string
	cachedTree   *Tree
	rawTree      []byte
	graph        graph.Graph[int64, int64]
	adjacencyMap map[int64]map[int64]graph.Edge[int64]
}

const cdnTreeBase = "https://go-pob-data.pages.dev/data/%s/tree/data.json.br"

func (v *TreeVersionData) Tree() *Tree {
	if v.cachedTree != nil {
		return v.cachedTree
	}

	var outTree Tree
	if err := json.Unmarshal(v.RawTree(), &outTree); err != nil {
		panic(errors.Wrap(err, "failed to decode file"))
	}
	v.cachedTree = &outTree

	return v.cachedTree
}

func (v *TreeVersionData) RawTree() []byte {
	if v.rawTree != nil {
		return v.rawTree
	}

	treeURL := fmt.Sprintf(cdnTreeBase, v.Display)
	var compressedTree []byte
	if cache.Disk().Exists(treeURL) {
		var err error
		compressedTree, err = cache.Disk().Get(treeURL)
		if err != nil {
			panic(err)
		}
	} else {
		log.Debug().Str("url", treeURL).Msg("fetching")
		response, err := http.DefaultClient.Get(treeURL)
		if err != nil {
			panic(errors.Wrap(err, "failed to fetch url: "+treeURL))
		}
		defer response.Body.Close()

		compressedTree, err = io.ReadAll(response.Body)
		if err != nil {
			panic(errors.Wrap(err, "failed to read response body"))
		}

		defer func() {
			_ = cache.Disk().Set(treeURL, compressedTree)
		}()
	}

	unzipStream := brotli.NewReader(bytes.NewReader(compressedTree))

	var err error
	v.rawTree, err = io.ReadAll(unzipStream)
	if err != nil {
		panic(errors.Wrap(err, "failed to read unzipped data"))
	}

	return v.rawTree
}

func (v *TreeVersionData) getGraph() (graph.Graph[int64, int64], map[int64]map[int64]graph.Edge[int64]) {
	if v.graph != nil {
		return v.graph, v.adjacencyMap
	}

	g := graph.New(func(v int64) int64 {
		return v
	}, graph.Directed())

	for _, node := range v.Tree().Nodes {
		if node.Skill == nil {
			continue
		}

		_ = g.AddVertex(*node.Skill)
	}

	for _, node := range v.Tree().Nodes {
		if node.Skill == nil {
			continue
		}

		for _, target := range node.Out {
			targetID, err := strconv.ParseInt(target, 10, 64)
			if err != nil {
				continue
			}

			targetNode := v.Tree().Nodes[target]
			if targetNode.ClassStartIndex != nil {
				continue
			}

			if (targetNode.AscendancyName != nil && node.AscendancyName != nil && *targetNode.AscendancyName != *node.AscendancyName) ||
				(targetNode.AscendancyName == nil && node.AscendancyName != nil) ||
				(node.AscendancyName == nil && targetNode.AscendancyName != nil) {
				continue
			}

			_ = g.AddEdge(targetID, *node.Skill)
			_ = g.AddEdge(*node.Skill, targetID)
		}
	}

	v.graph = g

	// We can pre-calculate the adjacency map, as the graph won't change
	v.adjacencyMap, _ = v.graph.AdjacencyMap()

	return v.graph, v.adjacencyMap
}

func (v *TreeVersionData) CalculateTreePath(activeNodes []int64, target int64) []int64 {
	g, adjacencyMap := v.getGraph()
	//found := int64(-1)

	mappedNodes := make(map[int64]bool, len(activeNodes))
	for _, node := range activeNodes {
		mappedNodes[node] = true
	}

	resultPath, _ := BFS(g, adjacencyMap, target, func(value int64) bool {
		_, ok := mappedNodes[value]
		return ok
	})

	return resultPath
}

// BFS is an adapted version of graph.BFS that also returns the traversal path
func BFS[K comparable, T any](g graph.Graph[K, T], adjacencyMap map[K]map[K]graph.Edge[K], start K, visit func(K) bool) ([]K, error) {
	if _, ok := adjacencyMap[start]; !ok {
		return nil, fmt.Errorf("could not find start vertex with hash %v", start)
	}

	queue := make([]K, 0)
	visited := make(map[K]K)

	visited[start] = start
	queue = append(queue, start)

	found := false
	currentHash := start
	for len(queue) > 0 {
		currentHash = queue[0]

		queue = queue[1:]

		if stop := visit(currentHash); stop {
			found = true
			break
		}

		for adjacency := range adjacencyMap[currentHash] {
			if _, ok := visited[adjacency]; !ok {
				visited[adjacency] = currentHash
				queue = append(queue, adjacency)
			}
		}
	}

	if !found {
		return []K{}, nil
	}

	resultPath := make([]K, 0)
	resultPath = append(resultPath, currentHash)

	next := currentHash
	for next != start {
		next = visited[next]
		resultPath = append(resultPath, next)
	}

	return resultPath, nil
}

var TreeVersions = make(map[TreeVersion]*TreeVersionData)
