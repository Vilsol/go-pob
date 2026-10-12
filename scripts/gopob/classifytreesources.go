package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var (
	reDataID   = regexp.MustCompile(`\["id"\] = "([^"]+)"`)
	reTreeName = regexp.MustCompile(`"Tree:([A-Za-z][^"]*)"`)
)

type treePair struct{ build, id string }

func classifyTreeSources(args []string) int {
	pobData := filepath.Join(expandUser(envOr("POB_DIR", "~/PathOfBuilding")), "src", "Data")
	dumps := expandUser(envOr("HOME", "~") + "/go-pob-corpus/dumps-3.29")
	if len(args) > 0 {
		dumps = args[0]
	}

	tattooFile := filepath.Join(pobData, "TattooPassives.lua")
	legionFile := filepath.Join(pobData, "TimelessJewelData", "LegionPassives.lua")
	for _, f := range []string{tattooFile, legionFile} {
		if !readable(f) {
			fmt.Fprintf(os.Stderr, "REFUSING: cannot read %s\n", f)
			return 1
		}
	}
	if fi, err := os.Stat(dumps); err != nil || !fi.IsDir() { //nolint:gosec // operator-supplied
		fmt.Fprintf(os.Stderr, "REFUSING: no dump set at %s\n", dumps)
		return 1
	}

	tattoo, rc := dataIDs(tattooFile)
	if rc != 0 {
		return rc
	}
	legion, rc := dataIDs(legionFile)
	if rc != 0 {
		return rc
	}
	fmt.Printf("id universes:  tattoo %d   legion %d\n", len(tattoo), len(legion))

	overlap := intersectionSize(tattoo, legion)
	fmt.Printf("overlap: %d  (must be 0)\n", overlap)
	if overlap != 0 {
		fmt.Fprintln(os.Stderr, "REFUSING: id sets overlap; classification is not well defined")
		return 1
	}
	tattooSet, legionSet := toSet(tattoo), toSet(legion)

	files, err := filepath.Glob(filepath.Join(dumps, "*.json"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "REFUSING: %v\n", err)
		return 1
	}
	sort.Strings(files)

	var pairs []treePair
	for _, f := range files {
		b := strings.TrimSuffix(filepath.Base(f), ".json")
		src, readErr := os.ReadFile(f) //nolint:gosec // a path this tool globbed
		if readErr != nil {
			continue
		}
		seen := map[string]bool{}
		var ids []string
		for _, m := range reTreeName.FindAllStringSubmatch(string(src), -1) {
			if !seen[m[1]] {
				seen[m[1]] = true
				ids = append(ids, m[1])
			}
		}
		sort.Strings(ids)
		for _, id := range ids {
			pairs = append(pairs, treePair{b, id})
		}
	}

	fmt.Printf("dumps scanned: %d\n", len(files))
	if len(pairs) == 0 {
		fmt.Fprintf(os.Stderr, "REFUSING: not one named Tree: source in %d dumps — that is an unhealthy dump set,\n", len(files))
		fmt.Fprintln(os.Stderr, "          not an absence of tattoos and legion nodes.")
		return 1
	}
	fmt.Println()

	var pTattoo, pLegionDone, pLegionTodo []treePair
	for _, p := range pairs {
		switch {
		case tattooSet[p.id]:
			pTattoo = append(pTattoo, p)
		case legionSet[p.id]:
			if strings.Contains(p.id, "_keystone_") || p.id == "templar_devotion_node" ||
				p.id == "eternal_small_blank" {
				pLegionDone = append(pLegionDone, p)
			} else {
				pLegionTodo = append(pLegionTodo, p)
			}
		}
	}
	row := func(label string, ps []treePair) {
		builds := map[string]bool{}
		for _, p := range ps {
			builds[p.build] = true
		}
		fmt.Printf("  %-38s %4d builds   %5d pairs\n", label, len(builds), len(ps))
	}
	row("tattoo (ALL unported)", pTattoo)
	row("legion notable/vaal-small (unported)", pLegionTodo)
	row("legion keystone/devotion/blank (done)", pLegionDone)

	fmt.Println()
	fmt.Println("top named sources, classified:")
	counts := map[string]int{}
	for _, p := range pairs {
		counts[p.id]++
	}
	type tally struct {
		n  int
		id string
	}
	var ts []tally
	for id, n := range counts {
		ts = append(ts, tally{n, id})
	}
	sort.Slice(ts, func(i, j int) bool {
		if ts[i].n != ts[j].n {
			return ts[i].n > ts[j].n
		}
		return ts[i].id > ts[j].id
	})
	for _, t := range ts[:min(20, len(ts))] {
		kind := "UNKNOWN"
		if tattooSet[t.id] {
			kind = "TATTOO"
		}
		if legionSet[t.id] {
			kind = "LEGION"
		}
		fmt.Printf("  %4d builds  %-34s %s\n", t.n, t.id, kind)
	}
	return 0
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func readable(p string) bool {
	f, err := os.Open(p) //nolint:gosec // an operator-supplied data path
	if err != nil {
		return false
	}
	_ = f.Close()
	return true
}

func dataIDs(path string) ([]string, int) {
	b, err := os.ReadFile(path) //nolint:gosec // checked readable above
	if err != nil {
		fmt.Fprintf(os.Stderr, "REFUSING: cannot read %s\n", path)
		return nil, 1
	}
	seen := map[string]bool{}
	var ids []string
	for _, m := range reDataID.FindAllStringSubmatch(string(b), -1) {
		if !seen[m[1]] {
			seen[m[1]] = true
			ids = append(ids, m[1])
		}
	}
	sort.Strings(ids)
	return ids, 0
}

func toSet(xs []string) map[string]bool {
	m := make(map[string]bool, len(xs))
	for _, x := range xs {
		m[x] = true
	}
	return m
}
