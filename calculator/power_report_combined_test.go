package calculator

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestCombinedOffDefStatMatchesTheLua: CalcsTab:CalculateCombinedOffDefStat.
func TestCombinedOffDefStatMatchesTheLua(t *testing.T) {
	env := func(out map[string]float64, minion float64, withMinion bool) *Environment {
		e := &Environment{Player: &Actor{Output: out}}
		if withMinion {
			e.Minion = &ActorMinion{Actor: &Actor{Output: map[string]float64{"CombinedDPS": minion}}}
		}
		return e
	}
	o := map[string]float64{"LifeUnreserved": 5200, "Life": 5000, "Armour": 12000, "EnergyShield": 400, "Evasion": 3000, "LifeRegenRecovery": 300, "EnergyShieldRegenRecovery": 50, "CombinedDPS": 120000}
	m := map[string]float64{"LifeUnreserved": 5000, "Life": 5000, "Armour": 11000, "EnergyShield": 300, "Evasion": 3500, "LifeRegenRecovery": 250, "EnergyShieldRegenRecovery": 40, "CombinedDPS": 100000}
	off, def := combinedOffDefStat(env(o, 1000, true), env(m, 900, true))
	if off != 0.19920713577799801 || def != 0.22424242424242424 {
		t.Errorf("with a minion: %.17g %.17g, the Lua 0.19920713577799801 0.22424242424242424", off, def)
	}
	o["EnergyShieldRecoveryCap"], m["EnergyShieldRecoveryCap"] = 350, 200
	off, def = combinedOffDefStat(env(o, 0, false), env(m, 0, false))
	if off != 0.20000000000000001 || def != 0.24090909090909093 {
		t.Errorf("with the cap: %.17g %.17g, the Lua 0.20000000000000001 0.24090909090909093", off, def)
	}
}

// TestPowerReportCombinedMatchesPoB: power_combined_0280_2.tsv is PoB's PowerBuilder under Offence/Defence on the build.
func TestPowerReportCombinedMatchesPoB(t *testing.T) {
	raw, err := os.ReadFile("testdata/power_combined_0280_2.tsv")
	if err != nil {
		t.Fatal(err)
	}
	want := map[int64][2]float64{}
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		c := strings.Split(line, "\t")
		id, _ := strconv.ParseInt(c[0], 10, 64)
		off, _ := strconv.ParseFloat(c[1], 64)
		def, _ := strconv.ParseFloat(c[2], 64)
		want[id] = [2]float64{off, def}
	}
	if len(want) < 100 {
		t.Fatalf("REFUSING: %d expected rows", len(want))
	}
	m := GetMiscCalculator(parseCorpusBuild(t, "0280-elementalist-fireball.xml"))
	b := m.NewPowerBuilder(NodePowerStats()[0], 2)
	for b.Progress() < 1 {
		b.Step(time.Second)
	}
	near := func(a, b float64) bool { return math.Abs(a-b) <= 1e-6*math.Max(1, math.Abs(b)) }
	graded, nonzero := 0, 0
	var bad []string
	for _, e := range b.Report() {
		w, ok := want[e.ID]
		if e.Allocated || e.Type == "Mastery" || !ok {
			continue
		}
		graded++
		if w[0] != 0 || w[1] != 0 {
			nonzero++
		}
		if !near(e.Offence, w[0]) || !near(e.Defence, w[1]) || e.Power != e.Offence {
			bad = append(bad, fmt.Sprintf("%d %s: %g / %g, PoB %g / %g", e.ID, e.Name, e.Offence, e.Defence, w[0], w[1]))
		}
	}
	if graded < 100 || nonzero < 20 {
		t.Fatalf("graded %d nodes, %d nonzero: no control", graded, nonzero)
	}
	const ceiling = 0
	if len(bad) > ceiling {
		t.Errorf("%d of %d nodes differ:\n%s", len(bad), graded, strings.Join(bad, "\n"))
	}
}
