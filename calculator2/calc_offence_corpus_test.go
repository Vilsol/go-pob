package calculator2

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/Vilsol/go-pob/pob2"
)

// poe2SkillDataMatchedFloor is the main skill skillData entries that agree with the skills-off dumps.
const poe2SkillDataMatchedFloor = 30318

type poe2SkillDataDump struct {
	Meta struct {
		MainSkill string `json:"mainSkill"`
	} `json:"meta"`
	SkillData json.RawMessage `json:"skillData"`
}

func skillDataAgrees(want, got any) bool {
	switch w := want.(type) {
	case float64:
		g, ok := got.(float64)
		return ok && poe2OracleClose(g, w, poe2OracleTolerance)
	case bool:
		return truthy(got) == w
	case string:
		g, ok := got.(string)
		return ok && g == w
	}
	return false
}

// TestCorpusPoE2OffenceSkillDataVsDumps compares the main skill's skillData after offence with the
// skills-off dumps, which PoB-PoE2 writes after calcs.offence. Builds whose main skill differs are
// left out and counted. The dumps also carry keys the damage passes write, so a key below full
// agreement is not by itself a setup defect.
func TestCorpusPoE2OffenceSkillDataVsDumps(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("CORPUS=true reads the local PoE2 corpus")
	}
	oracleDir := filepath.Join("..", "testdata", "oracle-poe2-0.5")
	realOracle, err := filepath.EvalSymlinks(oracleDir)
	if err != nil {
		t.Skipf("no PoE2 oracle at %s", oracleDir)
	}
	dumpDir := filepath.Join(filepath.Dir(realOracle), "dumps-skillsoff")
	corpusDir := filepath.Join("..", "testdata", "corpus-poe2-0.5-skillsoff")
	entries, err := os.ReadDir(dumpDir)
	if err != nil {
		t.Skipf("no PoE2 dumps at %s", dumpDir)
	}
	var builds []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".json") {
			builds = append(builds, strings.TrimSuffix(e.Name(), ".json"))
		}
	}
	sort.Strings(builds)
	if len(builds) < 9000 {
		t.Fatalf("REFUSING: %d dumps, expected >=9000", len(builds))
	}
	tree := loadTestTree(t)

	type keyResult struct {
		key       string
		want, got any
		agrees    bool
	}
	type result struct {
		err       error
		otherMain bool
		keys      []keyResult
	}
	results := make([]result, len(builds))
	work := make(chan int)
	var wg sync.WaitGroup
	for range poe2Workers() {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range work {
				results[i] = func() result {
					raw, err := os.ReadFile(filepath.Join(dumpDir, builds[i]+".json"))
					if err != nil {
						return result{err: err}
					}
					var d poe2SkillDataDump
					if err := json.Unmarshal(raw, &d); err != nil {
						return result{err: err}
					}
					xml, err := os.ReadFile(filepath.Join(corpusDir, builds[i]+".xml"))
					if err != nil {
						return result{err: err}
					}
					build, err := pob2.ParseBuild(xml)
					if err != nil {
						return result{err: err}
					}
					var env *Environment
					func() {
						defer func() {
							if r := recover(); r != nil {
								err = fmt.Errorf("panic: %v", r)
							}
						}()
						env, err = BuildOutput(build, Options{Tree: tree, Items: NewItemsTab(build, tree, nil)})
					}()
					if err != nil {
						return result{err: err}
					}
					main := env.Player.MainSkill
					if main == nil || main.ActiveEffect.GrantedEffect.Name != d.Meta.MainSkill {
						return result{otherMain: true}
					}
					// An empty skillData table is dumped as [].
					var dumped map[string]any
					if len(d.SkillData) > 0 && d.SkillData[0] == '{' {
						if err := json.Unmarshal(d.SkillData, &dumped); err != nil {
							return result{err: err}
						}
					}
					var res result
					for key, want := range dumped {
						res.keys = append(res.keys, keyResult{key: key, want: want, got: main.SkillData[key], agrees: skillDataAgrees(want, main.SkillData[key])})
					}
					return res
				}()
			}
		}()
	}
	for i := range builds {
		work <- i
	}
	close(work)
	wg.Wait()

	type tally struct{ seen, ok int }
	perKey := map[string]*tally{}
	failed, otherMain, matched, total := 0, 0, 0, 0
	firstErr := ""
	for i, r := range results {
		switch {
		case r.err != nil:
			failed++
			if firstErr == "" {
				firstErr = fmt.Sprintf("%s: %v", builds[i], r.err)
			}
			continue
		case r.otherMain:
			otherMain++
			continue
		}
		for _, k := range r.keys {
			tl := perKey[k.key]
			if tl == nil {
				tl = &tally{}
				perKey[k.key] = tl
			}
			tl.seen++
			total++
			if k.agrees {
				tl.ok++
				matched++
			}
		}
	}
	keys := make([]string, 0, len(perKey))
	for k := range perKey {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(a, b int) bool {
		ma, mb := perKey[keys[a]].seen-perKey[keys[a]].ok, perKey[keys[b]].seen-perKey[keys[b]].ok
		if ma != mb {
			return ma > mb
		}
		return keys[a] < keys[b]
	})
	fmt.Printf("\n=== CORPUS PoE2 0.5 main skill skillData vs DUMPS (skills off) ===\n")
	fmt.Printf("%d builds, %d failed, %d with another main skill (left out)\n", len(builds), failed, otherMain)
	fmt.Printf("matched %d / %d  (%.4f%%)\n", matched, total, 100*float64(matched)/math.Max(1, float64(total)))
	fmt.Printf("  %-40s %8s %8s\n", "key", "seen", "ok")
	for _, k := range keys[:min(40, len(keys))] {
		fmt.Printf("  %-40s %8d %8d\n", k, perKey[k].seen, perKey[k].ok)
	}
	if path := os.Getenv("SKILLDATA_MISSES"); path != "" {
		var b strings.Builder
		for i, r := range results {
			for _, k := range r.keys {
				if !k.agrees {
					fmt.Fprintf(&b, "%s\t%s\t%v\t%v\n", builds[i], k.key, k.want, k.got)
				}
			}
		}
		if err := os.WriteFile(path, []byte(b.String()), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if failed > 0 {
		t.Errorf("%d builds failed (first: %s)", failed, firstErr)
	}
	if matched < poe2SkillDataMatchedFloor {
		t.Errorf("matched %d skillData entries, below floor %d", matched, poe2SkillDataMatchedFloor)
	} else if matched > poe2SkillDataMatchedFloor {
		fmt.Printf("matched %d is above the floor %d — raise poe2SkillDataMatchedFloor\n", matched, poe2SkillDataMatchedFloor)
	}
}
