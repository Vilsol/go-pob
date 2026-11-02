package calculator2

import (
	"bytes"
	"context"
	"encoding/json"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"testing"

	"github.com/Vilsol/go-pob/data/poe2skills"
	"github.com/Vilsol/go-pob/pob2"
)

// gemSortOracle is what testdata/gem_sort_oracle.lua writes from PoB-PoE2 for a gem slot of a build's group 1, sorted
// by Combined DPS.
type gemSortOracle struct {
	Base float64 `json:"base"`
	Gems []struct {
		ID         string  `json:"id"`
		CanSupport bool    `json:"canSupport"`
		DPS        float64 `json:"dps"`
	} `json:"gems"`
}

// gemSortCases are the oracle's slots: past Spark, the last gem, and over Spark's support.
var gemSortCases = []struct {
	oracle, fixture string
	slot            int
}{{"gem_sort", "skills_edit", 2}, {"gem_sort_replace", "offence_setup", 2}}

// The gems the gem list calculates, whether each supports the group, and the DPS it sorts by are PoB-PoE2's.
func TestGemSortMatchesPoB(t *testing.T) {
	for _, c := range gemSortCases {
		t.Run(c.oracle, func(t *testing.T) { runGemSort(t, c.oracle, c.fixture, c.slot) })
	}
}

func runGemSort(t *testing.T, name, fixture string, slot int) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", name+"_pob.json"))
	if err != nil {
		t.Fatal(err)
	}
	var oracle gemSortOracle
	if err := json.Unmarshal(raw, &oracle); err != nil {
		t.Fatal(err)
	}
	xml, err := os.ReadFile(filepath.Join("testdata", fixture+".xml"))
	if err != nil {
		t.Fatal(err)
	}
	build, err := pob2.ParseBuild(xml)
	if err != nil {
		t.Fatal(err)
	}
	tree := loadTestTree(t)
	env, err := BuildOutput(build, Options{Tree: tree, Items: NewItemsTab(build, tree, nil)})
	if err != nil {
		t.Fatal(err)
	}
	list, err := GemList()
	if err != nil {
		t.Fatal(err)
	}
	ids := make([]string, len(list))
	for i, g := range list {
		ids[i] = g.ID
	}
	canSupport, pending, err := GemSortCache(env, 1, ids)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]int{}
	for i, g := range oracle.Gems {
		want[g.ID] = i
	}
	var calc []string
	lazy := 0
	for i, id := range ids {
		j, ok := want[id]
		// hasGlobalEffect is also set by a stat map read, so a gem another test calculated first can be calculated
		// here where a fresh PoB-PoE2 does not
		if pending[i] && !ok && !canSupport[i] {
			lazy++
			continue
		}
		if pending[i] != ok {
			t.Errorf("%s: calculated %v, PoB-PoE2 %v", id, pending[i], ok)
		}
		if ok && canSupport[i] != oracle.Gems[j].CanSupport {
			t.Errorf("%s: canSupport %v, PoB-PoE2 %v", id, canSupport[i], oracle.Gems[j].CanSupport)
		}
		if pending[i] {
			calc = append(calc, id)
		}
	}
	misc := NewMiscCalculator(env)
	before := slices.Clone(env.SocketGroupList[0].GemList)
	beforeData := make([]*poe2skills.Gem, 0, len(before))
	for _, g := range before {
		beforeData = append(beforeData, g.GemData)
	}
	base, dps, err := GemSortDPS(build, env, misc, 1, slot, calc, "CombinedDPS")
	if err != nil {
		t.Fatal(err)
	}
	after := env.SocketGroupList[0].GemList
	if len(after) != len(before) {
		t.Fatalf("the sort left %d gems in the group, %d before", len(after), len(before))
	}
	for i, g := range after {
		if g != before[i] || g.GemData != beforeData[i] {
			t.Errorf("gem %d was not put back", i+1)
		}
	}
	if math.Abs(base-oracle.Base) > 1e-9*math.Max(1, math.Abs(oracle.Base)) {
		t.Errorf("base %v, PoB-PoE2 %v", base, oracle.Base)
	}
	wrong := 0
	for i, id := range calc {
		if _, ok := want[id]; !ok {
			continue
		}
		if w := oracle.Gems[want[id]].DPS; math.Abs(dps[i]-w) > 1e-9*math.Max(1, math.Abs(w)) {
			wrong++
			t.Errorf("%s: DPS %v, PoB-PoE2 %v", id, dps[i], w)
		}
	}
	t.Logf("%d gems calculated, %d wrong, %d left out as global effects read since the data loaded", len(calc), wrong, lazy)
}

// The committed oracle is what testdata/gem_sort_oracle.lua writes from the pinned PoB-PoE2; CORPUS=true runs it.
func TestGemSortOracleIsCurrent(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("CORPUS=true runs PoB-PoE2")
	}
	pob, luajit := pobPoE2Probe(t)
	probe, err := filepath.Abs(filepath.Join("testdata", "gem_sort_oracle.lua"))
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range gemSortCases {
		fixture, err := filepath.Abs(filepath.Join("testdata", c.fixture+".xml"))
		if err != nil {
			t.Fatal(err)
		}
		runGemSortOracle(t, pob, luajit, probe, fixture, c.oracle, c.slot)
	}
}

func runGemSortOracle(t *testing.T, pob, luajit, probe, fixture, name string, slot int) {
	t.Helper()
	out := filepath.Join(t.TempDir(), "gem_sort.json")
	cmd := exec.CommandContext(context.Background(), "setarch", "-R", luajit, probe, out, fixture, "1", strconv.Itoa(slot), "CombinedDPS")
	cmd.Dir = filepath.Join(pob, "src")
	cmd.Env = append(os.Environ(),
		"LUA_PATH="+pob+"/runtime/lua/?.lua;"+pob+"/runtime/lua/?/init.lua;"+pob+"/.luarocks/share/lua/5.1/?.lua;"+pob+"/.luarocks/share/lua/5.1/?/init.lua;;",
		"LUA_CPATH="+pob+"/.luarocks/lib/lua/5.1/?.so;;")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("the gem sort oracle failed: %v\n%s", err, stderr.String())
	}
	got, err := os.ReadFile(out) //nolint:gosec // the test's own temp dir
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(filepath.Join("testdata", name+"_pob.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("testdata/%s_pob.json is stale; PoB-PoE2 now writes:\n%s", name, got)
	}
}
