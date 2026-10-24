package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/root"
)

var (
	reLuaPresence = regexp.MustCompile(`\bif\s+(?:not\s+)?[A-Za-z_][\w.]*\.(\w+)\s+(?:then|and\b|or\b)`)
	reGoZeroGuard = regexp.MustCompile(`\.([A-Z]\w+)\s*(?:>\s*0|!=\s*0)\b`)
)

type goSite struct {
	file string
	line int
	text string
}

func zeroValueGuardSweep(args []string) int {
	if len(args) > 0 {
		fmt.Fprintf(os.Stderr, "REFUSING: zero-value-guard-sweep takes no arguments, got %q\n", args[0])
		return 64
	}
	repo, err := root.Resolve()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return root.RefusalCode
	}
	pob := filepath.Join(expandUser(pobPinDir()), "src")

	luaPresence := map[string]map[string]bool{}
	for _, pat := range []string{
		filepath.Join(pob, "Modules", "*.lua"), filepath.Join(pob, "Classes", "*.lua"),
	} {
		files, _ := filepath.Glob(pat)
		sort.Strings(files)
		for _, f := range files {
			b, readErr := os.ReadFile(f) //nolint:gosec // a path this tool globbed
			if readErr != nil {
				continue
			}
			for i, line := range strings.Split(string(b), "\n") {
				for _, m := range reLuaPresence.FindAllStringSubmatch(line, -1) {
					if luaPresence[m[1]] == nil {
						luaPresence[m[1]] = map[string]bool{}
					}
					luaPresence[m[1]][fmt.Sprintf("%s:%d", filepath.Base(f), i+1)] = true
				}
			}
		}
	}

	goHits := map[string][]goSite{}
	var goOrder []string
	for _, pat := range []string{
		filepath.Join(repo, "calculator", "*.go"), filepath.Join(repo, "moddb", "*.go"),
	} {
		files, _ := filepath.Glob(pat)
		sort.Strings(files)
		for _, f := range files {
			if strings.HasSuffix(f, "_test.go") {
				continue
			}
			b, readErr := os.ReadFile(f) //nolint:gosec // a path this tool globbed
			if readErr != nil {
				continue
			}
			for i, line := range strings.Split(string(b), "\n") {
				lt := strings.TrimLeft(line, " \t")
				if strings.HasPrefix(lt, "//") || strings.HasPrefix(lt, "*") {
					continue
				}
				for _, m := range reGoZeroGuard.FindAllStringSubmatch(line, -1) {
					if _, seen := goHits[m[1]]; !seen {
						goOrder = append(goOrder, m[1])
					}
					goHits[m[1]] = append(goHits[m[1]], goSite{
						filepath.Base(f), i + 1, truncRunes(strings.TrimSpace(line), 110)})
				}
			}
		}
	}

	if len(luaPresence) == 0 {
		fmt.Fprintf(os.Stderr, "REFUSING: no Lua presence tests found under %s — set POB_DIR to "+
			"the pinned checkout\n", pob)
		return root.RefusalCode
	}
	if len(goHits) == 0 {
		fmt.Fprintf(os.Stderr, "REFUSING: no `> 0` / `!= 0` field guards found under %s; that is a "+
			"broken scan, not a clean tree\n", repo)
		return root.RefusalCode
	}

	type cand struct {
		field string
		sites []goSite
		lua   []string
	}
	var cands []cand
	for _, field := range goOrder {
		luaKey := strings.ToLower(field[:1]) + field[1:]
		set, ok := luaPresence[luaKey]
		if !ok {
			continue
		}
		var lua []string
		for k := range set {
			lua = append(lua, k)
		}
		sort.Strings(lua)
		cands = append(cands, cand{field, goHits[field], lua[:min(3, len(lua))]})
	}
	sort.SliceStable(cands, func(i, j int) bool { return len(cands[i].sites) > len(cands[j].sites) })

	fmt.Printf("%d distinct Go fields guarded with >0/!=0; %d also appear as a bare Lua presence test\n\n",
		len(goHits), len(cands))
	for _, c := range cands {
		fmt.Printf("=== %s   (%d Go site(s))   lua: %s\n", c.field, len(c.sites), strings.Join(c.lua, ", "))
		for _, s := range c.sites[:min(4, len(c.sites))] {
			fmt.Printf("      %s:%d  %s\n", s.file, s.line, s.text)
		}
	}
	return 0
}
