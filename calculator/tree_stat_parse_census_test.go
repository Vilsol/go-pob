package calculator

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/data"
)

var (
	censusNodesAttrRe   = regexp.MustCompile(`<Spec[^>]*\snodes="([^"]*)"`)
	censusMasteryRe     = regexp.MustCompile(`masteryEffects="([^"]*)"`)
	censusMasteryPairRe = regexp.MustCompile(`\{(\d+),(\d+)\}`)
)

// TestTreeStatParseCensus ranks the passive-tree stat lines our ModParser cannot read, weighted by how many CORPUS BUILDS allocate the node carrying them.
func TestTreeStatParseCensus(t *testing.T) {
	requireOperatorRun(t, "report-only census")
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	tree := data.TreeVersions[data.LatestTreeVersion].Tree()
	if len(tree.Nodes) == 0 {
		t.Fatal("REFUSING: the loaded tree has no nodes")
	}

	alloc := map[string]int{}
	masteryPicks := map[[2]string]int{}
	corpus := corpusDirForLoadedData(t)
	files, err := filepath.Glob(filepath.Join(corpus, "*.xml"))
	if err != nil || len(files) == 0 {
		t.Fatalf("REFUSING: no corpus XMLs under %s (%v)", corpus, err)
	}
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		for _, m := range censusMasteryRe.FindAllStringSubmatch(string(raw), -1) {
			for _, pair := range censusMasteryPairRe.FindAllStringSubmatch(m[1], -1) {
				masteryPicks[[2]string{pair[1], pair[2]}]++
			}
		}
		seen := map[string]bool{}
		for _, m := range censusNodesAttrRe.FindAllStringSubmatch(string(raw), -1) {
			for _, id := range strings.Split(m[1], ",") {
				id = strings.TrimSpace(id)
				if id != "" && !seen[id] {
					seen[id] = true
					alloc[id]++
				}
			}
		}
	}
	if len(alloc) == 0 {
		t.Fatalf("REFUSING: parsed %d corpus files and found no <Spec nodes=\"...\"> — the "+
			"attribute this census reads has moved, and an empty result would read as a clean bill", len(files))
	}

	parses := func(line string) bool {
		e := ParseMod(line, false)
		return e != nil && len(e.ModList) > 0
	}

	type row struct {
		builds int
		nodes  []string
	}
	drops := map[string]*row{}
	nodesScanned, linesScanned := 0, 0

	ids := make([]string, 0, len(tree.Nodes))
	for id := range tree.Nodes {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	for _, id := range ids {
		node := tree.Nodes[id]
		type weighted struct {
			stats  []string
			builds int
		}
		groups := []weighted{{stats: node.Stats, builds: alloc[id]}}
		for _, e := range node.MasteryEffects {
			groups = append(groups, weighted{stats: e.Stats, builds: masteryPicks[[2]string{id, fmt.Sprint(e.Effect)}]})
		}
		nonEmpty := false
		for _, g := range groups {
			if len(g.stats) > 0 {
				nonEmpty = true
			}
		}
		if !nonEmpty {
			continue
		}
		nodesScanned++
		for _, g := range groups {
			if len(g.stats) == 0 {
				continue
			}
			var lines []string
			for _, s := range fixupTreeStatText(g.stats) {
				lines = append(lines, strings.Split(s, "\n")...)
			}
			for i := 0; i < len(lines); i++ {
				line := strings.TrimSpace(lines[i])
				if line == "" {
					continue
				}
				linesScanned++
				if parses(line) {
					continue
				}
				if i+1 < len(lines) && parses(line+" "+strings.TrimSpace(lines[i+1])) {
					i++
					continue
				}
				r := drops[line]
				if r == nil {
					r = &row{}
					drops[line] = r
				}
				r.builds += g.builds
				if len(r.nodes) < 3 {
					r.nodes = append(r.nodes, id)
				}
			}
		}
	}

	keys := make([]string, 0, len(drops))
	for k := range drops {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if drops[keys[i]].builds != drops[keys[j]].builds {
			return drops[keys[i]].builds > drops[keys[j]].builds
		}
		return keys[i] < keys[j]
	})

	total := 0
	for _, k := range keys {
		total += drops[k].builds
	}
	fmt.Printf("=== TREE STAT PARSE CENSUS === %d corpus builds, %d nodes with stats, %d stat lines\n",
		len(files), nodesScanned, linesScanned)
	fmt.Printf("DROPPED %d distinct lines; build-allocations summed %d (an UPPER BOUND — a build "+
		"allocating two dropped nodes is counted twice, and PoB may drop some of these too)\n", len(keys), total)
	fmt.Printf("%8s  %s\t%s\n", "BUILDS", "LINE", "NODE IDS")
	for i, k := range keys {
		if i >= 60 {
			fmt.Printf("... %d further lines not shown\n", len(keys)-60)
			break
		}
		fmt.Printf("%8d  %s\t%s\n", drops[k].builds, k, strings.Join(drops[k].nodes, ","))
	}
}
