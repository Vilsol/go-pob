package calculator

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Vilsol/go-pob/builds"
)

// TestPowerReportMatchesPoB: power_report_<build>_<stat>.tsv is PoB's own PowerBuilder + BuildPowerReportList run headless on the build.
func TestPowerReportMatchesPoB(t *testing.T) {
	t.Parallel()
	const file, stat = "0280-elementalist-fireball.xml", "Life"
	f, err := os.Open("testdata/power_report_0280_Life.tsv")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	type row struct {
		power, pathPower float64
		pathDist         int
		allocated        bool
		typ              string
		path             string
	}
	want := map[string]row{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		c := strings.Split(scanner.Text(), "\t")
		if len(c) != 8 {
			t.Fatalf("bad row %q", scanner.Text())
		}
		power, _ := strconv.ParseFloat(c[3], 64)
		pathPower, _ := strconv.ParseFloat(c[4], 64)
		dist, _ := strconv.Atoi(c[5])
		want[c[0]+"\t"+c[2]] = row{power, pathPower, dist, c[6] == "true", c[1], c[7]}
	}
	if len(want) < 1000 {
		t.Fatalf("REFUSING: %d expected rows", len(want))
	}

	build := parseCorpusBuild(t, file)
	m := GetMiscCalculator(build)
	var ps PowerStat
	for _, s := range PowerStats() {
		if s.Stat == stat {
			ps = s
		}
	}
	b := m.NewPowerBuilder(ps, 0)
	start := time.Now()
	for b.Progress() < 1 {
		b.Step(time.Second)
	}
	t.Logf("%d calculations in %s", len(b.tasks), time.Since(start).Round(time.Millisecond))

	near := func(a, b float64) bool { return math.Abs(a-b) <= 1e-6*math.Max(1, math.Abs(b)) }
	ceilings := map[string]int{"parser": 51, "path": 7, "depends": 1}
	classes := map[string][]string{}
	tree := b.tree
	unparsed := func(id string) bool {
		n, ok := tree.Nodes[id]
		if !ok {
			return false
		}
		if isMasteryNode(n) {
			for _, e := range n.MasteryEffects {
				if statsModKey(e.Stats) == "" {
					return true
				}
			}
			return false
		}
		return statsModKey(n.Stats) == ""
	}
	got := map[string]bool{}
	var bad []string
	nonzero, graded, tied := 0, 0, 0
	for _, e := range b.Report() {
		if e.Cluster {
			continue
		}
		key := strconv.FormatInt(e.ID, 10) + "\t" + e.Name
		got[key] = true
		w, ok := want[key]
		if !ok {
			bad = append(bad, "extra "+key)
			continue
		}
		if w.power != 0 {
			nonzero++
		}
		samePath := e.Allocated
		if !e.Allocated {
			path := allocationPath(b.paths, e.ID)
			slices.Sort(path)
			ids := make([]string, 0, len(path))
			for _, id := range path {
				ids = append(ids, strconv.FormatInt(id, 10))
			}
			samePath = strings.Join(ids, ",") == w.path
		}
		if samePath {
			graded++
		} else {
			tied++
		}
		if near(e.Power, w.power) && (!samePath || near(e.PathPower, w.pathPower)) && e.PathDist == w.pathDist && e.Allocated == w.allocated && e.Type == w.typ {
			continue
		}
		id := strconv.FormatInt(e.ID, 10)
		switch {
		case e.PathDist == 1 && w.pathDist == 1000 && unparsed(id):
			classes["parser"] = append(classes["parser"], key)
		case !e.Allocated && e.PathDist+1 == w.pathDist && near(e.Power, w.power):
			classes["path"] = append(classes["path"], key)
		case e.Allocated && e.PathDist == w.pathDist+1 && near(e.Power, w.power):
			classes["depends"] = append(classes["depends"], key)
		default:
			bad = append(bad, fmt.Sprintf("%s: %s %v %v %d %v, PoB %s %v %v %d %v", key, e.Type, e.Power, e.PathPower, e.PathDist, e.Allocated,
				w.typ, w.power, w.pathPower, w.pathDist, w.allocated))
		}
	}
	for key := range want {
		if got[key] {
			continue
		}
		if unparsed(strings.SplitN(key, "\t", 2)[0]) {
			classes["parser"] = append(classes["parser"], "missing "+key)
		} else {
			bad = append(bad, "missing "+key)
		}
	}
	t.Logf("%d rows with power; per-point power graded on %d, %d took another equally short path", nonzero, graded, tied)
	for class, ceiling := range ceilings {
		t.Logf("%s: %d (ceiling %d)", class, len(classes[class]), ceiling)
		if len(classes[class]) > ceiling {
			t.Errorf("%s divergences rose to %d: %v", class, len(classes[class]), classes[class])
		}
	}
	if nonzero < 100 || graded < len(want)/2 {
		t.Errorf("vacuity: %d rows with power, %d graded", nonzero, graded)
	}
	if len(bad) > 0 {
		for _, l := range bad[:min(40, len(bad))] {
			t.Error(l)
		}
		t.Fatalf("%d of %d rows differ from PoB", len(bad), len(want))
	}
}

