package exposition

import (
	"fmt"
	"maps"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/pob"
)

func sameStatOutputs(a, b StatOutputs) string {
	sameMap := func(what string, x, y map[string]float64) string {
		for _, k := range slices.Sorted(maps.Keys(y)) {
			v, ok := x[k]
			if !ok || !(v == y[k] || (math.IsNaN(v) && math.IsNaN(y[k]))) {
				return fmt.Sprintf("%s %s: %v (present %v), fresh %v", what, k, v, ok, y[k])
			}
		}
		if len(x) != len(y) {
			return fmt.Sprintf("%s: %d keys, fresh %d", what, len(x), len(y))
		}
		return ""
	}
	if d := sameMap("Output", a.Output, b.Output); d != "" {
		return d
	}
	if (a.Minion == nil) != (b.Minion == nil) {
		return "minion presence"
	}
	if d := sameMap("Minion", a.Minion, b.Minion); d != "" {
		return d
	}
	if !maps.Equal(a.SkillFlags, b.SkillFlags) {
		return "SkillFlags"
	}
	if !slices.Equal(a.GrantedPassives, b.GrantedPassives) {
		return "GrantedPassives"
	}
	return ""
}

// TestMiscCalculatorForMatchesFresh: The Tick's MAIN calculation stands in for the what-if base pass.
func TestMiscCalculatorForMatchesFresh(t *testing.T) {
	initItemData(t)
	files := []string{
		"0001-necromancer-unknown.xml", "0011-saboteur-cyclone.xml", "0012-occultist-unknown.xml",
		"0048-warden-frost-blades.xml", "0280-elementalist-fireball.xml", "0339-reliquarian-shield-crush-of-the-chieftain-2.xml",
	}
	ents, err := os.ReadDir(filepath.Join("..", "..", "testdata", "corpus-"+raw.LatestVersion))
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	for i, e := range ents {
		if i%997 == 500 {
			files = append(files, e.Name())
		}
	}
	compared := 0
	for _, file := range files {
		build := corpusBuild(t, file)
		groups := []int{build.Build.MainSocketGroup}
		if sets := build.Skills.SkillSets; len(sets) > 0 && len(sets[0].Skills) > 1 {
			groups = append(groups, 1+build.Build.MainSocketGroup%len(sets[0].Skills))
		}
		for _, g := range groups {
			build.SetMainSocketGroup(g)
			calc := calculator.NewCalculator(*build)
			calc.BuildOutput(calculator.OutputModeMain)
			reused := MiscCalculatorFor(build, calc)
			fresh := GetMiscCalculator(build)
			if d := sameStatOutputs(reused.Base(), fresh.Base()); d != "" {
				t.Errorf("%s group %d: base %s", file, g, d)
				continue
			}
			if len(build.Build.PassiveNodes) > 0 {
				drop := []int64{build.Build.PassiveNodes[0]}
				if d := sameStatOutputs(reused.CalcNodes(nil, drop), fresh.CalcNodes(nil, drop)); d != "" {
					t.Errorf("%s group %d: removing %d: %s", file, g, drop[0], d)
				}
			}
			if file == "0280-elementalist-fireball.xml" {
				report := func(m *MiscCalculator) []calculator.PowerReportEntry {
					p := NewPowerBuilder(m, "Life", 1)
					for p.Step(1000) < 1 {
					}
					return p.Report()
				}
				if a, b := report(reused), report(fresh); len(b) == 0 || !reflect.DeepEqual(a, b) {
					t.Errorf("%s group %d: power report differs (%d rows, fresh %d)", file, g, len(a), len(b))
				}
			}
			compared++
		}
	}
	if compared < len(files) {
		t.Fatalf("vacuity: %d comparisons over %d builds", compared, len(files))
	}
}

// TestMiscCalculatorForSeesAStaleEnvironment: The base is a CALCULATOR pass of the build as it stands.
func TestMiscCalculatorForSeesAStaleEnvironment(t *testing.T) {
	initItemData(t)
	build := corpusBuild(t, "0280-elementalist-fireball.xml")
	stale := calculator.NewCalculator(*build)
	if MiscCalculatorFor(build, stale) != nil {
		t.Fatal("a calculator that calculated nothing, yet a base")
	}
	main := stale.BuildOutput(calculator.OutputModeMain)
	if _, ok := main.Player.Output["TinctureLimit"]; !ok {
		t.Fatal("CONTROL FAILED: the MAIN pass carries no breakdown-only output to tell it from a CALCULATOR pass")
	}
	base := MiscCalculatorFor(build, stale).Base()
	if _, ok := base.Output["TinctureLimit"]; ok {
		t.Error("the misc base carries TinctureLimit, a MAIN-only output: it reuses the MAIN pass")
	}
	if d := sameStatOutputs(base, GetMiscCalculator(build).Base()); d != "" {
		t.Errorf("the misc base differs from a fresh CALCULATOR pass: %s", d)
	}
	edited := *build
	edited.Build.Level--
	if d := sameStatOutputs(MiscCalculatorFor(&edited, stale).Base(), GetMiscCalculator(&edited).Base()); d != "" {
		t.Errorf("after an edit the misc base still reads the stale calculation: %s", d)
	}
	var b *pob.PathOfBuilding
	if MiscCalculatorFor(b, stale) != nil {
		t.Fatal("no build, yet a calculator")
	}
}
