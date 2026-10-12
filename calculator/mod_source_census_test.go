package calculator

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestMissingItemSourceCensus answers ONE question over a build list: on how many of them does PoB build an `Item:` mod source that we build no mod for at all?
func TestMissingItemSourceCensus(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	listFile := os.Getenv("BUILDS")
	if listFile == "" {
		t.Skip("set BUILDS=<file of build names>")
	}
	requireGameData(t, "3.29")
	dir := diagDumpsDir(t)
	var wantNames []string
	for _, n := range strings.Split(os.Getenv("NAMES"), ",") {
		if n = strings.TrimSpace(n); n != "" {
			wantNames = append(wantNames, n)
		}
	}
	relevant := func(modName string) bool {
		if len(wantNames) == 0 {
			return true
		}
		for _, w := range wantNames {
			if strings.Contains(modName, w) {
				return true
			}
		}
		return false
	}

	raw, err := os.ReadFile(listFile)
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	var names []string
	for _, ln := range strings.Split(string(raw), "\n") {
		if ln = strings.TrimSpace(ln); ln != "" {
			names = append(names, strings.TrimSuffix(ln, ".xml"))
		}
	}
	if len(names) == 0 {
		t.Fatalf("REFUSING: %s lists no builds", listFile)
	}

	corpus := corpusDirForLoadedData(t)
	withMissingItem, checked, unhealthy, unusable := 0, 0, 0, 0
	perItem := map[string]int{}
	for _, name := range names {
		dump, healthy := diagLoadDump(dir, name)
		if dump == nil {
			unusable++
			continue
		}
		if !healthy {
			unhealthy++
			continue
		}
		xml, err := os.ReadFile(filepath.Join(corpus, name+".xml"))
		if err != nil {
			unusable++
			continue
		}
		build, err := builds.ParseBuild(xml)
		if err != nil {
			unusable++
			continue
		}
		env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, corpusCacheForTest(t))
		pobMods, ourDB, _ := diagActorModDB(dump, env, "player")
		if ourDB == nil {
			unusable++
			continue
		}
		ours := map[string]bool{}
		for _, mods := range ourDB.Mods {
			for _, m := range mods {
				ours[normalizeSource(string(m.GetSource()))] = true
			}
		}
		found := false
		for modName, mods := range pobMods {
			if !relevant(modName) {
				continue
			}
			for _, m := range mods {
				key := normalizeSource(m.Source)
				if !strings.HasPrefix(key, "Item:") || ours[key] {
					continue
				}
				found = true
				perItem[strings.TrimPrefix(key, "Item:")]++
			}
		}
		checked++
		if found {
			withMissingItem++
		}
	}

	if checked == 0 {
		t.Fatalf("REFUSING: 0 of %d builds were comparable (%d unhealthy dumps, %d unusable) — "+
			"empty must mean 'I looked and found nothing', never 'I did not look'",
			len(names), unhealthy, unusable)
	}
	fmt.Printf("\n=== MISSING ITEM SOURCES over %s (NAMES=%q) ===\n", filepath.Base(listFile), os.Getenv("NAMES"))
	fmt.Printf("listed %d · compared %d · unhealthy dump %d · unusable %d\n",
		len(names), checked, unhealthy, unusable)
	fmt.Printf("builds where PoB builds an Item: source we build NO mod for: %d of %d (%.1f%%)\n",
		withMissingItem, checked, float64(withMissingItem)/float64(checked)*100)
	type kv struct {
		name string
		n    int
	}
	items := make([]kv, 0, len(perItem))
	for k, v := range perItem {
		items = append(items, kv{k, v})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].n != items[j].n {
			return items[i].n > items[j].n
		}
		return items[i].name < items[j].name
	})
	fmt.Printf("%d distinct items; top 25:\n", len(items))
	for i, it := range items {
		if i >= 25 {
			break
		}
		fmt.Printf("  %4d  %s\n", it.n, it.name)
	}
}

