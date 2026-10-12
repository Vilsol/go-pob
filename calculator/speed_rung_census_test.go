package calculator

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/mod"
)

// TestSpeedRungCensus grades the INPUTS to CalcOffence.lua:2337 against PoB's own, rung by rung.
func TestSpeedRungCensus(t *testing.T) {
	requireOperatorRun(t, "report-only census")
	dumpPath := os.Getenv("SPEED_RUNGS")
	if dumpPath == "" {
		t.Skip("set SPEED_RUNGS=<jsonl from oracle/dump_speed_rungs.lua>")
	}
	requireGameData(t, "3.29")

	fh, err := os.Open(dumpPath)
	if err != nil {
		t.Fatalf("REFUSING: cannot read %s: %v", dumpPath, err)
	}
	defer fh.Close()

	type rec struct {
		Build                 string   `json:"build"`
		Error                 string   `json:"error"`
		Skill                 string   `json:"skill"`
		IsAttack              bool     `json:"isAttack"`
		AttackSpeedMultiplier *float64 `json:"attackSpeedMultiplier"`
		Inc                   float64  `json:"inc"`
		More                  float64  `json:"more"`
		BaseSpeedSum          float64  `json:"baseSpeedSum"`
		AttackRate1           float64  `json:"attackRate1"`
		Speed                 float64  `json:"speed"`
		MHSpeed               float64  `json:"mhSpeed"`
	}

	var rows *os.File
	if p := os.Getenv("SPEED_RUNG_OUT"); p != "" {
		var cerr error
		rows, cerr = os.Create(p)
		if cerr != nil {
			t.Fatalf("REFUSING: cannot write SPEED_RUNG_OUT=%s: %v", p, cerr)
		}
		defer rows.Close()
		fmt.Fprintln(rows, "build\trung\tpob\tours\thasMultiplier")
	}

	corpus := corpusDirForLoadedData(t)
	compared, skipped, skillMismatch := 0, 0, 0
	var mismatchExamples []string
	agree := map[string]int{}
	seen := map[string]int{}
	moreDiffWithMult, moreDiffWithoutMult := 0, 0
	withMult, withoutMult := 0, 0

	scanner := bufio.NewScanner(fh)
	scanner.Buffer(make([]byte, 0, 1<<20), 1<<22)
	for scanner.Scan() {
		var r rec
		if json.Unmarshal(scanner.Bytes(), &r) != nil || r.Build == "" || r.Error != "" {
			skipped++
			continue
		}
		xml, err := os.ReadFile(filepath.Join(corpus, r.Build))
		if err != nil {
			skipped++
			continue
		}
		b, err := builds.ParseBuild(xml)
		if err != nil {
			skipped++
			continue
		}
		env := (&Calculator{PoB: b}).BuildOutput(OutputModeMain)
		if env == nil || env.Player == nil || env.Player.MainSkill == nil {
			skipped++
			continue
		}
		as := env.Player.MainSkill

		ourSkill := ""
		if as.ActiveEffect != nil && as.ActiveEffect.GrantedEffect != nil {
			if raw := as.ActiveEffect.GrantedEffect.Raw.GetActiveSkill(); raw != nil {
				ourSkill = raw.DisplayedName
			}
		}
		if r.Skill != "" && ourSkill != "" && r.Skill != ourSkill {
			skillMismatch++
			mismatchExamples = append(mismatchExamples,
				fmt.Sprintf("%s: PoB %q vs ours %q", r.Build, r.Skill, ourSkill))
			continue
		}

		cfg := as.SkillCfg
		ourSpeed := env.Player.OutputTable[OutTableMainHand]["Speed"]
		if ourSpeed == 0 {
			ourSpeed = env.Player.Output["Speed"]
		}
		var ourRate float64
		if env.Player.WeaponData1 != nil {
			ourRate = env.Player.WeaponData1.AttackRate
		}
		wantSpeed := r.MHSpeed
		if wantSpeed == 0 {
			wantSpeed = r.Speed
		}
		compared++

		hasMult := r.AttackSpeedMultiplier != nil && *r.AttackSpeedMultiplier != 0
		compare := func(name string, want, got float64) bool {
			seen[name]++
			if rows != nil {
				fmt.Fprintf(rows, "%s\t%s\t%g\t%g\t%v\n", r.Build, name, want, got, hasMult)
			}
			if math.Abs(got-want) <= math.Abs(want)*0.005+1e-12 {
				agree[name]++
				return true
			}
			return false
		}
		compare("inc", r.Inc, as.SkillModList.Sum(mod.TypeIncrease, cfg, "Speed"))
		compare("baseSpeedSum", r.BaseSpeedSum, as.SkillModList.Sum(mod.TypeBase, cfg, "Speed"))
		compare("AttackRate", r.AttackRate1, ourRate)
		moreOK := compare("more", r.More, as.SkillModList.More(cfg, "Speed"))
		compare("Speed", wantSpeed, ourSpeed)

		if hasMult {
			withMult++
			if !moreOK {
				moreDiffWithMult++
			}
		} else {
			withoutMult++
			if !moreOK {
				moreDiffWithoutMult++
			}
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("REFUSING: cannot scan %s: %v", dumpPath, err)
	}
	if compared == 0 {
		t.Fatalf("REFUSING: compared 0 builds from %s (%d skipped) — empty must mean "+
			"'I looked and found nothing', never 'I did not look'", dumpPath, skipped)
	}

	fmt.Printf("=== SPEED RUNG CENSUS === %d builds compared, %d skipped, %d skipped for a "+
		"MAIN-SKILL MISMATCH\n", compared, skipped, skillMismatch)
	for i, e := range mismatchExamples {
		if i >= 5 {
			fmt.Printf("  ... %d further mismatches\n", len(mismatchExamples)-5)
			break
		}
		fmt.Printf("  skill mismatch: %s\n", e)
	}
	names := make([]string, 0, len(seen))
	for k := range seen {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, k := range names {
		fmt.Printf("  %-14s %5d compared  %5d agree  %5.1f%%\n",
			k, seen[k], agree[k], 100*float64(agree[k])/float64(seen[k]))
	}
	fmt.Printf("\n`more` DIFFERS, split by whether the skill has an attackSpeedMultiplier:\n")
	fmt.Printf("  with a multiplier    %5d of %5d\n", moreDiffWithMult, withMult)
	fmt.Printf("  without one          %5d of %5d\n", moreDiffWithoutMult, withoutMult)
	if withoutMult == 0 {
		fmt.Printf("  ⚠️ NO CONTROL GROUP IN THIS DUMP — every build carries a multiplier, so the\n" +
			"     split above says nothing. Re-run the dumper over a list that includes builds\n" +
			"     WITHOUT one before reading the first row as a cause.\n")
	}
	fmt.Printf("\n⚠️ `more` is expected to differ by exactly (1 + attackSpeedMultiplier/100) until\n" +
		"   calc_active_skill.go stops adding it as a Speed MORE mod AND calc_offence.go starts\n" +
		"   scaling the pass's AttackRate. Landing either half alone DOUBLE-APPLIES it.\n")
}
