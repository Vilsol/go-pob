package main

import (
	"fmt"
	"math"
	"math/bits"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const rcUsage = "usage: residual_clusters.py [MIN_BUILDS] [JACCARD]"

const rcGraded = 3208353

func rcGradedDenominator() (float64, error) {
	raw := os.Getenv("GRADED")
	if raw == "" {
		return rcGraded, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("REFUSING: GRADED must be a positive integer, not %s", raw)
	}
	return float64(n), nil
}

var reFloatArg = regexp.MustCompile(`^[0-9]*\.?[0-9]+$`)

type bitset []uint64

func (b bitset) set(i int) bitset {
	w := i / 64
	for len(b) <= w {
		b = append(b, 0)
	}
	b[w] |= 1 << (uint(i) % 64)
	return b
}

func (b bitset) count() int {
	n := 0
	for _, w := range b {
		n += bits.OnesCount64(w)
	}
	return n
}

func (b bitset) orInto(o bitset) bitset {
	if len(o) > len(b) {
		b = append(b, make(bitset, len(o)-len(b))...)
	}
	for i, w := range o {
		b[i] |= w
	}
	return b
}

func (b bitset) andCount(o bitset) int {
	n := 0
	for i := 0; i < len(b) && i < len(o); i++ {
		n += bits.OnesCount64(b[i] & o[i])
	}
	return n
}

func (b bitset) orCount(o bitset) int {
	n := 0
	for i := range b {
		if i < len(o) {
			n += bits.OnesCount64(b[i] | o[i])
		} else {
			n += bits.OnesCount64(b[i])
		}
	}
	for i := len(b); i < len(o); i++ {
		n += bits.OnesCount64(o[i])
	}
	return n
}

func (b bitset) key() string { return string(bitsetBytes(b)) }

func bitsetBytes(b bitset) []byte {
	n := len(b)
	for n > 0 && b[n-1] == 0 {
		n--
	}
	out := make([]byte, 0, n*8)
	for _, w := range b[:n] {
		for k := range 8 {
			out = append(out, byte((w>>(8*uint(k)))&0xff)) //nolint:gosec // masked to a byte
		}
	}
	return out
}

func pyRound4(v float64) float64 {
	r, _ := strconv.ParseFloat(strconv.FormatFloat(v, 'f', 4, 64), 64)
	return r
}

type rcCluster struct {
	mask   bitset
	stats  []string
	pairs  int
	builds int
	lead   string
	ratios int
	absent int
	top1   float64
	top5   float64
}

func residualClusters(argv []string) int {
	matrix := os.Getenv("MATRIX")
	if matrix == "" {
		matrix = "/tmp/residual_matrix.tsv"
	}
	minBuilds, jaccard := 25, 0.90
	if len(argv) > 0 {
		if !reNonNegI.MatchString(argv[0]) {
			fmt.Fprintf(os.Stderr, "REFUSING: MIN_BUILDS must be a non-negative integer, not %s\n%s\n", argv[0], rcUsage)
			return 1
		}
		minBuilds, _ = strconv.Atoi(argv[0])
	}
	if len(argv) > 1 {
		if !reFloatArg.MatchString(argv[1]) {
			fmt.Fprintf(os.Stderr, "REFUSING: JACCARD must be a number, not %s\n%s\n", argv[1], rcUsage)
			return 1
		}
		jaccard, _ = strconv.ParseFloat(argv[1], 64)
	}
	//go-only: GRADED is an override the deleted original had no concept of, so there is no
	rcDenom, gradedErr := rcGradedDenominator()
	if gradedErr != nil {
		fmt.Fprintf(os.Stderr, "%v\n%s\n", gradedErr, rcUsage)
		return 1
	}
	if _, err := os.Stat(matrix); err != nil { //nolint:gosec // an operator-supplied path
		fmt.Fprintf(os.Stderr, "REFUSING: no matrix at %s. Generate it first:\n"+
			"  CORPUS=true MATRIX_OUT=%s go test ./calculator/ -run TestResidualMatrixDump -count=1\n",
			matrix, matrix)
		return 1
	}
	raw, err := os.ReadFile(matrix) //nolint:gosec // ditto
	if err != nil {
		fmt.Fprintf(os.Stderr, "REFUSING: %s is not readable: %s\n", matrix, errnoText(err))
		return 1
	}
	lines := strings.Split(string(raw), "\n")
	header := lines[0]
	if len(lines) > 1 {
		header += "\n"
	}
	if !strings.HasPrefix(header, "build\tstat\tclass") {
		fmt.Fprintf(os.Stderr, "REFUSING: %s does not look like a residual matrix (header %s)\n",
			matrix, pyRepr(header))
		return 1
	}

	buildID := map[string]int{}
	statMask := map[string]bitset{}
	statOrder := []string{}
	statPairs := map[string]int{}
	statRatios := map[string]map[float64]bool{}
	statHist := map[string]map[float64]int{}
	statAbsent := map[string]int{}
	rows := 0
	for _, line := range lines[1:] {
		f := strings.Split(strings.TrimSuffix(line, "\n"), "\t")
		if len(f) < 5 {
			continue
		}
		b, st, cls, pob, ours := f[0], f[1], f[2], f[3], f[4]
		i, ok := buildID[b]
		if !ok {
			i = len(buildID)
			buildID[b] = i
		}
		if _, seen := statMask[st]; !seen {
			statOrder = append(statOrder, st)
			statMask[st] = bitset{}
		}
		statMask[st] = statMask[st].set(i)
		statPairs[st]++
		rows++
		if cls == "miss" || cls == "extra" || cls == "nan" {
			statAbsent[st]++
			continue
		}
		pv, e1 := strconv.ParseFloat(pob, 64)
		ov, e2 := strconv.ParseFloat(ours, 64)
		if e1 != nil || e2 != nil {
			continue
		}
		r := 0.0
		if pv != 0 {
			r = pyRound4(ov / pv)
		} else {
			r = rcNoneSentinel
		}
		if statRatios[st] == nil {
			statRatios[st] = map[float64]bool{}
			statHist[st] = map[float64]int{}
		}
		statRatios[st][r] = true
		statHist[st][r]++
	}
	if rows == 0 {
		fmt.Fprintln(os.Stderr, "REFUSING: matrix is empty; nothing to cluster")
		return 1
	}

	nbuilds := len(buildID)
	fmt.Printf("MATRIX %s\n", matrix)
	fmt.Printf("%d mismatch rows · %d failing stats · %d builds with >=1 mismatch\n",
		rows, len(statMask), nbuilds)
	fmt.Printf("clustering stats failing on >= %d builds, Jaccard >= %s\n\n", minBuilds, pyFloat(jaccard))

	var bigOrder []string
	for _, s := range statOrder {
		if statMask[s].count() >= minBuilds {
			bigOrder = append(bigOrder, s)
		}
	}
	bigSet := map[string]bool{}
	for _, s := range bigOrder {
		bigSet[s] = true
	}
	tailStats := len(statMask) - len(bigOrder)
	tailPairs := 0
	for s, p := range statPairs {
		if !bigSet[s] {
			tailPairs += p
		}
	}

	type grp struct {
		mask  bitset
		stats []string
	}
	groupIdx := map[string]int{}
	var groups []*grp
	for _, s := range bigOrder {
		k := statMask[s].key()
		if gi, ok := groupIdx[k]; ok {
			groups[gi].stats = append(groups[gi].stats, s)
			continue
		}
		groupIdx[k] = len(groups)
		groups = append(groups, &grp{mask: statMask[s], stats: []string{s}})
	}
	sort.SliceStable(groups, func(i, j int) bool {
		return rcSum(groups[i].stats, statPairs) > rcSum(groups[j].stats, statPairs)
	})

	var merged []*rcCluster
	for _, g := range groups {
		placed := false
		for _, mg := range merged {
			inter := mg.mask.andCount(g.mask)
			union := mg.mask.orCount(g.mask)
			if union != 0 && float64(inter)/float64(union) >= jaccard {
				mg.stats = append(mg.stats, g.stats...)
				mg.mask = mg.mask.orInto(g.mask)
				placed = true
				break
			}
		}
		if !placed {
			merged = append(merged, &rcCluster{mask: g.mask, stats: append([]string{}, g.stats...)})
		}
	}

	for _, mg := range merged {
		mg.pairs = rcSum(mg.stats, statPairs)
		mg.builds = mg.mask.count()
		lead := mg.stats[0]
		for _, s := range mg.stats[1:] {
			if statPairs[s] > statPairs[lead] {
				lead = s
			}
		}
		mg.lead = lead
		mg.ratios = len(statRatios[lead])
		var hist []int
		for _, v := range statHist[lead] {
			hist = append(hist, v)
		}
		sort.Sort(sort.Reverse(sort.IntSlice(hist)))
		tot := 0
		for _, h := range hist {
			tot += h
		}
		if tot == 0 {
			tot = 1
		}
		for _, s := range mg.stats {
			mg.absent += statAbsent[s]
		}
		if len(hist) > 0 {
			mg.top1 = float64(hist[0]) / float64(tot)
			sum5 := 0
			for _, h := range hist[:min(5, len(hist))] {
				sum5 += h
			}
			mg.top5 = float64(sum5) / float64(tot)
		}
	}
	sort.SliceStable(merged, func(i, j int) bool { return merged[i].pairs > merged[j].pairs })

	fmt.Printf("%7s %6s %6s %7s %7s  LEAD STAT / MEMBERS\n", "PAIRS", "pp", "STATS", "BUILDS", "RATIOS")
	for _, mg := range merged[:min(25, len(merged))] {
		sorted := append([]string{}, mg.stats...)
		sort.SliceStable(sorted, func(i, j int) bool { return statPairs[sorted[i]] > statPairs[sorted[j]] })
		members := strings.Join(sorted[:min(5, len(sorted))], ", ")
		if len(mg.stats) > 5 {
			members += fmt.Sprintf(", +%d more", len(mg.stats)-5)
		}
		fmt.Printf("%7d %6.3f %6d %7d %7d  %s\n", mg.pairs,
			float64(mg.pairs)/rcDenom*100, len(mg.stats), mg.builds, mg.ratios, members)
	}
	fmt.Printf("\n%7s %6s %6d %7s %7s  (tail: stats failing on < %d builds, %d pairs, %.3fpp)\n",
		"", "", tailStats, "", "", minBuilds, tailPairs, float64(tailPairs)/rcDenom*100)

	fmt.Println("\n=== CONTAINMENT (A's builds are >=90% inside B's -> A is plausibly UPSTREAM of B) ===")
	top := merged[:min(14, len(merged))]
	var roots []int
	for i, a := range top {
		var inside []int
		for j, b := range top {
			if i == j {
				continue
			}
			cov := float64(a.mask.andCount(b.mask)) / float64(max(1, a.builds))
			if cov >= 0.90 && b.builds > a.builds {
				inside = append(inside, j)
			}
		}
		if len(inside) == 0 {
			roots = append(roots, i)
			fmt.Printf("  [%2d] %-46s builds %5d  inside -> ROOT\n", i, truncRunes(a.lead, 44), a.builds)
		} else {
			fmt.Printf("  [%2d] %-46s builds %5d  inside -> %s\n", i, truncRunes(a.lead, 44), a.builds,
				pyIntList(inside))
		}
	}

	fmt.Println("\n=== ROOTS, ranked (contained in no larger cluster; fix these first) ===")
	rootsSorted := append([]int{}, roots...)
	sort.SliceStable(rootsSorted, func(i, j int) bool { return top[rootsSorted[i]].pairs > top[rootsSorted[j]].pairs })
	for _, i := range rootsSorted {
		a := top[i]
		fmt.Printf("  %7d pairs %6.3fpp  %5d builds  %5d distinct ratios  %s\n",
			a.pairs, float64(a.pairs)/rcDenom*100, a.builds, a.ratios, a.lead)
	}

	fmt.Println("\n=== EFFICIENCY: REACH@5 — what a cluster's five commonest ratios are worth ===")
	fmt.Printf("%8s %6s %6s %7s %7s %7s  LEAD STAT\n", "REACH@5", "top1", "top5", "PAIRS", "ABSENT", "BUILDS")
	eff := append([]*rcCluster{}, merged...)
	sort.SliceStable(eff, func(i, j int) bool {
		return float64(eff[i].pairs)*eff[i].top5 > float64(eff[j].pairs)*eff[j].top5
	})
	var effKept []*rcCluster
	for _, g := range eff {
		if g.pairs-g.absent > 0 {
			effKept = append(effKept, g)
		}
	}
	for _, mg := range effKept[:min(15, len(effKept))] {
		fmt.Printf("%8.0f %6.2f %6.2f %7d %7d %7d  %s\n",
			float64(mg.pairs)*mg.top5, mg.top1, mg.top5, mg.pairs, mg.absent, mg.builds, mg.lead)
	}
	fmt.Println("  REACH@5 = pairs x top5 share: what the FIVE commonest arithmetic relationships in that")
	fmt.Println("  cluster are worth. Both fixes measured so far landed their cluster's dominant ratio.")

	fmt.Println("\n=== REACHABLE BY EFFICIENCY BAND (cumulative, best-yield causes first) ===")
	allTotal := 0
	for _, p := range statPairs {
		allTotal += p
	}
	base := int(rcDenom) - allTotal
	fmt.Printf("  current gate ~ %.4f%%   (%d pairs wrong)\n", float64(base)/rcDenom*100, allTotal)
	fmt.Printf("%10s %9s %8s %8s %9s\n", "BAND", "CLUSTERS", "PAIRS", "~FIXES", "GATE AT")
	for _, lo := range []float64{0.8, 0.6, 0.4, 0.3, 0.2, 0.1, 0.0} {
		n, pairs := 0, 0
		for _, g := range eff {
			if g.top5 >= lo {
				n++
				pairs += int(float64(g.pairs) * g.top5)
			}
		}
		fmt.Printf("  top5>=%4.1f %9d %8d %8d %8.4f%%\n", lo, n, pairs, n*5,
			float64(base+pairs)/rcDenom*100)
	}
	fmt.Println("  ~FIXES assumes five fixes per cluster; both fixes measured so far cleared their")
	fmt.Println("  cluster's dominant ratio in ONE change, so this over-counts for concentrated clusters.")
	fmt.Println("  ⚠️ Clusters below the MIN_BUILDS cut are excluded entirely, so even the 0 row is an")
	fmt.Printf("     UNDER-estimate by the %d-pair tail.\n", tailPairs)

	fmt.Println()
	for _, l := range []string{
		"⚠️ BOTH CONCENTRATION MEASURES ARE BIASED, IN OPPOSITE DIRECTIONS, AND BOTH WERE MEASURED.",
		"   Distinct-ratio COUNT over-states the work by ~40x: the self-ailment-duration cluster",
		"   showed 88 distinct ratios and took ONE code change, the enemy-modifier cluster 45 and",
		"   took two rows. TOP5 SHARE under-states RECOVERABLE pairs for ADDITIVE causes: a missing",
		"   flat modifier produces a DIFFERENT ratio on every build, so one cause can scatter across",
		"   hundreds of ratios and score top5 ~= 0. The FireResist off-by-one did exactly that -- it",
		"   read as 0.9889 / 0.9882 / 0.9878 and was a single missing point of maximum resistance.",
		"   Use top1 == 1.00 as a near-certain single fix; treat a low top5 as UNKNOWN, not as hard.",
		"⚠️ Every figure is an UPPER BOUND. Only the gate measures a change.",
	} {
		fmt.Println(l)
	}
	return 0
}

const rcNoneSentinel = -1.7976931348623157e+308

func rcSum(stats []string, pairs map[string]int) int {
	n := 0
	for _, s := range stats {
		n += pairs[s]
	}
	return n
}

func pyFloat(v float64) string {
	switch {
	case math.IsNaN(v):
		return "nan"
	case math.IsInf(v, 1):
		return "inf"
	case math.IsInf(v, -1):
		return "-inf"
	}
	sci := strconv.FormatFloat(v, 'e', -1, 64)
	exp := 0
	if i := strings.IndexByte(sci, 'e'); i >= 0 {
		exp, _ = strconv.Atoi(sci[i+1:])
	}
	if exp < -4 || exp >= 16 {
		return sci
	}
	out := strconv.FormatFloat(v, 'f', -1, 64)
	if !strings.Contains(out, ".") {
		out += ".0"
	}
	return out
}

func pyIntList(xs []int) string {
	parts := make([]string, len(xs))
	for i, x := range xs {
		parts[i] = strconv.Itoa(x)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}
