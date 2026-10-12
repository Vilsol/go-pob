package calculator

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// TestCalcsHandRungCensus is TestCalcsEffMultCensus for the PER-HAND tables.
func TestCalcsHandRungCensus(t *testing.T) {
	requireOperatorRun(t, "report-only census")
	dumpDir := os.Getenv("CALCS_DUMP")
	if dumpDir == "" {
		t.Skip("set CALCS_DUMP=<dir> (oracle/dump_calcs_damage.lua)")
	}
	requireGameData(t, "3.29")

	files, err := filepath.Glob(filepath.Join(dumpDir, "*.json"))
	if err != nil || len(files) == 0 {
		t.Fatalf("REFUSING: %s holds no .json dumps — empty must mean 'I looked and found "+
			"nothing', never 'I did not look' (%v)", dumpDir, err)
	}

	type rec struct {
		Build            string `json:"build"`
		SkillMismatch    bool   `json:"skillMismatch"`
		MainHand         numMap `json:"outputMainHand"`
		OffHand          numMap `json:"outputOffHand"`
		ProductsMainHand map[string]struct {
			Inc  float64 `json:"inc"`
			More float64 `json:"more"`
		} `json:"productsMainHand"`
		ProductsOffHand map[string]struct {
			Inc  float64 `json:"inc"`
			More float64 `json:"more"`
		} `json:"productsOffHand"`
	}

	var rows *os.File
	if p := os.Getenv("HAND_RUNG_OUT"); p != "" {
		var err error
		rows, err = os.Create(p)
		if err != nil {
			t.Fatalf("REFUSING: cannot write HAND_RUNG_OUT=%s: %v", p, err)
		}
		defer rows.Close()
		fmt.Fprintln(rows, "build\thand\trung\ttype\tpob\tours")
	}

	corpus := corpusDirForLoadedData(t)
	type bucket struct {
		builds  int
		example string
	}
	clusters := map[string]*bucket{}
	compared, agreed, mismatchSkipped, noHand := 0, 0, 0, 0
	unusableBy := map[string]int{}
	unusable := func(why string) { unusableBy[why]++ }
	perRung, perRungOK := map[string]int{}, map[string]int{}
	firstBy, firstExample := map[string]int{}, map[string]string{}
	firstBuilds := map[string]map[string]bool{}
	chainsCompared, chainsClean := 0, 0
	rungs := []string{"MinBase", "MaxBase", "EffMult", "inc", "more", "F(modifiers)", "F | denom clean", "Min", "Max", "HitAverage"}

	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			unusable("dump unreadable")
			continue
		}
		var r rec
		if json.Unmarshal(raw, &r) != nil || r.Build == "" {
			unusable("dump unparseable")
			continue
		}
		if r.SkillMismatch {
			mismatchSkipped++
			continue
		}
		if len(r.MainHand) == 0 {
			noHand++
			continue
		}
		xml, err := os.ReadFile(filepath.Join(corpus, r.Build))
		if err != nil {
			unusable("corpus xml missing")
			continue
		}
		b, err := builds.ParseBuild(xml)
		if err != nil {
			unusable("ParseBuild failed")
			continue
		}
		env := (&Calculator{PoB: b, AlignCalcsSkillToMain: true}).BuildOutput(OutputModeCalcs)
		if env == nil || env.Player == nil {
			unusable("no CALCS env")
			continue
		}
		if len(env.Player.OutputTable[OutTableMainHand]) == 0 {
			unusable("our CALCS run has no MainHand table")
			continue
		}

		skill := env.Player.MainSkill
		for _, hand := range []struct {
			name string
			want map[string]float64
			tbl  OutTable
			prod map[string]struct {
				Inc  float64 `json:"inc"`
				More float64 `json:"more"`
			}
			cfg *moddb.ListCfg
		}{
			{"MainHand", r.MainHand, OutTableMainHand, r.ProductsMainHand, skill.Weapon1Cfg},
			{"OffHand", r.OffHand, OutTableOffHand, r.ProductsOffHand, skill.Weapon2Cfg},
		} {
			if len(hand.want) == 0 {
				continue
			}
			got := env.Player.OutputTable[hand.tbl]
			if got == nil {
				continue
			}
			for _, dt := range data.DamageType("").Values() {
				base, okB := hand.want[string(dt)+"MinBase"]
				mn, okM := hand.want[string(dt)+"Min"]
				em, okE := hand.want[string(dt)+"EffMult"]
				if okB && okM && okE && base != 0 && em != 0 {
					ourBase, ourMin, ourEm := got[string(dt)+"MinBase"], got[string(dt)+"Min"], got[string(dt)+"EffMult"]
					if ourBase != 0 && ourEm != 0 {
						wantF := mn / (base * em)
						gotF := ourMin / (ourBase * ourEm)
						if wantF != 0 {
							compared++
							perRung["F(modifiers)"]++
							if rows != nil {
								fmt.Fprintf(rows, "%s\t%s\t%s\t%s\t%g\t%g\n", r.Build, hand.name, "F", dt, wantF, gotF)
							}
							denomClean := math.Abs(ourBase-base) <= math.Abs(base)*0.005 &&
								math.Abs(ourEm-em) <= math.Abs(em)*0.005
							if denomClean {
								perRung["F | denom clean"]++
							}
							if math.Abs(gotF-wantF) <= math.Abs(wantF)*0.005 {
								agreed++
								perRungOK["F(modifiers)"]++
								if denomClean {
									perRungOK["F | denom clean"]++
								}
							} else {
								ck := fmt.Sprintf("%-12s %-8s %-10s %6.2f", hand.name, "F", dt,
									math.Round(gotF/wantF*100)/100)
								if clusters[ck] == nil {
									clusters[ck] = &bucket{example: r.Build}
								}
								clusters[ck].builds++
							}
						}
					}
				}
				if hand.prod != nil && hand.cfg != nil && skill.SkillModList != nil {
					if want, okP := hand.prod[string(dt)+"/"+string(dt)]; okP {
						names := data.DamageStatsForType(data.DamageTypeFlags[dt])
						ourInc := 1 + skill.SkillModList.Sum(mod.TypeIncrease, hand.cfg, names...)/100
						ourMore := skill.SkillModList.More(hand.cfg, names...)
						for _, f := range []struct {
							rung       string
							want, ours float64
						}{
							{"inc", want.Inc, ourInc},
							{"more", want.More, ourMore},
						} {
							if f.want == 0 {
								continue
							}
							compared++
							perRung[f.rung]++
							if rows != nil {
								fmt.Fprintf(rows, "%s\t%s\t%s\t%s\t%g\t%g\n", r.Build, hand.name, f.rung, dt, f.want, f.ours)
							}
							if math.Abs(f.ours-f.want) <= math.Abs(f.want)*0.005 {
								agreed++
								perRungOK[f.rung]++
								continue
							}
							ck := fmt.Sprintf("%-12s %-8s %-10s %6.2f", hand.name, f.rung, dt,
								math.Round(f.ours/f.want*100)/100)
							if clusters[ck] == nil {
								clusters[ck] = &bucket{example: r.Build}
							}
							clusters[ck].builds++
						}
					}
				}

				for _, rung := range []string{"MinBase", "MaxBase", "EffMult", "Min", "Max", "HitAverage"} {
					key := string(dt) + rung
					want, ok := hand.want[key]
					if !ok || want == 0 {
						continue
					}
					compared++
					perRung[rung]++
					if rows != nil {
						fmt.Fprintf(rows, "%s\t%s\t%s\t%s\t%g\t%g\n", r.Build, hand.name, rung, dt, want, got[key])
					}
					if math.Abs(got[key]-want) <= math.Abs(want)*0.005 {
						agreed++
						perRungOK[rung]++
						continue
					}
					ck := fmt.Sprintf("%-12s %-8s %-10s %6.2f", hand.name, rung, dt,
						math.Round(got[key]/want*100)/100)
					if clusters[ck] == nil {
						clusters[ck] = &bucket{example: r.Build}
					}
					clusters[ck].builds++
				}

				if first := calcsFirstDivergingRung(hand.want, got, string(dt)); first != "" {
					chainsCompared++
					firstBy[first]++
					if firstExample[first] == "" {
						firstExample[first] = r.Build + " " + hand.name + " " + string(dt)
					}
					if firstBuilds[first] == nil {
						firstBuilds[first] = map[string]bool{}
					}
					firstBuilds[first][r.Build] = true
				} else if _, seen := hand.want[string(dt)+"MinBase"]; seen {
					chainsCompared++
					chainsClean++
				}
			}
		}
	}

	if compared == 0 {
		t.Fatalf("REFUSING: compared 0 per-hand rungs across %d dumps (%d skill-mismatched, "+
			"%d with no MainHand table, unusable %v). Either the dump carries no per-hand "+
			"output or every build was skipped", len(files), mismatchSkipped, noHand, unusableBy)
	}

	keys := make([]string, 0, len(clusters))
	for k := range clusters {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if clusters[keys[i]].builds != clusters[keys[j]].builds {
			return clusters[keys[i]].builds > clusters[keys[j]].builds
		}
		return keys[i] < keys[j]
	})

	fmt.Printf("=== CALCS PER-HAND RUNG CENSUS === %d dumps; %d values compared, %d agree (%.1f%%), "+
		"%d skill-mismatched, %d without a MainHand table, unusable %v\n",
		len(files), compared, agreed, 100*float64(agreed)/float64(compared),
		mismatchSkipped, noHand, unusableBy)
	fmt.Printf("\nBY RUNG — where the per-hand damage chain diverges:\n")
	fmt.Printf("  (`F | denom clean` is F restricted to rows where MinBase AND EffMult agree — the\n" +
		"   only rows on which F is a statement about the modifier product at all.)\n")
	for _, rung := range rungs {
		if perRung[rung] == 0 {
			fmt.Printf("  %-12s  (no values in the dump)\n", rung)
			continue
		}
		fmt.Printf("  %-12s %6d compared  %6d agree  %5.1f%%\n",
			rung, perRung[rung], perRungOK[rung], 100*float64(perRungOK[rung])/float64(perRung[rung]))
	}
	fmt.Printf("\nFIRST DIVERGING RUNG — one cause per chain, ranked (this is the ATTRIBUTION;\n")
	fmt.Printf("  the table above counts one cause once per CONSUMER and so double-counts it)\n")
	fmt.Printf("  %d (build,hand,type) chains compared, %d agree end to end\n", chainsCompared, chainsClean)
	fmt.Printf("  %-14s %8s %8s  %s\n", "RUNG", "CHAINS", "BUILDS", "EXAMPLE")
	ordered := []string{"MinBase", "MaxBase", "EffMult", "F(modifiers)", "Min", "Max", "HitAverage"}
	byCount := append([]string{}, ordered...)
	sort.Slice(byCount, func(i, j int) bool { return firstBy[byCount[i]] > firstBy[byCount[j]] })
	for _, rung := range byCount {
		if firstBy[rung] == 0 {
			continue
		}
		fmt.Printf("  %-14s %8d %8d  %s\n", rung, firstBy[rung], len(firstBuilds[rung]), firstExample[rung])
	}

	fmt.Printf("\n%-12s %-8s %-10s %6s %8s  %s\n", "HAND", "RUNG", "TYPE", "RATIO", "BUILDS", "EXAMPLE")
	for i, k := range keys {
		if i >= 50 {
			fmt.Printf("... %d further clusters not shown\n", len(keys)-50)
			break
		}
		fmt.Printf("%s %8d  %s\n", k, clusters[k].builds, clusters[k].example)
	}
}

type numMap map[string]float64

func (n *numMap) UnmarshalJSON(b []byte) error {
	for _, c := range b {
		switch c {
		case ' ', '\t', '\r', '\n':
			continue
		case '[', 'n':
			*n = numMap{}
			return nil
		}
		break
	}
	var m map[string]float64
	if err := json.Unmarshal(b, &m); err != nil {
		return fmt.Errorf("numMap: %w", err)
	}
	*n = m
	return nil
}
