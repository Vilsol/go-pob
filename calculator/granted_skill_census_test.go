package calculator

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/pob"
	"github.com/Vilsol/go-pob/utils"
)

// TestGrantedSkillCensus sizes the unported granted-skill -> socket-group block.
func TestGrantedSkillCensus(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	corpusDir, files := corpusNamesForLoadedData(t)

	type agg struct {
		builds  int
		groups  int
		wrong   int
		example string
	}
	var mu sync.Mutex
	cls := map[string]*agg{}
	stats := map[string]*agg{}
	built, totalWrong := 0, 0

	runConcurrently(len(files), 12, func(i int, cache *EnvironmentCache) {
		rec, ok := oracle[files[i]]
		if !ok {
			return
		}
		name := strings.TrimSuffix(files[i], ".xml")
		file, err := os.ReadFile(filepath.Join(corpusDir, files[i]))
		if err != nil {
			return
		}
		build, err := builds.ParseBuild(file)
		if err != nil {
			return
		}
		var env *Environment
		func() {
			defer func() { _ = recover() }()
			env = BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, cache)
		}()
		if env == nil {
			return
		}

		wrong := 0
		for _, pair := range oracleTablePairs(rec, env) {
			for k, want := range pair.exp {
				got, ok := pair.got[k]
				if !ok {
					continue
				}
				if math.IsNaN(got) || math.IsInf(got, 0) || math.Abs(got-want) > 0.005*math.Abs(want) {
					wrong++
				}
			}
		}

		type gs struct {
			skillID string
			level   int
		}
		treeGranted := map[string][]gs{}
		unresolved := map[string]bool{}
		for nodeID, node := range env.AllocatedNodes {
			ml := buildModListForNode(env, nodeID, node)
			if ml == nil {
				continue
			}
			for _, v := range utils.CastSlice[mod.ExtraSkill](ml.List(nil, "ExtraSkill")) {
				if v.SkillID == "" {
					unresolved["Tree:"+nodeID] = true
					continue
				}
				treeGranted["Tree:"+nodeID] = append(treeGranted["Tree:"+nodeID], gs{v.SkillID, v.Level})
			}
		}

		local := map[string]int{}
		localStats := map[string]int{}
		seenTreeSrc := map[string]bool{}
		var groups []pob.Skill
		if si := activeSkillSetIndex(build.Skills); si >= 0 {
			groups = build.Skills.SkillSets[si].Skills
		}
		for gi := range groups {
			sg := groups[gi]
			if sg.Source == "" {
				continue
			}
			switch {
			case strings.HasPrefix(sg.Source, "Tree:"):
				seenTreeSrc[sg.Source] = true
				nodeID := strings.TrimPrefix(sg.Source, "Tree:")
				want, ok := treeGranted[sg.Source]
				if !ok {
					node, allocated := env.AllocatedNodes[nodeID]
					switch {
					case unresolved[sg.Source]:
						local["NO grantor: node allocated, ExtraSkill has EMPTY SkillID"]++
					case allocated:
						local["NO grantor: node ALLOCATED, no ExtraSkill mod at all"]++
						for _, st := range node.Stats {
							localStats[st]++
						}
					default:
						local["NO grantor: node NOT allocated (PoB DELETES the group)"]++
					}
					continue
				}
				if len(sg.Gems) == 0 {
					local["tree group with no gem"]++
					continue
				}
				matched, levelDiff := false, false
				for _, w := range want {
					if w.skillID == sg.Gems[0].SkillID {
						matched = true
						levelDiff = w.level != sg.Gems[0].Level
						break
					}
				}
				switch {
				case !matched:
					local["tree group whose grantor grants a DIFFERENT skill"]++
				case levelDiff:
					local["tree group with a STALE LEVEL (PoB overwrites it)"]++
				default:
					local["tree group AGREES"]++
				}
				if len(sg.Gems) > 1 {
					local["tree group carries EXTRA gems (PoB wipes the list)"]++
				}
			case strings.HasPrefix(sg.Source, "Item:"):
				local["item-sourced group (NOT ADJUDICATED — see comment)"]++
			case sg.Source == "Explode":
				local["Explode group (CalcSetup.lua:1416)"]++
			default:
				local["other-sourced group"]++
			}
		}
		for src := range treeGranted {
			if !seenTreeSrc[src] {
				local["grantor with NO group (PoB CREATES one)"]++
			}
		}

		mu.Lock()
		built++
		totalWrong += wrong
		bump := func(m map[string]*agg, k string, n int) {
			a := m[k]
			if a == nil {
				a = &agg{example: name}
				m[k] = a
			}
			a.builds++
			a.groups += n
			a.wrong += wrong
		}
		for k, n := range local {
			bump(cls, k, n)
		}
		for k, n := range localStats {
			bump(stats, k, n)
		}
		mu.Unlock()
	})

	if built < len(files)/2 {
		t.Fatalf("REFUSING: built %d of %d builds; a census over half a corpus is not a census",
			built, len(files))
	}

	report := func(title string, m map[string]*agg, width, limit int) {
		rows := make([]string, 0, len(m))
		for k := range m {
			rows = append(rows, k)
		}
		sort.Slice(rows, func(i, j int) bool { return m[rows[i]].wrong > m[rows[j]].wrong })
		fmt.Printf("\n=== %s ===\n", title)
		fmt.Printf("%-*s %7s %8s %9s %7s  %s\n", width, "CLASS", "BUILDS", "GROUPS", "WRONG", "W/BLD", "EXAMPLE")
		for i, k := range rows {
			if i >= limit {
				fmt.Printf("… +%d more\n", len(rows)-limit)
				break
			}
			a := m[k]
			fmt.Printf("%-*s %7d %8d %9d %7.1f  %s\n", width, k, a.builds, a.groups, a.wrong,
				float64(a.wrong)/float64(a.builds), a.example)
		}
	}

	fmt.Printf("\n=== GRANTED-SKILL CENSUS over %d builds ===\n", built)
	fmt.Printf("%d wrong pairs total · mean %.2f wrong/build\n", totalWrong, float64(totalWrong)/float64(built))
	report("SOURCED SOCKET GROUPS, BY WHETHER A GRANTOR STILL EXPLAINS THEM", cls, 56, 20)
	report("STAT LINES ON THE ALLOCATED NODES THAT PRODUCE NO ExtraSkill", stats, 120, 25)
}
