package calculator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/moddb"
)

// TestDiagSkillModList diffs the MAIN SKILL's mod list against the pinned PoB's, source-blind, as a multiset per mod name.
func TestDiagSkillModList(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.SkipNow()
	}
	dir := diagDumpsDir(t)
	want := strings.ToUpper(os.Getenv("TYPE"))

	if listFile := os.Getenv("LIST"); listFile != "" {
		raw, err := os.ReadFile(listFile)
		if err != nil {
			t.Fatalf("REFUSING: LIST=%s: %v", listFile, err)
		}
		var rows strings.Builder
		rows.WriteString("build\tactor\tkind\tkey\tmissing\textra\n")
		missByName, extraByName := map[string]int{}, map[string]int{}
		missBuilds, extraBuilds := map[string]map[string]bool{}, map[string]map[string]bool{}
		graded, skipped := 0, map[string]int{}
		for _, line := range strings.Split(string(raw), "\n") {
			b := strings.TrimSuffix(strings.TrimSpace(line), ".xml")
			if b == "" {
				continue
			}
			pob, ours, why := skillModListSidesKeyed(t, dir, b, want, os.Getenv("BYSOURCE") != "")
			if why != "" {
				skipped[why]++
				continue
			}
			graded++
			fmt.Fprintf(&rows, "%s.xml\t-\t-\t-\t0\t0\n", b)
			seen := map[string]bool{}
			for n := range pob {
				seen[n] = true
			}
			for n := range ours {
				seen[n] = true
			}
			for n := range seen {
				miss, extra := diffMultiset(pob[n], ours[n])
				if miss > 0 || extra > 0 {
					fmt.Fprintf(&rows, "%s.xml\tskill\tname\t%s\t%d\t%d\n", b, n, miss, extra)
				}
				if miss > 0 {
					missByName[n] += miss
					if missBuilds[n] == nil {
						missBuilds[n] = map[string]bool{}
					}
					missBuilds[n][b] = true
				}
				if extra > 0 {
					extraByName[n] += extra
					if extraBuilds[n] == nil {
						extraBuilds[n] = map[string]bool{}
					}
					extraBuilds[n][b] = true
				}
			}
		}
		if graded == 0 {
			t.Fatalf("REFUSING: graded 0 builds from LIST=%s — skipped: %v", listFile, skipped)
		}
		if out := os.Getenv("CENSUS_TSV"); out != "" {
			if err := os.WriteFile(out, []byte(rows.String()), 0o600); err != nil {
				t.Fatal(err)
			}
			t.Logf("wrote %s", out)
		}
		var ns []string
		for n := range missByName {
			ns = append(ns, n)
		}
		for n := range extraByName {
			if _, ok := missByName[n]; !ok {
				ns = append(ns, n)
			}
		}
		sort.Slice(ns, func(i, j int) bool {
			a, b := missByName[ns[i]]+extraByName[ns[i]], missByName[ns[j]]+extraByName[ns[j]]
			if a != b {
				return a > b
			}
			return ns[i] < ns[j]
		})
		key := "MOD NAME"
		if os.Getenv("BYSOURCE") != "" {
			key = "MOD NAME @ SOURCE"
		}
		fmt.Printf("\n=== SKILL MOD LIST CENSUS === %d builds graded, skipped %v (%s, keyed on %s)\n",
			graded, skipped, filterLabel(want), key)
		fmt.Printf("  %-56s %8s %10s %8s %10s\n", key, "MISSING", "MISSBUILDS", "EXTRA", "EXTRABUILDS")
		for _, n := range ns {
			fmt.Printf("  %-56s %8d %10d %8d %10d\n", n,
				missByName[n], len(missBuilds[n]), extraByName[n], len(extraBuilds[n]))
		}
		return
	}

	name := os.Getenv("BUILD")
	if name == "" {
		t.Skip("set BUILD=<name>, or LIST=<file> for the aggregate")
	}

	pob, oursByName, why := skillModListSides(t, dir, name, want)
	if why != "" {
		t.Fatalf("REFUSING: %s", why)
	}
	names := map[string]bool{}
	for n := range pob {
		names[n] = true
	}
	for n := range oursByName {
		names[n] = true
	}

	var sorted []string
	for n := range names {
		sorted = append(sorted, n)
	}
	sort.Strings(sorted)

	filter := filterLabel(want)
	fmt.Printf("\n=== SKILL MOD LIST / %s / main skill, local+base / %s — pob %d names, ours %d ===\n",
		name, filter, len(pob), len(oursByName))

	totalMiss, totalExtra := 0, 0
	for _, n := range sorted {
		miss, extra := diffMultiset(pob[n], oursByName[n])
		if miss == 0 && extra == 0 {
			continue
		}
		totalMiss += miss
		totalExtra += extra
		fmt.Printf("--- %s\n", n)
		fmt.Printf("   ONLY POB : %s\n", fmtCensusMods(pob[n]))
		fmt.Printf("   ONLY OURS: %s\n", fmtCensusMods(oursByName[n]))
	}
	if totalMiss == 0 && totalExtra == 0 {
		fmt.Printf("identical (%s)\n", filter)
	}
	fmt.Printf("\n%d name(s) differ — %d mod(s) PoB has that we do not, %d we have that PoB does not\n",
		countDifferingNames(pob, oursByName, sorted), totalMiss, totalExtra)
}