// TestDivergingModSourceCensus ranks, over a build list, WHICH mod SOURCE most often carries a differing value for one mod NAME.
func TestDivergingModSourceCensus(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	listFile, modName := os.Getenv("BUILDS"), os.Getenv("NAME")
	if listFile == "" || modName == "" {
		t.Skip("set BUILDS=<file> and NAME=<mod name>")
	}
	requireGameData(t, "3.29")
	dir := diagDumpsDir(t)
	actorName := os.Getenv("ACTOR")
	if actorName == "" {
		actorName = "player"
	}

	raw, err := os.ReadFile(listFile)
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	var names []string
	for _, ln := range strings.Split(string(raw), "\n") {
		if ln = strings.TrimSpace(ln); ln != "" {
			names = append(names, strings.TrimSuffix(ln, ".xml"))
		}
	}
	corpus := corpusDirForLoadedData(t)
	perSource := map[string]int{}
	checked, skipped := 0, 0
	for _, name := range names {
		dump, healthy := diagLoadDump(dir, name)
		if dump == nil || !healthy {
			skipped++
			continue
		}
		xml, err := os.ReadFile(filepath.Join(corpus, name+".xml"))
		if err != nil {
			skipped++
			continue
		}
		build, err := builds.ParseBuild(xml)
		if err != nil {
			skipped++
			continue
		}
		env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, corpusCacheForTest(t))
		pobMods, ourDB, _ := diagActorModDB(dump, env, actorName)
		if ourDB == nil {
			skipped++
			continue
		}
		pobBy, ourBy := map[string]map[string]int{}, map[string]map[string]int{}
		for _, m := range pobMods[modName] {
			src := normalizeSource(m.Source)
			if pobBy[src] == nil {
				pobBy[src] = map[string]int{}
			}
			pobBy[src][m.Type+" "+string(m.Value)]++
		}
		for _, m := range ourDB.Mods[modName] {
			src := normalizeSource(string(m.GetSource()))
			if ourBy[src] == nil {
				ourBy[src] = map[string]int{}
			}
			ourBy[src][string(m.Type())+" "+fmt.Sprintf("%v", m.Value().Float())]++
		}
		seen := map[string]bool{}
		for src := range pobBy {
			seen[src] = true
		}
		for src := range ourBy {
			seen[src] = true
		}
		for src := range seen {
			if !sameValueSet(pobBy[src], ourBy[src]) {
				perSource[src]++
			}
		}
		checked++
	}
	if checked == 0 {
		t.Fatalf("REFUSING: 0 of %d builds comparable (%d skipped)", len(names), skipped)
	}
	type kv struct {
		src string
		n   int
	}
	out := make([]kv, 0, len(perSource))
	for k, v := range perSource {
		out = append(out, kv{k, v})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].n != out[j].n {
			return out[i].n > out[j].n
		}
		return out[i].src < out[j].src
	})
	fmt.Printf("\n=== DIVERGING SOURCES for %q on actor=%s over %s ===\n", modName, actorName, filepath.Base(listFile))
	fmt.Printf("compared %d builds (%d skipped); %d distinct diverging sources\n", checked, skipped, len(out))
	for i, o := range out {
		if i >= 20 {
			break
		}
		fmt.Printf("  %4d builds (%5.1f%%)  %s\n", o.n, float64(o.n)/float64(checked)*100, o.src)
	}
}

func sameValueSet(a, b map[string]int) bool {
	norm := func(m map[string]int) []string {
		var out []string
		for k, n := range m {
			parts := strings.Fields(k)
			key := k
			if len(parts) == 2 {
				var f float64
				if _, err := fmt.Sscanf(parts[1], "%g", &f); err == nil {
					key = fmt.Sprintf("%s %g", parts[0], f)
				}
			}
			for i := 0; i < n; i++ {
				out = append(out, key)
			}
		}
		sort.Strings(out)
		return out
	}
	x, y := norm(a), norm(b)
	if len(x) != len(y) {
		return false
	}
	for i := range x {
		if x[i] != y[i] {
			return false
		}
	}
	return true
}
