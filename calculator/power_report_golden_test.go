package calculator

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Vilsol/go-pob/builds"
)

var powerBenchBuilds = []string{
	"0280-elementalist-fireball", "0651-elementalist-reap-5", "0215-slayer-cyclone-of-tumult",
	"0179-assassin-cold-snap", "0369-necromancer-automation",
}

func powerGoldenRows(b *PowerBuilder) []string {
	f := func(v float64) string { return strconv.FormatUint(math.Float64bits(v), 16) }
	var out []string
	for _, r := range b.Report() {
		out = append(out, fmt.Sprintf("R\t%d\t%q\t%s\t%s\t%s\t%d\t%v\t%v\t%s\t%s\t%q", r.ID, r.Name, r.Type, f(r.Power), f(r.PathPower), r.PathDist,
			r.Allocated, r.Cluster, f(r.Offence), f(r.Defence), r.Stats))
	}
	ids := make([]int64, 0, len(b.power))
	for id := range b.power {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	for _, id := range ids {
		p := b.power[id]
		out = append(out, fmt.Sprintf("N\t%d\t%s\t%s\t%s\t%v\t%v\t%d", id, f(p.single), f(p.path), f(p.defence), p.hasSingle, p.hasPath, p.distance))
		effects := make([]int64, 0, len(b.mastery[id]))
		for e := range b.mastery[id] {
			effects = append(effects, e)
		}
		slices.Sort(effects)
		for _, e := range effects {
			ep := b.mastery[id][e]
			out = append(out, fmt.Sprintf("E\t%d\t%d\t%s\t%s\t%s\t%v\t%v", id, e, f(ep.single), f(ep.path), f(ep.defence), ep.hasSingle, ep.hasPath))
		}
	}
	return out
}

func powerGoldenCases() []string {
	var cases []string
	for _, s := range NodePowerStats() {
		name := s.Stat
		if s.Combined {
			name = "OffDef"
		}
		cases = append(cases, name+":2")
	}
	return append(cases, "FullDPS:0", "Life:0", "OffDef:0", "CombinedDPS:0", "TotalEHP:0", "MinionLife:0")
}

// TestPowerReportGolden writes (POWER_GOLDEN=write) or compares bit for bit.
func TestPowerReportGolden(t *testing.T) {
	mode, dir := os.Getenv("POWER_GOLDEN"), os.Getenv("POWER_GOLDEN_DIR")
	if mode == "" {
		t.Skip("POWER_GOLDEN=write|check with POWER_GOLDEN_DIR=<dir>")
	}
	if dir == "" || (mode != "write" && mode != "check") {
		t.Fatalf("REFUSING: POWER_GOLDEN=%q POWER_GOLDEN_DIR=%q", mode, dir)
	}
	cases, buildNames := powerGoldenCases(), powerBenchBuilds
	if v := os.Getenv("POWER_GOLDEN_CASES"); v != "" {
		cases = strings.Fields(v)
	}
	if v := os.Getenv("POWER_GOLDEN_BUILDS"); v != "" {
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
				b := GetMiscCalculator(parseCorpusBuild(t, name+".xml")).NewPowerBuilder(powerOracleStat(stat), depth)
				for b.Progress() < 1 {
					b.Step(time.Second)
				}
				rows := powerGoldenRows(b)
				file := filepath.Join(dir, name+"_"+strings.ReplaceAll(c, ":", "_")+".txt")
				if mode == "write" {
					if err := os.WriteFile(file, []byte(strings.Join(rows, "\n")+"\n"), 0o600); err != nil {
						t.Fatal(err)
					}
					return
				}
				raw, err := os.ReadFile(file)
				if err != nil {
					t.Fatal(err)
				}
				want := strings.Split(strings.TrimSuffix(string(raw), "\n"), "\n")
				if len(want) < 100 {
					t.Fatalf("REFUSING: %d golden rows", len(want))
				}
				var bad []string
				for i := range max(len(want), len(rows)) {
					var w, g string
					if i < len(want) {
						w = want[i]
					}
					if i < len(rows) {
						g = rows[i]
					}
					if w != g {
						bad = append(bad, fmt.Sprintf("line %d:\n  was %s\n  now %s", i+1, w, g))
					}
				}
				if len(bad) > 0 {
					t.Fatalf("%d of %d rows changed:\n%s", len(bad), len(want), strings.Join(bad[:min(10, len(bad))], "\n"))
				}
			})
		}
	}
}

// BenchmarkPowerReportBuilds times the report of POWER_BENCH_STATS.
func BenchmarkPowerReportBuilds(b *testing.B) {
	stats := []string{"FullDPS", "Life"}
	if v := os.Getenv("POWER_BENCH_STATS"); v != "" {
		stats = strings.Fields(v)
	}
	names := powerBenchBuilds
	if v := os.Getenv("POWER_BENCH_BUILDS"); v != "" {
		names = strings.Fields(v)
	}
	for _, name := range names {
		raw, err := os.ReadFile(filepath.Join(corpusDirForLoadedData(b), name+".xml"))
		if err != nil {
			b.Fatal(err)
		}
		for _, stat := range stats {
			b.Run(strings.SplitN(name, "-", 2)[0]+"/"+stat, func(b *testing.B) {
				var before, after runtime.MemStats
				calcs, tasks := 0, 0
				runtime.ReadMemStats(&before)
				for b.Loop() {
					build, err := builds.ParseBuild(raw)
					if err != nil {
						b.Fatal(err)
					}
					pb := GetMiscCalculator(build).NewPowerBuilder(powerOracleStat(stat), 0)
					for pb.Progress() < 1 {
						pb.Step(time.Second)
					}
					if rows := len(pb.Report()); rows < 1000 {
						b.Fatalf("REFUSING: %d rows", rows)
					}
					calcs, tasks = pb.calcs, len(pb.tasks)
				}
				runtime.ReadMemStats(&after)
				b.ReportMetric(float64(calcs), "whatifs")
				b.ReportMetric(float64(tasks), "tasks")
				b.ReportMetric(float64(after.NumGC-before.NumGC)/float64(b.N), "gcs/op")
				b.ReportMetric(float64(b.Elapsed().Milliseconds())/float64(b.N)/float64(max(calcs, 1)), "ms/whatif")
			})
		}
	}
}
