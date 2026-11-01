package exposition

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

type calcsOracle struct {
	Output  map[string]json.RawMessage `json:"output"`
	Mods    map[string]float64         `json:"mods"`
	Entries map[string]struct {
		ModName   []string `json:"modName"`
		ModType   string   `json:"modType"`
		Cfg       string   `json:"cfg"`
		Enemy     bool     `json:"enemy"`
		ModSource string   `json:"modSource"`
		Actor     string   `json:"actor"`
	} `json:"entries"`
}

// calcsFixtures are the builds oracle/poe2_calcs.lua records (`mise run calc-sections-poe2`): a spell, and an attack
// dual wielding with damage over time.
var calcsFixtures = []string{"sorceress_spark", "warrior_dual_mace"}

func readCalcsFixture(t *testing.T, name string) (calcsOracle, *Calculator) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", "oracle", "poe2_calcs_"+name+".json"))
	if err != nil {
		t.Fatal(err)
	}
	var oracle calcsOracle
	if err := json.Unmarshal(raw, &oracle); err != nil {
		t.Fatal(err)
	}
	xml, err := os.ReadFile(filepath.Join("testdata", name+".xml"))
	if err != nil {
		t.Fatal(err)
	}
	calc, err := ParseBuild(string(xml))
	if err != nil {
		t.Fatal(err)
	}
	return oracle, calc
}

// The values are PoB-PoE2's Calcs tab entries over its main calculation.
func TestCalcsModsMatchPoB(t *testing.T) {
	for _, name := range calcsFixtures {
		t.Run(name, func(t *testing.T) {
			oracle, calc := readCalcsFixture(t, name)
			if _, err := calc.CalcsMods(CalcsModQuery{}); err == nil {
				t.Error("CalcsMods before Calculate: no error")
			}
			if _, err := calc.Calculate(); err != nil {
				t.Fatal(err)
			}
			query := CalcsModQuery{Entries: map[string]CalcsModEntry{}}
			for k, e := range oracle.Entries {
				query.Entries[k] = CalcsModEntry{Cfg: e.Cfg, ModSource: e.ModSource, Enemy: e.Enemy, ModName: e.ModName, ModType: e.ModType, Actor: e.Actor}
			}
			got, err := calc.CalcsMods(query)
			if err != nil {
				t.Fatal(err)
			}
			keys := make([]string, 0, len(query.Entries))
			for k := range query.Entries {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			nonzero, dot := 0, 0
			for _, k := range keys {
				want, inPoB := oracle.Mods[k]
				v, inOurs := got[k]
				if inPoB != inOurs {
					t.Errorf("%s %v: PoB-PoE2 has a value %v, ours %v", k, query.Entries[k].ModName, inPoB, inOurs)
					continue
				}
				if want != 0 {
					nonzero++
					if c := query.Entries[k].Cfg; c != "" && c != "skill" {
						dot++
					}
				}
				assertClose(t, name, k, v, want)
			}
			if nonzero == 0 {
				t.Error("no entry has a nonzero PoB-PoE2 value; the fixture tests nothing")
			}
			if name == "warrior_dual_mace" && dot == 0 {
				t.Error("no damage over time cfg entry has a nonzero PoB-PoE2 value")
			}
		})
	}
}

// PoB-PoE2's true and string outputs and its MainHand and OffHand tables, as the Calcs page reads them.
func TestCalcsOutputsMatchPoB(t *testing.T) {
	for _, name := range calcsFixtures {
		t.Run(name, func(t *testing.T) {
			oracle, calc := readCalcsFixture(t, name)
			if _, err := calc.CalcsOutputs(); err == nil {
				t.Error("CalcsOutputs before Calculate: no error")
			}
			if _, err := calc.Calculate(); err != nil {
				t.Fatal(err)
			}
			got, err := calc.CalcsOutputs()
			if err != nil {
				t.Fatal(err)
			}
			var wantFlags, gotFlags []string
			for k, raw := range oracle.Output {
				if string(raw) == "true" {
					wantFlags = append(wantFlags, k)
				}
			}
			for k := range got.Flags {
				gotFlags = append(gotFlags, k)
			}
			sort.Strings(wantFlags)
			sort.Strings(gotFlags)
			if strings.Join(gotFlags, " ") != strings.Join(wantFlags, " ") {
				t.Errorf("true outputs:\n PoB-PoE2 %v\n ours     %v", wantFlags, gotFlags)
			}
			wantStrings := map[string]string{}
			for k, raw := range oracle.Output {
				var s string
				if json.Unmarshal(raw, &s) == nil {
					wantStrings[k] = s
				}
			}
			if len(wantStrings) == 0 || !reflect.DeepEqual(got.Strings, wantStrings) {
				t.Errorf("string outputs:\n PoB-PoE2 %v\n ours     %v", wantStrings, got.Strings)
			}
			for _, hand := range []struct {
				ns   string
				ours map[string]float64
			}{{"MainHand", got.MainHand}, {"OffHand", got.OffHand}} {
				raw, inPoB := oracle.Output[hand.ns]
				if !inPoB {
					if hand.ours != nil {
						t.Errorf("%s: PoB-PoE2 has none, ours has %d entries", hand.ns, len(hand.ours))
					}
					continue
				}
				var want map[string]any
				if err := json.Unmarshal(raw, &want); err != nil {
					t.Fatal(err)
				}
				nonzero := 0
				for k, w := range want {
					f, ok := w.(float64)
					if !ok {
						continue
					}
					v, has := hand.ours[k]
					if !has {
						if f != 0 {
							t.Errorf("%s.%s: PoB-PoE2 %v, ours absent", hand.ns, k, f)
						}
						continue
					}
					if f != 0 {
						nonzero++
					}
					assertClose(t, name, hand.ns+"."+k, v, f)
				}
				if nonzero == 0 {
					t.Errorf("%s: no nonzero PoB-PoE2 value", hand.ns)
				}
			}
		})
	}
}
