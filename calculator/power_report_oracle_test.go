package calculator

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Vilsol/go-pob/data"
)

var powerOracleCases = []struct {
	build string
	stats []string
}{
	{"0280-elementalist-fireball", []string{"Life:0", "CombinedDPS:0", "OffDef:0", "FullDPS:0", "FullDPS:3"}},
	{"0179-assassin-cold-snap", []string{"Life:0", "FullDPS:0", "OffDef:2"}},
	{"0651-elementalist-reap-5", []string{"Life:0", "FullDPS:0", "OffDef:3"}},
	{"0215-slayer-cyclone-of-tumult", []string{"Life:3", "FullDPS:0", "OffDef:0", "CombinedDPS:0"}},
	{"0369-necromancer-automation", []string{"Life:0", "FullDPS:0", "OffDef:3"}},
	{"0021-assassin-blade-flurry", []string{"FullDPS:2"}},
	{"0036-elementalist-incinerate-of-venting", []string{"Life:2"}},
	{"0378-juggernaut-static-strike-of-gathering-lightning-2", []string{"Life:1"}},
}

type oracleNode struct {
	typ                            string
	alloc, granted                 bool
	distance                       string
	single, path, offence, defence string
	depends                        int
	pathIDs                        string
}

type oracleEffect struct{ single, path, offence, defence string }

type powerOracle struct {
	nodes   map[int64]oracleNode
	effects map[[2]int64]oracleEffect
	cluster map[int64]string
	max     [5]string
	report  [][]string
}

