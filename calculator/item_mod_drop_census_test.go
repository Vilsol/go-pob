package calculator

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestItemModDropCensus ranks the ITEM mod lines this parser discards whole.
func TestItemModDropCensus(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	{
		const good = "Rarity: RARE\nControl Ring\nIron Ring\n--------\n+10 to Strength\n"
		gi := ParseItem(good)
		if n := len(gi.ExplicitModLines); n != 1 {
			t.Fatalf("REFUSING: control item yielded %d explicit mod lines, want 1; the census "+
				"cannot see item mod lines at all", n)
		}
		if e := strings.TrimSpace(gi.ExplicitModLines[0].Extra); e != "" {
			t.Fatalf("REFUSING: a line that parses cleanly left Extra %q; this census would "+
				"report false drops", e)
		}

		const bad = "Rarity: RARE\nControl Ring\nIron Ring\n--------\nZz Not A Real Item Mod Zz\n"
		bi := ParseItem(bad)
		if n := len(bi.ExplicitModLines); n != 1 {
			t.Fatalf("REFUSING: unparseable control yielded %d explicit mod lines, want 1", n)
		}
		if strings.TrimSpace(bi.ExplicitModLines[0].Extra) == "" {
			t.Fatalf("REFUSING: an unparseable line left Extra empty; the census cannot see a " +
				"drop and would report that nothing is discarded")
		}
	}

	corpusDir, names := corpusNamesForLoadedData(t)

	type drop struct {
		builds   map[string]bool
		extra    string
		sample   string
		kinds    map[string]bool
		occurs   int
		variants int
	}

	var mu sync.Mutex
	drops := map[string]*drop{}
	byKind := map[string]int{}
	var parsedBuilds, parsedItems, totalLines, droppedLines, variantSkipped int

	runConcurrently(len(names), 12, func(i int, _ *EnvironmentCache) {
		name := names[i]
		file, err := os.ReadFile(filepath.Join(corpusDir, name))
		if err != nil {
			return
		}
		build, err := builds.ParseBuild(file)
		if err != nil {
			return
		}

		type hit struct{ kind, key, extra, line string }
		var hits []hit
		var nItems, nLines, nDropped, nVariant int

		for _, raw := range build.Items.Items {
			var item *Item
			func() {
				defer func() { _ = recover() }()
				item = ParseItem(raw.Text)
			}()
			if item == nil {
				continue
			}
			nItems++

			for _, group := range []struct {
				kind  string
				lines []ModLine
			}{
				{"Explicit", item.ExplicitModLines},
				{"Implicit", item.ImplicitModLines},
				{"Enchant", item.EnchantModLines},
				{"Scourge", item.ScourgeModLines},
				{"ClassRequirement", item.ClassRequirementModLines},
			} {
				for _, ml := range group.lines {
					nLines++
					if strings.TrimSpace(ml.Extra) == "" {
						continue
					}
					if !item.checkModLineVariant(ml) {
						nVariant++
						continue
					}
					nDropped++
					hits = append(hits, hit{
						kind:  group.kind,
						key:   normaliseItemModLine(ml.Line),
						extra: strings.TrimSpace(ml.Extra),
						line:  strings.ReplaceAll(ml.Line, "\n", " "),
					})
				}
			}
		}

		mu.Lock()
		parsedBuilds++
		parsedItems += nItems
		totalLines += nLines
		droppedLines += nDropped
		variantSkipped += nVariant
		for _, h := range hits {
			byKind[h.kind]++
			d, ok := drops[h.key]
			if !ok {
				d = &drop{builds: map[string]bool{}, kinds: map[string]bool{}, extra: h.extra, sample: h.line}
				drops[h.key] = d
			}
			d.occurs++
			d.builds[name] = true
			d.kinds[h.kind] = true
		}
		mu.Unlock()
	})

	if parsedBuilds < len(names)/2 {
		t.Fatalf("REFUSING: parsed %d of %d builds; a census over half a corpus understates "+
			"every row while still printing a confident total", parsedBuilds, len(names))
	}
	if parsedItems == 0 || totalLines == 0 {
		t.Fatalf("REFUSING: %d items and %d mod lines seen across %d builds; the census looked "+
			"at nothing", parsedItems, totalLines, parsedBuilds)
	}

	rows := make([]*drop, 0, len(drops))
	keys := make(map[*drop]string, len(drops))
	for k, d := range drops {
		rows = append(rows, d)
		keys[d] = k
	}
	sort.Slice(rows, func(a, b int) bool {
		if len(rows[a].builds) != len(rows[b].builds) {
			return len(rows[a].builds) > len(rows[b].builds)
		}
		return keys[rows[a]] < keys[rows[b]]
	})

	fmt.Printf("\n=== ITEM MOD DROP CENSUS ===\n")
	fmt.Printf("CORPUS %s -- %d builds, %d items, %d mod lines\n", corpusDir, parsedBuilds, parsedItems, totalLines)
	fmt.Printf("DROPPED %d lines (%.2f%% of all item mod lines) in %d distinct shapes\n",
		droppedLines, 100*float64(droppedLines)/float64(totalLines), len(drops))
	fmt.Printf("variant-skipped (NOT drops, excluded): %d\n", variantSkipped)

	kinds := make([]string, 0, len(byKind))
	for k := range byKind {
		kinds = append(kinds, k)
	}
	sort.Slice(kinds, func(a, b int) bool { return byKind[kinds[a]] > byKind[kinds[b]] })
	fmt.Printf("by kind: ")
	for _, k := range kinds {
		fmt.Printf("%s %d  ", k, byKind[k])
	}
	fmt.Printf("\n\nTOP 60 BY BUILDS AFFECTED\n")
	fmt.Printf("%7s %7s  %-11s %s\n", "BUILDS", "OCCURS", "KIND", "LINE  [leftover Extra]")
	for i, d := range rows {
		if i >= 60 {
			break
		}
		kk := make([]string, 0, len(d.kinds))
		for k := range d.kinds {
			kk = append(kk, k)
		}
		sort.Strings(kk)
		fmt.Printf("%7d %7d  %-11s %s  [%s]\n", len(d.builds), d.occurs, strings.Join(kk, "+"), d.sample, d.extra)
	}

	if out := os.Getenv("CENSUS_OUT"); out != "" {
		if err := os.MkdirAll(out, 0o755); err != nil {
			t.Fatalf("CENSUS_OUT: %v", err)
		}
		var b strings.Builder
		b.WriteString("builds\toccurrences\tkinds\tnormalised\tsample_line\tleftover_extra\n")
		for _, d := range rows {
			kk := make([]string, 0, len(d.kinds))
			for k := range d.kinds {
				kk = append(kk, k)
			}
			sort.Strings(kk)
			fmt.Fprintf(&b, "%d\t%d\t%s\t%s\t%s\t%s\n",
				len(d.builds), d.occurs, strings.Join(kk, "+"), keys[d], d.sample, d.extra)
		}
		p := filepath.Join(out, "item_mod_drops.tsv")
		if err := os.WriteFile(p, []byte(b.String()), 0o600); err != nil {
			t.Fatalf("CENSUS_OUT: %v", err)
		}
		fmt.Printf("\nwrote %s (%d rows)\n", p, len(rows))
	}
}

var reItemModNumber = regexp.MustCompile(`[+-]?\d+(\.\d+)?`)

func normaliseItemModLine(line string) string {
	s := strings.ReplaceAll(line, "\n", " ")
	s = reItemModNumber.ReplaceAllString(s, "#")
	return strings.ToLower(strings.Join(strings.Fields(s), " "))
}
