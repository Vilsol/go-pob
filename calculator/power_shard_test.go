package calculator

import (
	"fmt"
	"math"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

func powerShardRowLines(rows []PowerReportEntry) []string {
	f := func(v float64) string { return strconv.FormatUint(math.Float64bits(v), 16) }
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, fmt.Sprintf("%d\t%q\t%s\t%s\t%s\t%d\t%v\t%v\t%s\t%s\t%q", r.ID, r.Name, r.Type, f(r.Power), f(r.PathPower), r.PathDist,
			r.Allocated, r.Cluster, f(r.Offence), f(r.Defence), r.Stats))
	}
	return out
}

// TestPowerShardsMergeToTheReport: K shards of a report, merged, are the unsharded report bit for bit.
func TestPowerShardsMergeToTheReport(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("CORPUS=true")
	}
	cases := []string{"Life:2", "OffDef:2", "CombinedDPS:2", "FullDPS:2"}
	buildNames := append([]string{"0018-berserker-heavy-strike", "0040-trickster-exsanguinate-of-transmission", "0019-champion-herald-of-agony"},
		powerBenchBuilds...)
	if v := os.Getenv("POWER_SHARD_CASES"); v != "" {
		cases = strings.Fields(v)
	}
	if v := os.Getenv("POWER_SHARD_BUILDS"); v != "" {
		buildNames = strings.Fields(v)
	}
	for _, name := range buildNames {
		for _, c := range cases {
			t.Run(name+"/"+c, func(t *testing.T) {
				t.Parallel()
				stat, depthStr, _ := strings.Cut(c, ":")
				depth, err := strconv.Atoi(depthStr)
				if err != nil {
					t.Fatalf("bad case %q", c)
				}
				run := func(index, count int) PowerShardRows {
					b := GetMiscCalculator(parseCorpusBuild(t, name+".xml")).NewPowerBuilder(powerOracleStat(stat), depth)
					var owned []int64
					if count > 0 {
						owned = b.Shard(index, count)
					}
					for b.Progress() < 1 {
						b.Step(time.Second)
					}
					return PowerShardRows{Rows: b.Report(), Owned: owned, Ascendancy: b.AscendancyPower()}
				}
				single := run(0, 0)
				want := powerShardRowLines(single.Rows)
				wantAsc := powerShardRowLines(single.Ascendancy)
				if len(wantAsc) == 0 {
					t.Fatal("REFUSING: no ascendancy node")
				}
				if len(want) < 100 {
					t.Fatalf("REFUSING: %d rows", len(want))
				}
				for k := 1; k <= 4; k++ {
					shards := make([]PowerShardRows, k)
					for i := range k {
						shards[i] = run(i, k)
						if k > 1 && len(shards[i].Owned) == 0 {
							t.Fatalf("K=%d: shard %d owns no node", k, i)
						}
					}
					if got := powerShardRowLines(MergePowerAscendancy(shards)); !slices.Equal(got, wantAsc) {
						t.Errorf("K=%d: ascendancy entries differ (%d merged, %d in the report)", k, len(got), len(wantAsc))
					}
					got := powerShardRowLines(MergePowerReports(shards))
					var bad []string
					for i := range max(len(want), len(got)) {
						var w, g string
						if i < len(want) {
							w = want[i]
						}
						if i < len(got) {
							g = got[i]
						}
						if w != g {
							bad = append(bad, fmt.Sprintf("line %d:\n  want %s\n  got  %s", i+1, w, g))
						}
					}
					if len(bad) > 0 {
						t.Errorf("K=%d: %d of %d rows differ:\n%s", k, len(bad), len(want), strings.Join(bad[:min(5, len(bad))], "\n"))
					}
				}
			})
		}
	}
}
