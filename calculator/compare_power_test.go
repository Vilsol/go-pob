package calculator

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/pob"
)

func compareConfigDiffs(a, b *pob.PathOfBuilding) []pob.Input {
	find := func(p *pob.PathOfBuilding, name string) (pob.Input, bool) {
		for _, in := range p.Config.Inputs {
			if in.Name == name {
				return in, true
			}
		}
		return pob.Input{Name: name}, false
	}
	norm := func(in, other pob.Input) string {
		switch {
		case in.Boolean != nil:
			return strconv.FormatBool(*in.Boolean)
		case in.Number != nil:
			return strconv.FormatFloat(*in.Number, 'g', -1, 64)
		case in.String != nil:
			return "s:" + *in.String
		case other.Boolean != nil:
			return "false"
		case other.Number != nil:
			return "0"
		}
		return ""
	}
	var out []pob.Input
	for _, name := range configOrder {
		if _, ok := configurations[name]; !ok {
			if _, ok := configurationsWithBuild[name]; !ok {
				continue
			}
		}
		pa, _ := find(a, name)
		pb, _ := find(b, name)
		if norm(pa, pb) != norm(pb, pa) {
			out = append(out, pb)
		}
	}
	return out
}

func runComparePower(t *testing.T, a, b *pob.PathOfBuilding, stats []string) []ComparePowerEntry {
	t.Helper()
	cb := NewComparePowerBuilder(a, b, stats, ComparePowerCategories{TreeNodes: true, Items: true, SkillGems: true, SupportGems: true, Config: true},
		compareConfigDiffs(a, b))
	for cb.StepMs(1000) < 1 {
	}
	return cb.Report()
}

// TestComparePowerMatchesPoB: compare_power_<a>_<b>_<stat>.tsv is PoB's ComparePowerBuilder run headless.
func TestComparePowerMatchesPoB(t *testing.T) {
	t.Parallel()
	for _, pair := range [][2]string{
		{"0280-elementalist-fireball.xml", "0075-elementalist-winter-orb.xml"},
		{"0075-elementalist-winter-orb.xml", "0091-elementalist-winter-orb-2.xml"},
	} {
		t.Run(pair[0][:4]+"_"+pair[1][:4], func(t *testing.T) { compareToPoB(t, pair[0], pair[1]) })
	}
}

func compareToPoB(t *testing.T, fileA, fileB string) {
	stats := []string{"Life", "FullDPS", "TotalEHP"}
	decimals := map[string]int{"Life": 0, "FullDPS": 1, "TotalEHP": 0}

	a := parseCorpusBuild(t, fileA)
	b := parseCorpusBuild(t, fileB)
	before, err := builds.SerializeBuild(a)
	if err != nil {
		t.Fatal(err)
	}
	got := runComparePower(t, a, b, stats)
	after, err := builds.SerializeBuild(a)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("the report changed the primary build")
	}

	for si, stat := range stats {
		t.Run(stat, func(t *testing.T) {
			f, err := os.Open(fmt.Sprintf("testdata/compare_power_%s_%s_%s.tsv", fileA[:4], fileB[:4], stat))
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			var want []string
			var wantConfig []float64
			var excluded []string
			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				c := strings.Split(scanner.Text(), "\t")
				if len(c) != 3 {
					t.Fatalf("bad row %q", scanner.Text())
				}
				v, _ := strconv.ParseFloat(c[2], 64)
				switch {
				case c[0] == "Item" && strings.Contains(c[1], "Socket"):
					excluded = append(excluded, c[1])
				case c[0] == "Config":
					wantConfig = append(wantConfig, v)
				default:
					want = append(want, fmt.Sprintf("%s\t%s\t%.*f", c[0], c[1], decimals[stat], v))
				}
			}
			if len(want) < 10 {
				t.Fatalf("REFUSING: %d expected rows", len(want))
			}
			t.Logf("not ported, not compared: %q", excluded)

			var shown []string
			var shownConfig []float64
			for _, e := range got {
				v := e.Impact[si]
				text := strconv.FormatFloat(v, 'f', decimals[stat], 64)
				if z, _ := strconv.ParseFloat(text, 64); z == 0 {
					continue
				}
				if e.Category == "Config" {
					shownConfig = append(shownConfig, v)
					continue
				}
				shown = append(shown, fmt.Sprintf("%s\t%s\t%s", e.Category, e.Name, text))
			}
			slices.Sort(want)
			slices.Sort(shown)
			missing, extra := diffSorted(want, shown)
			for _, m := range missing {
				t.Errorf("PoB shows, Go does not: %s", m)
			}
			for _, x := range extra {
				t.Errorf("Go shows, PoB does not: %s", x)
			}
			slices.Sort(wantConfig)
			slices.Sort(shownConfig)
			if len(wantConfig) != len(shownConfig) {
				t.Errorf("config rows: PoB %v, Go %v", wantConfig, shownConfig)
			} else {
				for i := range wantConfig {
					if math.Abs(wantConfig[i]-shownConfig[i]) > 1e-6*math.Max(1, math.Abs(wantConfig[i])) {
						t.Errorf("config rows: PoB %v, Go %v", wantConfig, shownConfig)
						break
					}
				}
			}
		})
	}
}

func diffSorted(want, got []string) ([]string, []string) {
	var missing, extra []string
	i, j := 0, 0
	for i < len(want) || j < len(got) {
		switch {
		case j >= len(got) || (i < len(want) && want[i] < got[j]):
			missing = append(missing, want[i])
			i++
		case i >= len(want) || got[j] < want[i]:
			extra = append(extra, got[j])
			j++
		default:
			i++
			j++
		}
	}
	return missing, extra
}

// TestComparePowerOfItselfIsEmpty: A build compared with itself has nothing to report.
func TestComparePowerOfItselfIsEmpty(t *testing.T) {
	t.Parallel()
	a := parseCorpusBuild(t, "0280-elementalist-fireball.xml")
	b := parseCorpusBuild(t, "0280-elementalist-fireball.xml")
	if got := runComparePower(t, a, b, []string{"Life"}); len(got) != 0 {
		t.Fatalf("a build against itself reported %d rows: %+v", len(got), got)
	}
}
