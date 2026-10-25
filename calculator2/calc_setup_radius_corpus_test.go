package calculator2

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/pob2"
)

var treeSourceID = func(s string) (int, bool) {
	rest, ok := strings.CutPrefix(s, "Tree:")
	if !ok {
		return 0, false
	}
	if i := strings.IndexByte(rest, ' '); i >= 0 {
		rest = rest[:i]
	}
	id, err := strconv.Atoi(rest)
	return id, err == nil
}

func radiusModKey(name, typ string, value any) string {
	switch v := value.(type) {
	case float64:
		if typ == "FLAG" {
			return name + "|FLAG"
		}
		return fmt.Sprintf("%s|%s|%.6g", name, typ, v)
	case bool:
		return name + "|FLAG"
	default:
		return name + "|" + typ
	}
}

// TestCorpusPoE2RadiusJewelsVsOracle compares, per local PoE2 build, the mods PoB-PoE2's final modDB
// holds from jewel-socket sources (the radius jewel functions' output) with ours. LIMIT=<n> caps the
// builds; POE2_RADIUS_WORKLIST=<file> writes each disagreeing build's two mod lists.
func TestCorpusPoE2RadiusJewelsVsOracle(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("CORPUS=true reads the local PoE2 corpus")
	}
	corpusDir := filepath.Join("..", "testdata", "corpus-poe2-0.5-skillsoff")
	oracleDir := filepath.Join("..", "testdata", "oracle-poe2-0.5")
	realOracle, err := filepath.EvalSymlinks(oracleDir)
	if err != nil {
		t.Skipf("no PoE2 oracle at %s", oracleDir)
	}
	dumpDir := filepath.Join(filepath.Dir(realOracle), "dumps-skillsoff")
	if _, err := os.Stat(dumpDir); err != nil {
		t.Skipf("no PoE2 skills-off dumps beside the oracle at %s", dumpDir)
	}
	entries, err := os.ReadDir(corpusDir)
	if err != nil {
		t.Skipf("no PoE2 skills-off corpus at %s", corpusDir)
	}
	tree := loadTestTree(t)
	limit, _ := strconv.Atoi(os.Getenv("LIMIT"))
	var builds []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".xml") {
			builds = append(builds, e.Name())
		}
	}
	sort.Strings(builds)
	if limit > 0 && len(builds) > limit {
		builds = builds[:limit]
	}

	type result struct {
		err       error
		any, same bool
		pob, ours []string
	}
	results := make([]result, len(builds))
	work := make(chan int)
	var wg sync.WaitGroup
	for w := 0; w < min(8, runtime.GOMAXPROCS(0)); w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range work {
				res := &results[i]
				raw, err := os.ReadFile(filepath.Join(corpusDir, builds[i]))
				var build *pob2.PathOfBuilding2
				if err == nil {
					build, err = pob2.ParseBuild(raw)
				}
				var env *Environment
				if err == nil {
					env, err = BuildOutput(build, Options{Tree: tree, Items: NewItemsTab(build, tree, nil)})
				}
				var dump struct {
					ModDB map[string][]dumpMod `json:"modDB"`
				}
				if err == nil {
					var dr []byte
					dr, err = os.ReadFile(filepath.Join(dumpDir, strings.TrimSuffix(builds[i], ".xml")+".json"))
					if err == nil {
						err = json.Unmarshal(dr, &dump)
					}
				}
				if err != nil {
					res.err = err
					continue
				}
				for name, mods := range dump.ModDB {
					for _, m := range mods {
						if id, ok := treeSourceID(m.Source); ok && tree.Sockets[id] != nil {
							res.pob = append(res.pob, radiusModKey(name, m.Type, m.Value))
						}
					}
				}
				for name, mods := range env.ModDB.Mods {
					for _, m := range mods {
						id, ok := treeSourceID(string(m.GetSource()))
						if !ok || tree.Sockets[id] == nil {
							continue
						}
						var v any
						if mv := m.Value(); mv != nil {
							switch mv.Type() {
							case mod.ModValueMultiTypeFloat:
								v = mv.Float()
							case mod.ModValueMultiTypeFlag:
								v = mv.Flag()
							}
						}
						res.ours = append(res.ours, radiusModKey(name, string(m.Type()), v))
					}
				}
				sort.Strings(res.pob)
				sort.Strings(res.ours)
				res.any = len(res.pob) > 0 || len(res.ours) > 0
				res.same = strings.Join(res.pob, "\n") == strings.Join(res.ours, "\n")
			}
		}()
	}
	for i := range builds {
		work <- i
	}
	close(work)
	wg.Wait()

	var failed, withMods, agree, pobOnly, oursOnly int
	var lines []string
	for i, r := range results {
		if r.err != nil {
			failed++
			continue
		}
		if !r.any {
			continue
		}
		withMods++
		switch {
		case r.same:
			agree++
		case len(r.ours) == 0:
			pobOnly++
		case len(r.pob) == 0:
			oursOnly++
		}
		if !r.same {
			lines = append(lines, fmt.Sprintf("%s\tpob\t%s", builds[i], strings.Join(r.pob, " ; ")), fmt.Sprintf("%s\tours\t%s", builds[i], strings.Join(r.ours, " ; ")))
		}
	}
	t.Logf("builds %d, failed %d, with jewel-socket mods on either side %d: agree %d, PoB only %d, ours only %d", len(builds), failed, withMods, agree, pobOnly, oursOnly)
	if p := os.Getenv("POE2_RADIUS_WORKLIST"); p != "" {
		if err := os.WriteFile(p, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}
