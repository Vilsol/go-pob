package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var rmFamilies = []struct{ name, pat string }{
	{"EHP / max-hit-taken (downstream)",
		`EHP|MaximumHitTaken|NumberOfHits|NumberOfDamagingHits|NumberOfMitigated|` +
			`SurvivalTime|TotalHitPool|TotalPool|TakenHit`},
	{"pool inputs (Life/ES/Armour/Evasion/PDR)",
		`^(Life|EnergyShield|Armour|Evasion|Mana|PhysicalDamageReduction|SpellSuppression|` +
			`Block|Ward|Net)`},
	{"resistances", `Resist`},
	{"damage chain",
		`DPS|AverageDamage|AverageBurstDamage|AverageBurstHits|CombinedAvg|TotalDot|Damage$|` +
			`TotalMin|TotalMax|HitAverage|AverageHit|MinBase|MaxBase|StoredHitAvg`},
	{"ailments", `Ignite|Poison|Bleed|Shock|Chill|Freeze|Brittle|Sap|Scorch|Ailment|Stun`},
	{"crit", `Crit`},
	{"speed / time / duration", `Speed|Time|Duration|Cooldown|Frequency|Rate`},
	{"hit / accuracy / chance", `Accuracy|HitChance|Chance`},
	{"gems / skills / requirements", `^(Req|Gem|Skill|Level|Quality)`},
	{"leech / regen / recoup / reservation", `Leech|Regen|Recoup|Reserv|Recovery|Recharge`},
}

var rmExtraFamilies = []struct{ name, pat string }{
	{"damage taken (DoT and reflect)", `TakenDotMult|TakenReflect`},
	{"area of effect", `AreaOfEffect`},
	{"impale", `Impale`},
	{"limits / counts (minions, totems)", `Limit$|^Summoned`},
	{"warcry / exerted attacks", `Warcry|Exerted|FistOfWar|RuthlessBlow`},
	{"hit pools — MoM spellings EHP's TotalHitPool/TotalPool patterns miss", `HitPool`},
	{"pool inputs — PREFIXED spellings the ^-anchored family cannot see",
		`Armour|Evasion|EnergyShield|Mana`},
}

var rmExtraFamilyRE = func() []*regexp.Regexp {
	out := make([]*regexp.Regexp, len(rmExtraFamilies))
	for i, f := range rmExtraFamilies {
		out[i] = regexp.MustCompile(f.pat)
	}
	return out
}()

const rmUnclassifiedShown = 25

var (
	rmFamilyRE = func() []*regexp.Regexp {
		out := make([]*regexp.Regexp, len(rmFamilies))
		for i, f := range rmFamilies {
			out[i] = regexp.MustCompile(f.pat)
		}
		return out
	}()
	rmPrefix = regexp.MustCompile(`^(MainHand|OffHand|Minion|Enemy|Parent|PartyMember)\.`)
)

func rmClassify(stat string) string {
	bare := rmPrefix.ReplaceAllString(stat, "")
	for i, re := range rmFamilyRE {
		if re.MatchString(bare) {
			return rmFamilies[i].name
		}
	}
	//go-only: FAMILIES2 has no counterpart in the deleted python original and difftest sets only
	if rmExtendedFamilies() {
		for i, re := range rmExtraFamilyRE {
			if re.MatchString(bare) {
				return rmExtraFamilies[i].name
			}
		}
	}
	return "unclassified"
}

var rmExtendedFamilies = func() func() bool {
	var on bool
	var once bool
	return func() bool {
		if !once {
			on, once = os.Getenv("FAMILIES2") != "", true
		}
		return on
	}
}()

type rmTally struct {
	order []string
	v     map[string]int
}

func newTally() *rmTally { return &rmTally{v: map[string]int{}} }

func (t *rmTally) add(k string, n int) {
	if _, ok := t.v[k]; !ok {
		t.order = append(t.order, k)
	}
	t.v[k] += n
}

func (t *rmTally) byDesc() []string {
	out := append([]string{}, t.order...)
	sort.SliceStable(out, func(i, j int) bool { return t.v[out[i]] > t.v[out[j]] })
	return out
}

func rmRefuse(msg string) int {
	fmt.Fprintln(os.Stderr, msg)
	return 1
}

func rmReadTotals(wl string) (int, map[string]int, bool) {
	b, err := os.ReadFile(filepath.Join(wl, "totals.tsv")) //nolint:gosec // an operator-supplied worklist path
	if err != nil {
		return 0, nil, false
	}
	lines := strings.Split(strings.TrimRight(string(b), "\n"), "\n")
	if len(lines) < 2 || !strings.HasPrefix(lines[0], "scope\tseen") {
		return 0, nil, false
	}
	graded := 0
	byActor := map[string]int{}
	for _, line := range lines[1:] {
		c := strings.Split(line, "\t")
		if len(c) < 2 {
			continue
		}
		seen, err := strconv.Atoi(strings.TrimSpace(c[1]))
		if err != nil {
			return 0, nil, false
		}
		if c[0] == "ALL" {
			graded = seen
			continue
		}
		scope := c[0]
		if scope == "player" {
			scope = "player (unprefixed)"
		}
		byActor[scope] = seen
	}
	if graded <= 0 {
		return 0, nil, false
	}
	return graded, byActor, true
}

