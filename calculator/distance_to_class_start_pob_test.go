package calculator

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/storage"
)

// TestDistanceToClassStartMatchesPinnedPoB grades DistanceToClassStart against the PINNED PoB itself, not against our other implementation.
func TestDistanceToClassStartMatchesPinnedPoB(t *testing.T) {
	if raw.LatestVersion != "3.29" {
		t.Skipf("SKIP: the reference is recorded against 3.29 and this process loaded %s", raw.LatestVersion)
	}
	if err := poe.InitializeAll(raw.DataContext(t.Context()), raw.LatestVersion, raw.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}

	ref := filepath.Join("..", "testdata", "pob-probes", "node-distance-to-class-start-3.29.tsv")
	f, err := os.Open(ref)
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	defer f.Close()

	want := map[string]map[string]int{}
	order := []string{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		cols := strings.Split(line, "\t")
		if len(cols) != 4 || cols[0] != "DIST" {
			t.Fatalf("REFUSING: %s carries a line this test cannot read: %q", ref, line)
		}
		d, err := strconv.Atoi(cols[3])
		if err != nil {
			t.Fatalf("REFUSING: %s has a non-numeric distance in %q", ref, line)
		}
		if want[cols[1]] == nil {
			want[cols[1]] = map[string]int{}
			order = append(order, cols[1])
		}
		want[cols[1]][cols[2]] = d
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	if len(order) < 6 {
		t.Fatalf("REFUSING: %s covers %d builds; it was recorded over 6", ref, len(order))
	}
	sort.Strings(order)

	for _, name := range order {
		t.Run(name, func(t *testing.T) {
			xml, err := os.ReadFile(filepath.Join(corpusDirForLoadedData(t), name))
			if err != nil {
				t.Fatalf("REFUSING: %v", err)
			}
			build, err := builds.ParseBuild(xml)
			if err != nil {
				t.Fatal(err)
			}
			got := (&Calculator{PoB: build}).BuildOutput(OutputModeMain).Spec.DistanceToClassStart

			if len(got) != len(want[name]) {
				t.Errorf("%s: %d sockets measured, PoB has %d", name, len(got), len(want[name]))
			}
			ids := make([]string, 0, len(want[name]))
			for id := range want[name] {
				ids = append(ids, id)
			}
			sort.Strings(ids)
			for _, id := range ids {
				g, ok := got[id]
				if !ok {
					t.Errorf("%s node %s: absent, PoB says %d", name, id, want[name][id])
					continue
				}
				if g != want[name][id] {
					t.Errorf("%s node %s: %d, PoB says %d", name, id, g, want[name][id])
				}
			}
			for id := range got {
				if _, ok := want[name][id]; !ok {
					t.Errorf("%s node %s: we measure %d, PoB has no allocated socket there", name, id, got[id])
				}
			}
		})
	}
}
