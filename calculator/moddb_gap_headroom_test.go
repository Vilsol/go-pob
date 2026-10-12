package calculator

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestModDBGapHeadroom ranks the mod rows PoB has and we do not by the RESIDUAL CONCENTRATED IN THE BUILDS THAT CARRY THEM.
func TestModDBGapHeadroom(t *testing.T) {
	requireOperatorRun(t, "report-only census")
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	wl := os.Getenv("WORKLIST")
	if wl == "" {
		t.Skip("set WORKLIST=<dir> holding builds.tsv (write it with WORKLIST= on TestCorpus3_29VsOracle)")
	}
	requireGameData(t, "3.29")

	wrongByBuild, totalByBuild := readBuildsTSV(t, filepath.Join(wl, "builds.tsv"))

	dir := diagDumpsDir(t)
	names := diagCorpusNames(t)
	corpus := corpusDirForLoadedData(t)

	limit := len(names)
	if v := os.Getenv("LIMIT"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			t.Fatalf("REFUSING: LIMIT=%q is not a positive integer (%v)", v, err)
		}
		limit = min(n, len(names))
	}

	minBuilds := 25
	if v := os.Getenv("MINBUILDS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			t.Fatalf("REFUSING: MINBUILDS=%q is not a positive integer (%v)", v, err)
		}
		minBuilds = n
	}

	type row struct {
		builds  int
		occurs  int
		wrongIn float64
		totalIn float64
		example string
	}
	gaps := map[string]*row{}
	scanned, skipped, joined := 0, 0, 0
	var wrongAll, totalAll float64

	for _, name := range names {
		if scanned >= limit {
			break
		}
		d, healthy := diagLoadDump(dir, name)
		if d == nil || !healthy {
			skipped++
			continue
		}
		w, ok := wrongByBuild[name+".xml"]
		if !ok {
			skipped++
			continue
		}
		tot := totalByBuild[name+".xml"]

		file, err := os.ReadFile(filepath.Join(corpus, name+".xml"))
		if err != nil {
			skipped++
			continue
		}
		build, err := builds.ParseBuild(file)
		if err != nil {
			skipped++
			continue
		}
		env := (&Calculator{PoB: build}).BuildOutput(OutputModeMain)
		if env == nil || env.Player == nil || env.Player.ModDB == nil {
			skipped++
			continue
		}
		scanned++
		joined++
		wrongAll += w
		totalAll += tot

		key := func(modName, typ string, val float64) string {
			return fmt.Sprintf("%-34s %-8s %12.4f", modName, typ, val)
		}
		pobCount := map[string]int{}
		for modName, mods := range d.ModDB {
			for _, m := range mods {
				pobCount[key(modName, m.Type, pobModValue(m.Type, m.Value))]++
			}
		}
		ourCount := map[string]int{}
		for modName, mods := range env.Player.ModDB.Mods {
			for _, m := range mods {
				ourCount[key(modName, string(m.Type()), ourModValue(m))]++
			}
		}
		for k, n := range pobCount {
			if n-ourCount[k] <= 0 {
				continue
			}
			g := gaps[k]
			if g == nil {
				g = &row{example: name}
				gaps[k] = g
			}
			g.builds++
			g.occurs += n - ourCount[k]
			g.wrongIn += w
			g.totalIn += tot
		}
	}

	if scanned == 0 {
		t.Fatalf("REFUSING: scanned 0 builds. Either %s holds no usable dumps or no dump name "+
			"joined builds.tsv — empty must mean 'I looked and found nothing', never 'I did not look'", dir)
	}
	if totalAll == 0 {
		t.Fatalf("REFUSING: joined %d builds but their graded-pair total is 0; builds.tsv's "+
			"columns have moved", joined)
	}

	type ranked struct {
		key      string
		g        *row
		lift     float64
		headroom float64
	}
	var out []ranked
	universal, thin := 0, 0
	for k, g := range gaps {
		if g.builds >= scanned {
			universal++
			continue
		}
		if g.builds < minBuilds {
			thin++
			continue
		}
		totalOut := totalAll - g.totalIn
		if totalOut <= 0 || g.totalIn <= 0 {
			thin++
			continue
		}
		lift := g.wrongIn/g.totalIn - (wrongAll-g.wrongIn)/totalOut
		out = append(out, ranked{k, g, lift, lift * g.totalIn})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].headroom != out[j].headroom {
			return out[i].headroom > out[j].headroom
		}
		return out[i].key < out[j].key
	})

	byPop := make([]ranked, len(out))
	copy(byPop, out)
	sort.Slice(byPop, func(i, j int) bool {
		if byPop[i].g.builds != byPop[j].g.builds {
			return byPop[i].g.builds > byPop[j].g.builds
		}
		return byPop[i].key < byPop[j].key
	})
	popRank := map[string]int{}
	for i, r := range byPop {
		popRank[r.key] = i + 1
	}

	fmt.Printf("=== MODDB GAP HEADROOM === %d builds joined against builds.tsv; corpus residual "+
		"%.0f wrong of %.0f graded (%.3f%%)\n", joined, wrongAll, totalAll, 100*wrongAll/totalAll)
	fmt.Printf("%d rankable rows; %d universal rows excluded as base scaffolding; %d rows below "+
		"MINBUILDS=%d excluded as too thin to carry a lift\n", len(out), universal, thin, minBuilds)
	fmt.Printf("\n⚠️  HEADROOM IS AN ESTIMATE, NOT A BOUND, and LIFT IS CORRELATION. A carrying build\n" +
		"    is wrong for many reasons at once, so a port recovers a SUBSET of its headroom — but a\n" +
		"    fix also reaches whatever shares its CODE PATH, so the figure has been both undershot\n" +
		"    (73%%) and overshot (108%%) by confirmed fixes. Read it as an order of magnitude.\n")
	fmt.Printf("\n%10s %8s %7s %7s %8s  %-34s %-8s %12s  %s\n",
		"HEADROOM", "LIFT%", "SHARE%", "BUILDS", "POP-RANK", "MOD", "TYPE", "VALUE", "EXAMPLE BUILD")
	for i, r := range out {
		if i >= censusRowCap() {
			fmt.Printf("... %d further rows not shown (ROWS=n to change)\n", len(out)-censusRowCap())
			break
		}
		share := 100 * float64(r.g.builds) / float64(scanned)
		flag := " "
		if share >= 75 {
			flag = "!"
		}
		fmt.Printf("%10.0f %8.3f %6.1f%s %7d %8d  %s  %s\n",
			r.headroom, 100*r.lift, share, flag, r.g.builds, popRank[r.key], r.key, r.g.example)
	}
	fmt.Printf("\n! = SHARE >= 75%%: lift is measured against the remaining minority and is NOT " +
		"reliable for these rows.\n")
	fmt.Printf("⚠️  READ LIFT FIRST, HEADROOM SECOND. Measured against real gate deltas: lift 9.1%% " +
		"and 6.0%%\n    predicted within 73-108%%; lift 4.1%% predicted 7964 and delivered 2; lift " +
		"2.25%%\n    predicted 26280 and delivered 23. A LOW lift over a LARGE population is the " +
		"signature\n    of a correlate.\n")

	const reliableLift = 0.06
	band := make([]ranked, 0, 16)
	for _, r := range out {
		if r.lift >= reliableLift && float64(r.g.builds)/float64(scanned) < 0.75 {
			band = append(band, r)
		}
	}
	fmt.Printf("\n=== RELIABLE BAND (lift >= %.0f%%, share < 75%%) — %d rows ===\n",
		100*reliableLift, len(band))
	if len(band) == 0 {
		fmt.Printf("  none. Every row above the lift threshold has been worked, or the corpus has " +
			"no\n  concentrated gaps left — either way the next target is NOT in this ranking.\n")
	}
	for i, r := range band {
		if i >= censusRowCap() {
			fmt.Printf("... %d further band rows not shown\n", len(band)-censusRowCap())
			break
		}
		fmt.Printf("%10.0f %8.3f %6.1f  %7d %8d  %s  %s\n", r.headroom, 100*r.lift,
			100*float64(r.g.builds)/float64(scanned), r.g.builds, popRank[r.key], r.key, r.g.example)
	}
}

