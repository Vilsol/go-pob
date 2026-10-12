package calculator

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
)

// TestItemModDropVsPoB decides which of our item-mod drops are DEFECTS and which are FAITHFUL.
func TestItemModDropVsPoB(t *testing.T) {
	verdictPath := os.Getenv("POB_VERDICT")
	casesPath := os.Getenv("DROP_CASES")
	if verdictPath == "" || casesPath == "" {
		t.Skip("set POB_VERDICT=<pob_verdict.json> and DROP_CASES=<cases.json>")
	}
	requireGameData(t, "3.29")

	var verdicts []struct {
		Line   string   `json:"line"`
		Extra  string   `json:"extra"`
		Parsed bool     `json:"parsed"`
		Count  int      `json:"count"`
		Names  []string `json:"names"`
	}
	raw, err := os.ReadFile(verdictPath)
	if err != nil {
		t.Fatalf("POB_VERDICT: %v", err)
	}
	if err := json.Unmarshal(raw, &verdicts); err != nil {
		t.Fatalf("POB_VERDICT: %v", err)
	}
	if len(verdicts) == 0 {
		t.Fatalf("REFUSING: POB_VERDICT holds 0 lines; a zero here is indistinguishable from " +
			"'PoB agrees with us about everything'")
	}

	var cases []struct {
		Name string `json:"name"`
		Raw  string `json:"raw"`
	}
	raw, err = os.ReadFile(casesPath)
	if err != nil {
		t.Fatalf("DROP_CASES: %v", err)
	}
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatalf("DROP_CASES: %v", err)
	}
	weight := map[string]int{}
	for _, c := range cases {
		parts := strings.SplitN(c.Name, "|", 3)
		if len(parts) < 2 {
			continue
		}
		var n int
		if _, err := fmt.Sscanf(parts[1], "%d", &n); err == nil {
			lines := strings.Split(strings.TrimSpace(c.Raw), "\n")
			weight[lines[len(lines)-1]] = n
		}
	}

	if e := ParseMod("+10 to Strength", false); e == nil || strings.TrimSpace(modExtra(e)) != "" || len(e.ModList) == 0 {
		t.Fatalf("REFUSING: our parser failed a line it must handle; every row below would " +
			"read as a defect")
	}
	if e := ParseMod("Zz Not A Real Item Mod Zz", false); e != nil && strings.TrimSpace(modExtra(e)) == "" && len(e.ModList) > 0 {
		t.Fatalf("REFUSING: our parser accepted a nonsense line; this test cannot detect a drop")
	}

	var defectsLive, defectsInert, agree, weBetter []dropRow

	for _, v := range verdicts {
		e := ParseMod(v.Line, false)
		ourExtra := ""
		ourMods := 0
		if e != nil {
			ourExtra = strings.TrimSpace(modExtra(e))
			ourMods = len(e.ModList)
		}
		ourOK := e != nil && ourExtra == "" && ourMods > 0
		pobOK := v.Parsed && strings.TrimSpace(v.Extra) == ""

		r := dropRow{line: v.Line, ourExtra: ourExtra, builds: weight[v.Line], pobMods: v.Count, names: v.Names}
		switch {
		case pobOK && !ourOK && v.Count > 0:
			defectsLive = append(defectsLive, r)
		case pobOK && !ourOK:
			defectsInert = append(defectsInert, r)
		case !pobOK && ourOK:
			weBetter = append(weBetter, r)
		default:
			agree = append(agree, r)
		}
	}

	sum := func(rs []dropRow) int {
		n := 0
		for _, r := range rs {
			n += r.builds
		}
		return n
	}

	fmt.Printf("\n=== ITEM MOD DROPS: OURS vs THE PINNED PoB GRAMMAR ===\n")
	fmt.Printf("lines compared %d\n", len(verdicts))
	fmt.Printf("  DEFECT (PoB parses -> >=1 mod, we do not)  %4d shapes  %6d build-instances\n", len(defectsLive), sum(defectsLive))
	fmt.Printf("  measured zero (PoB parses -> 0 mods)       %4d shapes  %6d build-instances\n", len(defectsInert), sum(defectsInert))
	fmt.Printf("  faithful (both decline, or both accept)    %4d shapes  %6d build-instances\n", len(agree), sum(agree))
	fmt.Printf("  ⚠️ WE ACCEPT, PoB DECLINES                  %4d shapes  %6d build-instances\n", len(weBetter), sum(weBetter))
	if len(weBetter) > 0 {
		fmt.Printf("     (an OVER-APPLIED filter is a real defect too -- porting the ABSENCES is porting)\n")
	}

	byExtra := map[string]*struct {
		shapes, builds int
		sample         string
	}{}
	for _, r := range defectsLive {
		k := normaliseItemModLine(r.ourExtra)
		if k == "" {
			k = "(empty leftover -- we produced no mods at all)"
		}
		g, ok := byExtra[k]
		if !ok {
			g = &struct {
				shapes, builds int
				sample         string
			}{sample: r.line}
			byExtra[k] = g
		}
		g.shapes++
		g.builds += r.builds
	}
	type gk struct {
		key            string
		shapes, builds int
		sample         string
	}
	groups := make([]gk, 0, len(byExtra))
	for k, g := range byExtra {
		groups = append(groups, gk{k, g.shapes, g.builds, g.sample})
	}
	sort.Slice(groups, func(a, b int) bool {
		if groups[a].builds != groups[b].builds {
			return groups[a].builds > groups[b].builds
		}
		return groups[a].key < groups[b].key
	})

	fmt.Printf("\nLIVE DEFECTS GROUPED BY OUR LEFTOVER TEXT (the missing grammar), top 40\n")
	fmt.Printf("%7s %7s  %s\n", "BUILDS", "SHAPES", "LEFTOVER  ||  sample line")
	for i, g := range groups {
		if i >= 40 {
			break
		}
		fmt.Printf("%7d %7d  %.70s  ||  %.70s\n", g.builds, g.shapes, g.key, g.sample)
	}

	reportDropConsumerSplit(t, defectsLive)

	sort.Slice(defectsLive, func(a, b int) bool { return defectsLive[a].builds > defectsLive[b].builds })
	fmt.Printf("\nTOP 40 LIVE DEFECT LINES\n")
	for i, r := range defectsLive {
		if i >= 40 {
			break
		}
		fmt.Printf("%6d builds  pobMods=%d  %.95s\n     leftover: %.95s\n", r.builds, r.pobMods, r.line, r.ourExtra)
	}
}