func readPowerOracle(t *testing.T, file string) *powerOracle {
	t.Helper()
	f, err := os.Open(file)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	o := &powerOracle{nodes: map[int64]oracleNode{}, effects: map[[2]int64]oracleEffect{}, cluster: map[int64]string{}}
	scanner := bufio.NewScanner(gz)
	scanner.Buffer(make([]byte, 1<<20), 1<<20)
	for scanner.Scan() {
		c := strings.Split(scanner.Text(), "\t")
		id := func(s string) int64 {
			v, err := strconv.ParseInt(s, 10, 64)
			if err != nil {
				t.Fatalf("%s: bad id %q", file, s)
			}
			return v
		}
		switch {
		case c[0] == "N" && len(c) == 12:
			dep, _ := strconv.Atoi(c[10])
			o.nodes[id(c[1])] = oracleNode{c[2], c[3] == "true", c[4] == "true", c[5], c[6], c[7], c[8], c[9], dep, c[11]}
		case c[0] == "E" && len(c) == 7:
			o.effects[[2]int64{id(c[1]), id(c[2])}] = oracleEffect{c[3], c[4], c[5], c[6]}
		case c[0] == "C" && len(c) == 4:
			o.cluster[id(c[1])] = c[3]
		case c[0] == "X" && len(c) == 6:
			copy(o.max[:], c[1:])
		case c[0] == "R" && len(c) == 8:
			o.report = append(o.report, c[1:])
		default:
			t.Fatalf("%s: bad line %q", file, scanner.Text())
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	return o
}

func oracleNum(s string) (float64, bool) {
	if s == "nil" {
		return 0, false
	}
	v, err := strconv.ParseFloat(s, 64)
	return v, err == nil
}

func powerOracleStat(name string) PowerStat {
	for _, s := range NodePowerStats() {
		if (name == "OffDef" && s.Combined) || (s.Stat == name && !s.Combined) {
			return s
		}
	}
	panic("no power stat " + name)
}

func powerOracleCensus(b *PowerBuilder, o *powerOracle) (map[string][]string, int) {
	classes := map[string][]string{}
	add := func(class, format string, args ...any) {
		classes[class] = append(classes[class], fmt.Sprintf(format, args...))
	}
	near := func(a, b float64) bool { return math.Abs(a-b) <= 1e-6*math.Max(1, math.Abs(b)) }
	unparsed := func(id int64) bool {
		n := b.treeNode(id)
		if isMasteryNode(n) {
			return slices.ContainsFunc(n.MasteryEffects, func(e data.MasteryEffect) bool { return statsModKey(e.Stats) == "" })
		}
		return statsModKey(n.Stats) == ""
	}
	samePath := func(id int64, want string) bool {
		path := allocationPath(b.paths, id)
		slices.Sort(path)
		ids := make([]string, 0, len(path))
		for _, p := range path {
			ids = append(ids, strconv.FormatInt(p, 10))
		}
		return strings.Join(ids, ",") == want
	}
	graded := 0

	for id, w := range o.nodes {
		single, hasSingle := oracleNum(w.single)
		p := b.power[id]
		var gotSingle bool
		if p != nil {
			gotSingle = p.hasSingle
		}
		if hasSingle != gotSingle {
			cls := "missing"
			if !hasSingle {
				cls = "extra"
			}
			if unparsed(id) {
				cls = "parser"
			}
			if n, ok := b.tree.Nodes[strconv.FormatInt(id, 10)]; ok && n.Group == nil {
				cls = "data-groupless"
			}
			if id >= 1<<16 {
				cls = "cluster-subgraph"
			}
			add(cls, "%d %s single PoB %v ours %v", id, w.typ, hasSingle, gotSingle)
			continue
		}
		if !hasSingle || isMasteryNode(b.treeNode(id)) {
			continue
		}
		graded++
		if b.stat.Combined {
			off, _ := oracleNum(w.offence)
			def, _ := oracleNum(w.defence)
			if !near(p.single, off) || !near(p.defence, def) {
				add(valueClass(unparsed(id), "value"), "%d %s offence/defence %g / %g, PoB %g / %g", id, w.typ, p.single, p.defence, off, def)
			}
			continue
		}
		if !near(p.single, single) {
			add(valueClass(unparsed(id), "value"), "%d %s single %g, PoB %g (ratio %g)", id, w.typ, p.single, single, p.single/single)
			continue
		}
		path, hasPath := oracleNum(w.path)
		if hasPath != p.hasPath {
			add("path-set", "%d %s path PoB %v ours %v", id, w.typ, hasPath, p.hasPath)
			continue
		}
		if hasPath && (w.alloc || samePath(id, w.pathIDs)) && !near(p.path, path) {
			cls := valueClass(unparsed(id), "path-value")
			if w.alloc && w.depends != len(b.depends(id)) {
				cls = "depends"
			}
			add(cls, "%d %s path %g, PoB %g (depends %d, PoB %d)", id, w.typ, p.path, path, len(b.depends(id)), w.depends)
		}
	}
	for id, p := range b.power {
		if _, ok := o.nodes[id]; !ok && p.hasSingle {
			add("extra", "%d single ours %g, PoB none", id, p.single)
		}
	}
	for k, w := range o.effects {
		ep := b.mastery[k[0]][k[1]]
		single, hasSingle := oracleNum(w.single)
		if ep == nil {
			if hasSingle || b.stat.Combined {
				add(valueClass(unparsed(k[0]), "missing"), "mastery %d effect %d", k[0], k[1])
			}
			continue
		}
		graded++
		if b.stat.Combined {
			off, hasOff := oracleNum(w.offence)
			def, _ := oracleNum(w.defence)
			if hasOff && (!near(ep.single, off) || !near(ep.defence, def)) {
				add(valueClass(unparsed(k[0]), "value"), "mastery %d effect %d offence/defence %g / %g, PoB %g / %g", k[0], k[1], ep.single, ep.defence, off, def)
			}
			continue
		}
		if !near(ep.single, single) {
			add(valueClass(unparsed(k[0]), "value"), "mastery %d effect %d single %g, PoB %g", k[0], k[1], ep.single, single)
		}
	}
	for id, effects := range b.mastery {
		for e := range effects {
			if _, ok := o.effects[[2]int64{id, e}]; !ok {
				add("extra", "mastery %d effect %d", id, e)
			}
		}
	}
	for _, id := range b.clusterNodes {
		w, ok := o.cluster[id]
		v, has := b.cluster[id]
		if !ok {
			if has {
				cls := "extra"
				if id == 5029 {
					cls = "data-groupless"
				}
				add(cls, "cluster %d", id)
			}
			continue
		}
		want, _ := oracleNum(w)
		if !has {
			add("missing", "cluster %d", id)
		} else if !near(v, want) {
			add("value", "cluster %d %g, PoB %g", id, v, want)
		}
		graded++
	}
	for id := range o.cluster {
		if !slices.Contains(b.clusterNodes, id) {
			add("missing", "cluster %d not listed", id)
		}
	}

	report := b.Report()
	if !b.stat.Combined && len(o.report) > 0 {
		last, _ := oracleNum(o.report[min(20, len(o.report))-1][3])
		above := func(power float64) bool { return power > last && !near(power, last) }
		var want, got []string
		for _, r := range o.report {
			if v, _ := oracleNum(r[3]); above(v) {
				want = append(want, r[0])
			}
		}
		for _, r := range report {
			if above(r.Power) {
				got = append(got, strconv.FormatInt(r.ID, 10))
			}
		}
		sort.Strings(want)
		sort.Strings(got)
		if !slices.Equal(got, want) {
			add("top20", "ours %v, PoB %v", got, want)
		}
	}

	maxWant, _ := oracleNum(o.max[0])
	if b.stat.Combined {
		maxWant, _ = oracleNum(o.max[1])
	}
	values := map[int64]float64{}
	var maxGot float64
	for _, r := range append(slices.Clone(report), b.AscendancyPower()...) {
		if r.Allocated || r.Cluster {
			continue
		}
		v := r.Power
		if b.stat.Combined {
			v = r.Offence
		}
		if old, ok := values[r.ID]; !ok || v > old {
			values[r.ID] = v
		}
		n := b.treeNode(r.ID)
		if r.PathDist < 1000 && (n.AscendancyName == nil || *n.AscendancyName == "") {
			maxGot = max(maxGot, v)
		}
	}
	if !near(maxGot, maxWant) {
		add("heat-max", "largest %g, PoB %g", maxGot, maxWant)
	}
	channel := func(v, m float64) float64 {
		if m <= 0 {
			return 0
		}
		return min(1, math.Sqrt(max(v, 0)/m*1.5))
	}
	for id, w := range o.nodes {
		if w.alloc || w.typ == "ClassStart" || w.typ == "AscendClassStart" {
			continue
		}
		v, _ := oracleNum(w.single)
		if b.stat.Combined {
			v, _ = oracleNum(w.offence)
		}
		want, got := channel(v, maxWant), channel(values[id], maxGot)
		if math.Abs(want-got) > 1e-6 {
			cls := "heat-colour"
			if n := b.treeNode(id); n.AscendancyName != nil && *n.AscendancyName != "" {
				cls = "heat-ascendancy"
				if b.power[id] == nil {
					cls = "heat-groupless"
				}
			}
			if id >= 1<<16 {
				cls = "cluster-subgraph"
			}
			add(cls, "%d %s channel %.4f, PoB %.4f", id, w.typ, got, want)
		}
	}
	return classes, graded
}

func valueClass(unparsed bool, class string) string {
	if unparsed {
		return "parser"
	}
	return class
}

// TestPowerReportMatchesPoBOracle grades the report against PoB's PowerBuilder on the benchmark builds, by class; every class has a ceiling that only goes down.
func TestPowerReportMatchesPoBOracle(t *testing.T) {
	ceilings := map[string]int{
		"0021-assassin-blade-flurry/FullDPS:2/data-groupless":                          1,
		"0021-assassin-blade-flurry/FullDPS:2/parser":                                  1,
		"0021-assassin-blade-flurry/FullDPS:2/path-value":                              6,
		"0021-assassin-blade-flurry/FullDPS:2/top20":                                   1,
		"0021-assassin-blade-flurry/FullDPS:2/value":                                   1,
		"0036-elementalist-incinerate-of-venting/Life:2/data-groupless":                1,
		"0036-elementalist-incinerate-of-venting/Life:2/path-value":                    6,
		"0179-assassin-cold-snap/FullDPS:0/data-groupless":                             7,
		"0179-assassin-cold-snap/FullDPS:0/heat-groupless":                             1,
		"0179-assassin-cold-snap/FullDPS:0/heat-colour":                                1,
		"0179-assassin-cold-snap/FullDPS:0/parser":                                     1,
		"0179-assassin-cold-snap/Life:0/data-groupless":                                7,
		"0179-assassin-cold-snap/Life:0/parser":                                        1,
		"0215-slayer-cyclone-of-tumult/CombinedDPS:0/data-groupless":                   7,
		"0215-slayer-cyclone-of-tumult/CombinedDPS:0/heat-groupless":                   1,
		"0215-slayer-cyclone-of-tumult/CombinedDPS:0/parser":                           1,
		"0215-slayer-cyclone-of-tumult/FullDPS:0/data-groupless":                       7,
		"0215-slayer-cyclone-of-tumult/FullDPS:0/heat-groupless":                       1,
		"0215-slayer-cyclone-of-tumult/FullDPS:0/heat-colour":                          4,
		"0215-slayer-cyclone-of-tumult/FullDPS:0/parser":                               5,
		"0215-slayer-cyclone-of-tumult/Life:3/data-groupless":                          1,
		"0215-slayer-cyclone-of-tumult/OffDef:0/data-groupless":                        6,
		"0215-slayer-cyclone-of-tumult/OffDef:0/heat-groupless":                        1,
		"0215-slayer-cyclone-of-tumult/OffDef:0/parser":                                1,
		"0280-elementalist-fireball/CombinedDPS:0/data-groupless":                      7,
		"0280-elementalist-fireball/CombinedDPS:0/heat-colour":                         1,
		"0280-elementalist-fireball/CombinedDPS:0/parser":                              1,
		"0280-elementalist-fireball/FullDPS:0/data-groupless":                          7,
		"0280-elementalist-fireball/FullDPS:0/heat-colour":                             1,
		"0280-elementalist-fireball/FullDPS:0/parser":                                  1,
		"0280-elementalist-fireball/FullDPS:3/data-groupless":                          1,
		"0280-elementalist-fireball/Life:0/data-groupless":                             7,
		"0280-elementalist-fireball/Life:0/parser":                                     1,
		"0280-elementalist-fireball/OffDef:0/data-groupless":                           6,
		"0280-elementalist-fireball/OffDef:0/heat-colour":                              1,
		"0280-elementalist-fireball/OffDef:0/parser":                                   1,
		"0378-juggernaut-static-strike-of-gathering-lightning-2/Life:1/data-groupless": 1,
		"0378-juggernaut-static-strike-of-gathering-lightning-2/Life:1/path-value":     4,
		"0369-necromancer-automation/FullDPS:0/data-groupless":                         7,
		"0369-necromancer-automation/FullDPS:0/parser":                                 1,
		"0369-necromancer-automation/Life:0/data-groupless":                            7,
		"0369-necromancer-automation/Life:0/parser":                                    1,
		"0651-elementalist-reap-5/FullDPS:0/data-groupless":                            7,
		"0651-elementalist-reap-5/FullDPS:0/heat-groupless":                            2,
		"0651-elementalist-reap-5/FullDPS:0/heat-colour":                               1,
		"0651-elementalist-reap-5/FullDPS:0/parser":                                    1,
		"0651-elementalist-reap-5/Life:0/data-groupless":                               7,
		"0651-elementalist-reap-5/Life:0/parser":                                       1,
	}
	for _, tc := range powerOracleCases {
		for _, sd := range tc.stats {
			t.Run(tc.build+"/"+sd, func(t *testing.T) {
				if sd == "FullDPS:0" && os.Getenv("POWER_ORACLE_FULL") == "" {
					t.Skip("POWER_ORACLE_FULL=1 runs the Full DPS reports over the whole tree")
				}
				t.Parallel()
				name, depthStr, _ := strings.Cut(sd, ":")
				depth, _ := strconv.Atoi(depthStr)
				o := readPowerOracle(t, filepath.Join("testdata", "power_oracle", fmt.Sprintf("%s_%s_%d.tsv.gz", tc.build, name, depth)))
				if len(o.nodes) < 100 {
					t.Fatalf("REFUSING: %d PoB nodes", len(o.nodes))
				}
				b := GetMiscCalculator(parseCorpusBuild(t, tc.build+".xml")).NewPowerBuilder(powerOracleStat(name), depth)
				start := time.Now()
				for b.Progress() < 1 {
					b.Step(time.Second)
				}
				classes, graded := powerOracleCensus(b, o)
				t.Logf("%d tasks in %s, %d graded", len(b.tasks), time.Since(start).Round(time.Millisecond), graded)
				names := make([]string, 0, len(classes))
				for c := range classes {
					names = append(names, c)
				}
				sort.Strings(names)
				counts := make([]string, 0, len(names))
				for _, c := range names {
					counts = append(counts, fmt.Sprintf("%q: %d,", tc.build+"/"+sd+"/"+c, len(classes[c])))
				}
				t.Logf("census %s", strings.Join(counts, " "))
				for _, c := range names {
					rows := classes[c]
					sort.Strings(rows)
					t.Logf("%s: %d", c, len(rows))
					shown := rows[:min(8, len(rows))]
					if os.Getenv("POWER_ORACLE_VERBOSE") != "" {
						shown = rows
					}
					for _, r := range shown {
						t.Logf("  %s", r)
					}
					if ceiling := ceilings[tc.build+"/"+sd+"/"+c]; len(rows) > ceiling {
						t.Errorf("%s: %d over ceiling %d", c, len(rows), ceiling)
					} else if len(rows) < ceiling {
						t.Logf("%s: %d under ceiling %d, lower it", c, len(rows), ceiling)
					}
				}
			})
		}
	}
}
