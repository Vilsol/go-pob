package calculator

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// TestDiagBreakdown diffs OUR per-slot defence breakdown against PoB's, slot by slot.
func TestDiagBreakdown(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	build := os.Getenv("BUILD")
	if build == "" {
		t.Skip("set BUILD=<name without .xml>")
	}
	stat := os.Getenv("STAT")
	if stat == "" {
		stat = "Armour"
	}
	requireGameData(t, "3.29")

	dir := dumpsDirForLoadedData(t)
	raw, err := os.ReadFile(filepath.Join(dir, build+".json"))
	if err != nil {
		t.Fatalf("REFUSING: no PoB dump for %s in %s (%v).\nWithout PoB's side there is nothing "+
			"to diff, and printing our slots alone reads exactly like agreement.", build, dir, err)
	}
	var d struct {
		Breakdown map[string]json.RawMessage `json:"breakdown"`
		Output    map[string]json.RawMessage `json:"output"`
	}
	if err := json.Unmarshal(raw, &d); err != nil {
		t.Fatalf("REFUSING: cannot parse %s: %v", build, err)
	}

	pob, err := pobBreakdownSlots(d.Breakdown[stat])
	if err != nil {
		t.Fatalf("REFUSING: PoB's breakdown for %q on %s is not the slot form this diffs (%v).\n"+
			"Only the per-slot defence stats (Armour, EnergyShield, Evasion, Ward) use it.", stat, build, err)
	}
	if len(pob) == 0 {
		t.Fatalf("REFUSING: PoB emits no breakdown slots for %q on %s. That is a measurement about "+
			"the STAT or the BUILD, not a clean diff — pick a stat PoB breaks down by slot.", stat, build)
	}

	env := corpusEnvForLoadedData(t, build+".xml")
	entry := env.Player.Breakdown.Data[stat]
	if entry == nil {
		t.Fatalf("REFUSING: we record no breakdown entry for %q on %s, while PoB records %d slot(s). "+
			"That is itself the finding — report it rather than reading this as agreement.",
			stat, build, len(pob))
	}

	fmt.Printf("\n=== BREAKDOWN DIFF: %s / %s ===\n", build, stat)
	var pobVal float64
	_ = json.Unmarshal(d.Output[stat], &pobVal)
	fmt.Printf("PoB %s = %.4g · ours = %.4g\n\n", stat, pobVal, env.Player.Output[stat])
	fmt.Printf("%-22s %-30s %s\n", "SLOT", "PoB", "OURS")

	ours := map[string]breakdownSlot{}
	for _, s := range entry.Slots {
		ours[s.Source] = breakdownSlot{
			base:  s.Base,
			inc:   parseMult(s.Inc),
			more:  parseMult(s.More),
			total: parseFloat(s.Total),
		}
	}

	seen := map[string]bool{}
	var differ int
	for _, name := range slotOrderOf(pob, entry) {
		p, hasP := pob[name]
		o, hasO := ours[name]
		seen[name] = true
		mark := "  "
		switch {
		case !hasP:
			mark, differ = "**", differ+1
		case !hasO:
			mark, differ = "**", differ+1
		case !nearly(p.base, o.base) || !nearly(p.inc, o.inc) || !nearly(p.more, o.more):
			mark, differ = "**", differ+1
		}
		fmt.Printf("%s %-20s %-30s %s\n", mark, name, fmtSlot(p, hasP), fmtSlot(o, hasO))
	}
	fmt.Printf("\n%d of %d slot(s) differ.\n", differ, len(seen))
	fmt.Printf("READ IT: a differing BASE is a missing or extra ITEM/source. A differing INC or MORE\n" +
		"with an identical base is a TAG that evaluates differently — which `STEP=mods` cannot see,\n" +
		"because it compares the rows that exist and not what they evaluate to.\n")
}

type breakdownSlot struct{ base, inc, more, total float64 }

var pobSlotRe = regexp.MustCompile(`^\s*slot (.+?): base=([0-9.]+)((?: x [0-9.]+)*) total=([0-9.]+)`)
var pobMultRe = regexp.MustCompile(` x ([0-9.]+)`)

func pobBreakdownSlots(raw json.RawMessage) (map[string]breakdownSlot, error) {
	out := map[string]breakdownSlot{}
	if len(raw) == 0 {
		return out, nil
	}
	var lines []string
	if err := json.Unmarshal(raw, &lines); err != nil {
		return nil, fmt.Errorf("breakdown is not a list of slot lines: %w", err)
	}
	for _, ln := range lines {
		m := pobSlotRe.FindStringSubmatch(ln)
		if m == nil {
			continue
		}
		s := breakdownSlot{base: parseFloat(m[2]), total: parseFloat(m[4]), inc: 1, more: 1}
		var vals []float64
		for _, g := range pobMultRe.FindAllStringSubmatch(m[3], -1) {
			vals = append(vals, parseFloat(g[1]))
		}
		if len(vals) > 0 {
			s.inc = vals[0]
		}
		if len(vals) > 1 {
			s.more = vals[1]
		}
		if want := s.base * s.inc * s.more; math.Abs(want-s.total) > math.Max(0.02, s.total*1e-4) {
			return nil, fmt.Errorf("parse self-check FAILED on %q: base %g x %g x %g = %g, but the "+
				"line states total=%g. The parser is wrong, not the port", ln, s.base, s.inc, s.more, want, s.total)
		}
		out[m[1]] = s
	}
	return out, nil
}

func slotOrderOf(pob map[string]breakdownSlot, entry *Entry) []string {
	var order []string
	seen := map[string]bool{}
	for _, s := range entry.Slots {
		if !seen[s.Source] {
			order = append(order, s.Source)
			seen[s.Source] = true
		}
	}
	for name := range pob {
		if !seen[name] {
			order = append(order, name)
			seen[name] = true
		}
	}
	return order
}

func fmtSlot(s breakdownSlot, ok bool) string {
	if !ok {
		return "(absent)"
	}
	return fmt.Sprintf("base=%g x%.2f x%.2f = %.2f", s.base, s.inc, s.more, s.total)
}

func parseMult(s string) float64 {
	if s == "" {
		return 1
	}
	return parseFloat(strings.TrimPrefix(strings.TrimSpace(s), "x "))
}

func parseFloat(s string) float64 {
	f, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return f
}

func nearly(a, b float64) bool { return math.Abs(a-b) < 0.005 }
