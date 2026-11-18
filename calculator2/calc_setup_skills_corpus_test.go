package calculator2

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// poe2MainSkillFloors are the builds whose main skill name, and whose ordered support names, agree with
// PoB-PoE2's dump, per corpus.
var poe2MainSkillFloors = map[string][2]int{
	"skills-off": {10000, 10000},
	"skills-on":  {10000, 10000},
}

// poe2GemRequirementStats are the requirement outputs the gem requirements feed, with the skills-off
// builds whose nonzero value agrees with PoB-PoE2's oracle.
var poe2GemRequirementFloors = map[string]int{"ReqStr": 6546, "ReqDex": 6961, "ReqInt": 8755}

type poe2SkillDump struct {
	Meta struct {
		MainSkill *string `json:"mainSkill"`
	} `json:"meta"`
	MinionMeta struct {
		Name string `json:"name"`
	} `json:"minionMeta"`
	Supports []string `json:"supports"`
}

// TestCorpusPoE2MainSkillVsDumps compares, per build, our main skill's granted effect name and its
// support list with the main skill and supports PoB-PoE2's dump records.
func TestCorpusPoE2MainSkillVsDumps(t *testing.T) {
	runPoE2Job(t, "TestCorpusPoE2MainSkillVsDumps")
}

func prepPoE2MainSkill() (*poe2Job, error) {
	if os.Getenv("CORPUS") != "true" {
		return nil, poe2SkipError("CORPUS=true reads the local PoE2 corpus")
	}
	oracleDir := filepath.Join("..", "testdata", "oracle-poe2-0.5")
	realOracle, err := filepath.EvalSymlinks(oracleDir)
	if err != nil {
		return nil, poe2SkipError(fmt.Sprintf("no PoE2 oracle at %s", oracleDir))
	}
	root := filepath.Dir(realOracle)
	if _, err := poe2TestTree(); err != nil {
		return nil, poe2SkipError(poe2TreeSkip(err))
	}
	limit, _ := strconv.Atoi(os.Getenv("LIMIT"))
	recs, err := sharedPoE2Oracle(filepath.Join(oracleDir, "output-poe2-0.5-skillsoff.jsonl.gz"))
	if err != nil {
		return nil, err
	}
	oracle := make(map[string]map[string]any, len(recs))
	for _, r := range recs {
		oracle[strings.TrimSuffix(r.Build, ".xml")] = r.Output
	}
	job := &poe2Job{}
	var reports []func(t *testing.T)
	for _, c := range []struct{ name, corpus, dumps string }{
		{"skills-off", "corpus-poe2-0.5-skillsoff", "dumps-skillsoff"},
		{"skills-on", "corpus-poe2-0.5", "dumps"},
	} {
		dumpDir := filepath.Join(root, c.dumps)
		entries, err := os.ReadDir(dumpDir)
		if err != nil {
			return nil, poe2SkipError(fmt.Sprintf("no PoE2 dumps at %s", dumpDir))
		}
		var builds []string
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".json") {
				builds = append(builds, strings.TrimSuffix(e.Name(), ".json"))
			}
		}
		sort.Strings(builds)
		if limit > 0 && len(builds) > limit {
			builds = builds[:limit]
		}
		type result struct {
			err                  error
			wantMain, gotMain    string
			gotID                string
			noTemplate, template bool
			restart              bool
			reqOK                map[string]bool
			wantSupp, gotSupp    []string
			mainOK, suppOK, seen bool
		}
		results := make([]result, len(builds))
		visit := func(_, i int, calc *poe2Calc) {
			results[i] = func() result {
				var res result
				raw, err := os.ReadFile(filepath.Join(dumpDir, builds[i]+".json"))
				if err != nil {
					return result{err: err}
				}
				var d poe2SkillDump
				if err := json.Unmarshal(raw, &d); err != nil {
					return result{err: err}
				}
				if d.Meta.MainSkill != nil {
					res.wantMain = *d.Meta.MainSkill
				}
				res.wantSupp = d.Supports
				if calc.err != nil {
					res.err = calc.err
					return res
				}
				env := calc.env
				res.seen = true
				res.restart = env.weaponSetRestart
				if c.name == "skills-off" {
					res.reqOK = map[string]bool{}
					for stat := range poe2GemRequirementFloors {
						if want, ok := oracle[builds[i]][stat].(float64); ok && want != 0 {
							res.reqOK[stat] = poe2OracleClose(env.Player.Output[stat], want, poe2OracleTolerance)
						}
					}
				}
				if m := env.Player.MainSkill; m != nil {
					ge := m.ActiveEffect.GrantedEffect
					res.gotMain, res.gotID = ge.Name, ge.ID
					res.noTemplate = ge.Overlay.NoTemplate
					// buildActiveSkillModList puts the minion's name into a "{0}" effect name.
					if before, after, ok := strings.Cut(strings.TrimSpace(ge.Name), "{0}"); ok {
						res.template = strings.HasPrefix(res.wantMain, before) && strings.HasSuffix(res.wantMain, after)
						// CalcDefence.lua:207,213 write "Spectre: "/"Companion: " plus the minion's name into the shared
						// grantedEffect.name the dump reads, so with two such gems it names whichever reserved last; the
						// port keeps data.skills read-only. Compare the main skill's own minion with the dump's instead.
						if (before == "Spectre: " || before == "Companion: ") && m.Minion != nil && m.Minion.MinionData != nil && d.MinionMeta.Name != "" {
							res.gotMain = before + m.Minion.MinionData.Name
							res.wantMain = before + d.MinionMeta.Name
						}
					}
					for _, s := range m.SupportList {
						res.gotSupp = append(res.gotSupp, s.GrantedEffect.Name)
					}
				}
				res.mainOK = res.gotMain == res.wantMain
				res.suppOK = strings.Join(res.gotSupp, "|") == strings.Join(res.wantSupp, "|")
				return res
			}()
		}
		names := make([]string, len(builds))
		for i, b := range builds {
			names[i] = b + ".xml"
		}
		job.parts = append(job.parts, &poe2Part{corpus: c.corpus, names: names, visit: visit})
		reports = append(reports, func(t *testing.T) {

			var seen, mainOK, suppOK, failed, noTemplate, template, restart int
			reqOK, reqNonzero := map[string]int{}, map[string]int{}
			var mainMiss, suppMiss []string
			for i, r := range results {
				if r.err != nil {
					failed++
					if failed <= 5 {
						t.Logf("%s %s: %v", c.name, builds[i], r.err)
					}
					continue
				}
				seen++
				if r.restart {
					restart++
				}
				for stat, ok := range r.reqOK {
					reqNonzero[stat]++
					if ok {
						reqOK[stat]++
					}
				}
				switch {
				case r.mainOK:
					mainOK++
				case r.noTemplate && r.gotMain == "":
					noTemplate++
					mainMiss = append(mainMiss, fmt.Sprintf("%s\tnoTemplate=%s\tpob=%q", builds[i], r.gotID, r.wantMain))
				case r.template:
					template++
					mainMiss = append(mainMiss, fmt.Sprintf("%s\ttemplate=%q\tpob=%q", builds[i], r.gotMain, r.wantMain))
				default:
					mainMiss = append(mainMiss, fmt.Sprintf("%s\tours=%q\tpob=%q", builds[i], r.gotMain, r.wantMain))
				}
				if r.suppOK {
					suppOK++
				} else {
					suppMiss = append(suppMiss, fmt.Sprintf("%s\t%s\tours=%q\tpob=%q", builds[i], r.wantMain, r.gotSupp, r.wantSupp))
				}
			}
			t.Logf("=== %s: %d builds, %d failed; main skill %d / %d (+%d an effect with no data template, +%d a minion name); supports %d / %d; %d need the weapon set restart",
				c.name, len(builds), failed, mainOK, seen, noTemplate, template, suppOK, seen, restart)
			for _, l := range mainMiss[:min(len(mainMiss), 25)] {
				t.Logf("  main  %s", l)
			}
			for _, l := range suppMiss[:min(len(suppMiss), 25)] {
				t.Logf("  supp  %s", l)
			}
			if out := os.Getenv("WORKLIST"); out != "" {
				_ = os.MkdirAll(out, 0o750)
				_ = os.WriteFile(filepath.Join(out, "mainskill-"+c.name+".tsv"), []byte(strings.Join(append(mainMiss, suppMiss...), "\n")+"\n"), 0o600)
			}
			for _, stat := range []string{"ReqStr", "ReqDex", "ReqInt"} {
				if c.name != "skills-off" {
					break
				}
				t.Logf("  %s nonzero matched %d / %d", stat, reqOK[stat], reqNonzero[stat])
				if limit == 0 && reqOK[stat] < poe2GemRequirementFloors[stat] {
					t.Errorf("%s: %s nonzero matched %d, floor %d", c.name, stat, reqOK[stat], poe2GemRequirementFloors[stat])
				}
			}
			if limit == 0 {
				floor := poe2MainSkillFloors[c.name]
				if mainOK < floor[0] || suppOK < floor[1] {
					t.Errorf("%s: main skill %d (floor %d), supports %d (floor %d)", c.name, mainOK, floor[0], suppOK, floor[1])
				}
			}
		})
	}
	job.report = func(t *testing.T) {
		for _, r := range reports {
			r(t)
		}
	}
	return job, nil
}