type dropRow struct {
	line     string
	ourExtra string
	builds   int
	pobMods  int
	names    []string
}

func reportDropConsumerSplit(t *testing.T, defects []dropRow) {
	t.Helper()
	bodies := reverseSweepReaderBodies(t, "..")
	stripped := make([]string, 0, len(bodies))
	for _, b := range bodies {
		stripped = append(stripped, stripGoComments(b))
	}

	readable := map[string]bool{}
	read := func(name string) bool {
		if v, ok := readable[name]; ok {
			return v
		}
		needles := []string{`"` + name + `"`}
		if bare := strings.TrimPrefix(name, "Condition:"); bare != name {
			needles = append(needles, `"`+bare+`"`)
		}
		v := false
		for _, b := range stripped {
			for _, n := range needles {
				if strings.Contains(b, n) {
					v = true
					break
				}
			}
			if v {
				break
			}
		}
		readable[name] = v
		return v
	}

	var withShapes, withBuilds, noShapes, noBuilds int
	type nrow struct {
		builds int
		names  string
		line   string
	}
	var noRows, withRows []nrow
	for _, r := range defects {
		all := len(r.names) > 0
		for _, n := range r.names {
			if !read(n) {
				all = false
				break
			}
		}
		e := nrow{builds: r.builds, names: strings.Join(r.names, ","), line: r.line}
		if all {
			withShapes++
			withBuilds += r.builds
			withRows = append(withRows, e)
		} else {
			noShapes++
			noBuilds += r.builds
			noRows = append(noRows, e)
		}
	}

	fmt.Printf("\nLIVE DEFECTS SPLIT BY CONSUMER-EXISTENCE (rank by THIS, not by the count above)\n")
	fmt.Printf("  every emitted mod name has a reader here  %4d shapes  %6d build-instances\n", withShapes, withBuilds)
	fmt.Printf("  at least one name has NO reader here      %4d shapes  %6d build-instances\n", noShapes, noBuilds)
	fmt.Printf("     (the second row is a measured zero at best; see this function's header)\n")

	sort.Slice(withRows, func(a, b int) bool { return withRows[a].builds > withRows[b].builds })
	sort.Slice(noRows, func(a, b int) bool { return noRows[a].builds > noRows[b].builds })
	fmt.Printf("\n  TOP 20 WITH A READER — the candidate worklist, still to be adjudicated\n")
	for i, r := range withRows {
		if i >= 20 {
			break
		}
		fmt.Printf("  %6d  %-40.40s  %.60s\n", r.builds, r.names, r.line)
	}
	fmt.Printf("\n  TOP 10 WITH NO READER — port the CONSUMER first, or do not port at all\n")
	for i, r := range noRows {
		if i >= 10 {
			break
		}
		fmt.Printf("  %6d  %-40.40s  %.60s\n", r.builds, r.names, r.line)
	}
}

func stripGoComments(src string) string {
	var b strings.Builder
	b.Grow(len(src))
	for i := 0; i < len(src); {
		switch {
		case strings.HasPrefix(src[i:], "/*"):
			if j := strings.Index(src[i+2:], "*/"); j >= 0 {
				i += 2 + j + 2
			} else {
				i = len(src)
			}
		case strings.HasPrefix(src[i:], "//"):
			j := strings.IndexByte(src[i:], '\n')
			if j < 0 {
				i = len(src)
			} else {
				i += j
			}
		default:
			b.WriteByte(src[i])
			i++
		}
	}
	return b.String()
}
