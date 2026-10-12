package calculator

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/mod"
)

// TestFRungConversionSplit asks whether the F rung is a MODIFIER PRODUCT at all on the row being measured.
func TestFRungConversionSplit(t *testing.T) {
	requireOperatorRun(t, "report-only census")
	dumpDir := os.Getenv("CALCS_DUMP")
	if dumpDir == "" {
		t.Skip("set CALCS_DUMP=<dir> (task calcs-chain)")
	}
	requireGameData(t, "3.29")

	files, err := filepath.Glob(filepath.Join(dumpDir, "*.json"))
	if err != nil || len(files) == 0 {
		t.Fatalf("REFUSING: %s holds no .json dumps — empty must mean 'I looked and found "+
			"nothing', never 'I did not look' (%v)", dumpDir, err)
	}

	type prod struct {
		Inc  float64 `json:"inc"`
		More float64 `json:"more"`
	}
	type rec struct {
		Build            string             `json:"build"`
		SkillMismatch    bool               `json:"skillMismatch"`
		MainHand         numMap             `json:"outputMainHand"`
		OffHand          numMap             `json:"outputOffHand"`
		ProductsMainHand map[string]prod    `json:"productsMainHand"`
		ProductsOffHand  map[string]prod    `json:"productsOffHand"`
		_                map[string]float64 `json:"-"`
	}

	corpus := corpusDirForLoadedData(t)
	type tally struct{ compared, agreed int }
	var pure, converted tally
	pureClusters := map[string]int{}
	convClusters := map[string]int{}
	incDelta := map[string]int{}
	incDeltaEx := map[string]string{}
	moreRatio := map[string]int{}
	incAgree, incCompared := 0, 0
	unusable, mismatch := 0, 0

	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			unusable++
			continue
		}
		var r rec
		if json.Unmarshal(raw, &r) != nil || r.Build == "" {
			unusable++
			continue
		}
		if r.SkillMismatch {
			mismatch++
			continue
		}
		if len(r.MainHand) == 0 {
			continue
		}
		xml, err := os.ReadFile(filepath.Join(corpus, r.Build))
		if err != nil {
			unusable++
			continue
		}
		b, err := builds.ParseBuild(xml)
		if err != nil {
			unusable++
			continue
		}
		env := (&Calculator{PoB: b, AlignCalcsSkillToMain: true}).BuildOutput(OutputModeCalcs)
		if env == nil || env.Player == nil {
			unusable++
			continue
		}
		if len(env.Player.OutputTable[OutTableMainHand]) == 0 {
			unusable++
			continue
		}
		skill := env.Player.MainSkill

		for _, hand := range []struct {
			name string
			want map[string]float64
			tbl  OutTable
			prod map[string]prod
		}{
			{"MainHand", r.MainHand, OutTableMainHand, r.ProductsMainHand},
			{"OffHand", r.OffHand, OutTableOffHand, r.ProductsOffHand},
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
				if !okB || !okM || !okE || base == 0 || em == 0 {
					continue
				}
				ourBase, ourMin, ourEm := got[string(dt)+"MinBase"], got[string(dt)+"Min"], got[string(dt)+"EffMult"]
				if ourBase == 0 || ourEm == 0 {
					continue
				}
				if math.Abs(ourBase-base) > math.Abs(base)*0.005 ||
					math.Abs(ourEm-em) > math.Abs(em)*0.005 {
					continue
				}
				wantF := mn / (base * em)
				if wantF == 0 {
					continue
				}
				gotF := ourMin / (ourBase * ourEm)

				convPresent := false
				for k := range hand.prod {
					a, bb, ok := strings.Cut(k, "/")
					if ok && a == string(dt) && bb != string(dt) {
						convPresent = true
						break
					}
				}

				if !convPresent {
					if w, ok := hand.prod[string(dt)+"/"+string(dt)]; ok && skill != nil {
						names := data.DamageStatsForType(data.DamageTypeFlags[dt])
						cfg := skill.Weapon1Cfg
						if hand.tbl == OutTableOffHand {
							cfg = skill.Weapon2Cfg
						}
						if cfg != nil && skill.SkillModList != nil {
							ourInc := 1 + skill.SkillModList.Sum(mod.TypeIncrease, cfg, names...)/100
							ourMore := skill.SkillModList.More(cfg, names...)
							incCompared++
							if math.Abs(ourInc-w.Inc) <= math.Abs(w.Inc)*0.005 {
								incAgree++
							} else {
								dk := fmt.Sprintf("%+.1f", (ourInc-w.Inc)*100)
								incDelta[dk]++
								if incDeltaEx[dk] == "" {
									incDeltaEx[dk] = fmt.Sprintf("%s %s %s", strings.TrimSuffix(r.Build, ".xml"), hand.name, dt)
								}
							}
							if w.More != 0 && math.Abs(ourMore-w.More) > math.Abs(w.More)*0.005 {
								moreRatio[fmt.Sprintf("%.4f", ourMore/w.More)]++
							}
						}
					}
				}

				agree := math.Abs(gotF-wantF) <= math.Abs(wantF)*0.005
				cl := fmt.Sprintf("%.4f", gotF/wantF)
				if convPresent {
					converted.compared++
					if agree {
						converted.agreed++
					} else {
						convClusters[cl]++
					}
				} else {
					pure.compared++
					if agree {
						pure.agreed++
					} else {
						pureClusters[cl]++
					}
				}
			}
		}
	}

	if pure.compared+converted.compared == 0 {
		t.Fatalf("REFUSING: 0 comparable F rows across %d dumps (%d skill-mismatched, %d "+
			"unusable) — empty must mean 'I looked and found nothing', never 'I did not look'",
			len(files), mismatch, unusable)
	}

	pct := func(t tally) float64 {
		if t.compared == 0 {
			return 0
		}
		return float64(t.agreed) / float64(t.compared) * 100
	}
	fmt.Printf("\n=== F RUNG, SPLIT BY WHETHER IT IS A PRODUCT AT ALL ===\n")
	fmt.Printf("(denominator-clean rows only; %d dumps, %d skill-mismatched, %d unusable)\n",
		len(files), mismatch, unusable)
	fmt.Printf("  NO CONVERSION (F really is inc*more*gMore*mMin*iMin)  %4d compared  %4d agree  %5.1f%%\n",
		pure.compared, pure.agreed, pct(pure))
	fmt.Printf("  CONVERSION PRESENT (F carries an ADDITIVE term)       %4d compared  %4d agree  %5.1f%%\n",
		converted.compared, converted.agreed, pct(converted))

	top := func(label string, m map[string]int) {
		type kv struct {
			s string
			n int
		}
		rows := make([]kv, 0, len(m))
		for k, v := range m {
			rows = append(rows, kv{k, v})
		}
		sort.Slice(rows, func(i, j int) bool {
			if rows[i].n != rows[j].n {
				return rows[i].n > rows[j].n
			}
			return rows[i].s < rows[j].s
		})
		fmt.Printf("  %s (%d distinct ratios)\n", label, len(rows))
		for i, r := range rows {
			if i >= 10 {
				fmt.Printf("    (%d further ratios)\n", len(rows)-i)
				break
			}
			fmt.Printf("    %-10s %d\n", r.s, r.n)
		}
	}
	top("disagreeing ratios, NO-CONVERSION rows:", pureClusters)
	top("disagreeing ratios, CONVERSION rows:", convClusters)
	fmt.Printf("\n  --- the same rows split into the ORIGINAL's own terms (no-conversion only) ---\n")
	fmt.Printf("  inc: %d compared, %d agree\n", incCompared, incAgree)
	{
		type kv struct {
			s string
			n int
		}
		rows := make([]kv, 0, len(incDelta))
		for k, v := range incDelta {
			rows = append(rows, kv{k, v})
		}
		sort.Slice(rows, func(i, j int) bool {
			if rows[i].n != rows[j].n {
				return rows[i].n > rows[j].n
			}
			return rows[i].s < rows[j].s
		})
		fmt.Printf("  inc disagreements, as ADDITIVE INC POINTS (ours - pob) — %d distinct\n", len(rows))
		for i, r := range rows {
			if i >= 10 {
				fmt.Printf("    (%d further deltas)\n", len(rows)-i)
				break
			}
			fmt.Printf("    %-10s %3d   e.g. %s\n", r.s, r.n, incDeltaEx[r.s])
		}
	}
	top("more disagreements, as a RATIO (ours / pob):", moreRatio)
	fmt.Printf("  ⚠️ Read the two ratio tables against each other, not on their own. The claim under\n" +
		"     test is that the NO-CONVERSION side is clusterable and the CONVERSION side is not;\n" +
		"     a long tail on BOTH would refute it and say F's residual is coverage debt either way.\n")
}
