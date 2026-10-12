package calculator

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/data"
)

// TestTattooOverrideCensus grades the mods a build's <Override> nodes contribute.
func TestTattooOverrideCensus(t *testing.T) {
	requireOperatorRun(t, "report-only census")
	requireGameData(t, "3.29")
	dumps := diagDumpsDir(t)
	corpus := corpusDirForLoadedData(t)

	var names []string
	if listFile := os.Getenv("BUILDS"); listFile != "" {
		raw, err := os.ReadFile(listFile)
		if err != nil {
			t.Fatalf("REFUSING: BUILDS=%s: %v", listFile, err)
		}
		for _, line := range strings.Split(string(raw), "\n") {
			if line = strings.TrimSpace(line); line != "" {
				names = append(names, strings.TrimSuffix(line, ".xml"))
			}
		}
	} else {
		files, _ := filepath.Glob(filepath.Join(corpus, "*.xml"))
		for _, f := range files {
			names = append(names, strings.TrimSuffix(filepath.Base(f), ".xml"))
		}
		sort.Strings(names)
	}
	if len(names) == 0 {
		t.Fatalf("REFUSING: no builds to grade (BUILDS=%q, corpus %s)", os.Getenv("BUILDS"), corpus)
	}

	skipped := map[string]int{}
	type tally struct{ builds, pobMods, missing, extra int }
	byDN := map[string]*tally{}
	missByName, extraByName := map[string]int{}, map[string]int{}
	missBuilds, extraBuilds := map[string]map[string]bool{}, map[string]map[string]bool{}
	graded, clean := 0, 0

	for _, name := range names {
		xml, err := os.ReadFile(filepath.Join(corpus, name+".xml"))
		if err != nil {
			skipped["corpus xml unreadable"]++
			continue
		}
		build, err := builds.ParseBuild(xml)
		if err != nil {
			skipped["build unparseable"]++
			continue
		}
		specs := build.Tree.Specs
		if len(specs) == 0 {
			skipped["no tree spec"]++
			continue
		}
		idx := build.Tree.ActiveSpec - 1
		if idx < 0 || idx >= len(specs) {
			idx = 0
		}
		overrides := specs[idx].Overrides
		if len(overrides) == 0 {
			skipped["no <Override> nodes"]++
			continue
		}
		dump, healthy := diagLoadDump(dumps, name)
		if dump == nil {
			skipped["no dump"]++
			continue
		}
		if !healthy {
			skipped["dump not healthy"]++
			continue
		}

		labels := map[string]bool{}
		dns := map[string]bool{}
		for _, o := range overrides {
			row, found := data.TattooNodes[o.DN]
			if !found {
				skipped["override dn not in TattooNodes"]++
				continue
			}
			dns[o.DN] = true
			labels["Tree:"+strconv.FormatInt(int64(o.NodeID), 10)] = true
			labels["Tree:"+row.ID] = true
		}
		if len(dns) == 0 {
			continue
		}

		env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())

		if len(socketedLegionJewels(env)) > 0 {
			skipped["timeless jewel: override labels are not comparable"]++
			continue
		}
		pobMods, ourDB, _ := diagActorModDB(dump, env, "player")
		if ourDB == nil {
			skipped["no player actor"]++
			continue
		}

		pobByName := map[string][]censusMod{}
		for modName, mods := range pobMods {
			for _, m := range mods {
				if labels[normalizeSource(m.Source)] {
					pobByName[modName] = append(pobByName[modName], censusMod{typ: m.Type, val: pobModValue(m.Type, m.Value)})
				}
			}
		}
		oursByName := map[string][]censusMod{}
		for modName, mods := range ourDB.Mods {
			for _, m := range mods {
				if labels[normalizeSource(string(m.GetSource()))] {
					oursByName[modName] = append(oursByName[modName], censusMod{typ: string(m.Type()), val: ourModValue(m)})
				}
			}
		}

		allNames := map[string]bool{}
		for n := range pobByName {
			allNames[n] = true
		}
		for n := range oursByName {
			allNames[n] = true
		}
		miss, extra, pobTotal := 0, 0, 0
		for n := range allNames {
			m, e := diffMultiset(pobByName[n], oursByName[n])
			miss += m
			extra += e
			pobTotal += len(pobByName[n])
			if m > 0 {
				missByName[n] += m
				if missBuilds[n] == nil {
					missBuilds[n] = map[string]bool{}
				}
				missBuilds[n][name] = true
			}
			if e > 0 {
				extraByName[n] += e
				if extraBuilds[n] == nil {
					extraBuilds[n] = map[string]bool{}
				}
				extraBuilds[n][name] = true
			}
		}
		graded++
		if miss == 0 && extra == 0 {
			clean++
		}
		for dn := range dns {
			if byDN[dn] == nil {
				byDN[dn] = &tally{}
			}
			b := byDN[dn]
			b.builds++
			b.pobMods += pobTotal
			b.missing += miss
			b.extra += extra
		}
	}

	if graded == 0 {
		t.Fatalf("REFUSING: graded 0 builds of %d — nothing was compared. skipped: %v", len(names), skipped)
	}

	fmt.Printf("\n=== TATTOO / RUNEGRAFT OVERRIDE CENSUS === %d builds graded, %d with a PERFECT override set (%.1f%%)\n",
		graded, clean, float64(clean)/float64(graded)*100)
	fmt.Printf("skipped, by cause: %v\n", skipped)

	var mnames []string
	for n := range missByName {
		mnames = append(mnames, n)
	}
	for n := range extraByName {
		if _, ok := missByName[n]; !ok {
			mnames = append(mnames, n)
		}
	}
	sort.Slice(mnames, func(i, j int) bool {
		a, b := missByName[mnames[i]]+extraByName[mnames[i]], missByName[mnames[j]]+extraByName[mnames[j]]
		if a != b {
			return a > b
		}
		return mnames[i] < mnames[j]
	})
	if want := os.Getenv("NAME"); want != "" {
		fmt.Printf("\nBUILDS BEHIND NAME=%q (missing %d, extra %d):\n", want, missByName[want], extraByName[want])
		bs := make([]string, 0, len(missBuilds[want])+len(extraBuilds[want]))
		for b := range missBuilds[want] {
			bs = append(bs, b+"  [missing]")
		}
		for b := range extraBuilds[want] {
			bs = append(bs, b+"  [extra]")
		}
		sort.Strings(bs)
		if len(bs) == 0 {
			fmt.Printf("  none — this name disagrees on no graded build\n")
		}
		for _, b := range bs {
			fmt.Printf("  %s\n", b)
		}
	}

	if len(mnames) > 0 {
		fmt.Printf("\nBY MOD NAME — this axis IS attributed, unlike the dn ranking below:\n")
		fmt.Printf("  %-46s %8s %10s %8s %10s\n", "MOD NAME", "MISSING", "MISSBUILDS", "EXTRA", "EXTRABUILDS")
		for _, n := range mnames {
			fmt.Printf("  %-46s %8d %10d %8d %10d\n", truncateDN(n, 46),
				missByName[n], len(missBuilds[n]), extraByName[n], len(extraBuilds[n]))
		}
	}

	type row struct {
		dn string
		t  *tally
	}
	var rows []row
	for dn, b := range byDN {
		if b.missing == 0 && b.extra == 0 {
			continue
		}
		rows = append(rows, row{dn, b})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].t.missing+rows[i].t.extra != rows[j].t.missing+rows[j].t.extra {
			return rows[i].t.missing+rows[i].t.extra > rows[j].t.missing+rows[j].t.extra
		}
		return rows[i].dn < rows[j].dn
	})
	if len(rows) == 0 {
		fmt.Printf("\nNo override contributes a wrong mod on any graded build.\n")
		return
	}
	fmt.Printf("\n⚠️  PRESENT-ON-DISAGREEING-BUILDS, not a cause (see the note in the source):\n")
	fmt.Printf("  %-46s %7s %8s %8s %8s\n", "OVERRIDE (dn)", "BUILDS", "POBMODS", "MISSING", "EXTRA")
	for _, r := range rows {
		if len(rows) > 40 && r.t.missing+r.t.extra < 2 {
			continue
		}
		fmt.Printf("  %-46s %7d %8d %8d %8d\n", truncateDN(r.dn, 46), r.t.builds, r.t.pobMods, r.t.missing, r.t.extra)
	}
}

func truncateDN(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
