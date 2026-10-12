package calculator

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/storage"
)

// TestModRangeWalkValueArrival: "MEASURED EXACTLY 0" HAS TWO READINGS AND THE PARSE CANNOT SEPARATE THEM: the output is not graded, or the mod never lands.
func TestModRangeWalkValueArrival(t *testing.T) {
	dir := os.Getenv("SAMPLE")
	if dir == "" {
		t.Skip("set SAMPLE=<dir of build XMLs>")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	var files []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".xml") {
			files = append(files, e.Name())
		}
	}
	if len(files) == 0 {
		t.Fatalf("REFUSING: no .xml builds under %s", dir)
	}
	sort.Strings(files)

	if err := poe.InitializeAll(raw.DataContext(t.Context()), raw.LatestVersion, raw.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}

	modKey := func(m mod.Mod) string {
		return m.Name() + "|" + string(m.Type())
	}
	fingerprint := func(i *Item) map[string]float64 {
		out := map[string]float64{}
		i.BuildModList()
		for _, m := range i.Mods {
			v := m.Value()
			if v == nil || v.Type() != mod.ModValueMultiTypeFloat {
				continue
			}
			out[modKey(m)] += v.Float()
		}
		return out
	}

	var (
		itemsAll, itemsEquipped         int
		changedAll, changedEquipped     int
		changedRanged                   int
		sampleLines                     []string
		buildsSeen, buildsWithEquipDiff int

		controlRangesDiffer, controlRangedDiffer int
		rangedLines                              []string
	)

	for _, f := range files {
		xmlBytes, err := os.ReadFile(filepath.Join(dir, f))
		if err != nil {
			t.Fatalf("REFUSING: %v", err)
		}
		build, err := builds.ParseBuild(xmlBytes)
		if err != nil {
			t.Fatalf("REFUSING: %s: %v", f, err)
		}
		buildsSeen++

		equipped := map[int]bool{}
		for _, set := range build.Items.ItemSets {
			if len(build.Items.ItemSets) > 1 && set.ID != strconv.Itoa(build.Items.ActiveItemSet) {
				continue
			}
			for _, sl := range set.Slots {
				equipped[sl.ItemID] = true
			}
		}

		buildDiff := false
		for _, it := range build.Items.Items {
			itemsAll++
			isEquipped := equipped[it.ID]
			if isEquipped {
				itemsEquipped++
			}

			withWalk := ParseItem(it.Text)
			noWalk := &Item{}
			noWalk.ParseRaw(data.SanitiseItemText(xmlEntityDecoder.Replace(it.Text)))

			rangesDiffer, rangedDiffer := false, false
			for _, pair := range [][2][]ModLine{
				{withWalk.EnchantModLines, noWalk.EnchantModLines},
				{withWalk.ScourgeModLines, noWalk.ScourgeModLines},
				{withWalk.ImplicitModLines, noWalk.ImplicitModLines},
				{withWalk.ExplicitModLines, noWalk.ExplicitModLines},
			} {
				for k := range pair[0] {
					if k >= len(pair[1]) {
						break
					}
					if rangeStr(pair[0][k].Range) != rangeStr(pair[1][k].Range) {
						rangesDiffer = true
						if rolledRangeRE.MatchString(pair[0][k].Line) {
							rangedDiffer = true
						}
					}
				}
			}
			if rangesDiffer {
				controlRangesDiffer++
			}
			if rangedDiffer {
				controlRangedDiffer++
				for _, pair := range [][2][]ModLine{
					{withWalk.EnchantModLines, noWalk.EnchantModLines},
					{withWalk.ScourgeModLines, noWalk.ScourgeModLines},
					{withWalk.ImplicitModLines, noWalk.ImplicitModLines},
					{withWalk.ExplicitModLines, noWalk.ExplicitModLines},
				} {
					for k := range pair[0] {
						if k >= len(pair[1]) {
							break
						}
						if rangeStr(pair[0][k].Range) != rangeStr(pair[1][k].Range) && rolledRangeRE.MatchString(pair[0][k].Line) {
							rangedLines = append(rangedLines, fmt.Sprintf(
								"%s item %d %q equipped=%v: %q walk=%s nowalk=%s extra=%q variants=%v",
								f, it.ID, withWalk.Title, isEquipped, pair[0][k].Line,
								rangeStr(pair[0][k].Range), rangeStr(pair[1][k].Range),
								pair[0][k].Extra, pair[0][k].VariantList))
						}
					}
				}
			}

			a := fingerprint(withWalk)
			b := fingerprint(noWalk)
			diff := false
			for k, v := range a {
				if b[k] != v {
					diff = true
					if len(sampleLines) < 25 && isEquipped {
						sampleLines = append(sampleLines,
							fmt.Sprintf("%s item %d %q: %s  walk=%v nowalk=%v", f, it.ID, withWalk.Title, k, v, b[k]))
					}
				}
			}
			for k, v := range b {
				if a[k] != v {
					diff = true
				}
			}
			if diff {
				changedAll++
				if isEquipped {
					changedEquipped++
					buildDiff = true
				}
				if strings.Contains(it.Text, "(") {
					changedRanged++
				}
			}
		}
		if buildDiff {
			buildsWithEquipDiff++
		}
	}

	t.Logf("builds %d", buildsSeen)
	t.Logf("items parsed %d, of which EQUIPPED in the active set %d", itemsAll, itemsEquipped)
	t.Logf("items whose mod VALUES change when the walk is applied: %d (equipped: %d)", changedAll, changedEquipped)
	t.Logf("builds with at least one EQUIPPED item changing:        %d", buildsWithEquipDiff)
	t.Logf("CONTROL -- items whose stored ModLine RANGES differ between the arms: %d", controlRangesDiffer)
	t.Logf("CONTROL -- ... where the differing line carries a (min-max) roll:     %d", controlRangedDiffer)
	for _, l := range rangedLines {
		t.Logf("    %s", l)
	}
	if controlRangesDiffer == 0 {
		t.Fatalf("REFUSING: the two arms are identical upstream of the mods, so the zero below measures the probe, not the walk")
	}
	for _, s := range sampleLines {
		t.Logf("  %s", s)
	}
	if changedAll == 0 {
		t.Logf("VERDICT: the walk is INERT on mod production -- the gate's zero means the mod never changes,")
		t.Logf("         not that the changed mod is ungraded.")
	}
}
