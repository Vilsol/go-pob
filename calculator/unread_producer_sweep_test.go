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
	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// TestUnreadProducerSweep hunts the defect class that every other instrument here points AWAY from: a CORRECT PRODUCER WITH NO READER.
func TestUnreadProducerSweep(t *testing.T) {
	root := ".."

	producers := map[string]map[string]bool{}
	add := func(name, surface string) {
		if name == "" {
			return
		}
		if producers[name] == nil {
			producers[name] = map[string]bool{}
		}
		producers[name][surface] = true
	}
	tagged := map[string]bool{}
	addMods := func(mods []mod.Mod, surface string) {
		for _, m := range mods {
			if m != nil {
				add(m.Name(), surface)
				if len(m.Tags()) > 0 {
					tagged[m.Name()] = true
				}
			}
		}
	}
	for _, byStat := range raw.SkillStatMaps {
		for _, sm := range byStat {
			if sm != nil {
				addMods(sm.Mods, "skill_stat_maps")
			}
		}
	}
	for _, mods := range raw.SkillBaseMods {
		addMods(mods, "skill_base_mods")
	}
	for _, mi := range raw.Minions {
		if mi != nil {
			addMods(mi.ModList, "minion_data")
		}
	}

	parserPath := filepath.Join(root, "calculator", "mod_parser.go")
	parserSrc, err := os.ReadFile(parserPath)
	if err != nil {
		t.Fatalf("REFUSING: cannot read %s: %v", parserPath, err)
	}
	ctor := regexp.MustCompile(`(?:mod\.NewFloat|mod\.NewFlag|mod\.NewList|MOD|FLAG)\("([A-Za-z][A-Za-z0-9:_' -]*)"`)
	for _, m := range ctor.FindAllSubmatch(parserSrc, -1) {
		add(string(m[1]), "mod_parser")
	}

	dynamicallyRead := map[string]string{}
	for _, mi := range raw.Minions {
		if mi != nil && mi.Limit != "" {
			dynamicallyRead[mi.Limit] = "minion_data.Limit"
		}
	}
	noteTagStats := func(mods []mod.Mod) {
		for _, m := range mods {
			if m == nil {
				continue
			}
			for _, tg := range m.Tags() {
				switch tt := tg.(type) {
				case *mod.PerStatTag:
					for _, st := range tt.StatList {
						dynamicallyRead[st] = "PerStat tag"
					}
				case *mod.StatThresholdTag:
					dynamicallyRead[tt.Stat] = "StatThreshold tag"
				}
			}
		}
	}
	for _, byStat := range raw.SkillStatMaps {
		for _, sm := range byStat {
			if sm != nil {
				noteTagStats(sm.Mods)
			}
		}
	}
	for _, mods := range raw.SkillBaseMods {
		noteTagStats(mods)
	}
	for _, mi := range raw.Minions {
		if mi != nil {
			noteTagStats(mi.ModList)
		}
	}
	dynTag := regexp.MustCompile(`mod\.(?:PerStat\([^,]+,|StatThreshold\()\s*"([A-Za-z][A-Za-z0-9:_' -]*)"`)
	for _, m := range dynTag.FindAllSubmatch(parserSrc, -1) {
		dynamicallyRead[string(m[1])] = "PerStat/StatThreshold tag"
	}

	if len(producers) == 0 {
		t.Fatalf("REFUSING: enumerated 0 producer names — the tables moved and an empty sweep " +
			"reports 'nothing unread', which is the most reassuring possible way to be broken")
	}

	var files []string
	for _, dir := range []string{"calculator", "moddb", "mod", "data", "pob", "builds", "wasm", "utils"} {
		_ = filepath.WalkDir(filepath.Join(root, dir), func(p string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil //nolint:nilerr
			}
			if !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
				return nil
			}
			if strings.Contains(p, string(filepath.Separator)+"raw"+string(filepath.Separator)) {
				return nil
			}
			files = append(files, p)
			return nil
		})
	}
	sort.Strings(files)
	if len(files) == 0 {
		t.Fatalf("REFUSING: found 0 source files to search for readers under %s", root)
	}

	type fileBody struct {
		path string
		body string
	}
	bodies := make([]fileBody, 0, len(files))
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("REFUSING: cannot read %s: %v", f, err)
		}
		bodies = append(bodies, fileBody{f, string(b)})
	}

	concatFrag := regexp.MustCompile(`(?:\+\s*"([A-Za-z][A-Za-z0-9:_ ]{1,})"|"([A-Za-z][A-Za-z0-9:_ ]{1,})"\s*\+)`)
	frags := map[string]bool{}
	fragsShort := map[string]bool{}
	for _, fb := range bodies {
		for _, m := range concatFrag.FindAllStringSubmatch(fb.body, -1) {
			for _, g := range m[1:] {
				if g == "" {
					continue
				}
				fragsShort[g] = true
				if len(g) >= 6 {
					frags[g] = true
				}
			}
		}
	}
	if len(frags) == 0 {
		t.Fatalf("REFUSING: harvested 0 concatenation fragments — without them every per-type key " +
			"reports as unread and the ranking is noise")
	}
	anchored := func(name string) string {
		best := ""
		for f := range frags {
			if f == name || len(f) >= len(name) {
				continue
			}
			if strings.HasPrefix(name, f) || strings.HasSuffix(name, f) {
				if len(f) > len(best) {
					best = f
				}
			}
		}
		return best
	}
	damageTypes, ailments := data.DamageType("").Values(), data.Ailment("").Values()
	enums := make([]string, 0, len(damageTypes)+len(ailments))
	for _, d := range damageTypes {
		enums = append(enums, string(d))
	}
	for _, a := range ailments {
		enums = append(enums, string(a))
	}
	concatReachable := func(name string) string {
		if f := anchored(name); f != "" {
			return f
		}
		for _, e := range enums {
			trimmed := name
			if strings.HasPrefix(trimmed, e) {
				trimmed = strings.TrimPrefix(trimmed, e)
			} else if strings.HasSuffix(trimmed, e) {
				trimmed = strings.TrimSuffix(trimmed, e)
			} else {
				continue
			}
			if trimmed == "" || trimmed == name {
				continue
			}
			if fragsShort[trimmed] {
				return e + `|"` + trimmed + `"`
			}
			if f := anchored(trimmed); f != "" {
				return e + "|" + f
			}
			for _, e2 := range enums {
				if strings.HasSuffix(trimmed, e2) && len(trimmed) > len(e2) {
					return e + "|" + e2
				}
			}
		}
		return ""
	}

	type row struct {
		name             string
		surfaces         []string
		readersElsewhere int
		readersInParser  int
		viaFragment      string
	}
	var unread []row
	var viaConcat []row
	var viaDynamic []string
	for name, surf := range producers {
		lit := `"` + name + `"`
		elsewhere, inParser := 0, 0
		for _, fb := range bodies {
			n := strings.Count(fb.body, lit)
			if n == 0 {
				continue
			}
			if strings.HasSuffix(fb.path, "mod_parser.go") {
				inParser += n
			} else {
				elsewhere += n
			}
		}
		if elsewhere > 0 {
			continue
		}
		s := make([]string, 0, len(surf))
		for k := range surf {
			s = append(s, k)
		}
		sort.Strings(s)
		r := row{name, s, elsewhere, inParser, concatReachable(name)}
		if r.viaFragment != "" {
			viaConcat = append(viaConcat, r)
			continue
		}
		if via, dyn := dynamicallyRead[name]; dyn {
			viaDynamic = append(viaDynamic, name+"  via "+via)
			continue
		}
		unread = append(unread, r)
	}

	sort.Slice(unread, func(a, b int) bool { return unread[a].name < unread[b].name })

	fmt.Printf("\n=== UNREAD PRODUCER SWEEP === %d producer names enumerated, %d source files searched\n",
		len(producers), len(files))
	fmt.Printf("%d names have NO reader outside their own producer.\n", len(unread))
	fmt.Printf("%d further names had no exact-literal reader but ARE reachable through a "+
		"concatenation fragment harvested from the tree — counted, not ranked, because they are the "+
		"dominant false-positive class. Examples:\n", len(viaConcat))
	sort.Slice(viaConcat, func(a, b int) bool { return viaConcat[a].name < viaConcat[b].name })
	for i, r := range viaConcat {
		if i >= 6 {
			break
		}
		fmt.Printf("    %-44s via %q\n", r.name, r.viaFragment)
	}
	sort.Strings(viaDynamic)
	fmt.Printf("%d further names are read DYNAMICALLY — through a data field or a tag's stat name, "+
		"never as a literal — so an exact-literal reader search cannot see them. They are NOT "+
		"findings:\n", len(viaDynamic))
	for i, d := range viaDynamic {
		if i >= 8 {
			fmt.Printf("    (+%d more)\n", len(viaDynamic)-8)
			break
		}
		fmt.Printf("    %s\n", d)
	}
	fmt.Println()

	pop := map[string]int{}
	popActor := map[string]int{}
	popSkill := map[string]int{}
	if os.Getenv("SWEEP_POPULATION") == "1" {
		if os.Getenv("CORPUS") != "true" {
			t.Fatalf("REFUSING: SWEEP_POPULATION=1 needs CORPUS=true; ranking by a population " +
				"nobody measured is exactly the folklore this sweep exists to replace")
		}
		requireGameData(t, "3.29")

		scopeOfSurface := map[string]string{
			"skill_stat_maps": "skill",
			"skill_base_mods": "skill",
			"minion_data":     "skill",
			"mod_parser":      "actor",
		}
		for _, r := range unread {
			for _, surf := range r.surfaces {
				if _, ok := scopeOfSurface[surf]; !ok {
					t.Fatalf("REFUSING: producer surface %q has no counted scope, so every name "+
						"from it would report 0 by construction. Map it in scopeOfSurface and "+
						"extend the walk below, or this column lies about %q.", surf, r.name)
				}
			}
		}

		_, names := corpusNamesForLoadedData(t)
		want := map[string]bool{}
		for _, r := range unread {
			want[r.name] = true
		}
		for _, bname := range names {
			env := corpusEnvForLoadedData(t, bname)
			seenActor := map[string]bool{}
			seenSkill := map[string]bool{}
			dbs := []*moddb.ModDB{env.Player.ModDB}
			actors := []*Actor{env.Player}
			if env.Minion != nil && env.Minion.Actor != nil {
				actors = append(actors, env.Minion.Actor)
				if env.Minion.ModDB != nil {
					dbs = append(dbs, env.Minion.ModDB)
				}
			}
			for _, db := range dbs {
				for n := range db.Mods {
					if want[n] {
						seenActor[n] = true
					}
				}
			}
			for _, a := range actors {
				for _, as := range a.ActiveSkillList {
					if as == nil || as.SkillModList == nil {
						continue
					}
					for n := range want {
						if skillCarries(as, n) {
							seenSkill[n] = true
						}
					}
				}
			}
			for n := range seenActor {
				popActor[n]++
			}
			for n := range seenSkill {
				popSkill[n]++
			}
			for n := range seenActor {
				pop[n]++
			}
			for n := range seenSkill {
				if !seenActor[n] {
					pop[n]++
				}
			}
		}

		if _, tracked := want["FlameTouched"]; tracked {
			if popActor["FlameTouched"] == 0 || popSkill["FlameTouched"] == 0 {
				t.Fatalf("REFUSING: control FlameTouched is GlobalEffect-tagged and must be nonzero "+
					"in BOTH scopes; got actor %d skill %d. The counting pass is broken.",
					popActor["FlameTouched"], popSkill["FlameTouched"])
			}
		}
		if _, tracked := want["Dummy"]; tracked && popActor["Dummy"] == 0 {
			t.Fatalf("REFUSING: control Dummy comes from mod_parser via items and must be nonzero " +
				"in the ACTOR scope. The actor half of the counting pass is broken.")
		}

		sort.Slice(unread, func(a, b int) bool {
			if pop[unread[a].name] != pop[unread[b].name] {
				return pop[unread[a].name] > pop[unread[b].name]
			}
			return unread[a].name < unread[b].name
		})
		nTagged := 0
		for _, r := range unread {
			if tagged[r.name] {
				nTagged++
			}
		}
		fmt.Printf("⚠️ SKILL column may UNDERCOUNT: %d of %d unread names have a TAGGED producing "+
			"mod, and skillCarries queries with a nil cfg. See its comment.\n\n", nTagged, len(unread))
		fmt.Printf("%-44s %7s %7s %7s %8s  %s\n", "NAME", "BUILDS", "ACTOR", "SKILL", "inParser", "PRODUCED BY")
	} else {
		fmt.Printf("%-44s %8s  %s\n", "NAME", "inParser", "PRODUCED BY")
	}

	for _, r := range unread {
		if os.Getenv("SWEEP_POPULATION") == "1" {
			fmt.Printf("%-44s %7d %7d %7d %8d  %s\n", r.name, pop[r.name], popActor[r.name],
				popSkill[r.name], r.readersInParser, strings.Join(r.surfaces, ","))
		} else {
			fmt.Printf("%-44s %8d  %s\n", r.name, r.readersInParser, strings.Join(r.surfaces, ","))
		}
	}
}

func skillCarries(as *ActiveSkill, name string) bool {
	l := as.SkillModList
	if l.Flag(nil, name) {
		return true
	}
	if l.Override(nil, name) != nil {
		return true
	}
	if l.Sum(mod.TypeBase, nil, name) != 0 || l.Sum(mod.TypeIncrease, nil, name) != 0 {
		return true
	}
	if l.More(nil, name) != 1 {
		return true
	}
	return len(l.List(nil, name)) > 0
}