func fmtCensusMods(mods []censusMod) string {
	if len(mods) == 0 {
		return "(none)"
	}
	parts := make([]string, 0, len(mods))
	for _, m := range mods {
		parts = append(parts, fmt.Sprintf("%s %g", m.typ, m.val))
	}
	sort.Strings(parts)
	return strings.Join(parts, ", ")
}

func countDifferingNames(pob, ours map[string][]censusMod, names []string) int {
	n := 0
	for _, name := range names {
		if miss, extra := diffMultiset(pob[name], ours[name]); miss != 0 || extra != 0 {
			n++
		}
	}
	return n
}

func filterLabel(want string) string {
	if want == "" {
		return "every type"
	}
	return want + " only"
}

func skillModListSides(t *testing.T, dir, name, want string) (map[string][]censusMod, map[string][]censusMod, string) {
	pob, ours, why := skillModListSidesKeyed(t, dir, name, want, false)
	return pob, ours, why
}

func skillModListSidesKeyed(t *testing.T, dir, name, want string, bySource bool) (map[string][]censusMod, map[string][]censusMod, string) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dir, name+".json"))
	if err != nil {
		return nil, nil, fmt.Sprintf("no dump for %s in %s", name, dir)
	}
	var dump struct {
		SkillModList     map[string][]diagDumpMod `json:"skillModList"`
		BaseSkillModList map[string][]diagDumpMod `json:"baseSkillModList"`
	}
	if json.Unmarshal(raw, &dump) != nil {
		return nil, nil, "dump unparseable: " + name
	}
	if len(dump.SkillModList) == 0 && len(dump.BaseSkillModList) == 0 {
		return nil, nil, "dump carries neither skillModList nor baseSkillModList: " + name
	}
	file, err := os.ReadFile(filepath.Join(corpusDirForLoadedData(t), name+".xml"))
	if err != nil {
		return nil, nil, "corpus xml missing: " + name
	}
	build, err := builds.ParseBuild(file)
	if err != nil {
		return nil, nil, "build unparseable: " + name
	}
	env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())
	if env.Player == nil || env.Player.MainSkill == nil || env.Player.MainSkill.SkillModList == nil {
		return nil, nil, "we build no main skill: " + name
	}
	keep := func(typ string) bool { return want == "" || strings.EqualFold(typ, want) }

	pob := map[string][]censusMod{}
	for _, src := range []map[string][]diagDumpMod{dump.SkillModList, dump.BaseSkillModList} {
		for modName, mods := range src {
			for _, m := range mods {
				if keep(m.Type) {
					k := modName
					if bySource {
						k = modName + " @ " + m.Source
					}
					pob[k] = append(pob[k], censusMod{typ: m.Type, val: pobModValue(m.Type, m.Value)})
				}
			}
		}
	}
	ourLists := []*moddb.ModList{env.Player.MainSkill.SkillModList}
	if env.Player.MainSkill.BaseSkillModList != nil {
		ourLists = append(ourLists, env.Player.MainSkill.BaseSkillModList)
	}
	ours := map[string][]censusMod{}
	for _, list := range ourLists {
		for _, m := range list.Mods() {
			if typ := string(m.Type()); keep(typ) {
				k := m.Name()
				if bySource {
					k += " @ " + string(m.GetSource())
				}
				ours[k] = append(ours[k], censusMod{typ: typ, val: ourModValue(m)})
			}
		}
	}
	return pob, ours, ""
}
