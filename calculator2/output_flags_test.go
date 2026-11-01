package calculator2

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/pob2"
)

// TestOutputFlagsVsPoBPoE2Sidebar compares the boolean outputs Go sets true with the true outputs
// PoB-PoE2's sidebar oracle recorded for the same builds.
func TestOutputFlagsVsPoBPoE2Sidebar(t *testing.T) {
	for _, name := range []string{"sorceress_spark", "sorceress_three_spells"} {
		raw, err := os.ReadFile(filepath.Join("..", "testdata", "oracle", "poe2_sidebar_"+name+".json"))
		testza.AssertNoError(t, err)
		var oracle struct {
			Output map[string]any `json:"output"`
		}
		testza.AssertNoError(t, json.Unmarshal(raw, &oracle))
		var want []string
		for k, v := range oracle.Output {
			if v == true {
				want = append(want, k)
			}
		}
		xml, err := os.ReadFile(filepath.Join("..", "wasm2", "exposition", "testdata", name+".xml"))
		testza.AssertNoError(t, err)
		build, err := pob2.ParseBuild(xml)
		testza.AssertNoError(t, err)
		tree := loadTestTree(t)
		env, err := BuildOutput(build, Options{Tree: tree, Items: NewItemsTab(build, tree, nil)})
		testza.AssertNoError(t, err)
		var got []string
		for k, v := range env.Player.OutputFlags {
			if v {
				got = append(got, k)
			}
		}
		sort.Strings(want)
		sort.Strings(got)
		if len(want) == 0 {
			t.Fatalf("%s: the oracle holds no true output", name)
		}
		testza.AssertEqual(t, want, got, name)
	}
}

// TestOutputStringsVsPoBPoE2Calcs compares the string outputs with the ones PoB-PoE2's Calcs tab oracle
// recorded over the same builds' main calculation, and checks no string key is also a number.
func TestOutputStringsVsPoBPoE2Calcs(t *testing.T) {
	for _, name := range []string{"sorceress_spark", "warrior_dual_mace"} {
		raw, err := os.ReadFile(filepath.Join("..", "testdata", "oracle", "poe2_calcs_"+name+".json"))
		testza.AssertNoError(t, err)
		var oracle struct {
			Output map[string]any `json:"output"`
		}
		testza.AssertNoError(t, json.Unmarshal(raw, &oracle))
		want := map[string]string{}
		for k, v := range oracle.Output {
			if s, ok := v.(string); ok {
				want[k] = s
			}
		}
		if len(want) == 0 {
			t.Fatalf("%s: the oracle holds no string output", name)
		}
		xml, err := os.ReadFile(filepath.Join("..", "wasm2", "exposition", "testdata", name+".xml"))
		testza.AssertNoError(t, err)
		build, err := pob2.ParseBuild(xml)
		testza.AssertNoError(t, err)
		tree := loadTestTree(t)
		env, err := BuildOutput(build, Options{Tree: tree, Items: NewItemsTab(build, tree, nil)})
		testza.AssertNoError(t, err)
		got := map[string]string{}
		for k, v := range env.Player.OutputStrings {
			got[k] = v
			if _, ok := env.Player.Output[k]; ok {
				t.Errorf("%s: %s is both a string and a number", name, k)
			}
		}
		testza.AssertEqual(t, want, got, name)
	}
}
