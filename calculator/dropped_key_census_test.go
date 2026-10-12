package calculator

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/utils"
)

// TestDroppedSkillDataKeyCensus: The silently-dropped skill-data key census.
func TestDroppedSkillDataKeyCensus(t *testing.T) {
	t.Parallel()

	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	dir := corpusDirForLoadedData(t)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("REFUSING: cannot read the corpus directory %s: %v", dir, err)
	}
	builds := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".xml") {
			builds = append(builds, e.Name())
		}
	}
	if len(builds) == 0 {
		t.Fatalf("REFUSING: no .xml builds under %s — an empty census is not a finding", dir)
	}

	if n := os.Getenv("CENSUS_LIMIT"); n != "" {
		var lim int
		if _, err := fmt.Sscanf(n, "%d", &lim); err == nil && lim > 0 && lim < len(builds) {
			builds = builds[:lim]
		}
	}

	type row struct{ builds, occurrences int }
	perKey := map[string]*row{}
	for _, build := range builds {
		before := droppedSetCounts()
		corpusEnvForLoadedData(t, build)
		for key, after := range droppedSetCounts() {
			delta := after - before[key]
			if delta <= 0 {
				continue
			}
			r := perKey[key]
			if r == nil {
				r = &row{}
				perKey[key] = r
			}
			r.builds++
			r.occurrences += delta
		}
	}

	if len(perKey) == 0 {
		t.Fatal("REFUSING: not one dropped key over the whole corpus. utils.recordDropped is the " +
			"only instrument that sees this class, so a zero here means the probe is broken, not " +
			"that the class is empty.")
	}

	mentions := packageMentions(t)
	keys := make([]string, 0, len(perKey))
	for k := range perKey {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if perKey[keys[i]].builds != perKey[keys[j]].builds {
			return perKey[keys[i]].builds > perKey[keys[j]].builds
		}
		return keys[i] < keys[j]
	})

	var starved int
	fmt.Printf("\n=== SILENTLY-DROPPED SKILL-DATA KEYS  (%d distinct, over %d corpus builds) ===\n",
		len(keys), len(builds))
	fmt.Printf("%-26s %-40s %7s %7s  %s\n", "TARGET TYPE", "KEY", "BUILDS", "DROPS", "CONSUMER")
	for _, k := range keys {
		typ, key, _ := strings.Cut(k, "\x00")
		has := mentions(strings.ToUpper(key[:1]) + key[1:])
		if !has {
			starved++
		}
		fmt.Printf("%-26s %-40s %7d %7d  %s\n", typ, key, perKey[k].builds, perKey[k].occurrences,
			map[bool]string{true: "yes", false: "no"}[has])
	}
	fmt.Printf("\n%d of %d have NO consumer either — both ends absent, so a field alone measures "+
		"zero.\n", starved, len(keys))
	fmt.Println("BUILDS is the effect rung FOR THE DROP: it happened on that many real evaluations.")
	fmt.Println("⚠️  It is an UPPER BOUND on the fix — ask what the consumer would compute if the")
	fmt.Println("    value arrived. Herald of Ash reads 1439 here and is worth 3.")
	fmt.Println("⚠️  A `consumer=yes` row is NOT work — the concept is wired by another route.")
	fmt.Println("⚠️  The weaponData *DPS rows are DECLARED drops, not lost inputs: item_setup.go")
	fmt.Println("    :287-297 records that Item.lua writes them for the item tooltip and no calc")
	fmt.Println("    module reads one. They head the table by count and are not work.")
}

func droppedSetCounts() map[string]int {
	_, counts := utils.DroppedKeys()
	out := make(map[string]int, len(counts))
	for _, c := range counts {
		if c.Op == "set" {
			out[c.Type+"\x00"+c.Key] += c.Count
		}
	}
	return out
}

func packageMentions(t *testing.T) func(string) bool {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("REFUSING: cannot read the package directory: %v", err)
	}
	var blob strings.Builder
	for _, e := range entries {
		n := e.Name()
		if e.IsDir() || !strings.HasSuffix(n, ".go") || strings.HasSuffix(n, "_test.go") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(".", n))
		if err != nil {
			t.Fatalf("REFUSING: cannot read %s: %v", n, err)
		}
		blob.Write(b)
	}
	if blob.Len() == 0 {
		t.Fatal("REFUSING: read no package source, so every key would report consumer=no and the " +
			"whole census would present as work to do")
	}
	src := blob.String()
	return func(name string) bool { return strings.Contains(src, name) }
}
