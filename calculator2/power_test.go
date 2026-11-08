package calculator2

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/data/tree2"
)

// powerRow is one row of testdata/power_pob.jsonl.gz, which testdata/power_oracle.lua writes from PoB-PoE2's power
// builder: per build and stat, each calculated node's [id, distance, singleStat, pathPower, offence, defence] (nil as
// null), the maxima and the report rows with a power, as [id, name, power, powerStr, pathPower, pathPowerStr,
// allocated, pathDist].
type powerRow struct {
	Build      string    `json:"build"`
	Stat       string    `json:"stat"`
	Depth      int       `json:"depth"`
	Max        []float64 `json:"max"`
	Nodes      [][]any   `json:"nodes"`
	ReportRows int       `json:"reportRows"`
	Report     [][]any   `json:"report"`
}

func readPowerRows(t *testing.T) []powerRow {
	t.Helper()
	f, err := os.Open(filepath.Join("testdata", "power_pob.jsonl.gz"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	var rows []powerRow
	sc := bufio.NewScanner(zr)
	sc.Buffer(nil, 1<<26)
	for sc.Scan() {
		var r powerRow
		if err := json.Unmarshal(sc.Bytes(), &r); err != nil {
			t.Fatal(err)
		}
		rows = append(rows, r)
	}
	return rows
}

func powerStat(name string) *PowerStat {
	for i := range PowerStatList {
		if s := &PowerStatList[i]; (name == "OffDef" && s.CombinedOffDef) || s.Stat == name {
			return s
		}
	}
	return nil
}

func near(a, b float64) bool { return math.Abs(a-b) <= 1e-6*math.Max(1, math.Abs(b)) }

// field compares one of our node's fields with PoB-PoE2's (nil when PoB leaves it unset).
func powerField(want any, got float64, has bool) bool {
	if want == nil {
		return !has
	}
	w, ok := want.(float64)
	return ok && has && near(got, w)
}

func TestPowerBuilderMatchesPoB(t *testing.T) {
	misc := map[string]*MiscCalculator{}
	nodes, reports, ties := 0, 0, 0
	for _, r := range readPowerRows(t) {
		m := misc[r.Build]
		if m == nil {
			m = nodeCompareBuild(t, r.Build)
			misc[r.Build] = m
		}
		stat := powerStat(r.Stat)
		if stat == nil {
			t.Fatalf("no power stat %s", r.Stat)
		}
		b := NewPowerBuilder(m, stat, r.Depth)
		for !b.Step(100) {
		}
		label := r.Build + " " + r.Stat
		want := map[int]bool{}
		tiePath := map[int]float64{}
		for _, n := range r.Nodes {
			id := int(n[0].(float64))
			want[id] = true
			nodes++
			p := b.Power[id]
			if p == nil {
				t.Errorf("%s: node %d has no power", label, id)
				continue
			}
			pathPower := p.PathPower
			if node := m.main.Spec.Nodes[id]; p.HasPath && n[3] != nil {
				path := node.Path
				if node.Alloc {
					path = node.Depends
				}
				ours := make([]string, 0, len(path))
				for _, pn := range path {
					ours = append(ours, strconv.Itoa(pn.ID))
				}
				slices.SortFunc(ours, func(a, b string) int { x, _ := strconv.Atoi(a); y, _ := strconv.Atoi(b); return x - y })
				pobPath := strings.Split(n[6].(string), ",")
				if strings.Join(ours, ",") != n[6].(string) && len(ours) == len(pobPath) && len(ours) > 1 {
					// an equal-length route PoB-PoE2 picked in pairs() order: grade the path power over its route
					ties++
					set := map[*tree2.SpecNode]bool{}
					for _, sid := range pobPath {
						pid, _ := strconv.Atoi(sid)
						set[m.main.Spec.Nodes[pid]] = true
					}
					o := NodeOverride{AddNodes: set}
					if node.Alloc {
						o = NodeOverride{RemoveNodes: set}
					}
					pathPower = calculatePowerStat(stat, powerOutput(m.CalcNodes(o)), b.base)
				}
			}
			tiePath[id] = pathPower
			ok := near(float64(p.Distance), n[1].(float64)) && powerField(n[2], p.SingleStat, p.HasSingle) &&
				powerField(n[3], pathPower, p.HasPath) && powerField(n[4], p.Offence, p.HasOffDef) && powerField(n[5], p.Defence, p.HasOffDef)
			if !ok {
				t.Errorf("%s: node %d power %+v (path power %v), PoB-PoE2 %v", label, id, *p, pathPower, n)
			}
		}
		for id, p := range b.Power {
			if (p.HasSingle || p.HasOffDef) && !want[id] {
				t.Errorf("%s: node %d has a power PoB-PoE2 does not calculate: %+v", label, id, *p)
			}
		}
		gotMax := []float64{b.Max.SingleStat, b.Max.Offence, b.Max.OffencePerPoint, b.Max.Defence, b.Max.DefencePerPoint}
		for i := range gotMax {
			if !near(gotMax[i], r.Max[i]) {
				t.Errorf("%s: power max %v, PoB-PoE2 %v", label, gotMax, r.Max)
				break
			}
		}
		// the stats graded here have no display scale, so the report's power is the node's
		rows := b.PowerReportList()
		if len(rows) != r.ReportRows {
			t.Errorf("%s: %d report rows, PoB-PoE2 %d", label, len(rows), r.ReportRows)
		}
		byID := map[int]PowerReportRow{}
		for _, row := range rows {
			if pp, ok := tiePath[row.ID]; ok {
				row.PathPower = pp / float64(row.PathDist)
			}
			byID[row.ID] = row
		}
		withPower := 0
		for _, row := range byID {
			if row.Power != 0 || row.PathPower != 0 {
				withPower++
			}
		}
		if withPower != len(r.Report) {
			t.Errorf("%s: %d report rows with a power, PoB-PoE2 %d", label, withPower, len(r.Report))
		}
		for _, w := range r.Report {
			reports++
			id := int(w[0].(float64))
			g, ok := byID[id]
			if !ok || g.Name != w[1].(string) || !near(g.Power, w[2].(float64)) || !near(g.PathPower, w[4].(float64)) ||
				g.Allocated != w[6].(bool) || w[7].(string) != strconv.Itoa(g.PathDist) {
				t.Errorf("%s: report row %+v, PoB-PoE2 %v", label, g, w)
			}
		}
	}
	t.Logf("%d node powers (%d over PoB-PoE2's equal-length route), %d report rows graded", nodes, ties, reports)
	if nodes == 0 || reports == 0 {
		t.Fatal("nothing graded")
	}
}