// BenchmarkPowerReport times the whole Life report (misc calculator.
func BenchmarkPowerReport(b *testing.B) {
	for _, file := range []string{"0280-elementalist-fireball.xml", "0001-necromancer-unknown.xml"} {
		b.Run(strings.SplitN(file, "-", 2)[0], func(b *testing.B) {
			raw, err := os.ReadFile(filepath.Join(corpusDirForLoadedData(b), file))
			if err != nil {
				b.Fatal(err)
			}
			var stat PowerStat
			for _, s := range PowerStats() {
				if s.Stat == "Life" {
					stat = s
				}
			}
			b.ReportAllocs()
			for b.Loop() {
				build, err := builds.ParseBuild(raw)
				if err != nil {
					b.Fatal(err)
				}
				pb := GetMiscCalculator(build).NewPowerBuilder(stat, 0)
				for pb.Progress() < 1 {
					pb.Step(time.Second)
				}
				if rows := len(pb.Report()); rows < 1000 {
					b.Fatalf("REFUSING: %d rows", rows)
				}
			}
		})
	}
}

// TestPowerReportReuseMatchesFresh: The report reuses the build's base mod DBs and cluster subgraphs across its calculations.
func TestPowerReportReuseMatchesFresh(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		file  string
		extra []int64
	}{
		{"0280-elementalist-fireball.xml", nil},
		{"0001-necromancer-unknown.xml", nil},
		{"0011-saboteur-cyclone.xml", nil},
		{"0012-occultist-unknown.xml", nil},
		{"0048-warden-frost-blades.xml", []int64{63727}},
	} {
		raw, err := os.ReadFile(filepath.Join(corpusDirForLoadedData(t), tc.file))
		if err != nil {
			t.Fatal(err)
		}
		build, err := builds.ParseBuild(raw)
		if err != nil {
			t.Fatal(err)
		}
		misc := GetMiscCalculator(build)
		var stat PowerStat
		for _, s := range PowerStats() {
			if s.Stat == "Life" {
				stat = s
			}
		}
		b := misc.NewPowerBuilder(stat, 0)
		var overrides []*CalcOverride
		for i, id := range b.nodes {
			if b.keys[id] == "" || b.granted[id] || i%6 != 0 {
				continue
			}
			if b.alloc[id] {
				overrides = append(overrides, &CalcOverride{RemoveNodes: []int64{id}})
			} else {
				overrides = append(overrides, &CalcOverride{AddNodes: []int64{id}})
			}
		}
		for _, id := range tc.extra {
			overrides = append(overrides, &CalcOverride{AddNodes: []int64{id}})
		}
		compared, differ := 0, 0
		for _, o := range overrides {
			reused := b.run(o)
			fresh := initEnvOnly(misc.build, misc.cache, OutputModeCalculator, o)
			PerformCalc(fresh)
			outputs := [][2]map[string]float64{{reused.Player.Output, fresh.Player.Output}}
			if reused.Minion != nil || fresh.Minion != nil {
				if reused.Minion == nil || fresh.Minion == nil {
					t.Fatalf("%s %+v: minion on one side only", tc.file, *o)
				}
				outputs = append(outputs, [2]map[string]float64{reused.Minion.Output, fresh.Minion.Output})
			}
			for _, pair := range outputs {
				compared++
				if len(pair[0]) != len(pair[1]) {
					differ++
					t.Errorf("%s %+v: %d outputs reused, %d fresh", tc.file, *o, len(pair[0]), len(pair[1]))
					continue
				}
				for k, v := range pair[1] {
					if r, ok := pair[0][k]; !ok || math.Float64bits(r) != math.Float64bits(v) {
						differ++
						t.Errorf("%s %+v: %s reused %v, fresh %v", tc.file, *o, k, r, v)
						break
					}
				}
			}
		}
		if compared < 50 {
			t.Fatalf("vacuity: %s compared %d outputs", tc.file, compared)
		}
		if differ > 0 {
			t.Fatalf("%s: %d of %d outputs differ", tc.file, differ, compared)
		}
	}
}