func readBuildsTSV(t *testing.T, path string) (map[string]float64, map[string]float64) {
	t.Helper()
	fh, err := os.Open(path) //nolint:gosec // operator-supplied worklist path
	if err != nil {
		t.Fatalf("REFUSING: cannot read %s (%v). Write it with "+
			"`CORPUS=true WORKLIST=<dir> go test ./calculator/ -run TestCorpus3_29VsOracle`", path, err)
	}
	defer func() { _ = fh.Close() }()

	wrong, total := map[string]float64{}, map[string]float64{}
	sc := bufio.NewScanner(fh)
	sc.Buffer(make([]byte, 0, 1<<20), 1<<22)
	first := true
	for sc.Scan() {
		f := strings.Split(sc.Text(), "\t")
		if first {
			first = false
			if len(f) < 3 || f[0] != "build" || f[1] != "wrong" || f[2] != "total" {
				t.Fatalf("REFUSING: %s's header is %q, want build/wrong/total — the gate's "+
					"worklist format has moved and this join would be silently wrong", path, sc.Text())
			}
			continue
		}
		if len(f) < 3 {
			continue
		}
		w, err1 := strconv.ParseFloat(f[1], 64)
		tt, err2 := strconv.ParseFloat(f[2], 64)
		if err1 != nil || err2 != nil {
			continue
		}
		wrong[f[0]], total[f[0]] = w, tt
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("REFUSING: reading %s: %v", path, err)
	}
	if len(wrong) == 0 {
		t.Fatalf("REFUSING: %s parsed to 0 builds", path)
	}
	return wrong, total
}
