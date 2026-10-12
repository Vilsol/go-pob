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
)

// TestCalcsEffMultCensus ranks the per-damage-type <type>EffMult divergence against PoB, across a dump produced by oracle/dump_calcs_damage.lua.
func TestCalcsEffMultCensus(t *testing.T) {
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
		Build         string             `json:"build"`
		SkillMismatch bool               `json:"skillMismatch"`
		Output        map[string]float64 `json:"output"`
	}

	corpus := corpusDirForLoadedData(t)
	type bucket struct {
		builds  int
		example string
		ratio   float64
	}
	clusters := map[string]*bucket{}
	compared, agreed, mismatchSkipped, unusable := 0, 0, 0, 0
	perRung, perRungOK := map[string]int{}, map[string]int{}

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
			mismatchSkipped++
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
		for _, dt := range data.DamageType("").Values() {
			base, okB := r.Output[string(dt)+"MinBase"]
			mn, okM := r.Output[string(dt)+"Min"]
			em, okE := r.Output[string(dt)+"EffMult"]
			if !okB || !okM || !okE || base == 0 || em == 0 {
				continue
			}
			ourBase, ourMin, ourEm := env.Player.Output[string(dt)+"MinBase"], env.Player.Output[string(dt)+"Min"], env.Player.Output[string(dt)+"EffMult"]
			if ourBase == 0 || ourEm == 0 {
				continue
			}
			wantF := mn / (base * em)
			gotF := ourMin / (ourBase * ourEm)
			if wantF == 0 {
				continue
			}
			compared++
			perRung["F(modifiers)"]++
			if math.Abs(gotF-wantF) <= math.Abs(wantF)*0.005 {
				agreed++
				perRungOK["F(modifiers)"]++
				continue
			}
			ratio := math.Round(gotF/wantF*100) / 100
			ck := fmt.Sprintf("%-8s %-10s %6.2f", "F", dt, ratio)
			c := clusters[ck]
			if c == nil {
				c = &bucket{example: r.Build, ratio: ratio}
				clusters[ck] = c
			}
			c.builds++
		}
		for _, rung := range []string{"MinBase", "MaxBase", "Min", "Max", "EffMult"} {
			for _, dt := range data.DamageType("").Values() {
				key := string(dt) + rung
				want, ok := r.Output[key]
				if !ok || want == 0 {
					continue
				}
				got := env.Player.Output[key]
				compared++
				perRung[rung]++
				if math.Abs(got-want) <= math.Abs(want)*0.005 {
					agreed++
					perRungOK[rung]++
					continue
				}
				ratio := math.Round(got/want*100) / 100
				ck := fmt.Sprintf("%-8s %-10s %6.2f", rung, dt, ratio)
				c := clusters[ck]
				if c == nil {
					c = &bucket{example: r.Build, ratio: ratio}
					clusters[ck] = c
				}
				c.builds++
			}
		}
	}

	if compared == 0 {
		t.Fatalf("REFUSING: compared 0 EffMult values across %d dumps (%d skill-mismatched, "+
			"%d unusable). Either the dump carries no <type>EffMult or every build was skipped",
			len(files), mismatchSkipped, unusable)
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

	fmt.Printf("=== CALCS EFFMULT CENSUS === %d dumps; %d values compared, %d agree (%.1f%%), "+
		"%d builds skipped for PoB-side skill mismatch, %d unusable\n",
		len(files), compared, agreed, 100*float64(agreed)/float64(compared), mismatchSkipped, unusable)
	fmt.Printf("\nBY RUNG — where the damage chain actually diverges:\n")
	for _, rung := range []string{"MinBase", "MaxBase", "EffMult", "F(modifiers)", "Min", "Max"} {
		if perRung[rung] == 0 {
			fmt.Printf("  %-8s  (no values in the dump)\n", rung)
			continue
		}
		fmt.Printf("  %-8s %6d compared  %6d agree  %5.1f%%\n",
			rung, perRung[rung], perRungOK[rung], 100*float64(perRungOK[rung])/float64(perRung[rung]))
	}
	fmt.Printf("\n%-8s %-10s %6s %8s  %s\n", "RUNG", "TYPE", "RATIO", "BUILDS", "EXAMPLE")
	for i, k := range keys {
		if i >= 40 {
			fmt.Printf("... %d further clusters not shown\n", len(keys)-40)
			break
		}
		fmt.Printf("%s %8d  %s\n", k, clusters[k].builds, clusters[k].example)
	}
}
