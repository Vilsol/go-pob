package calculator

import (
	"bufio"
	"math"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestPowerReportClustersMatchPoB: power_report_<build>_<stat>_clusters.tsv is the cluster-notable rows of PoB's own report on the build.
func TestPowerReportClustersMatchPoB(t *testing.T) {
	t.Parallel()
	f, err := os.Open("testdata/power_report_0280_Life_clusters.tsv")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	want := map[string]float64{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		c := strings.Split(scanner.Text(), "\t")
		if len(c) != 4 {
			t.Fatalf("bad row %q", scanner.Text())
		}
		want[c[0]+"\t"+c[1]+"\t"+c[2]], _ = strconv.ParseFloat(c[3], 64)
	}
	if len(want) < 200 {
		t.Fatalf("REFUSING: %d expected rows", len(want))
	}

	var stat PowerStat
	for _, s := range PowerStats() {
		if s.Stat == "Life" {
			stat = s
		}
	}
	b := GetMiscCalculator(parseCorpusBuild(t, "0280-elementalist-fireball.xml")).NewPowerBuilder(stat, 0)
	for b.Progress() < 1 {
		b.Step(time.Second)
	}
	got, nonzero := 0, 0
	seen := map[string]bool{}
	for _, e := range b.Report() {
		if !e.Cluster {
			continue
		}
		key := strconv.FormatInt(e.ID, 10) + "\t" + e.Type + "\t" + e.Name
		w, ok := want[key]
		if !ok {
			if e.ID != 5029 {
				t.Errorf("extra %s", key)
			}
			continue
		}
		got++
		seen[key] = true
		if w != 0 {
			nonzero++
		}
		if math.Abs(e.Power-w) > 1e-6*math.Max(1, math.Abs(w)) {
			t.Errorf("%s: %v, PoB %v", key, e.Power, w)
		}
	}
	for key := range want {
		if !seen[key] {
			t.Errorf("missing %s", key)
		}
	}
	if got != len(want) || nonzero == 0 {
		t.Fatalf("%d cluster rows of PoB's %d (%d with power)", got, len(want), nonzero)
	}
}
