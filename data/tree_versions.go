package data

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/andybalholm/brotli"
	"github.com/dominikbraun/graph"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/pob"
	"github.com/Vilsol/go-pob/storage"
)

const (
	TreeVersion3_10 = pob.TreeVersion("3_10")
	TreeVersion3_11 = pob.TreeVersion("3_11")
	TreeVersion3_12 = pob.TreeVersion("3_12")
	TreeVersion3_13 = pob.TreeVersion("3_13")
	TreeVersion3_14 = pob.TreeVersion("3_14")
	TreeVersion3_15 = pob.TreeVersion("3_15")
	TreeVersion3_16 = pob.TreeVersion("3_16")
	TreeVersion3_17 = pob.TreeVersion("3_17")
	TreeVersion3_18 = pob.TreeVersion("3_18")
	TreeVersion3_20 = pob.TreeVersion("3_20")
	TreeVersion3_29 = pob.TreeVersion("3_29")
)

var LatestTreeVersion = treeForGameData(raw.LatestVersion)

func init() {
	builds.NewBuildTreeVersion = LatestTreeVersion
}

func treeForGameData(version string) pob.TreeVersion {
	switch version {
	case "3.20":
		return TreeVersion3_20
	case "3.29":
		return TreeVersion3_29
	default:
		return TreeVersion3_18
	}
}

const DefaultTreeVersion = TreeVersion3_10

type TreeVersionData struct {
	Display       string
	Num           float64
	URL           string
	cachedTree    *Tree
	rawTree       []byte
	graph         graph.Graph[int64, int64]
	adjacencyMap  map[int64]map[int64]graph.Edge[int64]
	notableMap    map[string]string
	ascendancyMap map[string]string

	treeMu       sync.Mutex
	rawMu        sync.Mutex
	notableMu    sync.Mutex
	ascendancyMu sync.Mutex
	graphMu      sync.Mutex
}

func (v *TreeVersionData) Tree() *Tree {
	v.treeMu.Lock()
	defer v.treeMu.Unlock()

	if v.cachedTree != nil {
		return v.cachedTree
	}

	var outTree Tree
	if err := json.Unmarshal(v.RawTree(), &outTree); err != nil {
		panic(fmt.Errorf("failed to decode file: %w", err))
	}
	v.cachedTree = &outTree

	return v.cachedTree
}

func (v *TreeVersionData) RawTree() []byte {
	v.rawMu.Lock()
	defer v.rawMu.Unlock()

	if v.rawTree != nil {
		return v.rawTree
	}

	treeURL := raw.TreeURL(v.Display)

	treeCacheKey := raw.CacheKey(treeURL)

	var compressedTree []byte
	if storage.Get().ExistsInCache(treeCacheKey) {
		var err error
		compressedTree, err = storage.Get().GetCache(treeCacheKey)
		if err != nil {
			panic(fmt.Errorf("failed reading compressed tree: %w", err))
		}
	} else {
		slog.Debug("fetching", slog.String("url", treeURL))
		req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, treeURL, nil)
		response, err := http.DefaultClient.Do(req)
		if err != nil {
			panic(fmt.Errorf("failed to fetch url: %s: %w", treeURL, err))
		}
		defer response.Body.Close()

		compressedTree, err = io.ReadAll(response.Body)
		if err != nil {
			panic(fmt.Errorf("failed to read response body: %w", err))
		}

		defer func() {
			_ = storage.Get().SetCache(treeCacheKey, compressedTree)
		}()
	}

	unzipStream := brotli.NewReader(bytes.NewReader(compressedTree))

	var err error
	v.rawTree, err = io.ReadAll(unzipStream)
	if err != nil {
		panic(fmt.Errorf("failed to read unzipped data: %w", err))
	}

	return v.rawTree
}

func IsTreeNotable(node Node) bool {
	switch {
	case node.ClassStartIndex != nil:
		return false
	case node.IsAscendancyStart != nil && *node.IsAscendancyStart:
		return false
	case node.IsMastery != nil && *node.IsMastery:
		return false
	case node.IsJewelSocket != nil && *node.IsJewelSocket:
		return false
	case node.IsKeystone != nil && *node.IsKeystone:
		return false
	}
	return node.IsNotable != nil && *node.IsNotable
}

func (v *TreeVersionData) NotableMap() map[string]string {
	v.notableMu.Lock()
	defer v.notableMu.Unlock()

	if v.notableMap != nil {
		return v.notableMap
	}

	nodes := v.Tree().Nodes

	ids := make([]string, 0, len(nodes))
	for id := range nodes {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	notableMap := make(map[string]string)
	for _, id := range ids {
		node := nodes[id]
		if node.Name == nil || node.AscendancyName != nil || !IsTreeNotable(node) {
			continue
		}

		name := strings.ToLower(*node.Name)
		if _, exists := notableMap[name]; !exists || node.Group != nil {
			notableMap[name] = id
		}
	}

	v.notableMap = notableMap

	return v.notableMap
}

func ascendancyMapNormalArm(node Node) bool {
	if node.AscendancyName == nil || node.Name == nil {
		return false
	}
	switch {
	case node.ClassStartIndex != nil:
		return false
	case node.IsAscendancyStart != nil && *node.IsAscendancyStart:
		return false
	case node.IsMastery != nil && *node.IsMastery:
		return false
	case node.IsJewelSocket != nil && *node.IsJewelSocket:
		return false
	case node.IsKeystone != nil && *node.IsKeystone:
		return false
	case node.IsNotable != nil && *node.IsNotable:
		return false
	}
	if *node.AscendancyName == "Reliquarian" || *node.AscendancyName == "Luminary" {
		return false
	}
	if node.IsMultipleChoiceOption != nil && *node.IsMultipleChoiceOption {
		return true
	}
	if *node.AscendancyName != "Ascendant" {
		return false
	}
	for _, skip := range []string{"Dexterity", "Intelligence", "Strength", "Passive"} {
		if strings.Contains(*node.Name, skip) {
			return false
		}
	}
	return true
}

func (v *TreeVersionData) AscendancyMap() map[string]string {
	v.ascendancyMu.Lock()
	defer v.ascendancyMu.Unlock()

	if v.ascendancyMap != nil {
		return v.ascendancyMap
	}

	nodes := v.Tree().Nodes

	ids := make([]string, 0, len(nodes))
	for id := range nodes {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	ascendancyMap := make(map[string]string)
	for _, id := range ids {
		node := nodes[id]
		if node.Name == nil {
			continue
		}

		ascendancy := node.AscendancyName != nil && (IsTreeNotable(node) || node.Group == nil)
		if !ascendancy && !ascendancyMapNormalArm(node) {
			continue
		}

		name := strings.ToLower(*node.Name)
		if _, exists := ascendancyMap[name]; !exists {
			ascendancyMap[name] = id
		}
	}

	v.ascendancyMap = ascendancyMap

	return v.ascendancyMap
}

func (v *TreeVersionData) getGraph() (graph.Graph[int64, int64], map[int64]map[int64]graph.Edge[int64]) {
	v.graphMu.Lock()
	defer v.graphMu.Unlock()

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

			_ = g.AddEdge(*node.Skill, targetID)

			if targetNode.IsMastery == nil || !*targetNode.IsMastery {
				_ = g.AddEdge(targetID, *node.Skill)
			}
		}
	}

	v.graph = g

	v.adjacencyMap, _ = v.graph.AdjacencyMap()

	return v.graph, v.adjacencyMap
}

var TreeVersions = make(map[pob.TreeVersion]*TreeVersionData)