func residualMap(args []string) int {
	if len(args) > 0 {
		fmt.Fprintf(os.Stderr, "REFUSING: residual-map takes no arguments, got %q\n", args[0])
		return 64
	}
	wl := strings.TrimSpace(os.Getenv("WL"))
	if wl == "" {
		return rmRefuse("REFUSING: set WL=<dir> to a worklist directory " +
			"(WORKLIST=<dir> on TestCorpus3_29VsOracle writes one).")
	}
	statsPath := filepath.Join(wl, "stats.tsv")
	buildsPath := filepath.Join(wl, "builds.tsv")
	for _, p := range []string{statsPath, buildsPath} {
		if _, err := os.Stat(p); err != nil { //nolint:gosec // an operator-supplied worklist path
			return rmRefuse("REFUSING: " + p + " does not exist. This tool cannot answer without it.")
		}
	}

	blines, rc := rmReadTSV(buildsPath)
	if rc != 0 {
		return rc
	}
	h := strings.Split(blines[0], "\t")
	if len(h) < 3 || h[0] != "build" || h[1] != "wrong" || h[2] != "total" {
		return rmRefuse(fmt.Sprintf("REFUSING: builds.tsv columns are %s, expected build/wrong/total.",
			pyList(h, 3)))
	}
	graded := 0
	for i, line := range blines[1:] {
		c := strings.Split(line, "\t")
		if len(c) >= 3 {
			n, rc := rmCellInt(buildsPath, i+2, c[2])
			if rc != 0 {
				return rc
			}
			graded += n
		}
	}
	if graded == 0 {
		return rmRefuse("REFUSING: builds.tsv reports zero graded pairs.")
	}

	slines, rc := rmReadTSV(statsPath)
	if rc != 0 {
		return rc
	}
	if !strings.HasPrefix(slines[0], "stat") {
		return rmRefuse(fmt.Sprintf("REFUSING: %s does not look like a worklist stats.tsv "+
			"(header: %s)", statsPath, pyRepr(slines[0]+"\n")))
	}
	fam, famn := newTally(), newTally()
	uncl := newTally()
	act, actn, actseen := newTally(), newTally(), newTally()
	totalSeen, totalWrong := 0, 0
	for i, line := range slines[1:] {
		c := strings.Split(line, "\t")
		if len(c) < 3 {
			continue
		}
		seen, rc := rmCellInt(statsPath, i+2, c[1])
		if rc != 0 {
			return rc
		}
		ok, rc := rmCellInt(statsPath, i+2, c[2])
		if rc != 0 {
			return rc
		}
		stat := c[0]
		totalSeen += seen
		w := seen - ok
		if w <= 0 {
			continue
		}
		totalWrong += w
		f := rmClassify(stat)
		fam.add(f, w)
		famn.add(f, 1)
		if f == "unclassified" {
			uncl.add(stat, w)
		}
		a := "player (unprefixed)"
		if m := rmPrefix.FindStringSubmatch(stat); m != nil {
			a = m[1] + "."
		}
		act.add(a, w)
		actn.add(a, 1)
		actseen.add(a, seen)
	}
	if totalWrong == 0 {
		return rmRefuse("REFUSING: zero wrong pairs across the whole table. That is not a clean " +
			"bill of health, it is a broken read — check the worklist is from a real gate run.")
	}

	totalsGraded, totalsByActor, haveTotals := rmReadTotals(wl)
	if haveTotals {
		graded = totalsGraded
	}

	pp := func(n int) float64 { return float64(n) * 100.0 / float64(graded) }

	fmt.Printf("WORKLIST %s\n", wl)
	fmt.Printf("%d graded pairs · %d wrong · %.4fpp residual\n", graded, totalWrong, pp(totalWrong))
	fmt.Printf("1pp = %.0f pairs\n", float64(graded)/100)
	if haveTotals {
		fmt.Printf("  (denominator read from totals.tsv — the gate's own graded total. %d pairs "+
			"sit in stats that are PERFECT and so are absent from stats.tsv)\n\n",
			graded-totalSeen)
	} else {
		fmt.Printf("  (%d pairs sit in stats that are PERFECT and therefore absent from "+
			"stats.tsv — that is why the denominator is read from builds.tsv)\n\n", graded-totalSeen)
	}

	fmt.Println("=== BY FAMILY (first-match-wins; a total is an UPPER BOUND, families share causes) ===")
	for _, k := range fam.byDesc() {
		fmt.Printf("  %-42s%7d pairs  %6.3fpp  (%d stats)\n", k, fam.v[k], pp(fam.v[k]), famn.v[k])
	}
	if fam.v["unclassified"] != 0 {
		fmt.Printf("  ⚠️ %d pairs are UNCLASSIFIED — extend FAMILIES or argue with it.\n",
			fam.v["unclassified"])
		if os.Getenv("UNCLASSIFIED") != "" {
			ranked := uncl.byDesc()
			shown := len(ranked)
			if shown > rmUnclassifiedShown {
				shown = rmUnclassifiedShown
			}
			fmt.Printf("     top %d of %d unclassified stats:\n", shown, len(ranked))
			for _, k := range ranked[:shown] {
				fmt.Printf("       %-52s%7d\n", k, uncl.v[k])
			}
			if rest := len(ranked) - shown; rest > 0 {
				var tail int
				for _, k := range ranked[shown:] {
					tail += uncl.v[k]
				}
				fmt.Printf("       (%d further stats holding %d pairs, largest %d)\n",
					rest, tail, uncl.v[ranked[shown]])
			}
		}
	}

	fmt.Println("\n=== BY ACTOR TABLE (error RATE is the column to read, not the total) ===")
	for _, k := range act.byDesc() {
		seen := actseen.v[k]
		if haveTotals {
			if v, found := totalsByActor[k]; found {
				seen = v
			}
		}
		fmt.Printf("  %-24s%7d wrong  %6.3fpp  %4d stats  %5.1f%% of its own graded pairs\n",
			k, act.v[k], pp(act.v[k]), actn.v[k], 100.0*float64(act.v[k])/float64(seen))
	}

	type rmRow struct {
		wrong int
		frac  float64
		build string
	}
	var rows []rmRow
	for i, line := range blines[1:] {
		c := strings.Split(line, "\t")
		if len(c) < 3 {
			continue
		}
		total, rc := rmCellInt(buildsPath, i+2, c[2])
		if rc != 0 {
			return rc
		}
		if total == 0 {
			continue
		}
		w, rc := rmCellInt(buildsPath, i+2, c[1])
		if rc != 0 {
			return rc
		}
		rows = append(rows, rmRow{w, float64(w) / float64(total), c[0]})
	}
	sort.Slice(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		switch {
		case a.wrong != b.wrong:
			return a.wrong > b.wrong
		case a.frac != b.frac:
			return a.frac > b.frac
		default:
			return a.build > b.build
		}
	})
	frac := make([]float64, len(rows))
	for i, r := range rows {
		frac[i] = r.frac
	}
	sort.Float64s(frac)
	q := func(p float64) float64 {
		if len(frac) == 0 {
			return 0
		}
		return frac[int(p*float64(len(frac)-1))]
	}
	tw := 0
	for _, r := range rows {
		tw += r.wrong
	}
	fmt.Printf("\n=== BY BUILD (%d builds) ===\n", len(rows))
	fmt.Printf("  wrong fraction: p10 %.1f%%  p25 %.1f%%  median %.1f%%  p75 %.1f%%  p90 %.1f%%\n",
		q(.10)*100, q(.25)*100, q(.50)*100, q(.75)*100, q(.90)*100)
	for _, k := range []int{10, 100, 500, 1000} {
		if k <= len(rows) {
			s := 0
			for _, r := range rows[:k] {
				s += r.wrong
			}
			fmt.Printf("  top %4d builds hold %5.1f%% of the residual\n", k, 100*float64(s)/float64(tw))
		}
	}
	fmt.Print("  ⚠️ A FLAT DISTRIBUTION MEANS BUILD-FIRST IS THE WRONG AXIS. It has been flat every time\n" +
		"     it was measured; re-measure rather than inheriting that, it is one command.\n")
	return 0
}

func rmReadTSV(path string) ([]string, int) {
	b, err := os.ReadFile(path) //nolint:gosec // an operator-supplied worklist path
	if err != nil {
		return nil, rmRefuse(fmt.Sprintf("REFUSING: %s is not readable: %s", path, errnoText(err)))
	}
	lines := strings.Split(string(b), "\n")
	if len(lines) == 0 || lines[0] == "" {
		return nil, rmRefuse("REFUSING: " + path +
			" is empty; a worklist written by a real gate run has a header.")
	}
	return lines, 0
}

func rmCellInt(path string, lineno int, raw string) (int, int) {
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, rmRefuse(fmt.Sprintf("REFUSING: %s line %d: %s is not an integer.",
			path, lineno, pyRepr(raw)))
	}
	return n, 0
}

func pyList(xs []string, n int) string {
	if n > len(xs) {
		n = len(xs)
	}
	parts := make([]string, n)
	for i := range n {
		parts[i] = pyRepr(xs[i])
	}
	return "[" + strings.Join(parts, ", ") + "]"
}
